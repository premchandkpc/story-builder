package service

import (
	"context"
	"testing"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
)

func TestGenerate_NoJobRepo(t *testing.T) {
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   &mockGenRepo{},
		SceneRepo: newMockSceneRepo(),
	})
	_, err := svc.Generate(context.Background(), "scene-1")
	if err == nil {
		t.Fatal("expected error when no JobRepo configured")
	}
}

func TestGenerate_SequentialCallsSucceed(t *testing.T) {
	sceneRepo := newMockSceneRepo()
	sceneRepo.Create(context.Background(), &domain.Scene{ID: "scene-1", StoryID: "story-1"})

	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   &mockGenRepo{},
		SceneRepo: sceneRepo,
		JobRepo:   newMockJobRepo(),
	})

	_, err := svc.Generate(context.Background(), "scene-1")
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	_, err = svc.Generate(context.Background(), "scene-1")
	if err != nil {
		t.Fatalf("sequential calls should succeed (in-flight only guards setup): %v", err)
	}
}

func TestGenerate_SceneNotFound(t *testing.T) {
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   &mockGenRepo{},
		SceneRepo: newMockSceneRepo(),
		JobRepo:   newMockJobRepo(),
	})

	_, err := svc.Generate(context.Background(), "scene-nonexistent")
	if err == nil {
		t.Fatal("expected error for missing scene")
	}
}

func TestGenerate_EnqueuesJob(t *testing.T) {
	sceneRepo := newMockSceneRepo()
	sceneRepo.Create(context.Background(), &domain.Scene{
		ID:      "scene-1",
		StoryID: "story-1",
	})

	jobRepo := newMockJobRepo()

	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   &mockGenRepo{},
		SceneRepo: sceneRepo,
		JobRepo:   jobRepo,
	})

	gen, err := svc.Generate(context.Background(), "scene-1")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if gen == nil {
		t.Fatal("expected non-nil generation")
	}
	if gen.Status != domain.GenStatusPending {
		t.Fatalf("expected status pending, got %s", gen.Status)
	}

	job, err := jobRepo.Get(context.Background(), "job-scene-1")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job == nil {
		t.Fatal("expected job to be created")
	}
	if job.SceneID != "scene-1" {
		t.Fatalf("expected job.SceneID scene-1, got %s", job.SceneID)
	}
	if job.GenID != gen.ID {
		t.Fatalf("expected job.GenID %s, got %s", gen.ID, job.GenID)
	}
	if job.Type != domain.JobTypeGenerateScene {
		t.Fatalf("expected job type %s, got %s", domain.JobTypeGenerateScene, job.Type)
	}
}

func TestAcceptGeneration_OnlyOneAccepted(t *testing.T) {
	genRepo := &mockGenRepo{}
	sceneRepo := newMockSceneRepo()

	ctx := context.Background()
	sceneRepo.Create(ctx, &domain.Scene{
		ID:      "scene-1",
		StoryID: "story-1",
		Status:  domain.SceneStatusGenerated,
	})

	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   genRepo,
		SceneRepo: sceneRepo,
		EventBus:  events.NewInMemoryBus(),
	})

	gen1 := &domain.Generation{ID: "gen-1", SceneID: "scene-1", StoryID: "story-1", Status: domain.GenStatusSuccess}
	gen2 := &domain.Generation{ID: "gen-2", SceneID: "scene-1", StoryID: "story-1", Status: domain.GenStatusSuccess}
	genRepo.Create(ctx, gen1)
	genRepo.Create(ctx, gen2)

	if err := svc.AcceptGeneration(ctx, "scene-1", "gen-1"); err != nil {
		t.Fatalf("AcceptGeneration failed: %v", err)
	}

	gens, _ := genRepo.ListByScene(ctx, "scene-1")
	for _, g := range gens {
		if g.ID == "gen-1" && !g.Accepted {
			t.Fatal("gen-1 should be accepted")
		}
		if g.ID == "gen-2" && g.Accepted {
			t.Fatal("gen-2 should not be accepted")
		}
	}

	scene, _ := sceneRepo.Get(ctx, "scene-1")
	if scene.Status != domain.SceneStatusAccepted {
		t.Fatalf("expected scene status %s, got %s", domain.SceneStatusAccepted, scene.Status)
	}
}

func TestAcceptGeneration_InvalidTransition(t *testing.T) {
	genRepo := &mockGenRepo{}
	sceneRepo := newMockSceneRepo()

	ctx := context.Background()
	sceneRepo.Create(ctx, &domain.Scene{
		ID:     "scene-1",
		Status: domain.SceneStatusDraft,
	})

	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   genRepo,
		SceneRepo: sceneRepo,
	})

	gen1 := &domain.Generation{ID: "gen-1", SceneID: "scene-1", Status: domain.GenStatusSuccess}
	genRepo.Create(ctx, gen1)

	err := svc.AcceptGeneration(ctx, "scene-1", "gen-1")
	if err == nil {
		t.Fatal("expected error for invalid scene transition (draft -> accepted)")
	}
}

func TestAcceptGeneration_GenerationNotFound(t *testing.T) {
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo:   &mockGenRepo{},
		SceneRepo: newMockSceneRepo(),
	})
	err := svc.AcceptGeneration(context.Background(), "scene-1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing generation")
	}
}

func TestListGenerations(t *testing.T) {
	genRepo := &mockGenRepo{}
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo: genRepo,
	})

	ctx := context.Background()
	genRepo.Create(ctx, &domain.Generation{ID: "g1", SceneID: "s1"})
	genRepo.Create(ctx, &domain.Generation{ID: "g2", SceneID: "s1"})

	gens, err := svc.ListGenerations(ctx, "s1")
	if err != nil {
		t.Fatalf("ListGenerations failed: %v", err)
	}
	if len(gens) != 2 {
		t.Fatalf("expected 2 generations, got %d", len(gens))
	}
}

func TestGetGeneration(t *testing.T) {
	genRepo := &mockGenRepo{}
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo: genRepo,
	})

	ctx := context.Background()
	genRepo.Create(ctx, &domain.Generation{ID: "g1", SceneID: "s1"})

	gen, err := svc.GetGeneration(ctx, "g1")
	if err != nil {
		t.Fatalf("GetGeneration failed: %v", err)
	}
	if gen == nil {
		t.Fatal("expected generation")
	}
}

func TestGetGeneration_NotFound(t *testing.T) {
	svc := NewGenerationService(GenerationServiceConfig{
		GenRepo: &mockGenRepo{},
	})

	gen, err := svc.GetGeneration(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("GetGeneration failed: %v", err)
	}
	if gen != nil {
		t.Fatal("expected nil for missing generation")
	}
}

// TestRecoverStuckJobs removed — functionality moved to orchestration.Worker.
// See internal/orchestration/worker_test.go (if exists) for equivalent coverage.
