// Package api implements the HTTP transport layer.
// Handlers wire chi route callbacks to service-layer interfaces.
package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/premchand/story-builder/internal/agents"
	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/service"
)

// StoryService defines story CRUD + blueprint operations.
type StoryService interface {
	Create(ctx context.Context, title string) (*domain.Story, error)
	Get(ctx context.Context, id string) (*domain.Story, error)
	Update(ctx context.Context, id string, params service.UpdateStoryParams) (*domain.Story, error)
	List(ctx context.Context) ([]*domain.Story, error)
	Delete(ctx context.Context, id string) error
	GetBlueprint(ctx context.Context, id string) (*domain.StoryBlueprint, error)
	UpdateBlueprint(ctx context.Context, id string, bp *domain.StoryBlueprint) error
}

// SceneService manages DAG scene nodes.
type SceneService interface {
	Create(ctx context.Context, scene *domain.Scene) (*domain.Scene, error)
	Get(ctx context.Context, id string) (*domain.Scene, error)
	Update(ctx context.Context, scene *domain.Scene) (*domain.Scene, error)
	List(ctx context.Context, storyID string) ([]*domain.Scene, error)
	Delete(ctx context.Context, id string) error
	Topology(ctx context.Context, storyID string) ([]*domain.Scene, []*domain.SceneEdge, error)
}

// EdgeService manages directed edges between scene nodes.
type EdgeService interface {
	Create(ctx context.Context, e *domain.SceneEdge) (*domain.SceneEdge, error)
	List(ctx context.Context, storyID string) ([]*domain.SceneEdge, error)
	Delete(ctx context.Context, storyID, from, to string) error
	DeleteByID(ctx context.Context, edgeID string) error
}

// CharacterService manages character definitions and immutable version log.
type CharacterService interface {
	Create(ctx context.Context, c *domain.Character) (*domain.Character, error)
	Get(ctx context.Context, id string) (*domain.Character, error)
	GetLatest(ctx context.Context, charID string) (*domain.Character, error)
	Update(ctx context.Context, c *domain.Character) (*domain.Character, error)
	List(ctx context.Context, storyID string) ([]*domain.Character, error)
	MigrateCharacter(ctx context.Context, charID, targetStoryID string) (*domain.Character, error)
}

// GenerationWriteService triggers LLM generation and accepts results.
type GenerationWriteService interface {
	Generate(ctx context.Context, sceneID string) (*domain.Generation, error)
	AcceptGeneration(ctx context.Context, sceneID, genID string) error
}

// GenerationReadService queries generation artifacts.
type GenerationReadService interface {
	GetGeneration(ctx context.Context, genID string) (*domain.Generation, error)
	ListGenerations(ctx context.Context, sceneID string) ([]*domain.Generation, error)
	ListGenerationsByStory(ctx context.Context, storyID string) ([]*domain.Generation, error)
}

// TimelineService records and queries ordered story events.
type TimelineService interface {
	Create(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error)
	List(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error)
	CreateCrossStory(ctx context.Context, e *domain.TimelineEvent) (*domain.TimelineEvent, error)
	ListCrossStory(ctx context.Context, storyID string) ([]*domain.TimelineEvent, error)
}

// SummaryService retrieves cached summaries at various granularity levels.
type SummaryService interface {
	GetByLevel(ctx context.Context, storyID, level string) (*domain.Summary, error)
	GetSceneSummary(ctx context.Context, storyID, sceneID string) (*domain.Summary, error)
}

// MemoryService provides semantic memory storage and vector search.
type MemoryService interface {
	ListByCharacter(ctx context.Context, charID string) ([]*domain.CharacterMemory, error)
	Search(ctx context.Context, storyID, characterID, query string, limit int) ([]*domain.CharacterMemory, error)
}

// BibleService manages the generated story bible document.
type BibleService interface {
	Get(ctx context.Context, storyID string) (*domain.StoryBible, error)
	Generate(ctx context.Context, storyID string) (*domain.StoryBible, error)
	Update(ctx context.Context, bible *domain.StoryBible) error
	DeleteByStory(ctx context.Context, storyID string) error
	LinkBibleToStory(ctx context.Context, bibleID, targetStoryID string) error
	UnlinkBibleFromStory(ctx context.Context, bibleID, targetStoryID string) error
	ListReferencingBibles(ctx context.Context, storyID string) ([]*domain.StoryBible, error)
}

// ChapterService manages chapter groupings within acts.
type ChapterService interface {
	Create(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error)
	Get(ctx context.Context, id string) (*domain.Chapter, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Chapter, error)
	ListByAct(ctx context.Context, storyID string, actNumber int) ([]*domain.Chapter, error)
	Update(ctx context.Context, c *domain.Chapter) (*domain.Chapter, error)
	Delete(ctx context.Context, id string) error
}

