package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) LinkBibleToStory(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		BibleID string `json:"bibleId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.bibleSvc.LinkBibleToStory(r.Context(), body.BibleID, storyID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

func (h *Handlers) UnlinkBibleFromStory(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		BibleID string `json:"bibleId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if err := h.bibleSvc.UnlinkBibleFromStory(r.Context(), body.BibleID, storyID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

func (h *Handlers) ListReferencingBibles(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	bibles, err := h.bibleSvc.ListReferencingBibles(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bibles)
}

func (h *Handlers) CreateCrossStoryEvent(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var e domain.TimelineEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	e.StoryID = storyID
	result, err := h.tlSvc.CreateCrossStory(r.Context(), &e)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handlers) ListCrossStoryEvents(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	events, err := h.tlSvc.ListCrossStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handlers) MigrateCharacter(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	charID := chi.URLParam(r, "charID")
	result, err := h.charSvc.MigrateCharacter(r.Context(), charID, storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
