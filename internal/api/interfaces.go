package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/canon"
	"github.com/premchand/story-builder/internal/compiler"
	"github.com/premchand/story-builder/internal/graph"
)

type CharacterService interface {
	Create(ctx context.Context, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	Get(ctx context.Context, id uuid.UUID, version int) (*canon.Character, error)
	Update(ctx context.Context, id uuid.UUID, name, persona, backstory, moralAlignment string, personality, flaws, goals, traits, voiceSamples []string, parentID *uuid.UUID, relationships map[string]string) (*canon.Character, error)
	List(ctx context.Context) ([]canon.Character, error)
}

type ActorService interface {
	Create(ctx context.Context, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	Get(ctx context.Context, id uuid.UUID) (*canon.Actor, error)
	Update(ctx context.Context, id uuid.UUID, name, gender, ethnicity, race, skinTone, eyeColor, hairColor, hairStyle, build, nationality string, heightCm, weightKg, age int, traits map[string]interface{}) (*canon.Actor, error)
	List(ctx context.Context) ([]canon.Actor, error)
}

type TraitService interface {
	Create(ctx context.Context, name, category, description string) (*canon.CharacterTrait, error)
	Get(ctx context.Context, id uuid.UUID) (*canon.CharacterTrait, error)
	List(ctx context.Context) ([]canon.CharacterTrait, error)
	Assign(ctx context.Context, characterID, traitID uuid.UUID, intensity int, note string) error
	Unassign(ctx context.Context, characterID, traitID uuid.UUID) error
	GetAssignments(ctx context.Context, characterID uuid.UUID) ([]canon.TraitAssignment, error)
}

type CastingService interface {
	Create(ctx context.Context, storyID, actorID, characterID uuid.UUID, roleType string) (*canon.Casting, error)
	GetForStory(ctx context.Context, storyID uuid.UUID) ([]canon.Casting, error)
	GetForCharacter(ctx context.Context, characterID uuid.UUID) ([]canon.Casting, error)
	GetForActor(ctx context.Context, actorID uuid.UUID) ([]canon.Casting, error)
}

type LocationService interface {
	Create(ctx context.Context, name, description string, props []string) (*canon.Location, error)
	Get(ctx context.Context, id uuid.UUID, version int) (*canon.Location, error)
	Update(ctx context.Context, id uuid.UUID, description string, props []string) (*canon.Location, error)
	List(ctx context.Context) ([]canon.Location, error)
}

type LoreService interface {
	Create(ctx context.Context, tags []string, content string) (*canon.Lore, error)
	List(ctx context.Context) ([]canon.Lore, error)
	SearchByTags(ctx context.Context, tags []string) ([]canon.Lore, error)
	SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]canon.Lore, error)
}

type StoryService interface {
	Create(ctx context.Context, title string) (*graph.Story, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)
	List(ctx context.Context) ([]graph.Story, error)
	CreateEdge(ctx context.Context, storyID, fromNode, toNode uuid.UUID, edgeType string) error
	ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error)
	GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
	TopologicalSort(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
}

type NodeService interface {
	Create(ctx context.Context, storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*graph.Node, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	Update(ctx context.Context, id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *graph.SceneStructure) (*graph.Node, error)
	SetSceneStructure(ctx context.Context, id uuid.UUID, ss graph.SceneStructure) error
	List(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
}

type GenerationService interface {
	Generate(ctx context.Context, nodeID uuid.UUID) (*compiler.Generation, error)
	AcceptGeneration(ctx context.Context, nodeID, genID uuid.UUID) error
	ListGenerations(ctx context.Context, nodeID uuid.UUID) ([]compiler.Generation, error)
}

type StoryGeneratorService interface {
	GenerateStory(ctx context.Context, synopsis string) (*StoryGenerateResult, error)
}
