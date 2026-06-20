package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// GetGenerationStatus handles GET /api/v1/generations/{genID}/status.
func (h *Handlers) GetGenerationStatus(w http.ResponseWriter, r *http.Request) {
	genID := chi.URLParam(r, "genID")
	gen, err := h.genReadSvc.GetGeneration(r.Context(), genID)
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
