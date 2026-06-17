package service

import (
	"context"
	"fmt"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type StoryService struct {
	repo repository.StoryRepository
}

func NewStoryService(repo repository.StoryRepository) *StoryService {
	return &StoryService{repo: repo}
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
		return nil, err
	}
	if st == nil {
		return nil, fmt.Errorf("story not found")
	}
	if params.Title != "" {
		st.Title = params.Title
	}
	if params.Status != "" {
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

func (s *StoryService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

type SceneService struct {
	sceneRepo repository.SceneRepository
	edgeRepo  repository.SceneEdgeRepository
}

func NewSceneService(sceneRepo repository.SceneRepository, edgeRepo repository.SceneEdgeRepository) *SceneService {
	return &SceneService{sceneRepo: sceneRepo, edgeRepo: edgeRepo}
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
	if err := s.sceneRepo.Update(ctx, scene); err != nil {
		return nil, err
	}
	return scene, nil
}

func (s *SceneService) List(ctx context.Context, storyID string) ([]*domain.Scene, error) {
	return s.sceneRepo.ListByStory(ctx, storyID)
}

func (s *SceneService) Delete(ctx context.Context, id string) error {
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

type CharacterService struct {
	charRepo  repository.CharacterRepository
	stateRepo repository.CharacterStateRepository
}

func NewCharacterService(charRepo repository.CharacterRepository, stateRepo repository.CharacterStateRepository) *CharacterService {
	return &CharacterService{charRepo: charRepo, stateRepo: stateRepo}
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
	repo repository.MemoryRepository
}

func NewMemoryService(repo repository.MemoryRepository) *MemoryService {
	return &MemoryService{repo: repo}
}

func (s *MemoryService) ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error) {
	return s.repo.ListByCharacter(ctx, charID)
}
