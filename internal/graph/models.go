package graph

import (
	"time"

	"github.com/google/uuid"
)

type NodeStatus string

const (
	NodeStatusDraft     NodeStatus = "draft"
	NodeStatusGenerated NodeStatus = "generated"
	NodeStatusAccepted  NodeStatus = "accepted"
	NodeStatusStale     NodeStatus = "stale"
)

type EdgeType string

const (
	EdgeTypeSeq    EdgeType = "seq"
	EdgeTypeFork   EdgeType = "fork"
	EdgeTypeJoin   EdgeType = "join"
	EdgeTypeChoice EdgeType = "choice"
)

type FlowType string

const (
	FlowMonologue  FlowType = "monologue"
	FlowDialogue   FlowType = "dialogue"
	FlowRoundRobin FlowType = "round_robin"
	FlowParallel   FlowType = "parallel"
	FlowCustom     FlowType = "custom"
)

type SceneStructure struct {
	FlowType       FlowType    `json:"flow_type"`
	CharacterOrder []uuid.UUID `json:"character_order,omitempty"`
	SituationFlow  string      `json:"situation_flow"`
	MaxTurns       int         `json:"max_turns,omitempty"`
}

type Node struct {
	ID             uuid.UUID      `json:"id"`
	StoryID        uuid.UUID      `json:"story_id"`
	BeatIntent     string         `json:"beat_intent"`
	CharacterRefs  []uuid.UUID    `json:"character_refs"`
	LocationRef    *uuid.UUID     `json:"location_ref"`
	POV            string         `json:"pov"`
	Tone           string         `json:"tone"`
	TargetWords    int            `json:"target_words"`
	Status         NodeStatus     `json:"status"`
	SceneStructure *SceneStructure `json:"scene_structure,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Edge struct {
	StoryID   uuid.UUID `json:"story_id"`
	FromNode  uuid.UUID `json:"from_node"`
	ToNode    uuid.UUID `json:"to_node"`
	EdgeType  EdgeType  `json:"edge_type"`
}

type Story struct {
	ID        uuid.UUID              `json:"id"`
	Title     string                 `json:"title"`
	CanonPins map[string]interface{} `json:"canon_pins"`
	CreatedAt time.Time              `json:"created_at"`
}

type GraphService interface {
	CreateStory(title string) (*Story, error)
	GetStory(id uuid.UUID) (*Story, error)

	CreateNode(storyID uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int) (*Node, error)
	GetNode(id uuid.UUID) (*Node, error)
	UpdateNode(id uuid.UUID, beatIntent string, characterRefs []uuid.UUID, locationRef *uuid.UUID, pov, tone string, targetWords int, sceneStructure *SceneStructure) (*Node, error)
	SetNodeStatus(id uuid.UUID, status NodeStatus) error
	SetSceneStructure(id uuid.UUID, ss SceneStructure) error
	ListNodes(storyID uuid.UUID) ([]Node, error)

	CreateEdge(storyID, fromNode, toNode uuid.UUID, edgeType EdgeType) error
	ListEdges(storyID uuid.UUID) ([]Edge, error)
	GetOutgoingEdges(nodeID uuid.UUID) ([]Edge, error)
	GetIncomingEdges(nodeID uuid.UUID) ([]Edge, error)

	TopologicalSort(storyID uuid.UUID) ([]Node, error)
	Predecessors(nodeID uuid.UUID) ([]Node, error)
	IsForkJoin(storyID uuid.UUID) ([]Edge, error)

	BranchNodes(storyID uuid.UUID, forkNode uuid.UUID) ([]Node, error)
	ForkCharacterSets(storyID uuid.UUID, forkNode uuid.UUID) (map[uuid.UUID][]uuid.UUID, error)
}

func (ns NodeStatus) Valid() bool {
	switch ns {
	case NodeStatusDraft, NodeStatusGenerated, NodeStatusAccepted, NodeStatusStale:
		return true
	}
	return false
}

func (et EdgeType) Valid() bool {
	switch et {
	case EdgeTypeSeq, EdgeTypeFork, EdgeTypeJoin, EdgeTypeChoice:
		return true
	}
	return false
}

func (et EdgeType) IsBranching() bool {
	return et == EdgeTypeFork || et == EdgeTypeChoice
}

func (et EdgeType) IsConverging() bool {
	return et == EdgeTypeJoin
}
