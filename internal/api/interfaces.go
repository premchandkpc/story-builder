package api

import (
	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
)

type CharacterService interface {
	Create(name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	Get(id uuid.UUID, version int) (*canon.Character, error)
	Update(id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	List() ([]canon.Character, error)
}

type ActorService interface {
	Create(name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	Get(id uuid.UUID) (*canon.Actor, error)
	Update(id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	List() ([]canon.Actor, error)
}

type TraitService interface {
	Create(name, category, description string) (*canon.CharacterTrait, error)
	Get(id uuid.UUID) (*canon.CharacterTrait, error)
	List() ([]canon.CharacterTrait, error)
	Assign(characterID, traitID uuid.UUID, intensity int, note string) error
	Unassign(characterID, traitID uuid.UUID) error
	GetAssignments(characterID uuid.UUID) ([]canon.TraitAssignment, error)
}

type CastingService interface {
	Create(storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
	GetForStory(storyID uuid.UUID) ([]canon.Casting, error)
	GetForCharacter(characterID uuid.UUID) ([]canon.Casting, error)
	GetForActor(actorID uuid.UUID) ([]canon.Casting, error)
}

type LocationService interface {
	Create(name, description string, props []string) (*canon.Location, error)
	Get(id uuid.UUID, version int) (*canon.Location, error)
	Update(id uuid.UUID, description string, props []string) (*canon.Location, error)
	List() ([]canon.Location, error)
}

type LoreService interface {
	Create(tags []string, content string) (*canon.Lore, error)
	List() ([]canon.Lore, error)
	SearchByTags(tags []string) ([]canon.Lore, error)
	SearchSimilar(embedding []float32, limit int) ([]canon.Lore, error)
}

type StoryService interface {
	Create(title string) (*graph.Story, error)
	Get(id uuid.UUID) (*graph.Story, error)
	List() ([]graph.Story, error)
	CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType string) error
	ListEdges(storyID uuid.UUID) ([]graph.Edge, error)
	GetNode(id uuid.UUID) (*graph.Node, error)
	ListNodes(storyID uuid.UUID) ([]graph.Node, error)
	TopologicalSort(storyID uuid.UUID) ([]graph.Node, error)
}

type NodeService interface {
	Create(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error)
	Get(id uuid.UUID) (*graph.Node, error)
	Update(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error)
	SetSceneStructure(id uuid.UUID, ss graph.SceneStructure) error
	List(storyID uuid.UUID) ([]graph.Node, error)
}

type GenerationService interface {
	Generate(nodeID uuid.UUID) (*compiler.Generation, error)
	AcceptGeneration(nodeID, genID uuid.UUID) error
	ListGenerations(nodeID uuid.UUID) ([]compiler.Generation, error)
}

type StoryGeneratorService interface {
	GenerateStory(synopsis string) (*StoryGenerateResult, error)
}
