package api

import (
	"context"
	"log/slog"
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
	GetBlueprint(ctx context.Context, id string) (*domain.StoryBlueprint, error)
	UpdateBlueprint(ctx context.Context, id string, bp *domain.StoryBlueprint) error
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

type GenerationWriteService interface {
	Generate(ctx context.Context, sceneID string) (*domain.Generation, error)
	AcceptGeneration(ctx context.Context, sceneID, genID string) error
}

type GenerationReadService interface {
	GetGeneration(ctx context.Context, genID string) (*domain.Generation, error)
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
	Search(ctx context.Context, storyID, characterID, query string, limit int) ([]*domain.CharacterMemory, error)
}

type BibleService interface {
	Get(ctx context.Context, storyID string) (*domain.StoryBible, error)
	Generate(ctx context.Context, storyID string) (*domain.StoryBible, error)
	DeleteByStory(ctx context.Context, storyID string) error
}

type ChapterService interface {
	Create(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error)
	Get(ctx context.Context, id string) (*domain.Chapter, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error)
	ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error)
	Update(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error)
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
	storySvc     StoryService
	sceneSvc     SceneService
	edgeSvc      EdgeService
	charSvc      CharacterService
	genWriteSvc  GenerationWriteService
	genReadSvc   GenerationReadService
	tlSvc        TimelineService
	sumSvc       SummaryService
	memSvc       MemoryService
	locSvc       LocationService
	bibleSvc     BibleService
	chapterSvc   ChapterService
	outlineSvc   llm.OutlineService
	titleSvc     llm.TitleService
	progress     *ProgressHub
}

func NewHandlers(
	storySvc StoryService,
	sceneSvc SceneService,
	edgeSvc EdgeService,
	charSvc CharacterService,
	genSvc GenerationWriteService,
	genReadSvc GenerationReadService,
	tlSvc TimelineService,
	sumSvc SummaryService,
	memSvc MemoryService,
	locSvc LocationService,
	bibleSvc BibleService,
	chapterSvc ChapterService,
	outlineSvc llm.OutlineService,
	titleSvc llm.TitleService,
	progress *ProgressHub,
) *Handlers {
	return &Handlers{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, genWriteSvc: genSvc, genReadSvc: genReadSvc, tlSvc: tlSvc,
		sumSvc: sumSvc, memSvc: memSvc, locSvc: locSvc,
		bibleSvc: bibleSvc, chapterSvc: chapterSvc,
		outlineSvc: outlineSvc, titleSvc: titleSvc, progress: progress,
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	if status >= 500 {
		slog.Error("server error", "status", status, "msg", msg)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}
