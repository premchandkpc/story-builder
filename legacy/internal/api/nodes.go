package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/graph"
)

// graphNode is the V2 transport representation of a scene/DAG node.
type graphNode struct {
	ID             string         `json:"id"`
	StoryID        string         `json:"story_id"`
	ChapterID      string         `json:"chapter_id"`
	Title          string         `json:"title"`
	BeatIntent     string         `json:"beat_intent"`
	CharacterRefs  []string       `json:"character_refs"`
	LocationRef    string         `json:"location_ref"`
	POV            string         `json:"pov"`
	Tone           string         `json:"tone"`
	TargetWords    int            `json:"target_words"`
	Status         string         `json:"status"`
	SceneStructure map[string]any `json:"scene_structure"`
	PositionX      *float64       `json:"position_x"`
	PositionY      *float64       `json:"position_y"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// graphEdge is the V2 transport representation of a directed edge.
type graphEdge struct {
	ID       string `json:"id"`
	StoryID  string `json:"story_id"`
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	EdgeType string `json:"edge_type"`
}

// topologyResponse bundles nodes, edges, and a topological ordering.
type topologyResponse struct {
	Nodes            []graphNode `json:"nodes"`
	Edges            []graphEdge `json:"edges"`
	TopologicalOrder []string    `json:"topological_order"`
}

// sceneToNode converts a domain.Scene into the V2 graphNode shape.
func sceneToNode(s *domain.Scene) graphNode {
	return graphNode{
		ID:             s.ID,
		StoryID:        s.StoryID,
		ChapterID:      s.ChapterID,
		Title:          s.Title,
		BeatIntent:     s.BeatIntent,
		CharacterRefs:  s.Participants,
		LocationRef:    s.LocationRef,
		POV:            s.POV,
		Tone:           s.Tone,
		TargetWords:    s.TargetWords,
		Status:         s.Status,
		SceneStructure: s.SceneStructure,
		PositionX:      s.PositionX,
		PositionY:      s.PositionY,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

// edgeToGraphEdge converts a domain.SceneEdge into the V2 graphEdge shape.
func edgeToGraphEdge(e *domain.SceneEdge) graphEdge {
	return graphEdge{
		ID:       e.ID,
		StoryID:  e.StoryID,
		FromNode: e.FromSceneID,
		ToNode:   e.ToSceneID,
		EdgeType: e.Type,
	}
}

// extractIDs pulls non-nil scene IDs into a string slice.
func extractIDs(items []*domain.Scene) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

// ListNodes handles GET /api/v1/stories/{storyID}/nodes.
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

// GetNode handles GET /api/v1/stories/{storyID}/nodes/{nodeID}.
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

// CreateNode handles POST /api/v1/stories/{storyID}/nodes.
func (h *Handlers) CreateNode(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		BeatIntent     string         `json:"beat_intent"`
		CharacterRefs  []string       `json:"character_refs"`
		LocationRef    string         `json:"location_ref"`
		ChapterID      string         `json:"chapter_id"`
		POV            string         `json:"pov"`
		Tone           string         `json:"tone"`
		TargetWords    int            `json:"target_words"`
		SceneStructure map[string]any `json:"scene_structure"`
		PositionX      float64        `json:"position_x"`
		PositionY      float64        `json:"position_y"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.BeatIntent == "" {
		writeError(w, http.StatusBadRequest, "beat_intent is required")
		return
	}
	px := body.PositionX
	py := body.PositionY
	scene := &domain.Scene{
		StoryID:        storyID,
		BeatIntent:     body.BeatIntent,
		Participants:   body.CharacterRefs,
		LocationRef:    body.LocationRef,
		ChapterID:      body.ChapterID,
		POV:            body.POV,
		Tone:           body.Tone,
		TargetWords:    body.TargetWords,
		SceneStructure: body.SceneStructure,
		PositionX:      &px,
		PositionY:      &py,
	}
	created, err := h.sceneSvc.Create(r.Context(), scene)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sceneToNode(created))
}

// UpdateNode handles PUT /api/v1/stories/{storyID}/nodes/{nodeID}.
func (h *Handlers) UpdateNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeID")
	var body struct {
		BeatIntent     string         `json:"beat_intent"`
		CharacterRefs  []string       `json:"character_refs"`
		LocationRef    string         `json:"location_ref"`
		ChapterID      string         `json:"chapter_id"`
		POV            string         `json:"pov"`
		Tone           string         `json:"tone"`
		TargetWords    int            `json:"target_words"`
		SceneStructure map[string]any `json:"scene_structure"`
		PositionX      *float64       `json:"position_x,omitempty"`
		PositionY      *float64       `json:"position_y,omitempty"`
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
		ChapterID:      body.ChapterID,
		POV:            body.POV,
		Tone:           body.Tone,
		TargetWords:    body.TargetWords,
		SceneStructure: body.SceneStructure,
	}
	scene.PositionX = body.PositionX
	scene.PositionY = body.PositionY
	updated, err := h.sceneSvc.Update(r.Context(), scene)
	if handleSvcErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, sceneToNode(updated))
}

// DeleteNode handles DELETE /api/v1/stories/{storyID}/nodes/{nodeID}.
func (h *Handlers) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "nodeID")
	if err := h.sceneSvc.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// V2Topology handles GET /api/v1/stories/{storyID}/topology.
// Returns nodes, edges, and a topological ordering. Falls back to timeline position if sort fails.
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
	nodeIDs := extractIDs(scenes)
	adj := make(map[string][]string, len(edges))
	for _, e := range edges {
		adj[e.FromSceneID] = append(adj[e.FromSceneID], e.ToSceneID)
	}
	sortedIDs, err := graph.TopologicalSortStrings(nodeIDs, adj)
	if err != nil {
		slog.Warn("topological sort failed, falling back to timeline position", "storyId", storyID, "error", err)
		sorted := make([]*domain.Scene, len(scenes))
		copy(sorted, scenes)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].TimelinePosition < sorted[j].TimelinePosition
		})
		sortedIDs = extractIDs(sorted)
	}
	writeJSON(w, http.StatusOK, topologyResponse{
		Nodes:            nodes,
		Edges:            ge,
		TopologicalOrder: sortedIDs,
	})
}
