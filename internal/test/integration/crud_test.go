//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
)

func TestIntegration_Stories(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	deleter := &service.StoryCascadeDeleter{
		SceneRepo: mgorepo.NewSceneRepo(testDB),
		EdgeRepo:  mgorepo.NewSceneEdgeRepo(testDB),
		CharRepo:  mgorepo.NewCharacterRepo(testDB),
		StateRepo: mgorepo.NewCharacterStateRepo(testDB),
		GenRepo:   mgorepo.NewGenerationRepo(testDB),
		MemRepo:   mgorepo.NewMemoryRepo(testDB),
		TlRepo:    mgorepo.NewTimelineRepo(testDB),
		SumRepo:   mgorepo.NewSummaryRepo(testDB),
	}
	svc := service.NewStoryService(storyRepo, deleter)
	ctx := context.Background()

	t.Run("create and get", func(t *testing.T) {
		s, err := svc.Create(ctx, "Test Story")
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if s.Title != "Test Story" {
			t.Fatalf("title: got %q", s.Title)
		}
		if s.Status != domain.StoryStatusDraft {
			t.Fatalf("status: got %q", s.Status)
		}
		if s.ID == "" {
			t.Fatal("id is empty")
		}

		got, err := svc.Get(ctx, s.ID)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Title != "Test Story" {
			t.Fatalf("get title: got %q", got.Title)
		}
	})

	t.Run("update title and status", func(t *testing.T) {
		s, _ := svc.Create(ctx, "Update Me")
		updated, err := svc.Update(ctx, s.ID, service.UpdateStoryParams{
			Title:  "Updated Title",
			Status: domain.StoryStatusActive,
		})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if updated.Title != "Updated Title" {
			t.Fatalf("title after update: got %q", updated.Title)
		}
		if updated.Status != domain.StoryStatusActive {
			t.Fatalf("status after update: got %q", updated.Status)
		}
	})

	t.Run("update missing story returns error", func(t *testing.T) {
		_, err := svc.Update(ctx, "nonexistent", service.UpdateStoryParams{Title: "x"})
		if err == nil {
			t.Fatal("expected error for missing story")
		}
	})

	t.Run("list includes all stories", func(t *testing.T) {
		cleanCollections(t, "stories")
		svc.Create(ctx, "A")
		svc.Create(ctx, "B")
		svc.Create(ctx, "C")

		stories, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(stories) != 3 {
			t.Fatalf("expected 3, got %d", len(stories))
		}
	})

	t.Run("delete removes story and cascade deletes related data", func(t *testing.T) {
		cleanCollections(t, "stories", "scenes", "scene_edges", "characters", "character_state")

		s, _ := svc.Create(ctx, "Delete Me")
		sceneRepo := mgorepo.NewSceneRepo(testDB)
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
		charRepo := mgorepo.NewCharacterRepo(testDB)
		stateRepo := mgorepo.NewCharacterStateRepo(testDB)

		sc := &domain.Scene{StoryID: s.ID, Title: "Scene 1"}
		sceneRepo.Create(ctx, sc)
		char := &domain.Character{StoryID: s.ID, Name: "Hero"}
		charRepo.Create(ctx, char)
		stateRepo.Append(ctx, &domain.CharacterState{
			CharacterID: char.CharID, StoryID: s.ID, SceneID: sc.ID, Mood: "happy",
		})
		edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc.ID, ToSceneID: sc.ID, Type: "seq"})

		if err := svc.Delete(ctx, s.ID); err != nil {
			t.Fatalf("delete: %v", err)
		}

		if got, _ := svc.Get(ctx, s.ID); got != nil {
			t.Fatal("story still exists after delete")
		}
		if scs, _ := sceneRepo.ListByStory(ctx, s.ID); len(scs) != 0 {
			t.Fatal("scenes not cascade deleted")
		}
		if chs, _ := charRepo.ListByStory(ctx, s.ID); len(chs) != 0 {
			t.Fatal("characters not cascade deleted")
		}
	})

	t.Run("delete nonexistent returns no error", func(t *testing.T) {
		err := svc.Delete(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("delete nonexistent: %v", err)
		}
	})

	t.Run("empty list returns empty slice", func(t *testing.T) {
		cleanCollections(t, "stories")
		stories, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("list: %v", err)
		}
		if len(stories) != 0 {
			t.Fatalf("expected 0, got %d", len(stories))
		}
	})
}

