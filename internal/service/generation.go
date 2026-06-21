package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

type GenerationServiceConfig struct {
	GenRepo   repository.GenerationRepository
	SceneRepo repository.SceneRepository
	JobRepo   repository.JobRepository
	EventBus  events.Bus
	AgentSvc  *AgentService
	BudgetSvc *TokenBudgetService
}

type GenerationService struct {
	genRepo       repository.GenerationRepository
	sceneRepo     repository.SceneRepository
	jobRepo       repository.JobRepository
	agentSvc      *AgentService
	budgetSvc     *TokenBudgetService

	genInFlight    sync.Map
	acceptInFlight sync.Map
	progress       ProgressPublisher
	eventBus       events.Bus
}

func NewGenerationService(cfg GenerationServiceConfig) *GenerationService {
	return &GenerationService{
		genRepo:   cfg.GenRepo,
		sceneRepo: cfg.SceneRepo,
		jobRepo:   cfg.JobRepo,
		agentSvc:  cfg.AgentSvc,
		budgetSvc: cfg.BudgetSvc,
		eventBus:  cfg.EventBus,
	}
}

func (s *GenerationService) SetProgressPublisher(p ProgressPublisher) {
	s.progress = p
}

func (s *GenerationService) Generate(ctx context.Context, sceneID string) (*domain.Generation, error) {
	if s.jobRepo == nil {
		return nil, fmt.Errorf("generation queue not available (no JobRepo configured)")
	}

	if _, loaded := s.genInFlight.LoadOrStore(sceneID, true); loaded {
		return nil, fmt.Errorf("generation already in progress for scene %s", sceneID)
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("get scene: %w", err)
	}
	if scene == nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("scene not found")
	}

	if s.budgetSvc != nil {
		model := string(llm.ModelSonnet)
		if err := s.budgetSvc.CheckAndConsume(ctx, scene.StoryID, model, "generation", 1000); err != nil {
			s.genInFlight.Delete(sceneID)
			return nil, fmt.Errorf("budget check: %w", err)
		}
	}

	if s.agentSvc != nil && s.agentSvc.IsAgentScene(scene) {
		gen, err := s.agentSvc.GenerateScene(ctx, sceneID)
		s.genInFlight.Delete(sceneID)
		return gen, err
	}

	gen := &domain.Generation{
		StoryID:    scene.StoryID,
		SceneID:    sceneID,
		Model:      string(llm.ModelSonnet),
		StepStatus: map[string]string{},
		Status:     domain.GenStatusPending,
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("create generation: %w", err)
	}

	job := &domain.Job{
		Type:       domain.JobTypeGenerateScene,
		Status:     domain.JobStatusPending,
		StoryID:    scene.StoryID,
		SceneID:    sceneID,
		GenID:      gen.ID,
		MaxRetries: 3,
	}
	if err := s.jobRepo.Create(ctx, job); err != nil {
		s.genInFlight.Delete(sceneID)
		return nil, fmt.Errorf("enqueue generation job: %w", err)
	}

	slog.Info("generation enqueued", "sceneId", sceneID, "genId", gen.ID, "jobId", job.ID)
	return gen, nil
}



func (s *GenerationService) AcceptGeneration(ctx context.Context, sceneID, genID string) error {
	if _, loaded := s.acceptInFlight.LoadOrStore(sceneID, true); loaded {
		return fmt.Errorf("accept already in progress for scene %s", sceneID)
	}
	defer s.acceptInFlight.Delete(sceneID)

	gen, err := s.genRepo.Get(ctx, genID)
	if err != nil {
		return err
	}
	if gen == nil {
		return fmt.Errorf("generation not found")
	}

	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		return fmt.Errorf("accept gen: get scene: %w", err)
	}
	if scene == nil {
		return fmt.Errorf("scene not found")
	}
	if err := scene.CanTransitionTo(domain.SceneStatusAccepted); err != nil {
		return fmt.Errorf("accept gen: %w", err)
	}

	scene.AcceptedGenerationID = genID
	scene.Status = domain.SceneStatusAccepted
	if err := s.sceneRepo.Update(ctx, scene); err != nil {
		return fmt.Errorf("accept gen: update scene: %w", err)
	}

	if err := s.genRepo.SetAccepted(ctx, sceneID, genID); err != nil {
		slog.Error("accept gen: set accepted flag", "sceneId", sceneID, "genId", genID, "error", err)
	}

	s.publishEvent(ctx, events.Event{
		Type: events.EventGenerationAccepted, StoryID: scene.StoryID, SceneID: sceneID, GenID: genID,
	})

	return nil
}

func (s *GenerationService) GetGeneration(ctx context.Context, genID string) (*domain.Generation, error) {
	return s.genRepo.Get(ctx, genID)
}

func (s *GenerationService) ListGenerations(ctx context.Context, sceneID string) ([]*domain.Generation, error) {
	return s.genRepo.ListByScene(ctx, sceneID)
}

func (s *GenerationService) ListGenerationsByStory(ctx context.Context, storyID string) ([]*domain.Generation, error) {
	return s.genRepo.ListByStory(ctx, storyID)
}

func (s *GenerationService) publishEvent(ctx context.Context, evt events.Event) {
	if s.eventBus != nil {
		s.eventBus.Publish(ctx, evt)
	}
}
