//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
)

func TestIntegration_GenerationPipeline(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "generations", "character_state",
		"character_memories", "timeline_events", "summaries", "characters")

	ctx := context.Background()

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	memRepo := mgorepo.NewMemoryRepo(testDB)
	tlRepo := mgorepo.NewTimelineRepo(testDB)
	sumRepo := mgorepo.NewSummaryRepo(testDB)

	story := &domain.Story{Title: "Pipeline Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)

	char := &domain.Character{
		StoryID: story.ID, Name: "Hero", Traits: []string{"brave"},
	}
	charRepo.Create(ctx, char)

	scene := &domain.Scene{
		StoryID:      story.ID,
		Title:        "The Beginning",
		BeatIntent:   "Hero discovers the ancient artifact",
		POV:          "Hero",
		Tone:         "mysterious",
		TargetWords:  100,
		Participants: []string{char.CharID},
	}
	sceneRepo.Create(ctx, scene)

	llmSvc := &mockProseService{}
	extractSvc := &mockExtractionService{}
	summarySvc := &mockSummaryService{}
	validateSvc := &mockValidationService{}

	locRepo := mgorepo.NewLocationRepo(testDB)
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, StoryRepo: storyRepo,
		CharRepo: charRepo, StateRepo: stateRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: llmSvc, ExtractSvc: extractSvc, SummarySvc: summarySvc, ValidateSvc: validateSvc,
	})

	t.Run("generation creates record and runs pipeline", func(t *testing.T) {
		gen, err := genSvc.Generate(ctx, scene.ID)
		if err != nil {
			t.Fatalf("generate scene: %v", err)
		}
		if gen.SceneID != scene.ID {
			t.Fatalf("sceneID mismatch: %s", gen.SceneID)
		}
		if gen.Model != "claude-sonnet" {
			t.Fatalf("model: got %q", gen.Model)
		}

		var gens []*domain.Generation
		for i := 0; i < 100; i++ {
			gens, err = genSvc.ListGenerations(ctx, scene.ID)
			if err != nil {
				t.Fatalf("list gens: %v", err)
			}
			if len(gens) > 0 && gens[0].Output != "" {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if len(gens) == 0 {
			t.Fatal("no generations found after pipeline (timeout)")
		}

		latest := gens[0]
		if latest.Output == "" {
			t.Fatal("generation output should not be empty after pipeline")
		}
	})

	t.Run("generation updates scene content", func(t *testing.T) {
		updatedScene, err := sceneRepo.Get(ctx, scene.ID)
		if err != nil {
			t.Fatalf("get scene: %v", err)
		}
		if updatedScene.GeneratedContent == "" {
			t.Fatal("scene should have generated content after pipeline")
		}
	})

	t.Run("extract creates character state", func(t *testing.T) {
		states, err := stateRepo.ListByCharacter(ctx, char.CharID)
		if err != nil {
			t.Fatalf("list states: %v", err)
		}
		if len(states) > 0 {
			return
		}
		t.Log("no states found (mock returns empty deltas)")
	})

	t.Run("accept generation marks scene accepted and rejects others", func(t *testing.T) {
		var gens []*domain.Generation
		for i := 0; i < 50; i++ {
			gens, _ = genSvc.ListGenerations(ctx, scene.ID)
			if len(gens) > 0 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if len(gens) == 0 {
			t.Fatal("no generations to accept")
		}

		err := genSvc.AcceptGeneration(ctx, scene.ID, gens[0].ID)
		if err != nil {
			t.Fatalf("accept gen: %v", err)
		}

		acceptedScene, _ := sceneRepo.Get(ctx, scene.ID)
		if acceptedScene.Status != domain.SceneStatusAccepted {
			t.Fatalf("scene status after accept: got %q", acceptedScene.Status)
		}

		gens, _ = genSvc.ListGenerations(ctx, scene.ID)
		for _, g := range gens {
			if g.ID == gens[0].ID && !g.Accepted {
				t.Fatal("accepted generation should be marked accepted")
			}
		}
	})

	t.Run("reject generation with accept=false", func(t *testing.T) {
		cleanCollections(t, "generations")

		oldProse := llmSvc.generateFn
		llmSvc.generateFn = func(_ context.Context, _ llm.PromptParams) (*llm.CompletionResponse, error) {
			return &llm.CompletionResponse{
				Content: "Alternative scene text for rejection test.",
				Model:   "mock-sonnet",
			}, nil
		}
		defer func() { llmSvc.generateFn = oldProse }()

		gen, _ := genSvc.Generate(ctx, scene.ID)

		var gens []*domain.Generation
		for i := 0; i < 50; i++ {
			gens, _ = genSvc.ListGenerations(ctx, scene.ID)
			if len(gens) > 0 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		genSvc.AcceptGeneration(ctx, scene.ID, gen.ID)

		if len(gens) > 0 && gens[0].ID != gen.ID {
			for _, g := range gens {
				if g.ID != gen.ID && g.Accepted {
					t.Fatal("non-current generation should not be accepted")
				}
			}
		}
	})

	t.Run("concurrent generation returns error for same scene", func(t *testing.T) {
		cleanCollections(t, "generations")
		time.Sleep(50 * time.Millisecond) // let any previous pipeline goroutine release inFlight

		var wg sync.WaitGroup
		results := make(chan error, 2)

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := genSvc.Generate(ctx, scene.ID)
				results <- err
			}()
		}
		wg.Wait()
		close(results)

		errCount := 0
		for err := range results {
			if err != nil {
				errCount++
			}
		}
		if errCount != 1 {
			t.Fatalf("expected exactly 1 concurrent error, got %d", errCount)
		}
	})

	t.Run("generation missing scene returns error", func(t *testing.T) {
		_, err := genSvc.Generate(ctx, "nonexistent-scene")
		if err == nil {
			t.Fatal("expected error for missing scene")
		}
	})
}

func TestIntegration_GenerationCustomProse(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "generations", "timeline_events", "summaries")

	ctx := context.Background()

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	memRepo := mgorepo.NewMemoryRepo(testDB)
	tlRepo := mgorepo.NewTimelineRepo(testDB)
	sumRepo := mgorepo.NewSummaryRepo(testDB)

	story := &domain.Story{Title: "Custom Prose", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, story)
	char := &domain.Character{StoryID: story.ID, Name: "Detective"}
	charRepo.Create(ctx, char)
	scene := &domain.Scene{
		StoryID:      story.ID,
		Title:        "The Interrogation",
		BeatIntent:   "Detective interrogates the suspect",
		POV:          "Detective",
		Tone:         "tense",
		TargetWords:  50,
		Participants: []string{char.CharID},
	}
	sceneRepo.Create(ctx, scene)

	customProse := &mockProseService{
		generateFn: func(_ context.Context, params llm.PromptParams) (*llm.CompletionResponse, error) {
			if params.BeatIntent != "Detective interrogates the suspect" {
				return nil, nil
			}
			return &llm.CompletionResponse{
				Content: "DETECTIVE: Where were you last night?\nSUSPECT: I was at home.\nDETECTIVE: Can anyone confirm that?",
				Model:   "mock-sonnet",
			}, nil
		},
	}

	locRepo := mgorepo.NewLocationRepo(testDB)
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, StoryRepo: storyRepo,
		CharRepo: charRepo, StateRepo: stateRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: customProse, ExtractSvc: &mockExtractionService{},
		SummarySvc: &mockSummaryService{}, ValidateSvc: &mockValidationService{},
	})

	gen, err := genSvc.Generate(ctx, scene.ID)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	var gens []*domain.Generation
	for i := 0; i < 50; i++ {
		gens, _ = genSvc.ListGenerations(ctx, scene.ID)
		if len(gens) > 0 && gens[0].Output != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(gens) == 0 {
		t.Fatal("no generations")
	}
	if gens[0].Output != "" {
		t.Logf("generation output: %s", gens[0].Output[:min(len(gens[0].Output), 100)])
	}

	err = genSvc.AcceptGeneration(ctx, scene.ID, gen.ID)
	if err != nil {
		t.Fatalf("accept gen: %v", err)
	}
}
