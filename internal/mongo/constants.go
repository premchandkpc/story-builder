package mongo

const (
	collectionStories     = "stories"
	collectionScenes      = "scenes"
	collectionEdges       = "edges"
	collectionDrafts      = "drafts"
	collectionCharacters  = "characters"
	collectionCanonEvents = "canon_events"

	fieldID           = "_id"
	fieldStoryID      = "storyId"
	fieldSceneID      = "sceneId"
	fieldFromSceneID  = "fromSceneId"
	fieldToSceneID    = "toSceneId"
	fieldCharacterID  = "characterId"
	fieldVersion      = "version"
	fieldCanonVersion = "canonVersion"
	fieldContextHash  = "contextHash"
	fieldStatus       = "status"
	fieldExpiresAt    = "expiresAt"
	fieldCreatedAt    = "createdAt"
	fieldSequence     = "seq"
)
