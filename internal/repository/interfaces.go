package repository

import (
	"context"
	"time"

	"github.com/premchand/story-builder/internal/domain"
)

type StoryRepository interface {
	Create(ctx context.Context, s *domain.Story) error
	Get(ctx context.Context, id string) (*domain.Story, error)
	Update(ctx context.Context, s *domain.Story) error
	List(ctx context.Context) ([]*domain.Story, error)
	Delete(ctx context.Context, id string) error
}

type SceneRepository interface {
	Create(ctx context.Context, s *domain.Scene) error
	Get(ctx context.Context, id string) (*domain.Scene, error)
	Update(ctx context.Context, s *domain.Scene) error
	ListByStory(ctx context.Context, storyID string) ([]*domain.Scene, error)
	Delete(ctx context.Context, id string) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type SceneEdgeRepository interface {
	Create(ctx context.Context, e *domain.SceneEdge) error
	ListByStory(ctx context.Context, storyID string) ([]*domain.SceneEdge, error)
	ListFrom(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error)
	ListTo(ctx context.Context, sceneID string) ([]*domain.SceneEdge, error)
	Delete(ctx context.Context, storyID, fromSceneID, toSceneID string) error
	DeleteByStory(ctx context.Context, storyID string) error
	DeleteByID(ctx context.Context, edgeID string) error
}

type CharacterRepository interface {
	Create(ctx context.Context, c *domain.Character) error
	Get(ctx context.Context, id string) (*domain.Character, error)
	GetLatest(ctx context.Context, charID string) (*domain.Character, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Character, error)
	Update(ctx context.Context, c *domain.Character) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type CharacterStateRepository interface {
	Append(ctx context.Context, s *domain.CharacterState) error
	Get(ctx context.Context, characterID, sceneID string) (*domain.CharacterState, error)
	ListByCharacter(ctx context.Context, characterID string) ([]*domain.CharacterState, error)
	ListByScene(ctx context.Context, sceneID string) ([]*domain.CharacterState, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type GenerationRepository interface {
	Create(ctx context.Context, g *domain.Generation) error
	Get(ctx context.Context, id string) (*domain.Generation, error)
	Update(ctx context.Context, g *domain.Generation) error
	SetStepStatus(ctx context.Context, genID, step, status string) error
	SetAccepted(ctx context.Context, sceneID, genID string) error
	ListByScene(ctx context.Context, sceneID string) ([]*domain.Generation, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Generation, error)
	DeleteByScene(ctx context.Context, sceneID string) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type MemoryRepository interface {
	Create(ctx context.Context, m *domain.CharacterMemory) error
	ListByCharacter(ctx context.Context, characterID string) ([]*domain.CharacterMemory, error)
	Search(ctx context.Context, storyID, characterID string, query []float64, limit int) ([]*domain.CharacterMemory, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type TimelineRepository interface {
	Create(ctx context.Context, e *domain.TimelineEvent) error
	ListByStory(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error)
	ListByRelatedStories(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type AgentConfigRepository interface {
	Create(ctx context.Context, a *domain.AgentConfig) error
	Get(ctx context.Context, name string) (*domain.AgentConfig, error)
	List(ctx context.Context) ([]*domain.AgentConfig, error)
	Delete(ctx context.Context, name string) error
}

type LocationRepository interface {
	Create(ctx context.Context, l *domain.Location) error
	Get(ctx context.Context, id string) (*domain.Location, error)
	GetByName(ctx context.Context, storyID, name string) (*domain.Location, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error)
	Update(ctx context.Context, l *domain.Location) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type SummaryRepository interface {
	Upsert(ctx context.Context, s *domain.Summary) error
	GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error)
	GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error)
	ListByLevel(ctx context.Context, storyID, level string) ([]*domain.Summary, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type BibleRepository interface {
	Create(ctx context.Context, b *domain.StoryBible) error
	Get(ctx context.Context, id string) (*domain.StoryBible, error)
	GetByStory(ctx context.Context, storyID string) (*domain.StoryBible, error)
	ListByReferencingStory(ctx context.Context, storyID string) ([]*domain.StoryBible, error)
	Update(ctx context.Context, b *domain.StoryBible) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type JobRepository interface {
	Create(ctx context.Context, j *domain.Job) error
	Get(ctx context.Context, id string) (*domain.Job, error)
	Update(ctx context.Context, j *domain.Job) error
	PickPending(ctx context.Context, jobType string, leaseTime time.Duration) (*domain.Job, error)
	ListPending(ctx context.Context) ([]*domain.Job, error)
	ListStuck(ctx context.Context, threshold time.Duration) ([]*domain.Job, error)
}

type TokenBudgetRepository interface {
	Get(ctx context.Context, storyID string) (*domain.TokenBudget, error)
	Upsert(ctx context.Context, tb *domain.TokenBudget) error
	DeleteByStory(ctx context.Context, storyID string) error
}

type ChapterRepository interface {
	Create(ctx context.Context, c *domain.Chapter) error
	Get(ctx context.Context, id string) (*domain.Chapter, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error)
	ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error)
	Update(ctx context.Context, c *domain.Chapter) error
	Delete(ctx context.Context, id string) error
	DeleteByStory(ctx context.Context, storyID string) error
}