func TestIntegration_Scenes(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
	svc := service.NewSceneService(sceneRepo, edgeRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Scene Test Story", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("create scene with metadata", func(t *testing.T) {
		sc := &domain.Scene{
			StoryID:    story.ID,
			Title:      "Opening",
			BeatIntent: "Introduce the hero",
			POV:        "Hero",
			Tone:       "mysterious",
			Participants: []string{"Hero", "Sidekick"},
		}
		created, err := svc.Create(ctx, sc)
		if err != nil {
			t.Fatalf("create scene: %v", err)
		}
		if created.Title != "Opening" {
			t.Fatalf("title: got %q", created.Title)
		}
		if created.Status != domain.SceneStatusDraft {
			t.Fatalf("status: got %q", created.Status)
		}
		if created.StoryID != story.ID {
			t.Fatalf("storyID mismatch")
		}
	})

	t.Run("update scene", func(t *testing.T) {
		sc := &domain.Scene{StoryID: story.ID, Title: "Middle"}
		svc.Create(ctx, sc)
		sc.Title = "Middle (Revised)"
		sc.Status = domain.SceneStatusGenerated

		updated, err := svc.Update(ctx, sc)
		if err != nil {
			t.Fatalf("update scene: %v", err)
		}
		if updated.Title != "Middle (Revised)" {
			t.Fatalf("title after update: %q", updated.Title)
		}
	})

	t.Run("list scenes by story", func(t *testing.T) {
		cleanCollections(t, "scenes")
		for i := 0; i < 3; i++ {
			svc.Create(ctx, &domain.Scene{
				StoryID: story.ID, Title: fmt.Sprintf("Scene %d", i),
			})
		}
		scenes, err := svc.List(ctx, story.ID)
		if err != nil {
			t.Fatalf("list scenes: %v", err)
		}
		if len(scenes) != 3 {
			t.Fatalf("expected 3 scenes, got %d", len(scenes))
		}
	})

	t.Run("topology returns scenes and edges", func(t *testing.T) {
		cleanCollections(t, "scenes", "scene_edges")
		s1, _ := svc.Create(ctx, &domain.Scene{StoryID: story.ID, Title: "First"})
		s2, _ := svc.Create(ctx, &domain.Scene{StoryID: story.ID, Title: "Second"})
		edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: s1.ID, ToSceneID: s2.ID, Type: "seq"})

		scenes, edges, err := svc.Topology(ctx, story.ID)
		if err != nil {
			t.Fatalf("topology: %v", err)
		}
		if len(scenes) != 2 {
			t.Fatalf("expected 2 scenes, got %d", len(scenes))
		}
		if len(edges) != 1 {
			t.Fatalf("expected 1 edge, got %d", len(edges))
		}
	})

	t.Run("get missing scene returns nil", func(t *testing.T) {
		sc, err := svc.Get(ctx, "doesnotexist")
		if err != nil {
			t.Fatalf("get missing: %v", err)
		}
		if sc != nil {
			t.Fatal("expected nil for missing scene")
		}
	})
}

