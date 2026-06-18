package api

import (
	"net/http"

	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/service"
)

type Handlers struct {
	storySvc   *service.StoryService
	sceneSvc   *service.SceneService
	edgeSvc    *service.EdgeService
	charSvc    *service.CharacterService
	genSvc     *service.GenerationService
	tlSvc      *service.TimelineService
	sumSvc     *service.SummaryService
	memSvc     *service.MemoryService
	locSvc     *service.LocationService
	outlineSvc *llm.OutlineServiceImpl
}

func NewHandlers(
	storySvc *service.StoryService,
	sceneSvc *service.SceneService,
	edgeSvc *service.EdgeService,
	charSvc *service.CharacterService,
	genSvc *service.GenerationService,
	tlSvc *service.TimelineService,
	sumSvc *service.SummaryService,
	memSvc *service.MemoryService,
	locSvc *service.LocationService,
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
