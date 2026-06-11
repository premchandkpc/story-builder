package api

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
	"github.com/premchand/story-builder/internal/scene"
)

type charService struct {
	chars   []canon.Character
	version map[uuid.UUID]int
}

func NewCharService() *charService {
	return &charService{version: make(map[uuid.UUID]int)}
}

func (s *charService) Create(name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	c := canon.Character{
		ID:             uuid.New(),
		Version:        1,
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   voiceSamples,
		ParentID:       parentID,
		Relationships:  relationships,
		CreatedAt:      time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[c.ID] = 2
	return &c, nil
}

func (s *charService) Get(id uuid.UUID, version int) (*canon.Character, error) {
	var latest *canon.Character
	for i := range s.chars {
		if s.chars[i].ID == id {
			if version > 0 && s.chars[i].Version == version {
				return &s.chars[i], nil
			}
			if latest == nil || s.chars[i].Version > latest.Version {
				latest = &s.chars[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("character %s not found", id)
	}
	return latest, nil
}

func (s *charService) Update(id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("character %s not found", id)
	}
	c := canon.Character{
		ID:             id,
		Version:        next,
		Name:           name,
		Persona:        persona,
		Backstory:      backstory,
		MoralAlignment: moralAlignment,
		Personality:    personality,
		Flaws:          flaws,
		Goals:          goals,
		Traits:         traits,
		VoiceSamples:   voiceSamples,
		ParentID:       parentID,
		Relationships:  relationships,
		CreatedAt:      time.Now(),
	}
	s.chars = append(s.chars, c)
	s.version[id] = next + 1
	return &c, nil
}

func (s *charService) List() ([]canon.Character, error) {
	latest := make(map[uuid.UUID]canon.Character)
	for _, c := range s.chars {
		if existing, ok := latest[c.ID]; !ok || c.Version > existing.Version {
			latest[c.ID] = c
		}
	}
	result := make([]canon.Character, 0, len(latest))
	for _, c := range latest {
		result = append(result, c)
	}
	return result, nil
}

type actorService struct {
	actors []canon.Actor
}

func NewActorService() *actorService {
	return &actorService{}
}

func (s *actorService) Create(name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	if traits == nil {
		traits = make(map[string]interface{})
	}
	a := canon.Actor{
		ID:          uuid.New(),
		Name:        name,
		Gender:      gender,
		Ethnicity:   ethnicity,
		Race:        race,
		SkinTone:    skinTone,
		EyeColor:    eyeColor,
		HairColor:   hairColor,
		HairStyle:   hairStyle,
		Build:       build,
		HeightCm:    heightCm,
		WeightKg:    weightKg,
		Age:         age,
		Nationality: nationality,
		Traits:      traits,
		CreatedAt:   time.Now(),
	}
	s.actors = append(s.actors, a)
	return &a, nil
}

func (s *actorService) Get(id uuid.UUID) (*canon.Actor, error) {
	for i := range s.actors {
		if s.actors[i].ID == id {
			return &s.actors[i], nil
		}
	}
	return nil, fmt.Errorf("actor %s not found", id)
}

func (s *actorService) Update(id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error) {
	for i := range s.actors {
		if s.actors[i].ID == id {
			if traits == nil {
				traits = make(map[string]interface{})
			}
			s.actors[i].Name = name
			s.actors[i].Gender = gender
			s.actors[i].Ethnicity = ethnicity
			s.actors[i].Race = race
			s.actors[i].SkinTone = skinTone
			s.actors[i].EyeColor = eyeColor
			s.actors[i].HairColor = hairColor
			s.actors[i].HairStyle = hairStyle
			s.actors[i].Build = build
			s.actors[i].HeightCm = heightCm
			s.actors[i].WeightKg = weightKg
			s.actors[i].Age = age
			s.actors[i].Nationality = nationality
			s.actors[i].Traits = traits
			return &s.actors[i], nil
		}
	}
	return nil, fmt.Errorf("actor %s not found", id)
}

func (s *actorService) List() ([]canon.Actor, error) {
	r := make([]canon.Actor, len(s.actors))
	copy(r, s.actors)
	return r, nil
}

type characterTraitService struct {
	traits      []canon.CharacterTrait
	assignments []canon.TraitAssignment
}

func NewCharacterTraitService() *characterTraitService {
	return &characterTraitService{}
}

func (s *characterTraitService) Create(name, category, description string) (*canon.CharacterTrait, error) {
	t := canon.CharacterTrait{
		ID:          uuid.New(),
		Name:        name,
		Category:    category,
		Description: description,
		CreatedAt:   time.Now(),
	}
	s.traits = append(s.traits, t)
	return &t, nil
}

func (s *characterTraitService) Get(id uuid.UUID) (*canon.CharacterTrait, error) {
	for i := range s.traits {
		if s.traits[i].ID == id {
			return &s.traits[i], nil
		}
	}
	return nil, fmt.Errorf("trait %s not found", id)
}

func (s *characterTraitService) List() ([]canon.CharacterTrait, error) {
	r := make([]canon.CharacterTrait, len(s.traits))
	copy(r, s.traits)
	return r, nil
}

func (s *characterTraitService) Assign(characterID, traitID uuid.UUID, intensity int, note string) error {
	for i := range s.assignments {
		if s.assignments[i].CharacterID == characterID && s.assignments[i].TraitID == traitID {
			s.assignments[i].Intensity = intensity
			s.assignments[i].Note = note
			return nil
		}
	}
	s.assignments = append(s.assignments, canon.TraitAssignment{
		CharacterID: characterID,
		TraitID:     traitID,
		Intensity:   intensity,
		Note:        note,
	})
	return nil
}

func (s *characterTraitService) Unassign(characterID, traitID uuid.UUID) error {
	for i := range s.assignments {
		if s.assignments[i].CharacterID == characterID && s.assignments[i].TraitID == traitID {
			s.assignments = append(s.assignments[:i], s.assignments[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *characterTraitService) GetAssignments(characterID uuid.UUID) ([]canon.TraitAssignment, error) {
	var result []canon.TraitAssignment
	for _, a := range s.assignments {
		if a.CharacterID == characterID {
			result = append(result, a)
		}
	}
	return result, nil
}

type castingService struct {
	casts []canon.Casting
}

func NewCastingService() *castingService {
	return &castingService{}
}

func (s *castingService) Create(storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error) {
	c := canon.Casting{
		ID:          uuid.New(),
		StoryID:     storyID,
		ActorID:     actorID,
		CharacterID: characterID,
		RoleType:    roleType,
		CreatedAt:   time.Now(),
	}
	s.casts = append(s.casts, c)
	return &c, nil
}

func (s *castingService) GetForStory(storyID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.StoryID == storyID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *castingService) GetForCharacter(characterID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.CharacterID == characterID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (s *castingService) GetForActor(actorID uuid.UUID) ([]canon.Casting, error) {
	var result []canon.Casting
	for _, c := range s.casts {
		if c.ActorID == actorID {
			result = append(result, c)
		}
	}
	return result, nil
}

type memorySummaryService struct {
	scene map[uuid.UUID]string // nodeID → content
	act   map[uuid.UUID]string // storyID → content
	story map[uuid.UUID]string // storyID → content
}

func NewMemorySummaryService() *memorySummaryService {
	return &memorySummaryService{
		scene: make(map[uuid.UUID]string),
		act:   make(map[uuid.UUID]string),
		story: make(map[uuid.UUID]string),
	}
}

func (s *memorySummaryService) UpsertSceneSummary(storyID, nodeID uuid.UUID, content string) error {
	s.scene[nodeID] = content
	return nil
}

func (s *memorySummaryService) UpsertActSummary(storyID uuid.UUID, content string) error {
	s.act[storyID] = content
	return nil
}

func (s *memorySummaryService) UpsertStorySummary(storyID uuid.UUID, content string) error {
	s.story[storyID] = content
	return nil
}

func (s *memorySummaryService) GetSceneSummary(storyID, nodeID uuid.UUID) (*compiler.StorySummary, error) {
	content, ok := s.scene[nodeID]
	if !ok {
		return nil, fmt.Errorf("scene summary not found for node %s", nodeID)
	}
	return &compiler.StorySummary{
		ID:        uuid.New(),
		StoryID:   storyID,
		NodeID:    &nodeID,
		Level:     compiler.SummaryScene,
		Content:   content,
		WordCount: len(content),
	}, nil
}

func (s *memorySummaryService) GetSummaryByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (*compiler.StorySummary, error) {
	switch level {
	case compiler.SummaryAct:
		content, ok := s.act[storyID]
		if !ok {
			return nil, fmt.Errorf("act summary not found for story %s", storyID)
		}
		return &compiler.StorySummary{
			ID:        uuid.New(),
			StoryID:   storyID,
			Level:     compiler.SummaryAct,
			Content:   content,
			WordCount: len(content),
		}, nil
	case compiler.SummaryStory:
		content, ok := s.story[storyID]
		if !ok {
			return nil, fmt.Errorf("story summary not found for story %s", storyID)
		}
		return &compiler.StorySummary{
			ID:        uuid.New(),
			StoryID:   storyID,
			Level:     compiler.SummaryStory,
			Content:   content,
			WordCount: len(content),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported level %s", level)
	}
}

func (s *memorySummaryService) ListSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) ([]compiler.StorySummary, error) {
	summary, err := s.GetSummaryByLevel(storyID, level)
	if err != nil {
		return nil, err
	}
	return []compiler.StorySummary{*summary}, nil
}

func (s *memorySummaryService) CountSummariesByLevel(storyID uuid.UUID, level compiler.SummaryLevel) (int, error) {
	if level == compiler.SummaryScene {
		count := 0
		for _, v := range s.scene {
			if v != "" {
				count++
			}
		}
		return count, nil
	}
	return 0, nil
}

func (s *memorySummaryService) ShouldElevate(storyID uuid.UUID, level compiler.SummaryLevel, threshold int) (bool, error) {
	count, err := s.CountSummariesByLevel(storyID, level)
	if err != nil {
		return false, err
	}
	return count >= threshold, nil
}

type locService struct {
	locs    []canon.Location
	version map[uuid.UUID]int
}

func NewLocService() *locService {
	return &locService{version: make(map[uuid.UUID]int)}
}

func (s *locService) Create(name, description string, props []string) (*canon.Location, error) {
	l := canon.Location{
		ID:          uuid.New(),
		Version:     1,
		Name:        name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[l.ID] = 2
	return &l, nil
}

func (s *locService) Get(id uuid.UUID, version int) (*canon.Location, error) {
	var latest *canon.Location
	for i := range s.locs {
		if s.locs[i].ID == id {
			if version > 0 && s.locs[i].Version == version {
				return &s.locs[i], nil
			}
			if latest == nil || s.locs[i].Version > latest.Version {
				latest = &s.locs[i]
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("location %s not found", id)
	}
	return latest, nil
}

func (s *locService) Update(id uuid.UUID, description string, props []string) (*canon.Location, error) {
	next := s.version[id]
	if next == 0 {
		return nil, fmt.Errorf("location %s not found", id)
	}
	l := canon.Location{
		ID:          id,
		Version:     next,
		Name:        s.locs[0].Name,
		Description: description,
		Props:       props,
		CreatedAt:   time.Now(),
	}
	s.locs = append(s.locs, l)
	s.version[id] = next + 1
	return &l, nil
}

func (s *locService) List() ([]canon.Location, error) {
	latest := make(map[uuid.UUID]canon.Location)
	for _, l := range s.locs {
		if existing, ok := latest[l.ID]; !ok || l.Version > existing.Version {
			latest[l.ID] = l
		}
	}
	result := make([]canon.Location, 0, len(latest))
	for _, l := range latest {
		result = append(result, l)
	}
	return result, nil
}

type loreService struct {
	items []canon.Lore
}

func NewLoreService() *loreService {
	return &loreService{}
}

func (s *loreService) Create(tags []string, content string) (*canon.Lore, error) {
	l := canon.Lore{
		ID:        uuid.New(),
		Tags:      tags,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.items = append(s.items, l)
	return &l, nil
}

func (s *loreService) List() ([]canon.Lore, error) {
	r := make([]canon.Lore, len(s.items))
	copy(r, s.items)
	return r, nil
}

func (s *loreService) SearchByTags(tags []string) ([]canon.Lore, error) {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var result []canon.Lore
	for _, l := range s.items {
		for _, t := range l.Tags {
			if tagSet[t] {
				result = append(result, l)
				break
			}
		}
	}
	return result, nil
}

func (s *loreService) SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error) {
	if limit > len(s.items) {
		limit = len(s.items)
	}
	return s.items[:limit], nil
}

type graphStoryService struct {
	graph *graph.MemoryStore
}

func NewGraphStoryService(gs *graph.MemoryStore) *graphStoryService {
	return &graphStoryService{graph: gs}
}

func (s *graphStoryService) Create(title string) (*graph.Story, error) { return s.graph.CreateStory(title) }
func (s *graphStoryService) Get(id uuid.UUID) (*graph.Story, error)   { return s.graph.GetStory(id) }
func (s *graphStoryService) List() ([]graph.Story, error)              { return s.graph.ListStories() }

func (s *graphStoryService) CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error {
	et := graph.EdgeType(edgeType)
	if !et.Valid() {
		et = graph.EdgeTypeSeq
	}
	return s.graph.CreateEdge(storyID, fromNode, toNode, et)
}

func (s *graphStoryService) ListEdges(storyID uuid.UUID) ([]graph.Edge, error) {
	return s.graph.ListEdges(storyID)
}

func (s *graphStoryService) GetNode(id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphStoryService) ListNodes(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

func (s *graphStoryService) TopologicalSort(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.TopologicalSort(storyID)
}

type graphNodeService struct {
	graph *graph.MemoryStore
}

func NewGraphNodeService(gs *graph.MemoryStore) *graphNodeService {
	return &graphNodeService{graph: gs}
}

func (s *graphNodeService) Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error) {
	return s.graph.CreateNode(storyID, beatIntent, characterRefs, locationRef, pov, tone, targetWords)
}

func (s *graphNodeService) Get(id uuid.UUID) (*graph.Node, error) {
	return s.graph.GetNode(id)
}

func (s *graphNodeService) Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error) {
	return s.graph.UpdateNode(id, beatIntent, characterRefs, locationRef, pov, tone, targetWords, sceneStructure)
}

func (s *graphNodeService) SetSceneStructure(id uuid.UUID, ss graph.SceneStructure) error {
	return s.graph.SetSceneStructure(id, ss)
}

func (s *graphNodeService) List(storyID uuid.UUID) ([]graph.Node, error) {
	return s.graph.ListNodes(storyID)
}

type generationService struct {
	gens []compiler.Generation
}

func NewGenerationService() *generationService {
	return &generationService{}
}

func (s *generationService) Generate(nodeID uuid.UUID) (*compiler.Generation, error) {
	return nil, fmt.Errorf("generation requires LLM integration — not implemented in memory mode")
}

func (s *generationService) AcceptGeneration(nodeID, genID uuid.UUID) error {
	for i := range s.gens {
		if s.gens[i].ID == genID.String() && s.gens[i].NodeID == nodeID.String() {
			s.gens[i].Accepted = true
			return nil
		}
	}
	return fmt.Errorf("generation %s not found for node %s", genID, nodeID)
}

func (s *generationService) ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error) {
	var result []compiler.Generation
	for _, g := range s.gens {
		if g.NodeID == nodeID.String() {
			result = append(result, g)
		}
	}
	return result, nil
}

type memorySceneService struct {
	turns map[uuid.UUID][]scene.SceneTurn
}

func NewMemorySceneService() *memorySceneService {
	return &memorySceneService{turns: make(map[uuid.UUID][]scene.SceneTurn)}
}

func (s *memorySceneService) StartScene(nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration — not implemented in memory mode")
}

func (s *memorySceneService) NextTurn(nodeID uuid.UUID) (*scene.SceneTurn, error) {
	return nil, fmt.Errorf("multi-agent scene requires LLM integration — not implemented in memory mode")
}

func (s *memorySceneService) FinishScene(nodeID uuid.UUID) (string, error) {
	return "", fmt.Errorf("multi-agent scene requires LLM integration — not implemented in memory mode")
}

func (s *memorySceneService) GetTurns(nodeID uuid.UUID) ([]scene.SceneTurn, error) {
	turns := s.turns[nodeID]
	r := make([]scene.SceneTurn, len(turns))
	copy(r, turns)
	return r, nil
}

func (s *memorySceneService) SetSceneStructure(nodeID uuid.UUID, ss graph.SceneStructure) error {
	return nil
}

func (s *memorySceneService) GetSceneStructure(nodeID uuid.UUID) (*graph.SceneStructure, error) {
	return nil, nil
}
