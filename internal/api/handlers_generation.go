package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type LoreHandler struct {
	Service LoreService
}

type createLoreRequest struct {
	Tags    []string `json:"tags"`
	Content string   `json:"content"`
}

type searchLoreRequest struct {
	Tags      []string  `json:"tags"`
	Embedding []float32 `json:"embedding,omitempty"`
	Limit     int       `json:"limit"`
}

func (h *LoreHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	lore, err := h.Service.Create(r.Context(), req.Tags, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, lore)
}

func (h *LoreHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *LoreHandler) Search(w http.ResponseWriter, r *http.Request) {
	var req searchLoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Embedding) > 0 {
		results, err := h.Service.SearchSimilar(r.Context(), req.Embedding, req.Limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)
		return
	}
	results, err := h.Service.SearchByTags(r.Context(), req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

type GenerationHandler struct {
	Service GenerationService
}

type acceptGenerationRequest struct {
	GenerationID string `json:"generation_id"`
}

func (h *GenerationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	gen, err := h.Service.Generate(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

func (h *GenerationHandler) AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	var req acceptGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	genID, err := parseUUID(req.GenerationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid generation_id")
		return
	}
	if err := h.Service.AcceptGeneration(r.Context(), nodeID, genID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *GenerationHandler) ListGenerations(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	gens, err := h.Service.ListGenerations(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gens)
}

type SceneHandler struct {
	SceneService scene.SceneService
}

type setSceneStructureRequest struct {
	SceneStructure graph.SceneStructure `json:"scene_structure"`
}

func (h *SceneHandler) SetStructure(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	var req setSceneStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.SceneService.SetSceneStructure(r.Context(), nodeID, req.SceneStructure); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SceneHandler) GetStructure(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	ss, err := h.SceneService.GetSceneStructure(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ss)
}

func (h *SceneHandler) Start(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turn, err := h.SceneService.StartScene(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turn)
}

func (h *SceneHandler) Next(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turn, err := h.SceneService.NextTurn(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turn)
}

func (h *SceneHandler) Finish(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	output, err := h.SceneService.FinishScene(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": output})
}

func (h *SceneHandler) Turns(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turns, err := h.SceneService.GetTurns(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turns)
}

type SummaryHandler struct {
	Service compiler.SummaryService
}

func (h *SummaryHandler) GetSceneSummary(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	summary, err := h.Service.GetSceneSummary(r.Context(), storyID, nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *SummaryHandler) GetByLevel(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	if level != compiler.SummaryAct && level != compiler.SummaryStory {
		level = compiler.SummaryAct
	}
	summary, err := h.Service.GetSummaryByLevel(r.Context(), storyID, level)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *SummaryHandler) CountByLevel(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	if level != compiler.SummaryScene && level != compiler.SummaryAct && level != compiler.SummaryStory {
		level = compiler.SummaryScene
	}
	count, err := h.Service.CountSummariesByLevel(r.Context(), storyID, level)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *SummaryHandler) ShouldElevate(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	threshold := 10
	if t := r.URL.Query().Get("threshold"); t != "" {
		if v, err := parseInt(t); err == nil {
			threshold = v
		}
	}
	should, err := h.Service.ShouldElevate(r.Context(), storyID, level, threshold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"should_elevate": should,
		"level":          level,
		"threshold":      threshold,
	})
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
