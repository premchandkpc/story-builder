package event

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EvStoryCreated           EventType = "story.created"
	EvChapterCreated         EventType = "chapter.created"
	EvSceneCreated           EventType = "scene.created"
	EvSceneGenerated         EventType = "scene.generated"
	EvDialogueGenerated      EventType = "dialogue.generated"
	EvMemoryCreated          EventType = "memory.created"
	EvMemoryRetrieved        EventType = "memory.retrieved"
	EvRelationshipChanged    EventType = "relationship.changed"
	EvTimelineUpdated        EventType = "timeline.updated"
	EvPromptCompiled         EventType = "prompt.compiled"
	EvLocalizationApplied    EventType = "localization.applied"
	EvImageRendered          EventType = "image.rendered"
	EvVideoRendered          EventType = "video.rendered"
	EvGenerationCompleted    EventType = "generation.completed"
	EvGenerationFailed       EventType = "generation.failed"
	EvCanonViolationDetected EventType = "canon.violation_detected"
	EvCharacterMoved         EventType = "character.moved"
	EvEmotionChanged         EventType = "emotion.changed"
	EvStateDeltaApplied      EventType = "state_delta.applied"
)

type Event struct {
	ID          uuid.UUID       `json:"id"`
	Type        EventType       `json:"type"`
	AggregateID uuid.UUID       `json:"aggregate_id"`
	StoryID     uuid.UUID       `json:"story_id,omitempty"`
	SceneID     uuid.UUID       `json:"scene_id,omitempty"`
	CharID      uuid.UUID       `json:"character_id,omitempty"`
	GenID       uuid.UUID       `json:"generation_id,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	TraceID     string          `json:"trace_id,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

type Store interface {
	Append(evt *Event) error
	GetByAggregate(aggregateID uuid.UUID, evtType EventType) ([]Event, error)
	GetByStory(storyID uuid.UUID, evtType EventType, limit int) ([]Event, error)
	GetByType(evtType EventType, since time.Time, limit int) ([]Event, error)
	Replay(aggregateID uuid.UUID) ([]Event, error)
}

type Bus interface {
	Publish(evt *Event) error
	Subscribe(evtType EventType, handler func(*Event) error)
	Start()
	Stop()
}
