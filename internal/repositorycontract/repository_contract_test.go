package repositorycontract_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/memory"
	storymongo "github.com/premchand/story-builder/internal/mongo"
	"github.com/premchand/story-builder/internal/story"
)

type repositoryFactory struct {
	name string
	new  func(*testing.T) story.Repository
}

func TestRepositoryContract(t *testing.T) {
	factories := []repositoryFactory{{name: "memory", new: func(*testing.T) story.Repository { return memory.NewRepository() }}}
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		factories = append(factories, repositoryFactory{name: "mongo", new: mongoFactory(uri)})
	} else {
		t.Log("Mongo contract skipped: set MONGO_URI to a replica-set MongoDB URI")
	}
	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			testDraftBoundaries(t, factory.new)
			t.Run("completion_failure_retry", func(t *testing.T) { testCompletionFailure(t, factory.new) })
			t.Run("accept_is_idempotent_and_orders_parents", func(t *testing.T) {
				ctx := context.Background()
				svc := story.NewService(factory.new(t))
				st, _ := svc.CreateStory(ctx, "Map")
				parent, _ := svc.CreateScene(ctx, st.ID, "Parent", "", story.Position{})
				child, _ := svc.CreateScene(ctx, st.ID, "Child", "", story.Position{})
				if _, err := svc.CreateEdge(ctx, st.ID, parent.ID, child.ID); err != nil {
					t.Fatal(err)
				}
				childDraft, _ := svc.GenerateDraft(ctx, child.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); !errors.Is(err, story.ErrParentsPending) {
					t.Fatalf("want parents pending, got %v", err)
				}
				parentDraft, _ := svc.GenerateDraft(ctx, parent.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, parentDraft.ID); err != nil {
					t.Fatal(err)
				}
				childDraft, _ = svc.GenerateDraft(ctx, child.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); err != nil {
					t.Fatal(err)
				}
				latest, err := svc.Graph(ctx, st.ID)
				if err != nil || latest.Story.CanonVersion != 2 {
					t.Fatalf("want two canonical events, got %#v, %v", latest.Story, err)
				}
			})
			t.Run("concurrent_opposite_edges_remain_acyclic", func(t *testing.T) {
				ctx := context.Background()
				svc := story.NewService(factory.new(t))
				st, _ := svc.CreateStory(ctx, "Map")
				a, _ := svc.CreateScene(ctx, st.ID, "A", "", story.Position{})
				b, _ := svc.CreateScene(ctx, st.ID, "B", "", story.Position{})
				start := make(chan struct{})
				var wg sync.WaitGroup
				for _, edge := range [][2]string{{a.ID, b.ID}, {b.ID, a.ID}} {
					wg.Add(1)
					go func(from, to string) { defer wg.Done(); <-start; _, _ = svc.CreateEdge(ctx, st.ID, from, to) }(edge[0], edge[1])
				}
				close(start)
				wg.Wait()
				graph, err := svc.Graph(ctx, st.ID)
				if err != nil || story.HasCycle(graph.Edges) || len(graph.Edges) != 1 {
					t.Fatalf("invalid graph: %#v, %v", graph.Edges, err)
				}
			})
		})
	}
}

func mongoFactory(uri string) func(*testing.T) story.Repository {
	return func(t *testing.T) story.Repository {
		t.Helper()
		ctx := context.Background()
		database := "story_builder_contract_" + uuid.NewString()
		client, repo, err := storymongo.Connect(ctx, uri, database)
		if err != nil {
			t.Fatalf("connect Mongo contract database (requires a replica set): %v", err)
		}
		t.Cleanup(func() {
			_ = client.Database(database).Drop(ctx)
			_ = client.Disconnect(ctx)
		})
		return repo
	}
}