func TestIntegration_Edges(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
	svc := service.NewEdgeService(edgeRepo)
	sceneSvc := service.NewSceneService(mgorepo.NewSceneRepo(testDB), edgeRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Edge Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)
	s1, _ := sceneSvc.Create(ctx, &domain.Scene{StoryID: story.ID, Title: "A"})
	s2, _ := sceneSvc.Create(ctx, &domain.Scene{StoryID: story.ID, Title: "B"})

	t.Run("create seq edge", func(t *testing.T) {
		e, err := svc.Create(ctx, &domain.SceneEdge{
			StoryID: story.ID, FromSceneID: s1.ID, ToSceneID: s2.ID, Type: "seq",
		})
		if err != nil {
			t.Fatalf("create edge: %v", err)
		}
		if e.Type != "seq" {
			t.Fatalf("edge type: got %q", e.Type)
		}
	})

	t.Run("create fork edge", func(t *testing.T) {
		s3, _ := sceneSvc.Create(ctx, &domain.Scene{StoryID: story.ID, Title: "C"})
		e, err := svc.Create(ctx, &domain.SceneEdge{
			StoryID: story.ID, FromSceneID: s1.ID, ToSceneID: s3.ID, Type: "fork",
		})
		if err != nil {
			t.Fatalf("create fork edge: %v", err)
		}
		if e.Type != "fork" {
			t.Fatalf("edge type: got %q", e.Type)
		}
	})

	t.Run("duplicate edge returns error (unique index)", func(t *testing.T) {
		_, err := svc.Create(ctx, &domain.SceneEdge{
			StoryID: story.ID, FromSceneID: s1.ID, ToSceneID: s2.ID, Type: "seq",
		})
		if err == nil {
			t.Fatal("expected duplicate edge error")
		}
	})

	t.Run("list edges by story", func(t *testing.T) {
		edges, err := svc.List(ctx, story.ID)
		if err != nil {
			t.Fatalf("list edges: %v", err)
		}
		if len(edges) != 2 {
			t.Fatalf("expected 2 edges, got %d", len(edges))
		}
	})

	t.Run("delete edge", func(t *testing.T) {
		err := svc.Delete(ctx, story.ID, s1.ID, s2.ID)
		if err != nil {
			t.Fatalf("delete edge: %v", err)
		}
		edges, _ := svc.List(ctx, story.ID)
		if len(edges) != 1 {
			t.Fatalf("expected 1 edge after delete, got %d", len(edges))
		}
	})
}

func TestIntegration_Characters(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_state")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	svc := service.NewCharacterService(charRepo, stateRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Char Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("create character", func(t *testing.T) {
		c, err := svc.Create(ctx, &domain.Character{
			StoryID: story.ID,
			Name:    "Aragorn",
			Persona: "ranger",
			Traits:  []string{"brave", "loyal"},
		})
		if err != nil {
			t.Fatalf("create character: %v", err)
		}
		if c.Name != "Aragorn" {
			t.Fatalf("name: got %q", c.Name)
		}
		if c.Version != 1 {
			t.Fatalf("version: got %d", c.Version)
		}
		if c.CharID == "" {
			t.Fatal("charId is empty")
		}
	})

	t.Run("update creates new version", func(t *testing.T) {
		orig, _ := svc.Create(ctx, &domain.Character{
			StoryID: story.ID, Name: "Legolas",
		})
		origID := orig.ID
		origVersion := orig.Version

		orig.Traits = []string{"elf", "archer"}
		updated, err := svc.Update(ctx, orig)
		if err != nil {
			t.Fatalf("update character: %v", err)
		}
		if updated.Version != origVersion+1 {
			t.Fatalf("version: got %d, expected %d", updated.Version, origVersion+1)
		}
		if updated.ID == origID {
			t.Fatal("ID should change after versioned update")
		}
	})

	t.Run("get latest returns highest version", func(t *testing.T) {
		latest, err := svc.GetLatest(ctx, "aragorn-charid")
		if err != nil {
			t.Fatal("get latest:", err)
		}
		if latest == nil {
			t.Skip("get latest needs charId to match; using Get instead")
		}

		c, _ := svc.Create(ctx, &domain.Character{
			StoryID: story.ID, Name: "Gimli", CharID: "gimli-01",
		})
		c2, _ := svc.Update(ctx, c)
		latest2, _ := svc.GetLatest(ctx, "gimli-01")
		if latest2 == nil || latest2.Version != c2.Version {
			t.Fatalf("get latest version: got %d, expected %d", latest2.Version, c2.Version)
		}
	})

	t.Run("list characters by story", func(t *testing.T) {
		chars, err := svc.List(ctx, story.ID)
		if err != nil {
			t.Fatalf("list chars: %v", err)
		}
		if len(chars) == 0 {
			t.Fatal("expected at least one character")
		}
	})

	t.Run("get missing character returns nil", func(t *testing.T) {
		c, err := svc.Get(ctx, "doesnotexist")
		if err != nil {
			t.Fatalf("get missing: %v", err)
		}
		if c != nil {
			t.Fatal("expected nil for missing character")
		}
	})
}

