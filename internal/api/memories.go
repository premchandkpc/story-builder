package api

import (
	"encoding/json"
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

type searchMemoriesRequest struct {
	StoryID string `json:"story_id"`
	Query   string `json:"query"`
	Limit   int    `json:"limit,omitempty"`
}

func (h *Handlers) SearchMemories(w http.ResponseWriter, r *http.Request) {
	charID := chi.URLParam(r, "charID")

	var req searchMemoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.StoryID == "" {
		writeError(w, http.StatusBadRequest, "story_id is required")
		return
	}
	if req.Query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	mems, err := h.memSvc.Search(r.Context(), req.StoryID, charID, req.Query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if mems == nil {
		mems = []*domain.CharacterMemory{}
	}
	writeJSON(w, http.StatusOK, mems)
}
