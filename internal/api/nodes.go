package api

import (
	"encoding/json"
	"net/http"
	"sort"
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

func extractIDs(items []*domain.Scene) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
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
	w.WriteHeader(http.StatusNoContent)
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
	sorted := make([]*domain.Scene, len(scenes))
	copy(sorted, scenes)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].TimelinePosition < sorted[j].TimelinePosition
	})
	writeJSON(w, http.StatusOK, topologyResponse{
		Nodes:            nodes,
		Edges:            ge,
		TopologicalOrder: extractIDs(sorted),
	})
}
