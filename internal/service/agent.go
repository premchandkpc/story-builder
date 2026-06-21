package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/premchand/story-builder/internal/agents"
	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
	"github.com/premchand/story-builder/internal/scene"
)

type AgentServiceConfig struct {
	Registry     *agents.AgentRegistry
	Orchestrator *agents.Orchestrator
	TurnRepo     scene.TurnRepository
	ActorRepo    scene.ActorRepository
	CanonRepo    scene.CanonDeltaRepository
	GenRepo      repository.GenerationRepository
	StoryRepo    repository.StoryRepository
	SceneRepo    repository.SceneRepository
	CharRepo     repository.CharacterRepository
	StateRepo    repository.CharacterStateRepository
	BibleRepo    repository.BibleRepository
	EdgeRepo     repository.SceneEdgeRepository
	MemRepo      repository.MemoryRepository
	TlRepo       repository.TimelineRepository
	SumRepo      repository.SummaryRepository
}

type AgentService struct {
	registry     *agents.AgentRegistry
	orchestrator *agents.Orchestrator
	turnRepo     scene.TurnRepository
	actorRepo    scene.ActorRepository
	canonRepo    scene.CanonDeltaRepository
	genRepo      repository.GenerationRepository
	storyRepo    repository.StoryRepository
	sceneRepo    repository.SceneRepository
	charRepo     repository.CharacterRepository
	stateRepo    repository.CharacterStateRepository
	bibleRepo    repository.BibleRepository
	edgeRepo     repository.SceneEdgeRepository
	memRepo      repository.MemoryRepository
	tlRepo       repository.TimelineRepository
	sumRepo      repository.SummaryRepository
}

func NewAgentService(cfg AgentServiceConfig) *AgentService {
	return &AgentService{
		registry: cfg.Registry, orchestrator: cfg.Orchestrator,
		turnRepo: cfg.TurnRepo, actorRepo: cfg.ActorRepo, canonRepo: cfg.CanonRepo,
		genRepo: cfg.GenRepo, storyRepo: cfg.StoryRepo, sceneRepo: cfg.SceneRepo,
		charRepo: cfg.CharRepo, stateRepo: cfg.StateRepo, bibleRepo: cfg.BibleRepo,
		edgeRepo: cfg.EdgeRepo, memRepo: cfg.MemRepo, tlRepo: cfg.TlRepo, sumRepo: cfg.SumRepo,
	}
}

func (s *AgentService) GenerateScene(ctx context.Context, sceneID string) (*domain.Generation, error) {
	scene, err := s.sceneRepo.Get(ctx, sceneID)
	if err != nil {
		return nil, fmt.Errorf("get scene: %w", err)
	}
	if scene == nil {
		return nil, fmt.Errorf("scene %s not found", sceneID)
	}

	gen := &domain.Generation{
		StoryID:    scene.StoryID,
		SceneID:    sceneID,
		Model:      "agent-orchestrator",
		StepStatus: map[string]string{},
		Status:     domain.GenStatusPending,
	}
	if err := s.genRepo.Create(ctx, gen); err != nil {
		return nil, fmt.Errorf("create generation: %w", err)
	}

	agentCtx, err := s.buildContext(ctx, scene)
	if err != nil {
		s.genRepo.Update(ctx, &domain.Generation{ID: gen.ID, Status: domain.GenStatusFailed, Error: err.Error()})
		return nil, fmt.Errorf("build context: %w", err)
	}

	plan, err := s.orchestrator.Plan(ctx, scene)
	if err != nil {
		s.genRepo.Update(ctx, &domain.Generation{ID: gen.ID, Status: domain.GenStatusFailed, Error: err.Error()})
		return nil, fmt.Errorf("orchestrator plan: %w", err)
	}

	result, err := s.orchestrator.Execute(ctx, plan, agentCtx, s.turnRepo)
	if err != nil {
		slog.Error("orchestrator execute failed", "sceneId", sceneID, "error", err)
		gen.Status = domain.GenStatusFailed
		gen.Error = err.Error()
		s.genRepo.Update(ctx, gen)
		return gen, nil
	}

	if err := s.orchestrator.RunFinish(ctx, sceneID, agentCtx, s.turnRepo); err != nil {
		slog.Warn("orchestrator runfinish failed", "sceneId", sceneID, "error", err)
	}

	if len(result.Turns) > 0 {
		gen.Output = result.Turns[len(result.Turns)-1].Output
	}
	gen.Status = domain.GenStatusSuccess
	if err := s.genRepo.Update(ctx, gen); err != nil {
		return nil, fmt.Errorf("update generation: %w", err)
	}

	scene.Status = domain.SceneStatusGenerated
	scene.GeneratedContent = gen.Output
	if err := s.sceneRepo.Update(ctx, scene); err != nil {
		slog.Warn("update scene after agent generation", "sceneId", sceneID, "error", err)
	}

	slog.Info("agent generation complete", "sceneId", sceneID, "genId", gen.ID, "turns", len(result.Turns))
	return gen, nil
}

