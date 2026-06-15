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

type ChapterStatus string

const (
	ChapterStatusDraft     ChapterStatus = "draft"
	ChapterStatusActive    ChapterStatus = "active"
	ChapterStatusCompleted ChapterStatus = "completed"
	ChapterStatusArchived  ChapterStatus = "archived"
)

type EdgeType string

const (
	EdgeTypeSeq     EdgeType = "seq"
	EdgeTypeFork    EdgeType = "fork"
	EdgeTypeJoin    EdgeType = "join"
	EdgeTypeChoice  EdgeType = "choice"
	EdgeTypeParallel EdgeType = "parallel"
)

type FlowType string

const (
	FlowMonologue  FlowType = "monologue"
	FlowDialogue   FlowType = "dialogue"
	FlowRoundRobin FlowType = "round_robin"
	FlowParallel   FlowType = "parallel"
	FlowAction     FlowType = "action"
	FlowSilent     FlowType = "silent"
	FlowCustom     FlowType = "custom"
)

type SceneStructure struct {
	FlowType       FlowType    `json:"flow_type"`
	CharacterOrder []uuid.UUID `json:"character_order,omitempty"`
	SituationFlow  string      `json:"situation_flow"`
	MaxTurns       int         `json:"max_turns,omitempty"`
}

type Story struct {
	ID            uuid.UUID              `json:"id"`
	Title         string                 `json:"title"`
	Genre         string                 `json:"genre"`
	Theme         string                 `json:"theme"`
	MainPrompt    string                 `json:"main_prompt"`
	GeneralPrompt string                 `json:"general_prompt"`
	CanonPins     map[string]interface{} `json:"canon_pins"`
	CreatedAt     time.Time              `json:"created_at"`
}

type Chapter struct {
	ID         uuid.UUID      `json:"id"`
	StoryID    uuid.UUID      `json:"story_id"`
	Title      string         `json:"title"`
	Goal       string         `json:"goal"`
	OrderIndex int            `json:"order_index"`
	Summary    string         `json:"summary"`
	Status     ChapterStatus  `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type Node struct {
	ID               uuid.UUID      `json:"id"`
	StoryID          uuid.UUID      `json:"story_id"`
	ChapterID        uuid.UUID      `json:"chapter_id"`
	Title            string         `json:"title"`
	BeatIntent       string         `json:"beat_intent"`
	CharacterRefs    []uuid.UUID    `json:"character_refs"`
	LocationRef      *uuid.UUID     `json:"location_ref"`
	POV              string         `json:"pov"`
	Tone             string         `json:"tone"`
	TargetWords      int            `json:"target_words"`
	Status           NodeStatus     `json:"status"`
	SceneStructure   *SceneStructure `json:"scene_structure,omitempty"`
	ParentSceneID    *uuid.UUID     `json:"parent_scene_id,omitempty"`
	TimelinePosition string         `json:"timeline_position"`
	FlowType         FlowType       `json:"flow_type"`
	MaxTurns         int            `json:"max_turns"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type Edge struct {
	StoryID   uuid.UUID `json:"story_id"`
	FromNode  uuid.UUID `json:"from_node"`
	ToNode    uuid.UUID `json:"to_node"`
	EdgeType  EdgeType  `json:"edge_type"`
	Condition string    `json:"condition"`
}

type GraphService interface {
	CreateStory(title string) (*Story, error)
	GetStory(id uuid.UUID) (*Story, error)
	ListStories() ([]Story, error)

	CreateChapter(storyID uuid.UUID, title string, orderIndex int) (*Chapter, error)
	GetChapter(id uuid.UUID) (*Chapter, error)
	ListChapters(storyID uuid.UUID) ([]Chapter, error)

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
	case EdgeTypeSeq, EdgeTypeFork, EdgeTypeJoin, EdgeTypeChoice, EdgeTypeParallel:
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