func testDraftBoundaries(t *testing.T, factory func(*testing.T) story.Repository) {
	for _, kind := range []string{"stale_scene_update", "repeat_accept_after_graph_change", "edited_scene", "characters_changed", "expired_draft", "failed_generation_retry", "deduplicated_send_error"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			repo := factory(t)
			svc := story.NewService(repo)
			st, err := svc.CreateStory(ctx, "Boundary")
			if err != nil {
				t.Fatal(err)
			}
			sc, err := svc.CreateScene(ctx, st.ID, "Opening", "Before", story.Position{})
			if err != nil {
				t.Fatal(err)
			}
			if kind == "failed_generation_retry" {
				boom := errors.New("stream disconnected")
				if _, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return boom }); !errors.Is(err, boom) {
					t.Fatal(err)
				}
			}
			d, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "stale_scene_update":
				accepted, err := svc.AcceptDraft(ctx, d.ID)
				if err != nil {
					t.Fatal(err)
				}
				sc.Position = story.Position{X: 99, Y: 123}
				if err := repo.UpdateScene(ctx, sc); err != nil {
					t.Fatal(err)
				}
				latest, err := repo.GetScene(ctx, sc.ID)
				if err != nil {
					t.Fatal(err)
				}
				if latest.AcceptedDraftID != accepted.AcceptedDraftID || latest.Prose != accepted.Prose || latest.Position != sc.Position {
					t.Fatalf("scene edit overwrote canon: %+v", latest)
				}
			case "repeat_accept_after_graph_change":
				if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
					t.Fatal(err)
				}
				parent, err := svc.CreateScene(ctx, st.ID, "Late parent", "", story.Position{})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := svc.CreateEdge(ctx, st.ID, parent.ID, sc.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
					t.Fatalf("repeat acceptance lost idempotency: %v", err)
				}
			case "characters_changed":
				if _, err := svc.CreateCharacter(ctx, st.ID, "Mira", []string{"curious"}); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrStaleDraft) {
					t.Fatalf("draft without new character accepted: %v", err)
				}
				refreshed, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if err != nil {
					t.Fatal(err)
				}
				if refreshed.ID == d.ID || len(refreshed.ProposedEvents) != 1 {
					t.Fatal("new character missing from regenerated draft")
				}
				if _, err := svc.AcceptDraft(ctx, refreshed.ID); err != nil {
					t.Fatal(err)
				}
			case "edited_scene":
				brief := "After"
				if _, err := svc.UpdateScene(ctx, sc.ID, nil, &brief, nil); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrStaleDraft) {
					t.Fatalf("edited scene draft accepted: %v", err)
				}
			case "expired_draft":
				d.ExpiresAt = time.Now().Add(-time.Minute)
				if err := repo.UpdateDraft(ctx, d); err != nil {
					t.Fatal(err)
				}
				if _, err := repo.AcceptDraft(ctx, d.ID); !errors.Is(err, story.ErrStaleDraft) {
					t.Fatalf("expired draft accepted: %v", err)
				}
			case "deduplicated_send_error":
				boom := errors.New("stream disconnected")
				if _, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return boom }); !errors.Is(err, boom) {
					t.Fatalf("stream error lost: %v", err)
				}
			default:
				if _, err := svc.AcceptDraft(ctx, d.ID); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Simulate a storage failure after prose has streamed successfully. A retry must
// not remain blocked by the incomplete generation's unique active-draft slot.
type completionFailureRepository struct {
	story.Repository
	fail bool
}

func (r *completionFailureRepository) UpdateDraft(ctx context.Context, d story.Draft) error {
	if r.fail && d.Status == story.DraftReady {
		r.fail = false
		return errors.New("completion write failed")
	}
	return r.Repository.UpdateDraft(ctx, d)
}
func testCompletionFailure(t *testing.T, factory func(*testing.T) story.Repository) {
	ctx := context.Background()
	repo := &completionFailureRepository{Repository: factory(t), fail: true}
	svc := story.NewService(repo)
	st, err := svc.CreateStory(ctx, "Retry")
	if err != nil {
		t.Fatal(err)
	}
	sc, err := svc.CreateScene(ctx, st.ID, "Opening", "", story.Position{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return nil }); err == nil {
		t.Fatal("expected completion failure")
	}
	draft, err := svc.GenerateDraft(ctx, sc.ID, story.TemplateGenerator{}, func(string) error { return nil })
	if err != nil {
		t.Fatalf("retry blocked after completion failure: %v", err)
	}
	if _, err := svc.AcceptDraft(ctx, draft.ID); err != nil {
		t.Fatal(err)
	}
}
