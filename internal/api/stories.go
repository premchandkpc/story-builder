package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/service"
)

// CreateStory handles POST /api/v1/stories.
func (h *Handlers) CreateStory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	story, err := h.storySvc.Create(r.Context(), body.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.publishEntityEvent(r.Context(), events.EventStoryCreated, story.ID, map[string]any{"title": story.Title})
	writeJSON(w, http.StatusCreated, story)
}

// GetStory handles GET /api/v1/stories/{storyID}.
func (h *Handlers) GetStory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "storyID")
	story, err := h.storySvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if story == nil {
		writeError(w, http.StatusNotFound, "story not found")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

// UpdateStory handles PUT /api/v1/stories/{storyID}.
func (h *Handlers) UpdateStory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "storyID")
	var body struct {
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := h.storySvc.Update(r.Context(), id, service.UpdateStoryParams{
		Title:  body.Title,
		Status: body.Status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ListStories handles GET /api/v1/stories.
func (h *Handlers) ListStories(w http.ResponseWriter, r *http.Request) {
	stories, err := h.storySvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stories)
}

// DeleteStory handles DELETE /api/v1/stories/{storyID}.
func (h *Handlers) DeleteStory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "storyID")
	if err := h.storySvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GenerateStory handles POST /api/v1/stories/generate.
// Calls the LLM outline service, creates a story, characters, locations, scenes, and edges from the outline.
func (h *Handlers) GenerateStory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Synopsis string `json:"synopsis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Synopsis == "" {
		writeError(w, http.StatusBadRequest, "synopsis is required")
		return
	}

	outline, err := h.outlineSvc.GenerateOutline(r.Context(), body.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "outline generation failed: "+err.Error())
		return
	}
	title := outline.Title
	if title == "" {
		title = body.Synopsis
		if len(title) > 80 {
			title = title[:80]
		}
	}
	story, err := h.storySvc.Create(r.Context(), title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Build and store the story blueprint from the outline.
	acts := make([]domain.Act, 0)
	actMap := make(map[int]int)
	for _, b := range outline.Beats {
		if idx, ok := actMap[b.Act]; ok {
			_ = idx
		} else {
			actMap[b.Act] = len(acts)
			acts = append(acts, domain.Act{Number: b.Act})
		}
	}
	charArcs := make([]domain.CharacterArc, 0, len(outline.Characters))
	for _, c := range outline.Characters {
		charArcs = append(charArcs, domain.CharacterArc{
			CharacterName: c.Name,
			Want:          pickString(c.Goals, 0),
			ArcType:       "growth",
		})
	}
	bp := &domain.StoryBlueprint{
		Premise:       outline.Synopsis,
		Genre:         story.Genre,
		Acts:          acts,
		CharacterArcs: charArcs,
	}
	if err := h.storySvc.UpdateBlueprint(r.Context(), story.ID, bp); err != nil {
		slog.Error("generate story: store blueprint failed", "error", err)
	}

	charIDByName := make(map[string]string, len(outline.Beats))
	for _, c := range outline.Characters {
		personality := make(map[string]any)
		if len(c.Personality) > 0 {
			personality["traits"] = c.Personality
		}
		char := &domain.Character{
			StoryID:        story.ID,
			Name:           c.Name,
			Persona:        c.Persona,
			Backstory:      c.Backstory,
			MoralAlignment: c.MoralAlignment,
			Personality:    personality,
			Goals:          c.Goals,
			Flaws:          c.Flaws,
			Traits:         c.Personality,
			VoiceSamples:   c.VoiceSamples,
		}
		created, err := h.charSvc.Create(r.Context(), char)
		if err != nil {
			slog.Error("generate story: create character failed", "name", c.Name, "error", err)
			continue
		}
		charIDByName[c.Name] = created.ID
	}

	locIDByName := make(map[string]string)
	beatIDByTitle := make(map[string]string, len(outline.Beats))
	for i, b := range outline.Beats {
		if b.LocationName != "" {
			if _, ok := locIDByName[b.LocationName]; !ok {
				loc := &domain.Location{
					StoryID: story.ID,
					Name:    b.LocationName,
				}
				if err := h.locSvc.Create(r.Context(), loc); err != nil {
					slog.Error("generate story: create location failed", "name", b.LocationName, "error", err)
				} else {
					locIDByName[b.LocationName] = loc.ID
				}
			}
		}
		locRef := b.LocationName
		if id, ok := locIDByName[b.LocationName]; ok {
			locRef = id
		}
		scene := &domain.Scene{
			StoryID:          story.ID,
			Title:            b.Title,
			BeatIntent:       b.BeatIntent,
			POV:              b.POV,
			Tone:             b.Tone,
			TargetWords:      b.TargetWords,
			LocationRef:      locRef,
			TimelinePosition: i + 1,
		}
		for _, cn := range b.CharacterNames {
			if id, ok := charIDByName[cn]; ok {
				scene.Participants = append(scene.Participants, id)
			}
		}
		created, err := h.sceneSvc.Create(r.Context(), scene)
		if err != nil {
			slog.Error("generate story: create scene failed", "beat", b.Title, "error", err)
			continue
		}
		beatIDByTitle[b.Title] = created.ID
	}

	for _, e := range outline.Edges {
		fromID, ok1 := beatIDByTitle[e.From]
		toID, ok2 := beatIDByTitle[e.To]
		if !ok1 || !ok2 {
			continue
		}
		edgeType := e.Type
		if edgeType == "" {
			edgeType = "seq"
		}
		_, _ = h.edgeSvc.Create(r.Context(), &domain.SceneEdge{
			StoryID:     story.ID,
			FromSceneID: fromID,
			ToSceneID:   toID,
			Type:        edgeType,
		})
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"story_id": story.ID,
		"status":   "outlined",
	})
}

// GenerateTitle handles POST /api/v1/stories/generate-title.
// Uses the LLM title service to generate a title from a synopsis.
func (h *Handlers) GenerateTitle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Synopsis string `json:"synopsis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Synopsis == "" {
		writeError(w, http.StatusBadRequest, "synopsis is required")
		return
	}
	title, err := h.titleSvc.GenerateTitle(r.Context(), body.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "title generation failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"title": title})
}

// pickString returns s[idx] if in bounds, empty string otherwise.
func pickString(s []string, idx int) string {
	if idx < len(s) {
		return s[idx]
	}
	return ""
}

// GetBlueprint handles GET /api/v1/stories/{storyID}/blueprint.
func (h *Handlers) GetBlueprint(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	bp, err := h.storySvc.GetBlueprint(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if bp == nil {
		writeJSON(w, http.StatusOK, map[string]string{"blueprint": "none"})
		return
	}
	writeJSON(w, http.StatusOK, bp)
}

// UpdateBlueprint handles PUT /api/v1/stories/{storyID}/blueprint.
func (h *Handlers) UpdateBlueprint(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var bp domain.StoryBlueprint
	if err := json.NewDecoder(r.Body).Decode(&bp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.storySvc.UpdateBlueprint(r.Context(), storyID, &bp); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
