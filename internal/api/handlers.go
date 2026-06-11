package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type CharacterHandler struct {
	Service interface {
		Create(name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
		Get(id uuid.UUID, version int) (*canon.Character, error)
		Update(id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
		List() ([]canon.Character, error)
	}
}

type createCharacterRequest struct {
	Name           string            `json:"name"`
	Persona        string            `json:"persona"`
	Backstory      string            `json:"backstory"`
	MoralAlignment string            `json:"moral_alignment"`
	Personality    []string          `json:"personality"`
	Flaws          []string          `json:"flaws"`
	Goals          []string          `json:"goals"`
	Traits         []string          `json:"traits"`
	VoiceSamples   []string          `json:"voice_samples"`
	ParentID       *string           `json:"parent_id,omitempty"`
	Relationships  map[string]string `json:"relationships"`
}

func (h *CharacterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var parentID *uuid.UUID
	if req.ParentID != nil {
		parsed, err := uuid.Parse(*req.ParentID)
		if err == nil {
			parentID = &parsed
		}
	}
	char, err := h.Service.Create(req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, parentID, req.Relationships)
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
	var parentID *uuid.UUID
	if req.ParentID != nil {
		parsed, _ := uuid.Parse(*req.ParentID)
		parentID = &parsed
	}
	char, err := h.Service.Update(id, req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, parentID, req.Relationships)
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

type ActorHandler struct {
	Service interface {
		Create(name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
		Get(id uuid.UUID) (*canon.Actor, error)
		Update(id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
		List() ([]canon.Actor, error)
	}
}

type createActorRequest struct {
	Name        string                 `json:"name"`
	Gender      string                 `json:"gender"`
	Ethnicity   string                 `json:"ethnicity"`
	Race        string                 `json:"race"`
	SkinTone    string                 `json:"skin_tone"`
	EyeColor    string                 `json:"eye_color"`
	HairColor   string                 `json:"hair_color"`
	HairStyle   string                 `json:"hair_style"`
	Build       string                 `json:"build"`
	HeightCm    int                    `json:"height_cm"`
	WeightKg    int                    `json:"weight_kg"`
	Age         int                    `json:"age"`
	Nationality string                 `json:"nationality"`
	Traits      map[string]interface{} `json:"traits"`
}

func (h *ActorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createActorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	actor, err := h.Service.Create(req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, req.HeightCm, req.WeightKg, req.Age, req.Traits)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, actor)
}

func (h *ActorHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	actor, err := h.Service.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, actor)
}

func (h *ActorHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req createActorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	actor, err := h.Service.Update(id, req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, req.HeightCm, req.WeightKg, req.Age, req.Traits)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, actor)
}

func (h *ActorHandler) List(w http.ResponseWriter, r *http.Request) {
	actors, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, actors)
}

type CharacterTraitHandler struct {
	Service interface {
		Create(name, category, description string) (*canon.CharacterTrait, error)
		Get(id uuid.UUID) (*canon.CharacterTrait, error)
		List() ([]canon.CharacterTrait, error)
		Assign(characterID, traitID uuid.UUID, intensity int, note string) error
		Unassign(characterID, traitID uuid.UUID) error
		GetAssignments(characterID uuid.UUID) ([]canon.TraitAssignment, error)
	}
}

type createCharacterTraitRequest struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

type assignTraitRequest struct {
	TraitID   string `json:"trait_id"`
	Intensity int    `json:"intensity"`
	Note      string `json:"note"`
}

func (h *CharacterTraitHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCharacterTraitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trait, err := h.Service.Create(req.Name, req.Category, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, trait)
}

func (h *CharacterTraitHandler) List(w http.ResponseWriter, r *http.Request) {
	traits, err := h.Service.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, traits)
}

func (h *CharacterTraitHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	trait, err := h.Service.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, trait)
}

