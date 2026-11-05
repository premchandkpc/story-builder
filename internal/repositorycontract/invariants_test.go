package repositorycontract_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/story"
)

type generatorFunc func(context.Context, story.GenerationContext, func(string) error) error

func (generatorFunc) Version() string { return "test-v1" }
func (f generatorFunc) Generate(ctx context.Context, input story.GenerationContext, send func(string) error) error {
	return f(ctx, input, send)
}
func discard(string) error { return nil }
func fixture(t *testing.T, factory repositoryFactory) (story.Repository, *story.Service, story.Story, story.Scene) {
	t.Helper()
	repo := factory.new(t)
	svc := story.NewService(repo)
	ctx := context.Background()
	st, err := svc.CreateStory(ctx, "Map")
	if err != nil {
		t.Fatal(err)
	}
	scene, err := svc.CreateScene(ctx, st.ID, "Opening", "A door opens", story.Position{})
	if err != nil {
		t.Fatal(err)
	}
	return repo, svc, st, scene
}
func draftFor(t *testing.T, svc *story.Service, id string) story.Draft {
	t.Helper()
	d, err := svc.GenerateDraft(context.Background(), id, story.TemplateGenerator{}, discard)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func testInvariants(t *testing.T, factory repositoryFactory) {
	ctx := context.Background()
	t.Run("stale_scene_write_cannot_erase_acceptance", func(t *testing.T) {
		repo, svc, _, scene := fixture(t, factory)
		d := draftFor(t, svc, scene.ID)
		stale, err := repo.GetScene(ctx, scene.ID)
		if err != nil {
			t.Fatal(err)
		}
		accepted, err := svc.AcceptDraft(ctx, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		stale.Position = story.Position{X: 42}
		stale.Revision++
		if err := repo.UpdateScene(ctx, stale); !errors.Is(err, story.ErrConflict) {
			t.Fatalf("want conflict, got %v", err)
		}
		latest, _ := repo.GetScene(ctx, scene.ID)
		if latest.AcceptedDraftID != d.ID || latest.Prose != accepted.Prose {
			t.Fatal("canonical scene was overwritten")
		}
	})
	t.Run("content_edits_invalidate_drafts_but_layout_does_not", func(t *testing.T) {
		_, svc, _, scene := fixture(t, factory)
		d := draftFor(t, svc, scene.ID)
		position := story.Position{X: 45}
		if _, err := svc.UpdateScene(ctx, scene.ID, nil, nil, &position); err != nil {
			t.Fatal(err)
		}
		cached := draftFor(t, svc, scene.ID)
		if cached.ID != d.ID {
			t.Fatal("layout changed narrative cache")
		}
		brief := "The door remains closed"
		if _, err := svc.UpdateScene(ctx, scene.ID, nil, &brief, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrStaleDraft) {
			t.Fatalf("want stale draft, got %v", err)
		}
	})
	t.Run("accepted_content_and_incoming_edges_are_locked", func(t *testing.T) {
		repo, svc, st, scene := fixture(t, factory)
		d := draftFor(t, svc, scene.ID)
		if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
			t.Fatal(err)
		}
		direct, err := repo.GetScene(ctx, scene.ID)
		if err != nil {
			t.Fatal(err)
		}
		direct.Title = "Forbidden edit"
		direct.Revision++
		direct.ContentRevision++
		if err := repo.UpdateScene(ctx, direct); !errors.Is(err, story.ErrSceneLocked) {
			t.Fatalf("persistence accepted narrative edit: %v", err)
		}
		title := "Changed"
		if _, err := svc.UpdateScene(ctx, scene.ID, &title, nil, nil); !errors.Is(err, story.ErrSceneLocked) {
			t.Fatalf("want lock, got %v", err)
		}
		parent, err := svc.CreateScene(ctx, st.ID, "New parent", "", story.Position{})
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.CreateEdge(ctx, story.Edge{ID: "late-parent", StoryID: st.ID, FromSceneID: parent.ID, ToSceneID: scene.ID}); !errors.Is(err, story.ErrSceneLocked) {
			t.Fatalf("want locked graph, got %v", err)
		}
		position := story.Position{Y: 99}
		if _, err := svc.UpdateScene(ctx, scene.ID, nil, nil, &position); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
			t.Fatalf("accept retry failed: %v", err)
		}
		if err := svc.RejectDraft(ctx, d.ID); !errors.Is(err, story.ErrDraftNotReady) {
			t.Fatalf("accepted draft deleted: %v", err)
		}
		persisted, _ := repo.GetDraft(ctx, d.ID)
		if !persisted.ExpiresAt.IsZero() {
			t.Fatal("accepted draft still expires")
		}
	})
	t.Run("expiry_is_checked_before_acceptance", func(t *testing.T) {
		repo, _, st, scene := fixture(t, factory)
		d := story.Draft{ID: "expired", StoryID: st.ID, SceneID: scene.ID, Status: story.DraftReady, ExpiresAt: time.Now().Add(-time.Minute)}
		if err := repo.CreateDraft(ctx, d); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrDraftNotReady) && !errors.Is(err, story.ErrNotFound) {
			t.Fatalf("expired draft accepted: %v", err)
		}
	})
	t.Run("abandoned_generation_recovers_and_old_owner_is_fenced", func(t *testing.T) {
		repo, _, st, scene := fixture(t, factory)
		abandoned := story.Draft{ID: "abandoned", StoryID: st.ID, SceneID: scene.ID, ContextHash: "same", Status: story.DraftGenerating, CreatedAt: time.Now().Add(-time.Hour), LeaseUntil: time.Now().Add(-time.Minute), ExpiresAt: time.Now().Add(time.Hour)}
		if err := repo.CreateDraft(ctx, abandoned); err != nil {
			t.Fatal(err)
		}
		replacement := abandoned
		replacement.ID = "replacement"
		replacement.LeaseUntil = time.Now().Add(time.Minute)
		if err := repo.CreateDraft(ctx, replacement); err != nil {
			t.Fatalf("recovery failed: %v", err)
		}
		abandoned.Status = story.DraftReady
		if err := repo.UpdateDraft(ctx, abandoned); !errors.Is(err, story.ErrDraftNotReady) {
			t.Fatalf("old owner completed: %v", err)
		}
	})
	t.Run("generation_failure_releases_context", func(t *testing.T) {
		repo, svc, _, scene := fixture(t, factory)
		boom := errors.New("stream failed")
		generate := generatorFunc(func(_ context.Context, _ story.GenerationContext, send func(string) error) error { return send("text") })
		failed, err := svc.GenerateDraft(ctx, scene.ID, generate, func(string) error { return boom })
		if !errors.Is(err, boom) {
			t.Fatal(err)
		}
		stored, err := repo.GetDraft(ctx, failed.ID)
		if err != nil || stored.Status != story.DraftFailed {
			t.Fatalf("draft not failed: %#v %v", stored, err)
		}
		if _, err := svc.GenerateDraft(ctx, scene.ID, generate, discard); err != nil {
			t.Fatalf("retry blocked: %v", err)
		}
	})
	t.Run("character_versions_pin_context_and_validate_relationships", func(t *testing.T) {
		repo, svc, st, scene := fixture(t, factory)
		input := story.CharacterInput{Name: "Mira", Traits: []string{"observant"}, CharacterDefinition: story.CharacterDefinition{Role: "investigator", Motivation: "Find her brother", Goals: []string{"Find the key"}, Constraints: []string{"Never reads minds"}, InitialKnowledge: []string{"The door is locked"}, Secrets: []string{"Her brother has the key"}}}
		c, err := svc.DefineCharacter(ctx, st.ID, "", input)
		if err != nil {
			t.Fatal(err)
		}
		d := draftFor(t, svc, scene.ID)
		stored, err := repo.GetDraft(ctx, d.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Input == nil || len(stored.Input.Characters) != 1 || stored.Input.Characters[0].Motivation != input.Motivation {
			t.Fatal("character definition absent from snapshot")
		}
		if len(stored.Input.States) != 1 || len(stored.Input.States[0].Knowledge) != 1 || stored.Input.States[0].Knowledge[0] != input.InitialKnowledge[0] {
			t.Fatal("secret leaked into character knowledge")
		}
		stored.Input.Characters[0].Goals[0] = "mutated outside repository"
		original, err := repo.GetDraft(ctx, d.ID)
		if err != nil || original.Input.Characters[0].Goals[0] != "Find the key" {
			t.Fatal("historical snapshot shares mutable provider state")
		}
		if len(d.ProposedEvents) != 0 {
			t.Fatal("template invented canonical character facts")
		}
		mutating := generatorFunc(func(_ context.Context, input story.GenerationContext, send func(string) error) error {
			input.Characters[0].Goals[0] = "provider mutation"
			return send("A draft")
		})
		mutatedDraft, err := svc.GenerateDraft(ctx, scene.ID, mutating, discard)
		if err != nil {
			t.Fatal(err)
		}
		preserved, err := repo.GetDraft(ctx, mutatedDraft.ID)
		if err != nil || preserved.Input.Characters[0].Goals[0] != "Find the key" {
			t.Fatal("provider altered historical input")
		}
		input.ExpectedVersion = c.Version
		input.Motivation = "Protect her brother"
		next, err := svc.DefineCharacter(ctx, st.ID, c.CharacterID, input)
		if err != nil || next.Version != 2 {
			t.Fatalf("version update: %#v %v", next, err)
		}
		if _, err := svc.DefineCharacter(ctx, st.ID, c.CharacterID, input); !errors.Is(err, story.ErrConflict) {
			t.Fatalf("stale character write succeeded: %v", err)
		}
		if _, err := svc.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrStaleDraft) {
			t.Fatalf("old character snapshot accepted: %v", err)
		}
		chars, err := svc.ListCharacters(ctx, st.ID)
		if err != nil || len(chars) != 1 || chars[0].Version != 2 {
			t.Fatal("latest definitions not selected")
		}
		other, err := svc.CreateStory(ctx, "Other")
		if err != nil {
			t.Fatal(err)
		}
		invalid := story.CharacterInput{Name: "Intruder", CharacterDefinition: story.CharacterDefinition{Relationships: []story.Relationship{{CharacterID: c.CharacterID, Kind: "rival"}}}}
		if _, err := svc.DefineCharacter(ctx, other.ID, "", invalid); !errors.Is(err, story.ErrInvalidCharacter) {
			t.Fatalf("cross-story relationship accepted: %v", err)
		}
		invalid.Relationships[0].CharacterID = c.CharacterID
		if _, err := svc.DefineCharacter(ctx, st.ID, "", invalid); err != nil {
			t.Fatalf("valid relationship rejected: %v", err)
		}
	})
	t.Run("concurrent_character_edits_preserve_one_version", func(t *testing.T) {
		_, svc, st, _ := fixture(t, factory)
		c, err := svc.CreateCharacter(ctx, st.ID, "Mira", nil)
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, err := svc.DefineCharacter(ctx, st.ID, c.CharacterID, story.CharacterInput{Name: fmt.Sprintf("Mira %d", i), ExpectedVersion: 1})
				results <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflicts := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if errors.Is(err, story.ErrConflict) {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflicts != 1 {
			t.Fatalf("success=%d conflicts=%d", success, conflicts)
		}
	})
}

