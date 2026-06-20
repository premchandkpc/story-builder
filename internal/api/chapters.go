package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

// ListChapters handles GET /api/v1/stories/{storyID}/chapters.
func (h *Handlers) ListChapters(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	chapters, err := h.chapterSvc.ListByStory(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chapters)
}

// CreateChapter handles POST /api/v1/stories/{storyID}/chapters.
func (h *Handlers) CreateChapter(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		ActNumber  int    `json:"actNumber"`
		ChapterNum int    `json:"chapterNumber"`
		Title      string `json:"title"`
		Summary    string `json:"summary"`
		Goal       string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	chapter := &domain.Chapter{
		StoryID:    storyID,
		ActNumber:  body.ActNumber,
		ChapterNum: body.ChapterNum,
		Title:      body.Title,
		Summary:    body.Summary,
		Goal:       body.Goal,
		Status:     domain.ChapterStatusPlanned,
	}
	created, err := h.chapterSvc.Create(r.Context(), chapter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GetChapter handles GET /api/v1/stories/{storyID}/chapters/{chapterID}.
func (h *Handlers) GetChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chapterID")
	chapter, err := h.chapterSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chapter == nil {
		writeError(w, http.StatusNotFound, "chapter not found")
		return
	}
	writeJSON(w, http.StatusOK, chapter)
}

// DeleteChapter handles DELETE /api/v1/stories/{storyID}/chapters/{chapterID}.
func (h *Handlers) DeleteChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chapterID")
	if err := h.chapterSvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UpdateChapter handles PUT /api/v1/stories/{storyID}/chapters/{chapterID}.
func (h *Handlers) UpdateChapter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "chapterID")
	var body domain.Chapter
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body.ID = id
	updated, err := h.chapterSvc.Update(r.Context(), &body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
