package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

// GetBible handles GET /api/v1/stories/{storyID}/bible.
func (h *Handlers) GetBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	bible, err := h.bibleSvc.Get(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bible == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no bible found for this story"})
		return
	}
	writeJSON(w, http.StatusOK, bible)
}

// GenerateBible handles POST /api/v1/stories/{storyID}/bible/generate.
// Triggers LLM bible generation. Returns immediately with 202; generation is async.
func (h *Handlers) GenerateBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")

	bible, err := h.bibleSvc.Generate(r.Context(), storyID)
	if err != nil {
		slog.Error("generate bible", "storyId", storyID, "error", err)
		writeError(w, http.StatusInternalServerError, "bible generation failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, bible)
}

// DeleteBible handles DELETE /api/v1/stories/{storyID}/bible.
func (h *Handlers) DeleteBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	if err := h.bibleSvc.DeleteByStory(r.Context(), storyID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateBible handles PUT /api/v1/stories/{storyID}/bible.
func (h *Handlers) UpdateBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var bible domain.StoryBible
	if err := json.NewDecoder(r.Body).Decode(&bible); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	bible.StoryID = storyID
	if err := h.bibleSvc.Update(r.Context(), &bible); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, bible)
}
