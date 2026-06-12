package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
)

type StoryHandler struct {
	Service          StoryService
	BlueprintService BlueprintService
	TimelineService  TimelineService
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
	title, err := normalizeStoryTitle(req.Title)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	story, err := h.Service.Create(r.Context(), title)
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
	story, err := h.Service.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, story)
}

func (h *StoryHandler) List(w http.ResponseWriter, r *http.Request) {
	stories, err := h.Service.List(r.Context())
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
	if err := h.Service.CreateEdge(r.Context(), storyID, from, to, req.EdgeType); err != nil {
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
	edges, err := h.Service.ListEdges(r.Context(), storyID)
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
	nodes, err := h.Service.ListNodes(r.Context(), storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	edges, err := h.Service.ListEdges(r.Context(), storyID)
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
	Service NodeService
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
	charRefs, err := parseUUIDList(req.CharacterRefs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	locRef, err := parseOptionalUUID(req.LocationRef)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.Service.Create(r.Context(), storyID, req.BeatIntent, charRefs, locRef, req.POV, req.Tone, req.TargetWords)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.SceneStructure != nil {
		if err := h.Service.SetSceneStructure(r.Context(), node.ID, *req.SceneStructure); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
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
	node, err := h.Service.Get(r.Context(), id)
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
	charRefs, err := parseUUIDList(req.CharacterRefs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	locRef, err := parseOptionalUUID(req.LocationRef)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	node, err := h.Service.Update(r.Context(), id, req.BeatIntent, charRefs, locRef, req.POV, req.Tone, req.TargetWords, req.SceneStructure)
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
	nodes, err := h.Service.List(r.Context(), storyID)
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
	Service StoryGeneratorService
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
	result, err := h.Service.GenerateStory(r.Context(), req.Synopsis)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}
