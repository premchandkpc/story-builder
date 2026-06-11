package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/narrative"
)

type BlueprintService interface {
	Save(storyID uuid.UUID, bp *narrative.Blueprint) error
	Get(storyID uuid.UUID) (*narrative.Blueprint, error)
}

func (h *StoryHandler) UpsertBlueprint(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	if h.BlueprintService == nil {
		writeError(w, http.StatusServiceUnavailable, "blueprint service unavailable")
		return
	}
	if h.Service != nil {
		if _, err := h.Service.Get(storyID); err != nil {
			writeError(w, http.StatusNotFound, "story not found")
			return
		}
	}

	var req narrative.Blueprint
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.BlueprintService.Save(storyID, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (h *StoryHandler) GetBlueprint(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	if h.BlueprintService == nil {
		writeError(w, http.StatusServiceUnavailable, "blueprint service unavailable")
		return
	}
	bp, err := h.BlueprintService.Get(storyID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bp)
}
