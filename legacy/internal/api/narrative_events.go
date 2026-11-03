package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) ListNarrativeEvents(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	limit := queryInt(r, "limit", 100)
	events, err := h.narrativeSvc.ListByStory(r.Context(), storyID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []*domain.NarrativeEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handlers) ListNarrativeEventsByScene(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	limit := queryInt(r, "limit", 100)
	events, err := h.narrativeSvc.ListByScene(r.Context(), nodeID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if events == nil {
		events = []*domain.NarrativeEvent{}
	}
	writeJSON(w, http.StatusOK, events)
}
