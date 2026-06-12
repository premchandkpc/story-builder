package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/timeline"
)

func (h *StoryHandler) UpsertTimelineEvent(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	if h.TimelineService == nil {
		writeError(w, http.StatusServiceUnavailable, "timeline service unavailable")
		return
	}
	if h.Service != nil {
		if _, err := h.Service.Get(r.Context(), storyID); err != nil {
			writeError(w, http.StatusNotFound, "story not found")
			return
		}
	}
	var event timeline.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.TimelineService.Save(r.Context(), storyID, &event); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *StoryHandler) ListTimelineEvents(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	if h.TimelineService == nil {
		writeError(w, http.StatusServiceUnavailable, "timeline service unavailable")
		return
	}
	events, err := h.TimelineService.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}
