package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/llm"
	"github.com/premchand/story-builder/internal/service"
)

type Handlers struct {
	storySvc   *service.StoryService
	sceneSvc   *service.SceneService
	edgeSvc    *service.EdgeService
	charSvc    *service.CharacterService
	genSvc     *service.GenerationService
	tlSvc      *service.TimelineService
	sumSvc     *service.SummaryService
	memSvc     *service.MemoryService
	outlineSvc *llm.OutlineServiceImpl
}

func NewHandlers(
	storySvc *service.StoryService,
	sceneSvc *service.SceneService,
	edgeSvc *service.EdgeService,
	charSvc *service.CharacterService,
	genSvc *service.GenerationService,
	tlSvc *service.TimelineService,
	sumSvc *service.SummaryService,
	memSvc *service.MemoryService,
	outlineSvc *llm.OutlineServiceImpl,
) *Handlers {
	return &Handlers{
		storySvc: storySvc, sceneSvc: sceneSvc, edgeSvc: edgeSvc,
		charSvc: charSvc, genSvc: genSvc, tlSvc: tlSvc,
		sumSvc: sumSvc, memSvc: memSvc, outlineSvc: outlineSvc,
	}
}

// ─── Stories ───────────────────────────────────────────────

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
	writeJSON(w, http.StatusCreated, story)
}

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

func (h *Handlers) ListStories(w http.ResponseWriter, r *http.Request) {
	stories, err := h.storySvc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stories)
}

func (h *Handlers) DeleteStory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "storyID")
	if err := h.storySvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Scenes ────────────────────────────────────────────────

func (h *Handlers) CreateScene(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var scene domain.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene.StoryID = storyID

	created, err := h.sceneSvc.Create(r.Context(), &scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) GetScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	scene, err := h.sceneSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scene == nil {
		writeError(w, http.StatusNotFound, "scene not found")
		return
	}
	writeJSON(w, http.StatusOK, scene)
}

func (h *Handlers) UpdateScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	var scene domain.Scene
	if err := json.NewDecoder(r.Body).Decode(&scene); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene.ID = id

	updated, err := h.sceneSvc.Update(r.Context(), &scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) ListScenes(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	scenes, err := h.sceneSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scenes)
}

func (h *Handlers) DeleteScene(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "sceneID")
	if err := h.sceneSvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) Topology(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	scenes, edges, err := h.sceneSvc.Topology(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scenes == nil {
		scenes = []*domain.Scene{}
	}
	if edges == nil {
		edges = []*domain.SceneEdge{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": scenes,
		"edges": edges,
	})
}

// ─── Edges ─────────────────────────────────────────────────

func (h *Handlers) CreateEdge(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		FromScene string `json:"fromSceneId"`
		ToScene   string `json:"toSceneId"`
		Type      string `json:"type"`
		Condition string `json:"condition,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	edge := &domain.SceneEdge{
		StoryID:     storyID,
		FromSceneID: body.FromScene,
		ToSceneID:   body.ToScene,
		Type:        body.Type,
		Condition:   body.Condition,
	}

	created, err := h.edgeSvc.Create(r.Context(), edge)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) ListEdges(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	edges, err := h.edgeSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

func (h *Handlers) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		FromScene string `json:"from_scene"`
		ToScene   string `json:"to_scene"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.edgeSvc.Delete(r.Context(), storyID, body.FromScene, body.ToScene); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Characters ────────────────────────────────────────────

func (h *Handlers) CreateCharacter(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var char domain.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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

// ─── Generation ────────────────────────────────────────────

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

// ─── Timeline ──────────────────────────────────────────────

func (h *Handlers) CreateTimelineEvent(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var evt domain.TimelineEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	evt.StoryID = storyID

	created, err := h.tlSvc.Create(r.Context(), &evt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handlers) ListTimelineEvents(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	events, err := h.tlSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// ─── Summaries ─────────────────────────────────────────────

func (h *Handlers) GetSummaryByLevel(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "act"
	}

	sum, err := h.sumSvc.GetByLevel(r.Context(), storyID, level)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (h *Handlers) GetSceneSummary(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	sceneID := chi.URLParam(r, "sceneID")

	sum, err := h.sumSvc.GetSceneSummary(r.Context(), storyID, sceneID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sum == nil {
		writeError(w, http.StatusNotFound, "summary not found")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// ─── Memories ──────────────────────────────────────────────

func (h *Handlers) ListMemories(w http.ResponseWriter, r *http.Request) {
	charID := chi.URLParam(r, "charID")
	mems, err := h.memSvc.ListByCharacter(r.Context(), charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, mems)
}

func (h *Handlers) SearchMemories(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "vector search not yet wired")
}
