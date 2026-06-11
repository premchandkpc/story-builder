package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
)

type StoryHandler struct {
	Service interface {
		Create(title string) (*graph.Story, error)
		Get(id uuid.UUID) (*graph.Story, error)
		List() ([]graph.Story, error)
		CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error
		ListEdges(storyID uuid.UUID) ([]graph.Edge, error)
		GetNode(id uuid.UUID) (*graph.Node, error)
		ListNodes(storyID uuid.UUID) ([]graph.Node, error)
		TopologicalSort(storyID uuid.UUID) ([]graph.Node, error)
	}
}

type createStoryRequest struct {
	Title string `json:"title"`
}

func (h *StoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	story, err := h.Service.Create(req.Title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, story)
}

func (h *StoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	story, err := h.Service.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (h *StoryHandler) List(w http.ResponseWriter, r *http.Request) {
	stories, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stories)
}

type createEdgeRequest struct {
	FromNode string `json:"from_node"`
	ToNode   string `json:"to_node"`
	EdgeType string `json:"edge_type"`
}

func (h *StoryHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	var req createEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	from, err := uuid.Parse(req.FromNode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from_node")
		return
	}
	to, err := uuid.Parse(req.ToNode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to_node")
		return
	}
	if err := h.Service.CreateEdge(storyID, from, to, req.EdgeType); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *StoryHandler) ListEdges(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	edges, err := h.Service.ListEdges(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, edges)
}

func (h *StoryHandler) Topology(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	nodes, err := h.Service.ListNodes(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	edges, err := h.Service.ListEdges(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	type topology struct {
		Nodes []graph.Node `json:"nodes"`
		Edges []graph.Edge `json:"edges"`
	}
	writeJSON(w, http.StatusOK, topology{Nodes: nodes, Edges: edges})
}

type NodeHandler struct {
	Service interface {
		Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error)
		Get(id uuid.UUID) (*graph.Node, error)
		Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error)
		SetSceneStructure(id uuid.UUID, ss graph.SceneStructure) error
		List(storyID uuid.UUID) ([]graph.Node, error)
	}
}

type createNodeRequest struct {
	BeatIntent     string                `json:"beat_intent"`
	CharacterRefs  []string              `json:"character_refs"`
	LocationRef    *string               `json:"location_ref"`
	POV            string                `json:"pov"`
	Tone           string                `json:"tone"`
	TargetWords    int                   `json:"target_words"`
	SceneStructure *graph.SceneStructure `json:"scene_structure,omitempty"`
}

func (h *NodeHandler) Create(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	var req createNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	charRefs := make([]uuid.UUID, len(req.CharacterRefs))
	for i, s := range req.CharacterRefs {
		charRefs[i], err = uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid character_ref")
			return
		}
	}
	var locRef *uuid.UUID
	if req.LocationRef != nil {
		parsed, err := uuid.Parse(*req.LocationRef)
		if err == nil {
			locRef = &parsed
		}
	}
	node, err := h.Service.Create(storyID, req.BeatIntent, charRefs, locRef, req.POV, req.Tone, req.TargetWords)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.SceneStructure != nil {
		_ = h.Service.SetSceneStructure(node.ID, *req.SceneStructure)
		node.SceneStructure = req.SceneStructure
	}
	writeJSON(w, http.StatusCreated, node)
}

func (h *NodeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	node, err := h.Service.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *NodeHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req createNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	charRefs := make([]uuid.UUID, len(req.CharacterRefs))
	for i, s := range req.CharacterRefs {
		charRefs[i], _ = uuid.Parse(s)
	}
	var locRef *uuid.UUID
	if req.LocationRef != nil {
		parsed, _ := uuid.Parse(*req.LocationRef)
		locRef = &parsed
	}
	node, err := h.Service.Update(id, req.BeatIntent, charRefs, locRef, req.POV, req.Tone, req.TargetWords, req.SceneStructure)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (h *NodeHandler) List(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	nodes, err := h.Service.List(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, nodes)
}

type StoryGenerateResult struct {
	StoryID string `json:"story_id"`
	Status  string `json:"status"`
}

type StoryGeneratorHandler struct {
	Service interface {
		GenerateStory(synopsis string) (*StoryGenerateResult, error)
	}
}

type storyGenerateRequest struct {
	Synopsis string `json:"synopsis"`
}

func (h *StoryGeneratorHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req storyGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Synopsis == "" {
		writeError(w, http.StatusBadRequest, "synopsis is required")
		return
	}
	result, err := h.Service.GenerateStory(req.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}
