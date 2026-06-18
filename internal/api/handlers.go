package api

import (
	"context"
	"net/http"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/service"
)

type StoryService interface {
	Create(ctx context.Context, title string) (*domain.Story, error)
	Get(ctx context.Context, id string) (*domain.Story, error)
	Update(ctx context.Context, id string, params service.UpdateStoryParams) (*domain.Story, error)
	List(ctx context.Context) ([]*domain.Story, error)
	Delete(ctx context.Context, id string) error
}

type SceneService interface {
	Create(ctx context.Context, scene *domain.Scene) (*domain.Scene, error)
	Get(ctx context.Context, id string) (*domain.Scene, error)
	Update(ctx context.Context, scene *domain.Scene) (*domain.Scene, error)
	List(ctx context.Context, storyID string) ([]*domain.Scene, error)
	Delete(ctx context.Context, id string) error
	Topology(ctx context.Context, storyID string) ([]*domain.Scene, []*domain.SceneEdge, error)
}

type EdgeService interface {
	Create(ctx context.Context, e *domain.SceneEdge) (*domain.SceneEdge, error)
	List(ctx context.Context, storyID string) ([]*domain.SceneEdge, error)
	Delete(ctx context.Context, storyID, from, to string) error
}

type CharacterService interface {
	Create(ctx context.Context, c *domain.Character) (*domain.Character, error)
	Get(ctx context.Context, id string) (*domain.Character, error)
	GetLatest(ctx context.Context, charID string) (*domain.Character, error)
	Update(ctx context.Context, c *domain.Character) (*domain.Character, error)
	List(ctx context.Context, storyID string) ([]*domain.Character, error)
}

type GenerationService interface {
	Generate(ctx context.Context, sceneID string) (*domain.Generation, error)
	AcceptGeneration(ctx context.Context, sceneID, genID string) error
	ListGenerations(ctx context.Context, sceneID string) ([]*domain.Generation, error)
}

type TimelineService interface {
	Create(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error)
	List(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error)
}

type SummaryService interface {
	GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error)
	GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error)
}

type MemoryService interface {
	ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error)
}

type LocationService interface {
	Create(ctx context.Context, loc *domain.Location) error
	Get(ctx context.Context, id string) (*domain.Location, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error)
	Update(ctx context.Context, loc *domain.Location) error
	DeleteByStory(ctx context.Context, storyID string) error
	GetByName(ctx context.Context, storyID, name string) (*domain.Location, error)
}

type Handlers struct {
	storySvc   StoryService
	sceneSvc   SceneService
	edgeSvc    EdgeService
	charSvc    CharacterService
	genSvc     GenerationService
	tlSvc      TimelineService
	sumSvc     SummaryService
	memSvc     MemoryService
	locSvc     LocationService
	outlineSvc *llm.OutlineServiceImpl
}

func NewHandlers(
	storySvc StoryService,
	sceneSvc SceneService,
	edgeSvc EdgeService,
	charSvc CharacterService,
	genSvc GenerationService,
	tlSvc TimelineService,
	sumSvc SummaryService,
	memSvc MemoryService,
	locSvc LocationService,
	outlineSvc *llm.OutlineServiceImpl,
) *Handlers {
	return &Handlers{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, genSvc: genSvc, tlSvc: tlSvc,
		sumSvc: sumSvc, memSvc: memSvc, locSvc: locSvc,
		outlineSvc: outlineSvc,
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
