package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/premchand/story-builder/internal/domain"
)

// V2GenerateNode handles POST /api/v1/stories/{storyID}/nodes/{nodeID}/generate.
func (h *Handlers) V2GenerateNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gen, err := h.genWriteSvc.Generate(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

// V2ListNodeGenerations handles GET /api/v1/stories/{storyID}/nodes/{nodeID}/generations.
func (h *Handlers) V2ListNodeGenerations(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gens, err := h.genReadSvc.ListGenerations(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if gens == nil {
		gens = []*domain.Generation{}
	}
	writeJSON(w, http.StatusOK, gens)
}

// V2AcceptGeneration handles POST /api/v1/stories/{storyID}/nodes/{nodeID}/accept.
func (h *Handlers) V2AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	var body struct {
		GenerationID string `json:"generation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.genWriteSvc.AcceptGeneration(r.Context(), nodeID, body.GenerationID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}
