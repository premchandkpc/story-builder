package domain

import "time"

type NarrativeEvent struct {
	ID          string         `bson:"_id" json:"id"`
	StoryID     string         `bson:"storyId" json:"storyId"`
	SceneID     string         `bson:"sceneId,omitempty" json:"sceneId,omitempty"`
	SourceRunID string         `bson:"sourceRunId,omitempty" json:"sourceRunId,omitempty"`
	SourceAgent string         `bson:"sourceAgent,omitempty" json:"sourceAgent,omitempty"`
	EventType   string         `bson:"eventType" json:"eventType"`
	SubjectType string         `bson:"subjectType" json:"subjectType"`
	SubjectID   string         `bson:"subjectId" json:"subjectId"`
	Payload     map[string]any `bson:"payload,omitempty" json:"payload,omitempty"`
	Confidence  float64        `bson:"confidence" json:"confidence"`
	Version     int64          `bson:"version" json:"version"`
	CreatedAt   time.Time      `bson:"createdAt" json:"createdAt"`
}

const (
	NarrativeEventTypeCharLocation  = "character.location.changed"
	NarrativeEventTypeCharGoal      = "character.goal.updated"
	NarrativeEventTypeCharEmotion   = "character.emotion.changed"
	NarrativeEventTypeCharKnowledge = "character.knowledge.added"
	NarrativeEventTypeRelTrust      = "relationship.trust.changed"
	NarrativeEventTypeRelStatus     = "relationship.status.changed"
	NarrativeEventTypeTimeline      = "timeline.event.recorded"
	NarrativeEventTypePlotOpened    = "plot_thread.opened"
	NarrativeEventTypePlotAdvanced  = "plot_thread.advanced"
	NarrativeEventTypePlotResolved  = "plot_thread.resolved"
	NarrativeEventTypeCanonAsserted = "canon.fact.asserted"
	NarrativeEventTypeCanonRetract  = "canon.fact.retracted"
	NarrativeEventTypeWorldChange   = "world.state.changed"
)

const (
	NarrativeSubjectChar     = "character"
	NarrativeSubjectRel      = "relationship"
	NarrativeSubjectWorld    = "world"
	NarrativeSubjectTimeline = "timeline"
	NarrativeSubjectMemory   = "memory"
	NarrativeSubjectPlot     = "plot_thread"
	NarrativeSubjectCanon    = "canon"
)
