package service

import (
	"context"
	"fmt"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

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
		return nil, fmt.Errorf("scene not found: %w", ErrNotFound)
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
	scene, err := s.sceneRepo.Get(ctx, id)
	if err != nil {
		return err
	}
	if scene == nil {
		return s.sceneRepo.Delete(ctx, id)
	}

	fromEdges, _ := s.edgeRepo.ListFrom(ctx, id)
	for _, e := range fromEdges {
		_ = s.edgeRepo.Delete(ctx, e.StoryID, e.FromSceneID, e.ToSceneID)
	}
	toEdges, _ := s.edgeRepo.ListTo(ctx, id)
	for _, e := range toEdges {
		_ = s.edgeRepo.Delete(ctx, e.StoryID, e.FromSceneID, e.ToSceneID)
	}

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
