package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) CreateScene(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var scene domain.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene.StoryID = storyID

	created, err := h.sceneSvc.Create(r.Context(), &scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) GetScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	scene, err := h.sceneSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scene == nil {
		writeError(w, http.StatusNotFound, "scene not found")
		return
	}
	writeJSON(w, http.StatusOK, scene)
}

func (h *Handlers) UpdateScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	var scene domain.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene.ID = id

	updated, err := h.sceneSvc.Update(r.Context(), &scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) ListScenes(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	scenes, err := h.sceneSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scenes)
}

func (h *Handlers) DeleteScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	if err := h.sceneSvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	genID := chi.URLParam(r, "genID")
	gen, err := h.genSvc.GetGeneration(r.Context(), genID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if gen == nil {
		writeError(w, http.StatusNotFound, "generation not found")
		return
	}
	writeJSON(w, http.StatusOK, gen)
}


