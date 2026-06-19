package events

const (
	// Pipeline events
	EventSceneGenerated          = "scene.generated"
	EventCharacterStatesExtracted = "states.extracted"
	EventMemoriesCreated         = "memories.created"
	EventTimelineRecorded        = "timeline.recorded"
	EventSummaryUpdated          = "summary.updated"
	EventSceneValidated          = "scene.validated"
	EventGenerationAccepted      = "generation.accepted"
	EventPipelineComplete        = "pipeline.complete"
	EventPipelineFailed          = "pipeline.failed"

	// Entity lifecycle events
	EventStoryCreated    = "story.created"
	EventStoryUpdated    = "story.updated"
	EventStoryDeleted    = "story.deleted"
	EventCharacterCreated = "character.created"
	EventCharacterUpdated = "character.updated"
	EventCharacterDeleted = "character.deleted"
	EventSceneCreated    = "scene.created"
	EventSceneUpdated    = "scene.updated"
	EventSceneDeleted    = "scene.deleted"
	EventEdgeCreated     = "edge.created"
	EventEdgeDeleted     = "edge.deleted"
	EventChapterCreated  = "chapter.created"
	EventChapterUpdated  = "chapter.updated"
	EventChapterDeleted  = "chapter.deleted"
	EventLocationCreated = "location.created"
	EventLocationUpdated = "location.updated"
	EventLocationDeleted = "location.deleted"
	EventBibleCreated    = "bible.created"
	EventBibleUpdated    = "bible.updated"
	EventBibleDeleted    = "bible.deleted"
	EventGenerationStatusChanged = "generation.status_changed"
)
