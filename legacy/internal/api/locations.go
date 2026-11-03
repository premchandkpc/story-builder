package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/premchand/story-builder/internal/domain"
)

// ListStoryLocations handles GET /api/v1/stories/{storyID}/locations.
func (h *Handlers) ListStoryLocations(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	locs, err := h.locSvc.ListByStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, locs)
}

// CreateLocation handles POST /api/v1/stories/{storyID}/locations.
func (h *Handlers) CreateLocation(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Props       []string `json:"props"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	loc := &domain.Location{
		StoryID:     storyID,
		Name:        body.Name,
		Description: body.Description,
		Props:       body.Props,
	}
	if err := h.locSvc.Create(r.Context(), loc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "location name already exists in this story")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, loc)
}

// GetLocation handles GET /api/v1/locations/{id}.
func (h *Handlers) GetLocation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	loc, err := h.locSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

// UpdateLocation handles PUT /api/v1/locations/{id}.
// Name is not currently mutable via this endpoint.
func (h *Handlers) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Description string   `json:"description"`
		Props       []string `json:"props"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	loc, err := h.locSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if loc == nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}
	if body.Description != "" {
		loc.Description = body.Description
	}
	if body.Props != nil {
		loc.Props = body.Props
	}
	if err := h.locSvc.Update(r.Context(), loc); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, loc)
}