type failReadyRepository struct {
	story.Repository
	failed bool
}

func (r *failReadyRepository) UpdateDraft(ctx context.Context, value story.Draft) error {
	if value.Status == story.DraftReady && !r.failed {
		r.failed = true
		return errors.New("injected completion failure")
	}
	return r.Repository.UpdateDraft(ctx, value)
}
func testGenerationFailures(t *testing.T, factory repositoryFactory) {
	ctx := context.Background()
	t.Run("completion_write_failure_marks_failed_and_allows_retry", func(t *testing.T) {
		repo, _, _, scene := fixture(t, factory)
		svc := story.NewService(&failReadyRepository{Repository: repo})
		if _, err := svc.GenerateDraft(ctx, scene.ID, story.TemplateGenerator{}, discard); err == nil {
			t.Fatal("expected completion failure")
		}
		if _, err := svc.GenerateDraft(ctx, scene.ID, story.TemplateGenerator{}, discard); err != nil {
			t.Fatalf("retry blocked: %v", err)
		}
	})
	t.Run("generation_gets_parent_prose_and_supports_cancellation", func(t *testing.T) {
		_, svc, st, parent := fixture(t, factory)
		parentDraft := draftFor(t, svc, parent.ID)
		if _, err := svc.AcceptDraft(ctx, parentDraft.ID); err != nil {
			t.Fatal(err)
		}
		child, err := svc.CreateScene(ctx, st.ID, "Child", "", story.Position{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateEdge(ctx, st.ID, parent.ID, child.ID); err != nil {
			t.Fatal(err)
		}
		started := make(chan story.GenerationContext, 1)
		finished := make(chan error, 1)
		cancelCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		generator := generatorFunc(func(ctx context.Context, input story.GenerationContext, _ func(string) error) error {
			started <- input
			<-ctx.Done()
			return ctx.Err()
		})
		go func() { _, err := svc.GenerateDraft(cancelCtx, child.ID, generator, discard); finished <- err }()
		var input story.GenerationContext
		select {
		case input = <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("generation did not start")
		}
		if len(input.Parents) != 1 || input.Parents[0].Prose != parentDraft.Prose {
			t.Fatal("parent prose not captured")
		}
		if _, err := svc.GenerateDraft(ctx, child.ID, generator, discard); !errors.Is(err, story.ErrDraftInProgress) {
			t.Fatalf("overlap not rejected: %v", err)
		}
		cancel()
		select {
		case err := <-finished:
			if !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cancel did not finish")
		}
		if _, err := svc.GenerateDraft(ctx, child.ID, story.TemplateGenerator{}, discard); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("edge_and_accept_race_cannot_break_parent_invariant", func(t *testing.T) {
		repo, svc, st, child := fixture(t, factory)
		parent, err := svc.CreateScene(ctx, st.ID, "Parent", "", story.Position{})
		if err != nil {
			t.Fatal(err)
		}
		d := draftFor(t, svc, child.ID)
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); <-start; _, _ = svc.AcceptDraft(ctx, d.ID) }()
		go func() { defer wg.Done(); <-start; _, _ = svc.CreateEdge(ctx, st.ID, parent.ID, child.ID) }()
		close(start)
		wg.Wait()
		latest, err := repo.GetScene(ctx, child.ID)
		if err != nil {
			t.Fatal(err)
		}
		edges, err := repo.ListEdges(ctx, st.ID)
		if err != nil {
			t.Fatal(err)
		}
		if latest.AcceptedDraftID != "" && len(edges) > 0 {
			t.Fatal("accepted child has pending parent")
		}
	})
}
