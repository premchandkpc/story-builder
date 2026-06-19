package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

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

func (h *Handlers) DeleteBible(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	if err := h.bibleSvc.DeleteByStory(r.Context(), storyID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) UpdateBible(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "storyID")
	var body struct {
		Bible json.RawMessage `json:"bible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "not_implemented"})
}
