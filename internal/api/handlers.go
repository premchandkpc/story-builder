package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
)

type CharacterHandler struct {
	Service interface {
		Create(name string, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error)
		Get(id uuid.UUID, version int) (*canon.Character, error)
		Update(id uuid.UUID, traits, voiceSamples []string, relationships map[string]string) (*canon.Character, error)
		List() ([]canon.Character, error)
	}
}

type createCharacterRequest struct {
	Name          string            `json:"name"`
	Traits        []string          `json:"traits"`
	VoiceSamples  []string          `json:"voice_samples"`
	Relationships map[string]string `json:"relationships"`
}

func (h *CharacterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	char, err := h.Service.Create(req.Name, req.Traits, req.VoiceSamples, req.Relationships)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, char)
}

func (h *CharacterHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	char, err := h.Service.Get(id, 0)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, char)
}

func (h *CharacterHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req createCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	char, err := h.Service.Update(id, req.Traits, req.VoiceSamples, req.Relationships)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, char)
}

func (h *CharacterHandler) List(w http.ResponseWriter, r *http.Request) {
	chars, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chars)
}

type LocationHandler struct {
	Service interface {
		Create(name, description string, props []string) (*canon.Location, error)
		Get(id uuid.UUID, version int) (*canon.Location, error)
		Update(id uuid.UUID, description string, props []string) (*canon.Location, error)
		List() ([]canon.Location, error)
	}
}

type createLocationRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Props       []string `json:"props"`
}

func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	loc, err := h.Service.Create(req.Name, req.Description, req.Props)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, loc)
}

func (h *LocationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	loc, err := h.Service.Get(id, 0)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req createLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	loc, err := h.Service.Update(id, req.Description, req.Props)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	locs, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, locs)
}

type LoreHandler struct {
	Service interface {
		Create(tags []string, content string) (*canon.Lore, error)
		List() ([]canon.Lore, error)
		SearchByTags(tags []string) ([]canon.Lore, error)
		SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error)
	}
}

type createLoreRequest struct {
	Tags    []string `json:"tags"`
	Content string   `json:"content"`
}

type searchLoreRequest struct {
	Tags      []string  `json:"tags"`
	Embedding []float32 `json:"embedding,omitempty"`
	Limit     int       `json:"limit"`
}

func (h *LoreHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createLoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	lore, err := h.Service.Create(req.Tags, req.Content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, lore)
}

func (h *LoreHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *LoreHandler) Search(w http.ResponseWriter, r *http.Request) {
	var req searchLoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(req.Embedding) > 0 {
		results, err := h.Service.SearchSimilar(req.Embedding, req.Limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, results)
		return
	}
	results, err := h.Service.SearchByTags(req.Tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, results)
}

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
	from, _ := uuid.Parse(req.FromNode)
	to, _ := uuid.Parse(req.ToNode)
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
		Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error)
		List(storyID uuid.UUID) ([]graph.Node, error)
	}
}

type createNodeRequest struct {
	BeatIntent    string   `json:"beat_intent"`
	CharacterRefs []string `json:"character_refs"`
	LocationRef   *string  `json:"location_ref"`
	POV           string   `json:"pov"`
	Tone          string   `json:"tone"`
	TargetWords   int      `json:"target_words"`
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
		charRefs[i], _ = uuid.Parse(s)
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
	node, err := h.Service.Update(id, req.BeatIntent, charRefs, locRef, req.POV, req.Tone, req.TargetWords)
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

type GenerationHandler struct {
	Service interface {
		Generate(nodeID uuid.UUID) (*compiler.Generation, error)
		AcceptGeneration(nodeID, genID uuid.UUID) error
		ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error)
	}
}

type acceptGenerationRequest struct {
	GenerationID string `json:"generation_id"`
}

func (h *GenerationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	gen, err := h.Service.Generate(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, gen)
}

func (h *GenerationHandler) AcceptGeneration(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	var req acceptGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	genID, _ := uuid.Parse(req.GenerationID)
	if err := h.Service.AcceptGeneration(nodeID, genID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *GenerationHandler) ListGenerations(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	gens, err := h.Service.ListGenerations(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, gens)
}
