package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

type graphNode struct {
	ID             string         `json:"id"`
	StoryID        string         `json:"story_id"`
	BeatIntent     string         `json:"beat_intent"`
	CharacterRefs  []string       `json:"character_refs"`
	LocationRef    string         `json:"location_ref"`
	POV            string         `json:"pov"`
	Tone           string         `json:"tone"`
	TargetWords    int            `json:"target_words"`
	Status         string         `json:"status"`
	SceneStructure map[string]any `json:"scene_structure"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type graphEdge struct {
	StoryID  string `json:"story_id"`
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	EdgeType string `json:"edge_type"`
}

type topologyResponse struct {
	Nodes            []graphNode `json:"nodes"`
	Edges            []graphEdge `json:"edges"`
	TopologicalOrder []string    `json:"topological_order"`
}

func sceneToNode(s *domain.Scene) graphNode {
	return graphNode{
		ID:             s.ID,
		StoryID:        s.StoryID,
		BeatIntent:     s.BeatIntent,
		CharacterRefs:  s.Participants,
		LocationRef:    s.LocationRef,
		POV:            s.POV,
		Tone:           s.Tone,
		TargetWords:    s.TargetWords,
		Status:         s.Status,
		SceneStructure: s.SceneStructure,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

func edgeToGraphEdge(e *domain.SceneEdge) graphEdge {
	return graphEdge{
		StoryID:  e.StoryID,
		FromNode: e.FromSceneID,
		ToNode:   e.ToSceneID,
		EdgeType: e.Type,
	}
}

func (h *Handlers) ListNodes(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	scenes, err := h.sceneSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	nodes := make([]graphNode, 0, len(scenes))
	for _, s := range scenes {
		nodes = append(nodes, sceneToNode(s))
	}
	writeJSON(w, http.StatusOK, nodes)
}

func (h *Handlers) GetNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeID")
	scene, err := h.sceneSvc.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if scene == nil {
		writeError(w, http.StatusNotFound, "node not found")
		return
	}
	writeJSON(w, http.StatusOK, sceneToNode(scene))
}

func (h *Handlers) CreateNode(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		BeatIntent     string         `json:"beat_intent"`
		CharacterRefs  []string       `json:"character_refs"`
		LocationRef    string         `json:"location_ref"`
		POV            string         `json:"pov"`
		Tone           string         `json:"tone"`
		TargetWords    int            `json:"target_words"`
		SceneStructure map[string]any `json:"scene_structure"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene := &domain.Scene{
		StoryID:        storyID,
		BeatIntent:     body.BeatIntent,
		Participants:   body.CharacterRefs,
		LocationRef:    body.LocationRef,
		POV:            body.POV,
		Tone:           body.Tone,
		TargetWords:    body.TargetWords,
		SceneStructure: body.SceneStructure,
	}
	created, err := h.sceneSvc.Create(r.Context(), scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sceneToNode(created))
}

func (h *Handlers) UpdateNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeID")
	var body struct {
		BeatIntent     string         `json:"beat_intent"`
		CharacterRefs  []string       `json:"character_refs"`
		LocationRef    string         `json:"location_ref"`
		POV            string         `json:"pov"`
		Tone           string         `json:"tone"`
		TargetWords    int            `json:"target_words"`
		SceneStructure map[string]any `json:"scene_structure"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	scene := &domain.Scene{
		ID:             id,
		BeatIntent:     body.BeatIntent,
		Participants:   body.CharacterRefs,
		LocationRef:    body.LocationRef,
		POV:            body.POV,
		Tone:           body.Tone,
		TargetWords:    body.TargetWords,
		SceneStructure: body.SceneStructure,
	}
	updated, err := h.sceneSvc.Update(r.Context(), scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sceneToNode(updated))
}

func (h *Handlers) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeID")
	if err := h.sceneSvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) V2Topology(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	scenes, edges, err := h.sceneSvc.Topology(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	nodes := make([]graphNode, 0, len(scenes))
	for _, s := range scenes {
		nodes = append(nodes, sceneToNode(s))
	}
	ge := make([]graphEdge, 0, len(edges))
	for _, e := range edges {
		ge = append(ge, edgeToGraphEdge(e))
	}
	writeJSON(w, http.StatusOK, topologyResponse{
		Nodes:            nodes,
		Edges:            ge,
		TopologicalOrder: extractIDs(scenes),
	})
}

func (h *Handlers) V2CreateEdge(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		FromNode string `json:"from_node"`
		ToNode   string `json:"to_node"`
		EdgeType string `json:"edge_type"`
		Condition string `json:"condition,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	edge := &domain.SceneEdge{
		StoryID:     storyID,
		FromSceneID: body.FromNode,
		ToSceneID:   body.ToNode,
		Type:        body.EdgeType,
		Condition:   body.Condition,
	}
	created, err := h.edgeSvc.Create(r.Context(), edge)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, edgeToGraphEdge(created))
}

func (h *Handlers) V2ListEdges(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	edges, err := h.edgeSvc.List(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	ge := make([]graphEdge, 0, len(edges))
	for _, e := range edges {
		ge = append(ge, edgeToGraphEdge(e))
	}
	writeJSON(w, http.StatusOK, ge)
}

func (h *Handlers) V2ListCharacters(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []*domain.Character{})
}

func (h *Handlers) V2CreateCharacter(w http.ResponseWriter, r *http.Request) {
	var char domain.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
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
	writeError(w, http.StatusNotImplemented, "not implemented")
}

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

	charIDByName := make(map[string]string, len(outline.Beats))
	for _, c := range outline.Characters {
		char := &domain.Character{
			StoryID:   story.ID,
			Name:      c.Name,
			Persona:   c.Persona,
			Backstory: c.Backstory,
		}
		created, err := h.charSvc.Create(r.Context(), char)
		if err == nil {
			charIDByName[c.Name] = created.ID
		}
	}

	beatIDByTitle := make(map[string]string, len(outline.Beats))
	for _, b := range outline.Beats {
		scene := &domain.Scene{
			StoryID:    story.ID,
			Title:      b.Title,
			BeatIntent: b.BeatIntent,
			POV:        b.POV,
			Tone:       b.Tone,
			TargetWords: b.TargetWords,
		}
		for _, cn := range b.CharacterNames {
			if id, ok := charIDByName[cn]; ok {
				scene.Participants = append(scene.Participants, id)
			}
		}
		created, err := h.sceneSvc.Create(r.Context(), scene)
		if err != nil {
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

func (h *Handlers) V2GenerateNode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gen, err := h.genSvc.Generate(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

func (h *Handlers) V2ListNodeGenerations(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	gens, err := h.genSvc.ListGenerations(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gens)
}

func (h *Handlers) V2AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")
	var body struct {
		GenerationID string `json:"generation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.genSvc.AcceptGeneration(r.Context(), nodeID, body.GenerationID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *Handlers) EmptyArray(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *Handlers) NotImplemented(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not yet implemented")
}

func extractIDs(items []*domain.Scene) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
