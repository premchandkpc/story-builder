package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	canonsvc "github.com/premchand/story-builder/internal/service/canon"
)

type CharacterHandler struct {
	Service canonsvc.CharacterService
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
	parentID, err := parseOptionalUUID(req.ParentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid parent_id")
		return
	}
	char, err := h.Service.Create(r.Context(), req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, parentID, req.Relationships)
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
	char, err := h.Service.Get(r.Context(), id, 0)
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
	parentID, err := parseOptionalUUID(req.ParentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid parent_id")
		return
	}
	char, err := h.Service.Update(r.Context(), id, req.Name, req.Persona, req.Backstory, req.MoralAlignment, req.Personality, req.Flaws, req.Goals, req.Traits, req.VoiceSamples, parentID, req.Relationships)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, char)
}

func (h *CharacterHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	lim := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := parseInt(l); err == nil && v > 0 {
			lim = v
		}
	}
	chars, err := h.Service.Search(r.Context(), q, lim)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chars)
}

func (h *CharacterHandler) List(w http.ResponseWriter, r *http.Request) {
	chars, err := h.Service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, chars)
}

type ActorHandler struct {
	Service canonsvc.ActorService
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
	actor, err := h.Service.Create(r.Context(), req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, req.HeightCm, req.WeightKg, req.Age, req.Traits)
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
	actor, err := h.Service.Get(r.Context(), id)
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
	actor, err := h.Service.Update(r.Context(), id, req.Name, req.Gender, req.Ethnicity, req.Race, req.SkinTone, req.EyeColor, req.HairColor, req.HairStyle, req.Build, req.Nationality, req.HeightCm, req.WeightKg, req.Age, req.Traits)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, actor)
}

func (h *ActorHandler) List(w http.ResponseWriter, r *http.Request) {
	actors, err := h.Service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, actors)
}

type CharacterTraitHandler struct {
	Service canonsvc.TraitService
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
	trait, err := h.Service.Create(r.Context(), req.Name, req.Category, req.Description)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, trait)
}

func (h *CharacterTraitHandler) List(w http.ResponseWriter, r *http.Request) {
	traits, err := h.Service.List(r.Context())
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
	trait, err := h.Service.Get(r.Context(), id)
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
	traitID, err := parseUUID(req.TraitID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trait_id")
		return
	}
	if err := h.Service.Assign(r.Context(), charID, traitID, req.Intensity, req.Note); err != nil {
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
	if err := h.Service.Unassign(r.Context(), charID, traitID); err != nil {
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
	assignments, err := h.Service.GetAssignments(r.Context(), charID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, assignments)
}

type CastingHandler struct {
	Service canonsvc.CastingService
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
	actorID, err := parseUUID(req.ActorID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid actor_id")
		return
	}
	charID, err := parseUUID(req.CharacterID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid character_id")
		return
	}
	cast, err := h.Service.Create(r.Context(), storyID, actorID, charID, req.RoleType)
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
	cast, err := h.Service.GetForStory(r.Context(), storyID)
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
	cast, err := h.Service.GetForCharacter(r.Context(), charID)
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
	cast, err := h.Service.GetForActor(r.Context(), actorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cast)
}

type LocationHandler struct {
	Service canonsvc.LocationService
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
	loc, err := h.Service.Create(r.Context(), req.Name, req.Description, req.Props)
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
	loc, err := h.Service.Get(r.Context(), id, 0)
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
	loc, err := h.Service.Update(r.Context(), id, req.Description, req.Props)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, loc)
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	locs, err := h.Service.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, locs)
}
