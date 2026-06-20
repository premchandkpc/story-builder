package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

// DeleteEdge handles DELETE /stories/{storyID}/edges?from_scene=X&to_scene=Y.
func (h *Handlers) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	fromScene := r.URL.Query().Get("from_scene")
	toScene := r.URL.Query().Get("to_scene")
	if fromScene == "" || toScene == "" {
		writeError(w, http.StatusBadRequest, "from_scene and to_scene query params are required")
		return
	}
	if err := h.edgeSvc.Delete(r.Context(), storyID, fromScene, toScene); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// V2CreateEdge handles POST /stories/{storyID}/edges.
func (h *Handlers) V2CreateEdge(w http.ResponseWriter, r *http.Request) {
	storyID := chi.URLParam(r, "storyID")
	var body struct {
		FromNode  string `json:"from_node"`
		ToNode    string `json:"to_node"`
		EdgeType  string `json:"edge_type"`
		Condition string `json:"condition,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.FromNode == "" || body.ToNode == "" {
		writeError(w, http.StatusBadRequest, "from_node and to_node are required")
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

// V2ListEdges handles GET /stories/{storyID}/edges.
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