func (h *CharacterTraitHandler) Assign(w http.ResponseWriter, r *http.Request) {
	charID, err := uuid.Parse(chi.URLParam(r, "characterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid character id")
		return
	}
	var req assignTraitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	traitID, _ := uuid.Parse(req.TraitID)
	if err := h.Service.Assign(charID, traitID, req.Intensity, req.Note); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *CharacterTraitHandler) Unassign(w http.ResponseWriter, r *http.Request) {
	charID, err := uuid.Parse(chi.URLParam(r, "characterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid character id")
		return
	}
	traitID, err := uuid.Parse(chi.URLParam(r, "traitID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trait id")
		return
	}
	if err := h.Service.Unassign(charID, traitID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *CharacterTraitHandler) GetAssignments(w http.ResponseWriter, r *http.Request) {
	charID, err := uuid.Parse(chi.URLParam(r, "characterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid character id")
		return
	}
	assignments, err := h.Service.GetAssignments(charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, assignments)
}

type CastingHandler struct {
	Service interface {
		Create(storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
		GetForStory(storyID uuid.UUID) ([]canon.Casting, error)
		GetForCharacter(characterID uuid.UUID) ([]canon.Casting, error)
		GetForActor(actorID uuid.UUID) ([]canon.Casting, error)
	}
}

type createCastingRequest struct {
	ActorID     string `json:"actor_id"`
	CharacterID string `json:"character_id"`
	RoleType    string `json:"role_type"`
}

func (h *CastingHandler) Create(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	var req createCastingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	actorID, _ := uuid.Parse(req.ActorID)
	charID, _ := uuid.Parse(req.CharacterID)
	cast, err := h.Service.Create(storyID, actorID, charID, req.RoleType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cast)
}

func (h *CastingHandler) ListForStory(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	cast, err := h.Service.GetForStory(storyID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cast)
}

func (h *CastingHandler) ListForCharacter(w http.ResponseWriter, r *http.Request) {
	charID, err := uuid.Parse(chi.URLParam(r, "characterID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid character id")
		return
	}
	cast, err := h.Service.GetForCharacter(charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cast)
}

func (h *CastingHandler) ListForActor(w http.ResponseWriter, r *http.Request) {
	actorID, err := uuid.Parse(chi.URLParam(r, "actorID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid actor id")
		return
	}
	cast, err := h.Service.GetForActor(actorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cast)
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

type SceneHandler struct {
	SceneService scene.SceneService
}

type setSceneStructureRequest struct {
	SceneStructure graph.SceneStructure `json:"scene_structure"`
}

func (h *SceneHandler) SetStructure(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	var req setSceneStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.SceneService.SetSceneStructure(nodeID, req.SceneStructure); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *SceneHandler) GetStructure(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	ss, err := h.SceneService.GetSceneStructure(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ss)
}

func (h *SceneHandler) Start(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turn, err := h.SceneService.StartScene(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turn)
}

func (h *SceneHandler) Next(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turn, err := h.SceneService.NextTurn(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turn)
}

func (h *SceneHandler) Finish(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	output, err := h.SceneService.FinishScene(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"output": output})
}

func (h *SceneHandler) Turns(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	turns, err := h.SceneService.GetTurns(nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, turns)
}

type SummaryHandler struct {
	Service compiler.SummaryService
}

func (h *SummaryHandler) GetSceneSummary(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid node id")
		return
	}
	summary, err := h.Service.GetSceneSummary(storyID, nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *SummaryHandler) GetByLevel(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	if level != compiler.SummaryAct && level != compiler.SummaryStory {
		level = compiler.SummaryAct
	}
	summary, err := h.Service.GetSummaryByLevel(storyID, level)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *SummaryHandler) CountByLevel(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	if level != compiler.SummaryScene && level != compiler.SummaryAct && level != compiler.SummaryStory {
		level = compiler.SummaryScene
	}
	count, err := h.Service.CountSummariesByLevel(storyID, level)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func (h *SummaryHandler) ShouldElevate(w http.ResponseWriter, r *http.Request) {
	storyID, err := uuid.Parse(chi.URLParam(r, "storyID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid story id")
		return
	}
	level := compiler.SummaryLevel(r.URL.Query().Get("level"))
	threshold := 10
	if t := r.URL.Query().Get("threshold"); t != "" {
		if v, err := parseInt(t); err == nil {
			threshold = v
		}
	}
	should, err := h.Service.ShouldElevate(storyID, level, threshold)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"should_elevate": should,
		"level":          level,
		"threshold":      threshold,
	})
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
