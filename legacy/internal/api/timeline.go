package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

// CreateTimelineEvent handles POST /api/v1/stories/{storyID}/timeline.
func (h *Handlers) CreateTimelineEvent(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var evt domain.TimelineEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if evt.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	evt.StoryID = storyID

	created, err := h.tlSvc.Create(r.Context(), &evt)
	if handleSvcErr(w, err) {
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// ListTimelineEvents handles GET /api/v1/stories/{storyID}/timeline.
func (h *Handlers) ListTimelineEvents(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	events, err := h.tlSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}
