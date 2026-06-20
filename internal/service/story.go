package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/repository"
)

// StoryCascadeDeleter collects all repositories needed for cascade-deleting a story.
type StoryCascadeDeleter struct {
	SceneRepo   repository.SceneRepository
	EdgeRepo    repository.SceneEdgeRepository
	CharRepo    repository.CharacterRepository
	StateRepo   repository.CharacterStateRepository
	GenRepo     repository.GenerationRepository
	MemRepo     repository.MemoryRepository
	TlRepo      repository.TimelineRepository
	SumRepo     repository.SummaryRepository
	LocRepo     repository.LocationRepository
	BibleRepo   repository.BibleRepository
	ChapterRepo repository.ChapterRepository
}

type StoryService struct {
	repo    repository.StoryRepository
	deleter *StoryCascadeDeleter
}

func NewStoryService(repo repository.StoryRepository, deleter *StoryCascadeDeleter) *StoryService {
	return &StoryService{repo: repo, deleter: deleter}
}

func (s *StoryService) Create(ctx context.Context, title string) (*domain.Story, error) {
	st := &domain.Story{
		Title:     title,
		Status:    domain.StoryStatusDraft,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return nil, fmt.Errorf("create story: %w", err)
	}
	return st, nil
}

func (s *StoryService) Get(ctx context.Context, id string) (*domain.Story, error) {
	return s.repo.Get(ctx, id)
}

type UpdateStoryParams struct {
	Title  string
	Status string
}

func (s *StoryService) Update(ctx context.Context, id string, params UpdateStoryParams) (*domain.Story, error) {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get story for update: %w", err)
	}
	if st == nil {
		return nil, fmt.Errorf("story not found")
	}
	if params.Title != "" {
		st.Title = params.Title
	}
	if params.Status != "" {
		if err := st.CanTransitionTo(params.Status); err != nil {
			return nil, err
		}
		st.Status = params.Status
	}
	st.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, st); err != nil {
		return nil, err
	}
	return st, nil
}

func (s *StoryService) List(ctx context.Context) ([]*domain.Story, error) {
	return s.repo.List(ctx)
}

func (s *StoryService) GetBlueprint(ctx context.Context, id string) (*domain.StoryBlueprint, error) {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("story not found")
	}
	return st.Blueprint, nil
}

func (s *StoryService) UpdateBlueprint(ctx context.Context, id string, bp *domain.StoryBlueprint) error {
	st, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if st == nil {
		return fmt.Errorf("story not found")
	}
	st.Blueprint = bp
	st.UpdatedAt = time.Now()
	return s.repo.Update(ctx, st)
}

