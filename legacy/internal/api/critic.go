package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) ListCriticScores(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	if h.criticSvc == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	scores, err := h.criticSvc.ListByStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scores)
}
