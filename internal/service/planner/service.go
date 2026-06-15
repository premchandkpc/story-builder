package planner

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/planner"
)

type Service interface {
	PlanChapter(ctx context.Context, storyID, chapterID uuid.UUID, goal string, conflicts []string) (*planner.ChapterPlan, error)
	PlanScene(ctx context.Context, storyID, chapterID, sceneID uuid.UUID, sceneCtx planner.SceneContext) (*planner.ScenePlan, error)
	GetChapterPlan(ctx context.Context, chapterID uuid.UUID) (*planner.ChapterPlan, error)
	GetScenePlan(ctx context.Context, sceneID uuid.UUID) (*planner.ScenePlan, error)
}

type plannerService struct {
	svc planner.PlannerService
}

func NewService(svc planner.PlannerService) Service {
	return &plannerService{svc: svc}
}

func (s *plannerService) PlanChapter(ctx context.Context, storyID, chapterID uuid.UUID, goal string, conflicts []string) (*planner.ChapterPlan, error) {
	plan, err := s.svc.PlanChapter(storyID, chapterID, goal, conflicts)
	if err != nil {
		return nil, fmt.Errorf("planner service: chapter: %w", err)
	}
	return plan, nil
}

func (s *plannerService) PlanScene(ctx context.Context, storyID, chapterID, sceneID uuid.UUID, sceneCtx planner.SceneContext) (*planner.ScenePlan, error) {
	plan, err := s.svc.PlanScene(storyID, chapterID, sceneID, sceneCtx)
	if err != nil {
		return nil, fmt.Errorf("planner service: scene: %w", err)
	}
	return plan, nil
}

func (s *plannerService) GetChapterPlan(ctx context.Context, chapterID uuid.UUID) (*planner.ChapterPlan, error) {
	return s.svc.GetChapterPlan(chapterID)
}

func (s *plannerService) GetScenePlan(ctx context.Context, sceneID uuid.UUID) (*planner.ScenePlan, error) {
	return s.svc.GetScenePlan(sceneID)
}