func (s *StoryService) Delete(ctx context.Context, id string) error {
	if s.deleter != nil {
		if err := s.deleter.cascade(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.Delete(ctx, id)
}

func (d *StoryCascadeDeleter) cascade(ctx context.Context, storyID string) error {
	// Best-effort cascade delete across all related collections.
	// Order matters: child collections first, then parents.
	type step struct {
		name string
		fn   func(context.Context, string) error
	}
	steps := []step{
		{"character_state",   d.StateRepo.DeleteByStory},
		{"character_memories", d.MemRepo.DeleteByStory},
		{"generations",        d.GenRepo.DeleteByStory},
		{"summaries",          d.SumRepo.DeleteByStory},
		{"timeline_events",    d.TlRepo.DeleteByStory},
		{"scene_edges",        d.EdgeRepo.DeleteByStory},
		{"scenes",             d.SceneRepo.DeleteByStory},
		{"characters",         d.CharRepo.DeleteByStory},
		{"locations",          d.LocRepo.DeleteByStory},
		{"chapters",           d.ChapterRepo.DeleteByStory},
		{"bibles",             d.BibleRepo.DeleteByStory},
	}

	var firstErr error
	for _, st := range steps {
		if err := st.fn(ctx, storyID); err != nil {
			slog.Error("cascade delete failed", "collection", st.name, "storyId", storyID, "error", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("cascade delete %s: %w", st.name, err)
			}
		}
	}
	return firstErr
}

type SceneService struct {
	sceneRepo repository.SceneRepository
	edgeRepo  repository.SceneEdgeRepository
	genRepo   repository.GenerationRepository
}

func NewSceneService(sceneRepo repository.SceneRepository, edgeRepo repository.SceneEdgeRepository, genRepo repository.GenerationRepository) *SceneService {
	return &SceneService{sceneRepo: sceneRepo, edgeRepo: edgeRepo, genRepo: genRepo}
}

func (s *SceneService) Create(ctx context.Context, scene *domain.Scene) (*domain.Scene, error) {
	if err := s.sceneRepo.Create(ctx, scene); err != nil {
		return nil, fmt.Errorf("create scene: %w", err)
	}
	return scene, nil
}

func (s *SceneService) Get(ctx context.Context, id string) (*domain.Scene, error) {
	return s.sceneRepo.Get(ctx, id)
}

func (s *SceneService) Update(ctx context.Context, scene *domain.Scene) (*domain.Scene, error) {
	existing, err := s.sceneRepo.Get(ctx, scene.ID)
	if err != nil {
		return nil, fmt.Errorf("get scene for update: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("scene not found")
	}
	if scene.Title != "" {
		existing.Title = scene.Title
	}
	if scene.BeatIntent != "" {
		existing.BeatIntent = scene.BeatIntent
	}
	if scene.Summary != "" {
		existing.Summary = scene.Summary
	}
	if scene.GeneratedContent != "" {
		existing.GeneratedContent = scene.GeneratedContent
	}
	if scene.Participants != nil {
		existing.Participants = scene.Participants
	}
	if scene.LocationRef != "" {
		existing.LocationRef = scene.LocationRef
	}
	if scene.ChapterID != "" {
		existing.ChapterID = scene.ChapterID
	}
	if scene.POV != "" {
		existing.POV = scene.POV
	}
	if scene.Tone != "" {
		existing.Tone = scene.Tone
	}
	if scene.FlowType != "" {
		existing.FlowType = scene.FlowType
	}
	if scene.SceneStructure != nil {
		existing.SceneStructure = scene.SceneStructure
	}
	if scene.Metadata != nil {
		existing.Metadata = scene.Metadata
	}
	if scene.TargetWords != 0 {
		existing.TargetWords = scene.TargetWords
	}
	if scene.Status != "" {
		if scene.Status != existing.Status {
			if err := existing.CanTransitionTo(scene.Status); err != nil {
				return nil, err
			}
		}
		existing.Status = scene.Status
	}
	if scene.PositionX != nil {
		existing.PositionX = scene.PositionX
	}
	if scene.PositionY != nil {
		existing.PositionY = scene.PositionY
	}
	if err := s.sceneRepo.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *SceneService) List(ctx context.Context, storyID string) ([]*domain.Scene, error) {
	return s.sceneRepo.ListByStory(ctx, storyID)
}

func (s *SceneService) Delete(ctx context.Context, id string) error {
	// Determine storyId from the scene for edge cleanup.
	scene, err := s.sceneRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if scene == nil {
		return s.sceneRepo.Delete(ctx, id)
	}

	// Clean up edges referencing this scene.
	fromEdges, _ := s.edgeRepo.ListFrom(ctx, id)
	for _, e := range fromEdges {
		_ = s.edgeRepo.Delete(ctx, e.StoryID, e.FromSceneID, e.ToSceneID)
	}
	toEdges, _ := s.edgeRepo.ListTo(ctx, id)
	for _, e := range toEdges {
		_ = s.edgeRepo.Delete(ctx, e.StoryID, e.FromSceneID, e.ToSceneID)
	}

	// Clean up generations for this scene.
	_ = s.genRepo.DeleteByScene(ctx, id)

	return s.sceneRepo.Delete(ctx, id)
}

func (s *SceneService) Topology(ctx context.Context, storyID string) ([]*domain.Scene, []*domain.SceneEdge, error) {
	scenes, err := s.sceneRepo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, nil, err
	}
	edges, err := s.edgeRepo.ListByStory(ctx, storyID)
	if err != nil {
		return nil, nil, err
	}
	return scenes, edges, nil
}

type EdgeService struct {
	repo repository.SceneEdgeRepository
}

func NewEdgeService(repo repository.SceneEdgeRepository) *EdgeService {
	return &EdgeService{repo: repo}
}

func (s *EdgeService) Create(ctx context.Context, e *domain.SceneEdge) (*domain.SceneEdge, error) {
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create edge: %w", err)
	}
	return e, nil
}

func (s *EdgeService) List(ctx context.Context, storyID string) ([]*domain.SceneEdge, error) {
	return s.repo.ListByStory(ctx, storyID)
}

func (s *EdgeService) Delete(ctx context.Context, storyID, from, to string) error {
	return s.repo.Delete(ctx, storyID, from, to)
}

func (s *EdgeService) DeleteByID(ctx context.Context, edgeID string) error {
	return s.repo.DeleteByID(ctx, edgeID)
}

type CharacterService struct {
	charRepo repository.CharacterRepository
}

func NewCharacterService(charRepo repository.CharacterRepository) *CharacterService {
	return &CharacterService{charRepo: charRepo}
}

func (s *CharacterService) Create(ctx context.Context, c *domain.Character) (*domain.Character, error) {
	if err := s.charRepo.Create(ctx, c); err != nil {
		return nil, fmt.Errorf("create character: %w", err)
	}
	return c, nil
}

func (s *CharacterService) Get(ctx context.Context, id string) (*domain.Character, error) {
	return s.charRepo.Get(ctx, id)
}

func (s *CharacterService) GetLatest(ctx context.Context, charID string) (*domain.Character, error) {
	return s.charRepo.GetLatest(ctx, charID)
}

func (s *CharacterService) Update(ctx context.Context, c *domain.Character) (*domain.Character, error) {
	existing, err := s.charRepo.GetLatest(ctx, c.CharID)
	if err != nil {
		return nil, fmt.Errorf("get latest character: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("character not found")
	}
	if c.Name != "" {
		existing.Name = c.Name
	}
	if c.Persona != "" {
		existing.Persona = c.Persona
	}
	if c.Backstory != "" {
		existing.Backstory = c.Backstory
	}
	if c.Personality != nil {
		existing.Personality = c.Personality
	}
	if c.MoralAlignment != "" {
		existing.MoralAlignment = c.MoralAlignment
	}
	if c.Goals != nil {
		existing.Goals = c.Goals
	}
	if c.Flaws != nil {
		existing.Flaws = c.Flaws
	}
	if c.Traits != nil {
		existing.Traits = c.Traits
	}
	if c.VoiceSamples != nil {
		existing.VoiceSamples = c.VoiceSamples
	}
	if c.Relationships != nil {
		existing.Relationships = c.Relationships
	}
	if c.Want != "" {
		existing.Want = c.Want
	}
	if c.Need != "" {
		existing.Need = c.Need
	}
	if c.FalseBelief != "" {
		existing.FalseBelief = c.FalseBelief
	}
	if c.Fear != "" {
		existing.Fear = c.Fear
	}
	if c.ArcType != "" {
		existing.ArcType = c.ArcType
	}
	if err := s.charRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("update character: %w", err)
	}
	return existing, nil
}

func (s *CharacterService) List(ctx context.Context, storyID string) ([]*domain.Character, error) {
	return s.charRepo.ListByStory(ctx, storyID)
}

type TimelineService struct {
	repo repository.TimelineRepository
}

func NewTimelineService(repo repository.TimelineRepository) *TimelineService {
	return &TimelineService{repo: repo}
}

func (s *TimelineService) Create(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error) {
	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("create timeline event: %w", err)
	}
	return e, nil
}

func (s *TimelineService) List(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error) {
	return s.repo.ListByStory(ctx, storyID)
}

type SummaryService struct {
	repo repository.SummaryRepository
}

func NewSummaryService(repo repository.SummaryRepository) *SummaryService {
	return &SummaryService{repo: repo}
}

func (s *SummaryService) GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error) {
	return s.repo.GetByLevel(ctx, storyID, level)
}

func (s *SummaryService) GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error) {
	return s.repo.GetSceneSummary(ctx, storyID, sceneID)
}

type MemoryService struct {
	repo   repository.MemoryRepository
	embedSvc llm.EmbeddingService
}

func NewMemoryService(repo repository.MemoryRepository, embedSvc llm.EmbeddingService) *MemoryService {
	return &MemoryService{repo: repo, embedSvc: embedSvc}
}

func (s *MemoryService) ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error) {
	return s.repo.ListByCharacter(ctx, charID)
}

func (s *MemoryService) Search(ctx context.Context, storyID, characterID, query string, limit int) ([]*domain.CharacterMemory, error) {
	if s.embedSvc == nil {
		return nil, fmt.Errorf("embedding service not configured")
	}
	embedding, err := s.embedSvc.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	return s.repo.Search(ctx, storyID, characterID, embedding, limit)
}
