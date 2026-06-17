package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) ListMemories(w http.ResponseWriter, r *http.Request) {
	charID := chi.URLParam(r, "charID")
	mems, err := h.memSvc.ListByCharacter(r.Context(), charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mems == nil {
		mems = []*domain.CharacterMemory{}
	}
	writeJSON(w, http.StatusOK, mems)
}

func (h *Handlers) SearchMemories(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "vector search not yet wired")
}
