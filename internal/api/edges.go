package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/premchand/story-builder/internal/domain"
)

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
	if body.FromScene == "" || body.ToScene == "" {
		writeError(w, http.StatusBadRequest, "fromSceneId and toSceneId are required")
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
	if body.FromScene == "" || body.ToScene == "" {
		writeError(w, http.StatusBadRequest, "from_scene and to_scene are required")
		return
	}
	if err := h.edgeSvc.Delete(r.Context(), storyID, body.FromScene, body.ToScene); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

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
