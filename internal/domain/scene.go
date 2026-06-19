package domain

import (
	"fmt"
	"time"
)

type Scene struct {
	ID               string                 `bson:"_id" json:"id"`
	StoryID          string                 `bson:"storyId" json:"storyId"`
	ChapterID        string                 `bson:"chapterId,omitempty" json:"chapterId,omitempty"`
	Title            string                 `bson:"title,omitempty" json:"title,omitempty"`
	BeatIntent       string                 `bson:"beatIntent,omitempty" json:"beatIntent,omitempty"`
	Summary          string                 `bson:"summary,omitempty" json:"summary,omitempty"`
	GeneratedContent string                 `bson:"generatedContent,omitempty" json:"generatedContent,omitempty"`
	Participants     []string               `bson:"participants,omitempty" json:"participants,omitempty"`
	LocationRef      string                 `bson:"locationRef,omitempty" json:"locationRef,omitempty"`
	POV              string                 `bson:"pov,omitempty" json:"pov,omitempty"`
	Tone             string                 `bson:"tone,omitempty" json:"tone,omitempty"`
	TargetWords      int                    `bson:"targetWords,omitempty" json:"targetWords,omitempty"`
	FlowType         string                 `bson:"flowType,omitempty" json:"flowType,omitempty"`
	MaxTurns         int                    `bson:"maxTurns,omitempty" json:"maxTurns,omitempty"`
	TimelinePosition int                    `bson:"timelinePosition,omitempty" json:"timelinePosition,omitempty"`
	Status           string                 `bson:"status" json:"status"`
	SceneStructure   map[string]any         `bson:"sceneStructure,omitempty" json:"sceneStructure,omitempty"`
	Metadata         map[string]any         `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt        time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time              `bson:"updatedAt" json:"updatedAt"`
}

type SceneEdge struct {
	ID          string    `bson:"_id" json:"id"`
	StoryID     string    `bson:"storyId" json:"storyId"`
	FromSceneID string    `bson:"fromSceneId" json:"fromSceneId"`
	ToSceneID   string    `bson:"toSceneId" json:"toSceneId"`
	Type        string    `bson:"type" json:"type"`
	Condition   string    `bson:"condition,omitempty" json:"condition,omitempty"`
	CreatedAt   time.Time `bson:"createdAt" json:"createdAt"`
}

type Generation struct {
	ID               string            `bson:"_id" json:"id"`
	StoryID          string            `bson:"storyId" json:"storyId"`
	SceneID          string            `bson:"sceneId" json:"sceneId"`
	ContextHash      string            `bson:"contextHash,omitempty" json:"contextHash,omitempty"`
	PromptSnapshot   string            `bson:"promptSnapshot,omitempty" json:"promptSnapshot,omitempty"`
	Output           string            `bson:"output,omitempty" json:"output,omitempty"`
	Model            string            `bson:"model,omitempty" json:"model,omitempty"`
	Status           string            `bson:"status,omitempty" json:"status,omitempty"`
	Accepted         bool              `bson:"accepted" json:"accepted"`
	StepStatus       map[string]string `bson:"stepStatus,omitempty" json:"stepStatus,omitempty"`
	ValidationResult map[string]any    `bson:"validationResult,omitempty" json:"validationResult,omitempty"`
	Error            string            `bson:"error,omitempty" json:"error,omitempty"`
	PromptTokens     int               `bson:"promptTokens,omitempty" json:"promptTokens,omitempty"`
	CompletionTokens int               `bson:"completionTokens,omitempty" json:"completionTokens,omitempty"`
	TotalTokens      int               `bson:"totalTokens,omitempty" json:"totalTokens,omitempty"`
	DurationMs       int64             `bson:"durationMs,omitempty" json:"durationMs,omitempty"`
	CreatedAt        time.Time         `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time         `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

const (
	GenStatusPending        = "pending"
	GenStatusRunning        = "running"
	GenStatusPartialSuccess = "partial_success"
	GenStatusSuccess        = "success"
	GenStatusFailed         = "failed"
)

const (
	StepPending = "pending"
	StepRunning = "running"
	StepDone    = "done"
	StepFailed  = "failed"
)

const (
	EdgeTypeSeq      = "seq"
	EdgeTypeFork     = "fork"
	EdgeTypeJoin     = "join"
	EdgeTypeChoice   = "choice"
	EdgeTypeParallel = "parallel"

	FlowTypeMonologue  = "monologue"
	FlowTypeDialogue   = "dialogue"
	FlowTypeRoundRobin = "round_robin"
	FlowTypeParallel   = "parallel"
	FlowTypeAction     = "action"
	FlowTypeSilent     = "silent"
	FlowTypeCustom     = "custom"

	SceneStatusDraft     = "draft"
	SceneStatusGenerated = "generated"
	SceneStatusAccepted  = "accepted"
	SceneStatusStale     = "stale"
)

var validSceneTransitions = map[string][]string{
	SceneStatusDraft:     {SceneStatusGenerated},
	SceneStatusGenerated: {SceneStatusAccepted, SceneStatusStale},
	SceneStatusAccepted:  {SceneStatusStale},
	SceneStatusStale:     {SceneStatusGenerated},
}

func (s *Scene) CanTransitionTo(target string) error {
	allowed, ok := validSceneTransitions[s.Status]
	if !ok {
		return fmt.Errorf("unknown scene status: %s", s.Status)
	}
	if s.Status == target {
		return nil
	}
	for _, a := range allowed {
		if a == target {
			return nil
		}
	}
	return fmt.Errorf("cannot transition scene from %s to %s", s.Status, target)
}
