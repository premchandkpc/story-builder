package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GetLlmMetrics(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	if h.metricsSvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"total_prompt_tokens":     0,
			"total_completion_tokens": 0,
			"total_tokens":            0,
			"total_cost_estimate":     0,
			"turn_count":              0,
			"generation_count":        0,
			"by_model":                map[string]any{},
			"by_agent":                map[string]any{},
		})
		return
	}
	metrics, err := h.metricsSvc.GetLlmMetrics(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}
