package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GetSummaryByLevel(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "act"
	}

	sum, err := h.sumSvc.GetByLevel(r.Context(), storyID, level)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (h *Handlers) GetSceneSummary(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	sceneID := chi.URLParam(r, "sceneID")

	sum, err := h.sumSvc.GetSceneSummary(r.Context(), storyID, sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}
