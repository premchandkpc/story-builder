package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetSummaryByLevel handles GET /api/v1/stories/{storyID}/summaries/level.
// Query param "level" defaults to "act".
func (h *Handlers) GetSummaryByLevel(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	level := r.URL.Query().Get("level")
	if level == "" {
		writeError(w, http.StatusBadRequest, "level query param is required")
		return
	}
	if level != "scene" && level != "act" && level != "story" {
		writeError(w, http.StatusBadRequest, "invalid level: must be scene, act, or story")
		return
	}

	sum, err := h.sumSvc.GetByLevel(r.Context(), storyID, level)
	if handleSvcErr(w, err) {
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// GetSceneSummary handles GET /api/v1/stories/{storyID}/summaries/scenes/{sceneID}
// and GET /api/v1/stories/{storyID}/summaries/nodes/{nodeID}.
func (h *Handlers) GetSceneSummary(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	sceneID := chi.URLParam(r, "sceneID")
	if sceneID == "" {
		sceneID = chi.URLParam(r, "nodeID")
	}

	sum, err := h.sumSvc.GetSceneSummary(r.Context(), storyID, sceneID)
	if handleSvcErr(w, err) {
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}
