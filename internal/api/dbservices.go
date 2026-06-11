package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/db"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

func NewDBCharService(q *db.Queries) *dbCharService {
	return &dbCharService{q: q}
}

type dbCharService struct{ q *db.Queries }

func toUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func fromUUID(u pgtype.UUID) uuid.UUID {
	id, _ := uuid.FromBytes(u.Bytes[:])
	return id
}

func jsonBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (s *dbCharService) Create(name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.CreateCharacter(context.Background(), db.CreateCharacterParams{
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    jsonBytes(personality),
		Flaws:          jsonBytes(flaws),
		Goals:          jsonBytes(goals),
		Traits:         jsonBytes(traits),
		VoiceSamples:   voiceSamples,
		Relationships:  jsonBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) Get(id uuid.UUID, version int) (*canon.Character, error) {
	if version > 0 {
		c, err := s.q.GetCharacterAtVersion(context.Background(), db.GetCharacterAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainChar(c), nil
	}
	c, err := s.q.GetCharacterLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainCharFromLatest(c), nil
}

func (s *dbCharService) Update(id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	if voiceSamples == nil {
		voiceSamples = []string{}
	}
	var pid pgtype.UUID
	if parentID != nil {
		pid = toUUID(*parentID)
	}
	c, err := s.q.UpdateCharacter(context.Background(), db.UpdateCharacterParams{
		ID:             toUUID(id),
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Column6:        jsonBytes(personality),
		Column7:        jsonBytes(flaws),
		Column8:        jsonBytes(goals),
		Column9:        jsonBytes(traits),
		VoiceSamples:   voiceSamples,
		Column11:       jsonBytes(relationships),
		ParentID:       pid,
	})
	if err != nil {
		return nil, err
	}
	return toDomainChar(c), nil
}

func (s *dbCharService) List() ([]canon.Character, error) {
	chars, err := s.q.ListCharacters(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Character, len(chars))
	for i, c := range chars {
		result[i] = *toDomainCharFromLatest(c)
	}
	return result, nil
}

func toDomainChar(c db.Character) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var personality []string
	json.Unmarshal(c.Personality, &personality)
	var flaws []string
	json.Unmarshal(c.Flaws, &flaws)
	var goals []string
	json.Unmarshal(c.Goals, &goals)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := fromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             fromUUID(c.ID),
		Version:        int(c.Version),
		Name:           c.Name,
		Persona:        c.Persona,
		Backstory:      c.Backstory,
		MoralAlignment: c.MoralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   c.VoiceSamples,
		ParentID:       parentID,
		Relationships:  rel,
		CreatedAt:      c.CreatedAt.Time,
	}
}

func toDomainCharFromLatest(c db.LatestCharacter) *canon.Character {
	var traits []string
	json.Unmarshal(c.Traits, &traits)
	var personality []string
	json.Unmarshal(c.Personality, &personality)
	var flaws []string
	json.Unmarshal(c.Flaws, &flaws)
	var goals []string
	json.Unmarshal(c.Goals, &goals)
	var rel map[string]string
	json.Unmarshal(c.Relationships, &rel)
	var parentID *uuid.UUID
	if c.ParentID.Valid {
		p := fromUUID(c.ParentID)
		parentID = &p
	}
	return &canon.Character{
		ID:             fromUUID(c.ID),
		Version:        int(c.Version),
		Name:           c.Name,
		Persona:        c.Persona,
		Backstory:      c.Backstory,
		MoralAlignment: c.MoralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   c.VoiceSamples,
		ParentID:       parentID,
		Relationships:  rel,
		CreatedAt:      c.CreatedAt.Time,
	}
}

type dbLocService struct{ q *db.Queries }

func NewDBLocService(q *db.Queries) *dbLocService {
	return &dbLocService{q: q}
}

func (s *dbLocService) Create(name, description string, props []string) (*canon.Location, error) {
	if props == nil {
		props = []string{}
	}
	l, err := s.q.CreateLocation(context.Background(), db.CreateLocationParams{
		Name:        name,
		Description: description,
		Props:       jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) Get(id uuid.UUID, version int) (*canon.Location, error) {
	if version > 0 {
		l, err := s.q.GetLocationAtVersion(context.Background(), db.GetLocationAtVersionParams{
			ID:      toUUID(id),
			Version: int32(version),
		})
		if err != nil {
			return nil, err
		}
		return toDomainLoc(l), nil
	}
	l, err := s.q.GetLocationLatest(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainLocFromLatest(l), nil
}

func (s *dbLocService) Update(id uuid.UUID, description string, props []string) (*canon.Location, error) {
	l, err := s.q.UpdateLocation(context.Background(), db.UpdateLocationParams{
		ID:          toUUID(id),
		Description: description,
		Column3:     jsonBytes(props),
	})
	if err != nil {
		return nil, err
	}
	return toDomainLoc(l), nil
}

func (s *dbLocService) List() ([]canon.Location, error) {
	locs, err := s.q.ListLocations(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Location, len(locs))
	for i, l := range locs {
		result[i] = *toDomainLocFromLatest(l)
	}
	return result, nil
}

func toDomainLoc(l db.Location) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

func toDomainLocFromLatest(l db.LatestLocation) *canon.Location {
	var props []string
	json.Unmarshal(l.Props, &props)
	return &canon.Location{
		ID:          fromUUID(l.ID),
		Version:     int(l.Version),
		Name:        l.Name,
		Description: l.Description,
		Props:       props,
		CreatedAt:   l.CreatedAt.Time,
	}
}

type dbLoreService struct{ q *db.Queries }

func NewDBLoreService(q *db.Queries) *dbLoreService {
	return &dbLoreService{q: q}
}

func (s *dbLoreService) Create(tags []string, content string) (*canon.Lore, error) {
	if tags == nil {
		tags = []string{}
	}
	l, err := s.q.CreateLore(context.Background(), db.CreateLoreParams{
		Tags:    tags,
		Content: content,
	})
	if err != nil {
		return nil, err
	}
	return &canon.Lore{
		ID:        fromUUID(l.ID),
		Tags:      l.Tags,
		Content:   l.Content,
		CreatedAt: l.CreatedAt.Time,
	}, nil
}

func (s *dbLoreService) List() ([]canon.Lore, error) {
	items, err := s.q.ListLore(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchByTags(tags []string) ([]canon.Lore, error) {
	items, err := s.q.SearchLoreByTags(context.Background(), tags)
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbLoreService) SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error) {
	vec := pgvector.NewVector(embedding)
	items, err := s.q.SearchLoreSimilar(context.Background(), db.SearchLoreSimilarParams{
		Column1: vec,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]canon.Lore, len(items))
	for i, l := range items {
		result[i] = canon.Lore{
			ID:        fromUUID(l.ID),
			Tags:      l.Tags,
			Content:   l.Content,
			CreatedAt: l.CreatedAt.Time,
		}
	}
	return result, nil
}

type dbGraphStoryService struct{ q *db.Queries }

func NewDBGraphStoryService(q *db.Queries) *dbGraphStoryService {
	return &dbGraphStoryService{q: q}
}

func (s *dbGraphStoryService) Create(title string) (*graph.Story, error) {
	st, err := s.q.CreateStory(context.Background(), title)
	if err != nil {
		return nil, err
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: make(map[string]interface{}),
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbGraphStoryService) Get(id uuid.UUID) (*graph.Story, error) {
	st, err := s.q.GetStory(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	var pins map[string]interface{}
	json.Unmarshal(st.CanonPins, &pins)
	if pins == nil {
		pins = make(map[string]interface{})
	}
	return &graph.Story{
		ID:        fromUUID(st.ID),
		Title:     st.Title,
		CanonPins: pins,
		CreatedAt: st.CreatedAt.Time,
	}, nil
}

func (s *dbGraphStoryService) List() ([]graph.Story, error) {
	stories, err := s.q.ListStories(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]graph.Story, len(stories))
	for i, st := range stories {
		result[i] = graph.Story{
			ID:        fromUUID(st.ID),
			Title:     st.Title,
			CanonPins: make(map[string]interface{}),
			CreatedAt: st.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbGraphStoryService) CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	return s.q.CreateEdge(context.Background(), db.CreateEdgeParams{
		StoryID:  toUUID(storyID),
		FromNode: toUUID(fromNode),
		ToNode:   toUUID(toNode),
		EdgeType: edgeType,
	})
}

func (s *dbGraphStoryService) ListEdges(storyID uuid.UUID) ([]graph.Edge, error) {
	edges, err := s.q.ListEdges(context.Background(), toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Edge, len(edges))
	for i, e := range edges {
		result[i] = graph.Edge{
			StoryID:  fromUUID(e.StoryID),
			FromNode: fromUUID(e.FromNode),
			ToNode:   fromUUID(e.ToNode),
			EdgeType: graph.EdgeType(e.EdgeType),
		}
	}
	return result, nil
}

func (s *dbGraphStoryService) GetNode(id uuid.UUID) (*graph.Node, error) {
	return getNode(s.q, id)
}

func (s *dbGraphStoryService) ListNodes(storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(s.q, storyID)
}

func (s *dbGraphStoryService) TopologicalSort(storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := listNodes(s.q, storyID)
	if err != nil {
		return nil, err
	}
	edges, err := s.ListEdges(storyID)
	if err != nil {
		return nil, err
	}
	return graph.TopologicalSort(nodes, edges)
}

type dbGraphNodeService struct{ q *db.Queries }

func NewDBGraphNodeService(q *db.Queries) *dbGraphNodeService {
	return &dbGraphNodeService{q: q}
}

func (s *dbGraphNodeService) Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	n, err := s.q.CreateNode(context.Background(), db.CreateNodeParams{
		StoryID:       toUUID(storyID),
		BeatIntent:    beatIntent,
		CharacterRefs: refs,
		LocationRef:   locRef,
		Pov:           pov,
		Tone:          tone,
		TargetWords:   int32(targetWords),
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbGraphNodeService) Get(id uuid.UUID) (*graph.Node, error) {
	return getNode(s.q, id)
}

func (s *dbGraphNodeService) Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	refs := make([]pgtype.UUID, len(characterRefs))
	for i, r := range characterRefs {
		refs[i] = toUUID(r)
	}
	var locRef pgtype.UUID
	if locationRef != nil {
		locRef = toUUID(*locationRef)
	}
	ssBytes := jsonBytes(graph.SceneStructure{FlowType: graph.FlowMonologue, SituationFlow: ""})
	if sceneStructure != nil {
		ssBytes = jsonBytes(sceneStructure)
	}
	n, err := s.q.UpdateNode(context.Background(), db.UpdateNodeParams{
		ID:             toUUID(id),
		BeatIntent:     beatIntent,
		CharacterRefs:  refs,
		LocationRef:    locRef,
		Pov:            pov,
		Tone:           tone,
		TargetWords:    int32(targetWords),
		SceneStructure: ssBytes,
	})
	if err != nil {
		return nil, err
	}
	return toDomainNode(n), nil
}

func (s *dbGraphNodeService) SetSceneStructure(id uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateNodeSceneStructure(context.Background(), db.UpdateNodeSceneStructureParams{
		ID:             toUUID(id),
		SceneStructure: jsonBytes(ss),
	})
}

func (s *dbGraphNodeService) List(storyID uuid.UUID) ([]graph.Node, error) {
	return listNodes(s.q, storyID)
}

func toDomainNode(n db.Node) *graph.Node {
	refs := make([]uuid.UUID, len(n.CharacterRefs))
	for i, r := range n.CharacterRefs {
		refs[i] = fromUUID(r)
	}
	var locRef *uuid.UUID
	if n.LocationRef.Valid {
		l := fromUUID(n.LocationRef)
		locRef = &l
	}
	var ss *graph.SceneStructure
	if len(n.SceneStructure) > 0 {
		var s graph.SceneStructure
		if json.Unmarshal(n.SceneStructure, &s) == nil {
			ss = &s
		}
	}
	return &graph.Node{
		ID:             fromUUID(n.ID),
		StoryID:        fromUUID(n.StoryID),
		BeatIntent:     n.BeatIntent,
		CharacterRefs:  refs,
		LocationRef:    locRef,
		POV:            n.Pov,
		Tone:           n.Tone,
		TargetWords:    int(n.TargetWords),
		Status:         graph.NodeStatus(n.Status),
		SceneStructure: ss,
		CreatedAt:      n.CreatedAt.Time,
		UpdatedAt:      n.UpdatedAt.Time,
	}
}

func getNode(q *db.Queries, id uuid.UUID) (*graph.Node, error) {
	n, err := q.GetNode(context.Background(), toUUID(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("node %s not found", id)
		}
		return nil, err
	}
	return toDomainNode(n), nil
}

func listNodes(q *db.Queries, storyID uuid.UUID) ([]graph.Node, error) {
	nodes, err := q.ListNodes(context.Background(), toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]graph.Node, len(nodes))
	for i, n := range nodes {
		result[i] = *toDomainNode(n)
	}
	return result, nil
}

type dbGenerationService struct{ q *db.Queries }

func NewDBGenerationService(q *db.Queries) *dbGenerationService {
	return &dbGenerationService{q: q}
}

func (s *dbGenerationService) Generate(nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration — not implemented")
}

func (s *dbGenerationService) AcceptGeneration(nodeID, genID uuid.UUID) error {
	return s.q.AcceptGeneration(context.Background(), toUUID(genID))
}

func (s *dbGenerationService) ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error) {
	gens, err := s.q.ListGenerationsForNode(context.Background(), toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	result := make([]compiler.Generation, len(gens))
	for i, g := range gens {
		result[i] = compiler.Generation{
			ID:             fromUUID(g.ID).String(),
			NodeID:         fromUUID(g.NodeID).String(),
			ContextHash:    g.ContextHash,
			PromptSnapshot: g.PromptSnapshot,
			Output:         g.Output,
			Model:          g.Model,
			Accepted:       g.Accepted,
		}
	}
	return result, nil
}

type dbSceneService struct{ q *db.Queries }

func NewDBSceneService(q *db.Queries) *dbSceneService {
	return &dbSceneService{q: q}
}

func (s *dbSceneService) StartScene(nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration — not implemented")
}

func (s *dbSceneService) NextTurn(nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration — not implemented")
}

func (s *dbSceneService) FinishScene(nodeID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration — not implemented")
}

func (s *dbSceneService) GetTurns(nodeID uuid.UUID) ([]scene.SceneTurn, error) {
	turns, err := s.q.ListSceneTurns(context.Background(), toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	result := make([]scene.SceneTurn, len(turns))
	for i, t := range turns {
		actorIDs := make([]uuid.UUID, len(t.ActorIds))
		for j, a := range t.ActorIds {
			actorIDs[j] = fromUUID(a)
		}
		result[i] = scene.SceneTurn{
			ID:         fromUUID(t.ID),
			NodeID:     fromUUID(t.NodeID),
			TurnNumber: int(t.TurnNumber),
			ActorIDs:   actorIDs,
			Prompt:     t.Prompt,
			Output:     t.Output,
			Model:      t.Model,
			Status:     t.Status,
			CreatedAt:  t.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbSceneService) SetSceneStructure(nodeID uuid.UUID, ss graph.SceneStructure) error {
	return s.q.UpdateNodeSceneStructure(context.Background(), db.UpdateNodeSceneStructureParams{
		ID:             toUUID(nodeID),
		SceneStructure: jsonBytes(ss),
	})
}

func (s *dbSceneService) GetSceneStructure(nodeID uuid.UUID) (*graph.SceneStructure, error) {
	n, err := s.q.GetNode(context.Background(), toUUID(nodeID))
	if err != nil {
		return nil, err
	}
	if len(n.SceneStructure) == 0 {
		return nil, nil
	}
	var ss graph.SceneStructure
	if err := json.Unmarshal(n.SceneStructure, &ss); err != nil {
		return nil, err
	}
	return &ss, nil
}

// ── Actor (DB-backed) ──────────────────────────────────────────

func NewDBActorService(q *db.Queries) *dbActorService {
	return &dbActorService{q: q}
}

type dbActorService struct{ q *db.Queries }

func (s *dbActorService) Create(name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.CreateActor(context.Background(), db.CreateActorParams{
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    int32(heightCm),
		WeightKg:    int32(weightKg),
		Age:         int32(age),
		Nationality: nationality,
		Traits:      jsonBytes(traits),
	})
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) Get(id uuid.UUID) (*canon.Actor, error) {
	a, err := s.q.GetActor(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) Update(id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a, err := s.q.UpdateActor(context.Background(), db.UpdateActorParams{
		ID:          toUUID(id),
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    int32(heightCm),
		WeightKg:    int32(weightKg),
		Age:         int32(age),
		Nationality: nationality,
		Column15:    jsonBytes(traits),
	})
	if err != nil {
		return nil, err
	}
	return toDomainActor(a), nil
}

func (s *dbActorService) List() ([]canon.Actor, error) {
	actors, err := s.q.ListActors(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.Actor, len(actors))
	for i, a := range actors {
		result[i] = *toDomainActor(a)
	}
	return result, nil
}

func toDomainActor(a db.Actor) *canon.Actor {
	var traits map[string]interface{}
	json.Unmarshal(a.Traits, &traits)
	if traits == nil {
		traits = make(map[string]interface{})
	}
	return &canon.Actor{
		ID:          fromUUID(a.ID),
		Name:        a.Name,
		Gender:      a.Gender,
		Ethnicity:   a.Ethnicity,
		Race:        a.Race,
		SkinTone:    a.SkinTone,
		EyeColor:    a.EyeColor,
		HairColor:   a.HairColor,
		HairStyle:   a.HairStyle,
		Build:       a.Build,
		HeightCm:    int(a.HeightCm),
		WeightKg:    int(a.WeightKg),
		Age:         int(a.Age),
		Nationality: a.Nationality,
		Traits:      traits,
		CreatedAt:   a.CreatedAt.Time,
	}
}

// ── CharacterTrait (DB-backed) ──────────────────────────────────

func NewDBCharacterTraitService(q *db.Queries) *dbCharacterTraitService {
	return &dbCharacterTraitService{q: q}
}

type dbCharacterTraitService struct{ q *db.Queries }

func (s *dbCharacterTraitService) Create(name, category, description string) (*canon.CharacterTrait, error) {
	t, err := s.q.CreateCharacterTrait(context.Background(), db.CreateCharacterTraitParams{
		Name:        name,
		Category:    category,
		Description: description,
	})
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          fromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *dbCharacterTraitService) Get(id uuid.UUID) (*canon.CharacterTrait, error) {
	t, err := s.q.GetCharacterTrait(context.Background(), toUUID(id))
	if err != nil {
		return nil, err
	}
	return &canon.CharacterTrait{
		ID:          fromUUID(t.ID),
		Name:        t.Name,
		Category:    t.Category,
		Description: t.Description,
		CreatedAt:   t.CreatedAt.Time,
	}, nil
}

func (s *dbCharacterTraitService) List() ([]canon.CharacterTrait, error) {
	traits, err := s.q.ListCharacterTraits(context.Background())
	if err != nil {
		return nil, err
	}
	result := make([]canon.CharacterTrait, len(traits))
	for i, t := range traits {
		result[i] = canon.CharacterTrait{
			ID:          fromUUID(t.ID),
			Name:        t.Name,
			Category:    t.Category,
			Description: t.Description,
			CreatedAt:   t.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCharacterTraitService) Assign(characterID, traitID uuid.UUID, intensity int, note string) error {
	return s.q.AssignTrait(context.Background(), db.AssignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
		Intensity:   int32(intensity),
		Note:        note,
	})
}

func (s *dbCharacterTraitService) Unassign(characterID, traitID uuid.UUID) error {
	return s.q.UnassignTrait(context.Background(), db.UnassignTraitParams{
		CharacterID: toUUID(characterID),
		TraitID:     toUUID(traitID),
	})
}

func (s *dbCharacterTraitService) GetAssignments(characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	rows, err := s.q.GetTraitAssignments(context.Background(), toUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.TraitAssignment, len(rows))
	for i, r := range rows {
		result[i] = canon.TraitAssignment{
			CharacterID: fromUUID(r.CharacterID),
			TraitID:     fromUUID(r.TraitID),
			Intensity:   int(r.Intensity),
			Note:        r.Note,
		}
	}
	return result, nil
}

// ── Casting (DB-backed) ────────────────────────────────────────

func NewDBCastingService(q *db.Queries) *dbCastingService {
	return &dbCastingService{q: q}
}

type dbCastingService struct{ q *db.Queries }

func (s *dbCastingService) Create(storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c, err := s.q.CreateCasting(context.Background(), db.CreateCastingParams{
		StoryID:     toUUID(storyID),
		ActorID:     toUUID(actorID),
		CharacterID: toUUID(characterID),
		RoleType:    roleType,
	})
	if err != nil {
		return nil, err
	}
	return &canon.Casting{
		ID:          fromUUID(c.ID),
		StoryID:     fromUUID(c.StoryID),
		ActorID:     fromUUID(c.ActorID),
		CharacterID: fromUUID(c.CharacterID),
		RoleType:    c.RoleType,
		CreatedAt:   c.CreatedAt.Time,
	}, nil
}

func (s *dbCastingService) GetForStory(storyID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForStory(context.Background(), toUUID(storyID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCastingService) GetForCharacter(characterID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForCharacter(context.Background(), toUUID(characterID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

func (s *dbCastingService) GetForActor(actorID uuid.UUID) ([]canon.Casting, error) {
	rows, err := s.q.ListCastingForActor(context.Background(), toUUID(actorID))
	if err != nil {
		return nil, err
	}
	result := make([]canon.Casting, len(rows))
	for i, r := range rows {
		result[i] = canon.Casting{
			ID:          fromUUID(r.ID),
			StoryID:     fromUUID(r.StoryID),
			ActorID:     fromUUID(r.ActorID),
			CharacterID: fromUUID(r.CharacterID),
			RoleType:    r.RoleType,
			CreatedAt:   r.CreatedAt.Time,
		}
	}
	return result, nil
}

// ── SummaryService (DB-backed) ─────────────────────────────────

func NewDBSummaryService(q *db.Queries) *dbSummaryService {
	return &dbSummaryService{q: q}
}

type dbSummaryService struct{ q *db.Queries }

func (s *dbSummaryService) UpsertSceneSummary(storyID, nodeID uuid.UUID, content string) error {
	return s.q.UpsertSceneSummary(context.Background(), db.UpsertSceneSummaryParams{
		StoryID:   toUUID(storyID),
		NodeID:    toUUID(nodeID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertActSummary(storyID uuid.UUID, content string) error {
	return s.q.UpsertActSummary(context.Background(), db.UpsertActSummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) UpsertStorySummary(storyID uuid.UUID, content string) error {
	return s.q.UpsertStorySummary(context.Background(), db.UpsertStorySummaryParams{
		StoryID:   toUUID(storyID),
		Content:   content,
		WordCount: int32(len(strings.Fields(content))),
	})
}

func (s *dbSummaryService) GetSceneSummary(storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
	row, err := s.q.GetSceneSummary(context.Background(), db.GetSceneSummaryParams{
		StoryID: toUUID(storyID),
		NodeID:  toUUID(nodeID),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) GetSummaryByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	row, err := s.q.GetSummaryByLevel(context.Background(), db.GetSummaryByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	return toDomainSummary(row), nil
}

func (s *dbSummaryService) ListSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	rows, err := s.q.ListSummariesByLevel(context.Background(), db.ListSummariesByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return nil, err
	}
	result := make([]compiler.StorySummary, len(rows))
	for i, r := range rows {
		result[i] = *toDomainSummary(r)
	}
	return result, nil
}

func (s *dbSummaryService) CountSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	count, err := s.q.CountSummariesByLevel(context.Background(), db.CountSummariesByLevelParams{
		StoryID: toUUID(storyID),
		Level:   string(level),
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *dbSummaryService) ShouldElevate(storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

func toDomainSummary(row db.StorySummary) *compiler.StorySummary {
	var nodeID *uuid.UUID
	if row.NodeID.Valid {
		n := fromUUID(row.NodeID)
		nodeID = &n
	}
	return &compiler.StorySummary{
		ID:        fromUUID(row.ID),
		StoryID:   fromUUID(row.StoryID),
		NodeID:    nodeID,
		Level:     compiler.SummaryLevel(row.Level),
		Content:   row.Content,
		WordCount: int(row.WordCount),
		CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
	}
}