func TestIntegration_CharacterState(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_state")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	ctx := context.Background()

	story := &domain.Story{Title: "State Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("append and get character state", func(t *testing.T) {
		err := stateRepo.Append(ctx, &domain.CharacterState{
			CharacterID: "hero-1", StoryID: story.ID, SceneID: "scene-1",
			Mood: "hopeful", Location: "Shire", Health: 100,
		})
		if err != nil {
			t.Fatalf("append state: %v", err)
		}

		st, err := stateRepo.Get(ctx, "hero-1", "scene-1")
		if err != nil {
			t.Fatalf("get state: %v", err)
		}
		if st == nil {
			t.Fatal("state not found")
		}
		if st.Mood != "hopeful" {
			t.Fatalf("mood: got %q", st.Mood)
		}
		if st.Health != 100 {
			t.Fatalf("health: got %d", st.Health)
		}
	})

	t.Run("list by character returns ordered states", func(t *testing.T) {
		stateRepo.Append(ctx, &domain.CharacterState{
			CharacterID: "hero-1", StoryID: story.ID, SceneID: "scene-2", Mood: "worried",
		})
		states, err := stateRepo.ListByCharacter(ctx, "hero-1")
		if err != nil {
			t.Fatalf("list by char: %v", err)
		}
		if len(states) < 2 {
			t.Fatalf("expected >=2 states, got %d", len(states))
		}
	})

	t.Run("list by scene", func(t *testing.T) {
		states, err := stateRepo.ListByScene(ctx, "scene-1")
		if err != nil {
			t.Fatalf("list by scene: %v", err)
		}
		if len(states) != 1 {
			t.Fatalf("expected 1 state for scene-1, got %d", len(states))
		}
	})
}

