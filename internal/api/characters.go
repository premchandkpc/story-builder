package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

func (h *Handlers) CreateCharacter(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var char domain.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if char.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	char.StoryID = storyID

	created, err := h.charSvc.Create(r.Context(), &char)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) GetCharacter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "charID")
	char, err := h.charSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if char == nil {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	writeJSON(w, http.StatusOK, char)
}

func (h *Handlers) ListCharacters(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	chars, err := h.charSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chars)
}

func (h *Handlers) V2ListCharacters(w http.ResponseWriter, r *http.Request) {
	storyID := r.URL.Query().Get("story_id")
	if storyID == "" {
		writeJSON(w, http.StatusOK, []*domain.Character{})
		return
	}
	chars, err := h.charSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if chars == nil {
		chars = []*domain.Character{}
	}
	writeJSON(w, http.StatusOK, chars)
}

func (h *Handlers) V2CreateCharacter(w http.ResponseWriter, r *http.Request) {
	var char domain.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if char.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	created, err := h.charSvc.Create(r.Context(), &char)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) V2GetCharacter(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "charID")
	char, err := h.charSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if char == nil {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	writeJSON(w, http.StatusOK, char)
}

func (h *Handlers) V2UpdateCharacter(w http.ResponseWriter, r *http.Request) {
	charID := chi.URLParam(r, "charID")
	existing, err := h.charSvc.GetLatest(r.Context(), charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if existing == nil {
		writeError(w, http.StatusNotFound, "character not found")
		return
	}
	var body struct {
		Name           string            `json:"name,omitempty"`
		Persona        string            `json:"persona,omitempty"`
		Backstory      string            `json:"backstory,omitempty"`
		Personality    map[string]any    `json:"personality,omitempty"`
		MoralAlignment string            `json:"moralAlignment,omitempty"`
		Goals          []string          `json:"goals,omitempty"`
		Flaws          []string          `json:"flaws,omitempty"`
		Traits         []string          `json:"traits,omitempty"`
		VoiceSamples   []string          `json:"voiceSamples,omitempty"`
		Relationships  map[string]string `json:"relationships,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated := *existing
	if body.Name != "" {
		updated.Name = body.Name
	}
	if body.Persona != "" {
		updated.Persona = body.Persona
	}
	if body.Backstory != "" {
		updated.Backstory = body.Backstory
	}
	if body.Personality != nil {
		updated.Personality = body.Personality
	}
	if body.MoralAlignment != "" {
		updated.MoralAlignment = body.MoralAlignment
	}
	if body.Goals != nil {
		updated.Goals = body.Goals
	}
	if body.Flaws != nil {
		updated.Flaws = body.Flaws
	}
	if body.Traits != nil {
		updated.Traits = body.Traits
	}
	if body.VoiceSamples != nil {
		updated.VoiceSamples = body.VoiceSamples
	}
	if body.Relationships != nil {
		updated.Relationships = body.Relationships
	}
	result, err := h.charSvc.Update(r.Context(), &updated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