func (s *AgentService) buildContext(ctx context.Context, scene *domain.Scene) (*agents.AgentContext, error) {
	story, err := s.storyRepo.Get(ctx, scene.StoryID)
	if err != nil {
		return nil, fmt.Errorf("get story: %w", err)
	}

	chars, err := s.charRepo.ListByStory(ctx, scene.StoryID)
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}

	var states []*domain.CharacterState
	for _, c := range chars {
		st, err := s.stateRepo.Get(ctx, c.CharID, scene.ID)
		if err != nil {
			return nil, fmt.Errorf("get state %s: %w", c.CharID, err)
		}
		if st != nil {
			states = append(states, st)
		}
	}

	bible, _ := s.bibleRepo.GetByStory(ctx, scene.StoryID)
	edges, _ := s.edgeRepo.ListByStory(ctx, scene.StoryID)
	timeline, _ := s.tlRepo.ListByStory(ctx, scene.StoryID)
	summaries, _ := s.sumRepo.ListByLevel(ctx, scene.StoryID, "scene")

	memories := make(map[string][]*domain.CharacterMemory)
	for _, c := range chars {
		mems, err := s.memRepo.ListByCharacter(ctx, c.CharID)
		if err == nil && len(mems) > 0 {
			memories[c.CharID] = mems
		}
	}

	turns, _ := s.turnRepo.ListByScene(ctx, scene.ID)
	deltas, _ := s.canonRepo.ListByScene(ctx, scene.ID)

	var blueprint *domain.StoryBlueprint
	if story != nil {
		blueprint = story.Blueprint
	}

	return &agents.AgentContext{
		StoryID:        scene.StoryID,
		SceneID:        scene.ID,
		Story:          story,
		Scene:          scene,
		Characters:     chars,
		CharStates:     states,
		Bible:          bible,
		BluePrint:      blueprint,
		Edges:          edges,
		Turns:          turns,
		Timeline:       timeline,
		Memories:       memories,
		CanonDeltas:    deltas,
		Summaries:      summaries,
		ParticipantIDs: scene.Participants,
	}, nil
}

func (s *AgentService) GetTurns(ctx context.Context, sceneID string) ([]*domain.SceneTurn, error) {
	return s.turnRepo.ListByScene(ctx, sceneID)
}

func (s *AgentService) GetTurnsByRole(ctx context.Context, sceneID, role string) ([]*domain.SceneTurn, error) {
	return s.turnRepo.ListByRole(ctx, sceneID, role)
}

func (s *AgentService) GetCanonDeltas(ctx context.Context, sceneID string) ([]*domain.CanonDelta, error) {
	return s.canonRepo.ListByScene(ctx, sceneID)
}

func (s *AgentService) RecordStateDelta(ctx context.Context, d *domain.CanonDelta) error {
	return s.canonRepo.Create(ctx, d)
}

func (s *AgentService) IsAgentScene(scene *domain.Scene) bool {
	return scene.SceneStructure != nil || scene.FlowType != "" && scene.FlowType != domain.FlowTypeCustom
}