// AgentService provides agent-based scene generation and turn management.
type AgentService interface {
	GetTurns(ctx context.Context, sceneID string) ([]*domain.SceneTurn, error)
	GetTurnsByRole(ctx context.Context, sceneID, role string) ([]*domain.SceneTurn, error)
	GetCanonDeltas(ctx context.Context, sceneID string) ([]*domain.CanonDelta, error)
	RecordStateDelta(ctx context.Context, d *domain.CanonDelta) error
}

// CharAgentService provides runtime character agent inspection + control.
type CharAgentService interface {
	GetAgentState(ctx context.Context, charID string) (*agents.AgentStateSnapshot, error)
	BroadcastEvent(ctx context.Context, storyID, eventType string, data map[string]any) error
	GetProposals(ctx context.Context, sceneID string) ([]agents.ProposalSnapshot, error)
}

// LocationService manages story locations (hierarchical, named places).
type LocationService interface {
	Create(ctx context.Context, loc *domain.Location) error
	Get(ctx context.Context, id string) (*domain.Location, error)
	ListByStory(ctx context.Context, storyID string) ([]*domain.Location, error)
	Update(ctx context.Context, loc *domain.Location) error
	DeleteByStory(ctx context.Context, storyID string) error
	GetByName(ctx context.Context, storyID, name string) (*domain.Location, error)
}

// MetricsService aggregates LLM token usage and cost data for observability.
type MetricsService interface {
	GetLlmMetrics(ctx context.Context, storyID string) (*domain.LlmMetrics, error)
}

// CriticScoresService provides access to critic evaluation scores for generations.
type CriticScoresService interface {
	ListByStory(ctx context.Context, storyID string) ([]domain.CriticScoreEntry, error)
}

// AgentConfigService manages user-defined agent configurations.
type AgentConfigService interface {
	Create(ctx context.Context, cfg *domain.AgentConfig) error
	Get(ctx context.Context, name string) (*domain.AgentConfig, error)
	List(ctx context.Context) ([]*domain.AgentConfig, error)
	Export(ctx context.Context, name string) (*domain.AgentConfig, error)
	Import(ctx context.Context, cfg *domain.AgentConfig) error
	Delete(ctx context.Context, name string) error
	ListShared(ctx context.Context) ([]*domain.AgentConfig, error)
}

// Handlers groups all HTTP handler methods and their injected service dependencies.
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
	metricsSvc   MetricsService
	criticSvc    CriticScoresService
	agentCfgSvc  AgentConfigService
	progress     *ProgressHub
	eventBus     events.Bus
	agentSvc     AgentService
	charAgentSvc CharAgentService
}

// NewHandlers wires all service dependencies into a single Handlers struct.
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
	metricsSvc MetricsService,
	criticSvc CriticScoresService,
	agentCfgSvc AgentConfigService,
	progress *ProgressHub,
	eventBus events.Bus,
	agentSvc AgentService,
	charAgentSvc CharAgentService,
) *Handlers {
	return &Handlers{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, genWriteSvc: genSvc, genReadSvc: genReadSvc, tlSvc: tlSvc,
		sumSvc: sumSvc, memSvc: memSvc, locSvc: locSvc,
		bibleSvc: bibleSvc, chapterSvc: chapterSvc,
		outlineSvc: outlineSvc, titleSvc: titleSvc, metricsSvc: metricsSvc,
		criticSvc: criticSvc, agentCfgSvc: agentCfgSvc,
		progress: progress, eventBus: eventBus, agentSvc: agentSvc,
		charAgentSvc: charAgentSvc,
	}
}

// publishEntityEvent fires a domain event onto the event bus, if configured.
func (h *Handlers) publishEntityEvent(ctx context.Context, eventType, storyID string, data map[string]any) {
	if h.eventBus == nil {
		return
	}
	_ = h.eventBus.Publish(ctx, events.Event{
		Type:    eventType,
		StoryID: storyID,
		Data:    data,
	})
}

// writeError sends a JSON error response and logs 5xx server errors.
func writeError(w http.ResponseWriter, status int, msg string) {
	if status >= 500 {
		slog.Error("server error", "status", status, "msg", msg)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleSvcErr returns true if err is non-nil, writing 404 for service.ErrNotFound
// and 500 for other errors. Returns false when err is nil (caller proceeds).
func handleSvcErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, service.ErrNotFound) {
		writeError(w, http.StatusNotFound, err.Error())
		return true
	}
	writeError(w, http.StatusInternalServerError, err.Error())
	return true
}
