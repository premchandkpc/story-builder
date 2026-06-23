package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/service"
)

type Handlers struct {
	storySvc      StoryService
	sceneSvc      SceneService
	edgeSvc       EdgeService
	charSvc       CharacterService
	genWriteSvc   GenerationWriteService
	genReadSvc    GenerationReadService
	tlSvc         TimelineService
	sumSvc        SummaryService
	memSvc        MemoryService
	locSvc        LocationService
	bibleSvc      BibleService
	chapterSvc    ChapterService
	outlineSvc    llm.OutlineService
	titleSvc      llm.TitleService
	metricsSvc    MetricsService
	criticSvc     CriticScoresService
	agentCfgSvc   AgentConfigService
	progress      *ProgressHub
	eventBus      events.Bus
	agentSvc      AgentService
	charAgentSvc  CharAgentService
	runSvc        RunService
	narrativeSvc  NarrativeEventService
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
	metricsSvc MetricsService,
	criticSvc CriticScoresService,
	agentCfgSvc AgentConfigService,
	progress *ProgressHub,
	eventBus events.Bus,
	agentSvc AgentService,
	charAgentSvc CharAgentService,
	runSvc RunService,
	narrativeSvc NarrativeEventService,
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
		runSvc: runSvc, narrativeSvc: narrativeSvc,
	}
}

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

func writeError(w http.ResponseWriter, status int, msg string) {
	if status >= 500 {
		slog.Error("server error", "status", status, "msg", msg)
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

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
