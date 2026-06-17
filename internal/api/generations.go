package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handlers) GenerateScene(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "sceneID")

	gen, err := h.genSvc.Generate(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

func (h *Handlers) ListGenerations(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "sceneID")
	gens, err := h.genSvc.ListGenerations(r.Context(), sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gens)
}

func (h *Handlers) AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	sceneID := chi.URLParam(r, "sceneID")
	var body struct {
		GenerationID string `json:"generation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.genSvc.AcceptGeneration(r.Context(), sceneID, body.GenerationID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *Handlers) V2GenerateNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gen, err := h.genSvc.Generate(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

func (h *Handlers) V2ListNodeGenerations(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gens, err := h.genSvc.ListGenerations(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gens)
}

func (h *Handlers) V2AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	var body struct {
		GenerationID string `json:"generation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.genSvc.AcceptGeneration(r.Context(), nodeID, body.GenerationID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