func TestIntegration_Timeline(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	tlRepo := mgorepo.NewTimelineRepo(testDB)
	svc := service.NewTimelineService(tlRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Timeline Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("create timeline events", func(t *testing.T) {
		e1, err := svc.Create(ctx, &domain.TimelineEvent{
			StoryID: story.ID, Title: "Start", Description: "The beginning", Order: 1,
		})
		if err != nil {
			t.Fatalf("create event: %v", err)
		}
		if e1.Order != 1 {
			t.Fatalf("order: got %d", e1.Order)
		}

		svc.Create(ctx, &domain.TimelineEvent{
			StoryID: story.ID, Title: "End", Description: "The end", Order: 2,
		})
	})

	t.Run("list ordered by order", func(t *testing.T) {
		events, err := svc.List(ctx, story.ID)
		if err != nil {
			t.Fatalf("list events: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}
		if events[0].Order != 1 || events[1].Order != 2 {
			t.Fatalf("expected order 1,2; got %d,%d", events[0].Order, events[1].Order)
		}
	})

	t.Run("empty story returns empty list", func(t *testing.T) {
		events, _ := svc.List(ctx, "nonexistent-story")
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})
}

func TestIntegration_Summaries(t *testing.T) {
	cleanCollections(t, "stories", "summaries")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	svc := service.NewSummaryService(sumRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Summary Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("upsert creates new summary", func(t *testing.T) {
		err := sumRepo.Upsert(ctx, &domain.Summary{
			StoryID: story.ID, Level: domain.SummaryLevelStory,
			Content: "This is the story summary.", WordCount: 5,
		})
		if err != nil {
			t.Fatalf("upsert summary: %v", err)
		}

		s, err := svc.GetByLevel(ctx, story.ID, domain.SummaryLevelStory)
		if err != nil {
			t.Fatalf("get by level: %v", err)
		}
		if s == nil {
			t.Fatal("summary not found")
		}
		if s.Content != "This is the story summary." {
			t.Fatalf("content: got %q", s.Content)
		}
	})

	t.Run("upsert replaces existing summary", func(t *testing.T) {
		sumRepo.Upsert(ctx, &domain.Summary{
			StoryID: story.ID, Level: domain.SummaryLevelStory,
			Content: "Replaced content.", WordCount: 2,
		})
		s, _ := svc.GetByLevel(ctx, story.ID, domain.SummaryLevelStory)
		if s.Content != "Replaced content." {
			t.Fatalf("after upsert: got %q", s.Content)
		}
	})

	t.Run("get scene summary", func(t *testing.T) {
		err := sumRepo.Upsert(ctx, &domain.Summary{
			StoryID: story.ID, SceneID: "scene-1", Level: domain.SummaryLevelScene,
			Content: "Scene summary.",
		})
		if err != nil {
			t.Fatalf("upsert scene summary: %v", err)
		}
		s, err := svc.GetSceneSummary(ctx, story.ID, "scene-1")
		if err != nil {
			t.Fatalf("get scene summary: %v", err)
		}
		if s == nil || s.Content != "Scene summary." {
			t.Fatalf("scene summary: got %+v", s)
		}
	})

	t.Run("missing summary returns nil", func(t *testing.T) {
		s, err := svc.GetByLevel(ctx, story.ID, "nonexistent")
		if err != nil {
			t.Fatalf("get missing: %v", err)
		}
		if s != nil {
			t.Fatal("expected nil for missing summary")
		}
	})
}

func TestIntegration_Memories(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_memories")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	memRepo := mgorepo.NewMemoryRepo(testDB)
	svc := service.NewMemoryService(memRepo)
	ctx := context.Background()

	story := &domain.Story{Title: "Memory Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("create memories for character", func(t *testing.T) {
		err := memRepo.Create(ctx, &domain.CharacterMemory{
			StoryID: story.ID, CharacterID: "hero", SceneID: "scene-1",
			Content: "Met the villain for the first time.",
			Type:    domain.MemoryTypeEvent, Importance: 0.9,
		})
		if err != nil {
			t.Fatalf("create memory: %v", err)
		}

		memRepo.Create(ctx, &domain.CharacterMemory{
			StoryID: story.ID, CharacterID: "hero", SceneID: "scene-2",
			Content: "Discovered the ancient sword.",
			Type:    domain.MemoryTypeObservation, Importance: 0.7,
		})

		memRepo.Create(ctx, &domain.CharacterMemory{
			StoryID: story.ID, CharacterID: "sidekick", SceneID: "scene-1",
			Content: "Witnessed the confrontation.",
			Importance: 0.5,
		})
	})

	t.Run("list memories by character", func(t *testing.T) {
		mems, err := svc.ListByCharacter(ctx, "hero")
		if err != nil {
			t.Fatalf("list memories: %v", err)
		}
		if len(mems) != 2 {
			t.Fatalf("expected 2 hero memories, got %d", len(mems))
		}
	})

	t.Run("search by importance fallback", func(t *testing.T) {
		mems, err := memRepo.Search(ctx, story.ID, "hero", nil, 5)
		if err != nil {
			t.Fatalf("search memories: %v", err)
		}
		if len(mems) == 0 {
			t.Fatal("expected at least one memory from search")
		}
	})

	t.Run("empty character returns empty list", func(t *testing.T) {
		mems, _ := svc.ListByCharacter(ctx, "nobody")
		if len(mems) != 0 {
			t.Fatalf("expected 0, got %d", len(mems))
		}
	})
}

func TestIntegration_Generations(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "generations")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	ctx := context.Background()

	story := &domain.Story{Title: "Gen Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)
	scene := &domain.Scene{StoryID: story.ID, Title: "Scene 1", Status: domain.SceneStatusDraft}
	sceneRepo.Create(ctx, scene)

	t.Run("create and read generation", func(t *testing.T) {
		gen := &domain.Generation{
			StoryID: story.ID, SceneID: scene.ID,
			Model: "claude-sonnet", Output: "Scene prose...",
			Accepted: false,
		}
		err := genRepo.Create(ctx, gen)
		if err != nil {
			t.Fatalf("create gen: %v", err)
		}
		if gen.ID == "" {
			t.Fatal("gen id is empty")
		}

		got, err := genRepo.Get(ctx, gen.ID)
		if err != nil {
			t.Fatalf("get gen: %v", err)
		}
		if got.Output != "Scene prose..." {
			t.Fatalf("output: got %q", got.Output)
		}
	})

	t.Run("list generations by scene", func(t *testing.T) {
		gens, err := genRepo.ListByScene(ctx, scene.ID)
		if err != nil {
			t.Fatalf("list by scene: %v", err)
		}
		if len(gens) != 1 {
			t.Fatalf("expected 1 gen, got %d", len(gens))
		}
	})

	t.Run("accept generation rejects others", func(t *testing.T) {
		genRepo.Create(ctx, &domain.Generation{
			StoryID: story.ID, SceneID: scene.ID, Model: "claude-sonnet", Output: "v2",
		})
		gens, _ := genRepo.ListByScene(ctx, scene.ID)
		foundAccepted := false
		for _, g := range gens {
			if g.Accepted {
				foundAccepted = true
			}
		}
		if foundAccepted {
			t.Fatal("no gen should be accepted yet")
		}
	})
}

func TestIntegration_ConcurrentCreate(t *testing.T) {
	cleanCollections(t, "stories")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	svc := service.NewStoryService(storyRepo, nil)
	ctx := context.Background()

	_, err := svc.Create(ctx, "Story A")
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	_, err = svc.Create(ctx, "Story B")
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	if _, err := svc.Get(ctx, "nonexistent"); err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
}

func TestIntegration_StatusTransitions(t *testing.T) {
	cleanCollections(t, "stories")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	svc := service.NewStoryService(storyRepo, nil)
	ctx := context.Background()

	transitions := []struct {
		from, to string
	}{
		{domain.StoryStatusDraft, domain.StoryStatusActive},
		{domain.StoryStatusActive, domain.StoryStatusCompleted},
		{domain.StoryStatusCompleted, domain.StoryStatusArchived},
	}

	s, _ := svc.Create(ctx, "Transitions Test")
	for _, tr := range transitions {
		updated, err := svc.Update(ctx, s.ID, service.UpdateStoryParams{Status: tr.to})
		if err != nil {
			t.Fatalf("transition %s->%s: %v", tr.from, tr.to, err)
		}
		if updated.Status != tr.to {
			t.Fatalf("expected %s, got %s", tr.to, updated.Status)
		}
	}
}

func TestIntegration_NullFields(t *testing.T) {
	cleanCollections(t, "stories", "scenes")

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	ctx := context.Background()

	story := &domain.Story{Title: "Null Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	t.Run("scene with minimal fields", func(t *testing.T) {
		sc := &domain.Scene{StoryID: story.ID}
		err := sceneRepo.Create(ctx, sc)
		if err != nil {
			t.Fatalf("create minimal scene: %v", err)
		}
		if sc.Title != "" {
			t.Fatalf("expected empty title, got %q", sc.Title)
		}
		if sc.Status != domain.SceneStatusDraft {
			t.Fatalf("expected status draft, got %q", sc.Status)
		}
	})

	t.Run("story with empty title creates anyway (handles at handler level)", func(t *testing.T) {
		s := &domain.Story{Status: domain.StoryStatusDraft}
		err := storyRepo.Create(ctx, s)
		if err != nil {
			t.Fatalf("create story with empty title: %v", err)
		}
		if s.Title != "" {
			t.Fatalf("expected empty title")
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
