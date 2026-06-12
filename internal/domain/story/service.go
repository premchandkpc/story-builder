package story

import (
	"context"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/graph"
)

type StoryService interface {
	Create(ctx context.Context, title string) (*graph.Story, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)
	List(ctx context.Context) ([]graph.Story, error)
	CreateNode(ctx context.Context, storyID uuid.UUID, spec CreateNodeSpec) (*graph.Node, error)
	UpdateNode(ctx context.Context, id uuid.UUID, spec UpdateNodeSpec) (*graph.Node, error)
	GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
	CreateEdge(ctx context.Context, storyID, from, to uuid.UUID, edgeType graph.EdgeType) error
	ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error)
	Topology(ctx context.Context, storyID uuid.UUID) (*Topology, error)
	GetIncomingEdges(ctx context.Context, nodeID uuid.UUID) ([]graph.Edge, error)
	GetOutgoingEdges(ctx context.Context, nodeID uuid.UUID) ([]graph.Edge, error)
	SetSceneStructure(ctx context.Context, nodeID uuid.UUID, ss graph.SceneStructure) error
	GetSceneStructure(ctx context.Context, nodeID uuid.UUID) (*graph.SceneStructure, error)
}

type CreateNodeSpec struct {
	BeatIntent    string
	CharacterRefs []uuid.UUID
	LocationRef   *uuid.UUID
	POV           string
	Tone          string
	TargetWords   int
}

type UpdateNodeSpec struct {
	BeatIntent    string
	CharacterRefs []uuid.UUID
	LocationRef   *uuid.UUID
	POV           string
	Tone          string
	TargetWords   int
	SceneStructure *graph.SceneStructure
}

type Topology struct {
	Nodes []graph.Node
	Edges []graph.Edge
}

type StoryRepository interface {
	Create(ctx context.Context, title string) (*graph.Story, error)
	Get(ctx context.Context, id uuid.UUID) (*graph.Story, error)
	List(ctx context.Context) ([]graph.Story, error)
	CreateNode(ctx context.Context, storyID uuid.UUID, spec CreateNodeSpec) (*graph.Node, error)
	UpdateNode(ctx context.Context, id uuid.UUID, spec UpdateNodeSpec) (*graph.Node, error)
	GetNode(ctx context.Context, id uuid.UUID) (*graph.Node, error)
	ListNodes(ctx context.Context, storyID uuid.UUID) ([]graph.Node, error)
	DeleteNode(ctx context.Context, id uuid.UUID) error
	CreateEdge(ctx context.Context, storyID, from, to uuid.UUID, edgeType graph.EdgeType) error
	ListEdges(ctx context.Context, storyID uuid.UUID) ([]graph.Edge, error)
	GetIncomingEdges(ctx context.Context, nodeID uuid.UUID) ([]graph.Edge, error)
	GetOutgoingEdges(ctx context.Context, nodeID uuid.UUID) ([]graph.Edge, error)
	SetSceneStructure(ctx context.Context, nodeID uuid.UUID, ss graph.SceneStructure) error
	GetSceneStructure(ctx context.Context, nodeID uuid.UUID) (*graph.SceneStructure, error)
}
