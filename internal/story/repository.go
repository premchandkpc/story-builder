package story

import "context"

// Repository is the persistence boundary for the graph. It intentionally
// contains only operations that the graph workspace needs in Slice 1.
type Repository interface {
	CreateStory(context.Context, Story) error
	GetStory(context.Context, string) (Story, error)
	ListStories(context.Context) ([]Story, error)
	CreateScene(context.Context, Scene) error
	GetScene(context.Context, string) (Scene, error)
	UpdateScene(context.Context, Scene) error
	ListScenes(context.Context, string) ([]Scene, error)
	// CreateEdge must atomically revalidate the DAG before inserting the edge.
	// Service-level validation is for prompt errors; this is the concurrency boundary.
	CreateEdge(context.Context, Edge) error
	ListEdges(context.Context, string) ([]Edge, error)
	CreateDraft(context.Context, Draft) error
	FindDraft(context.Context, string, string) (Draft, error)
	GetDraft(context.Context, string) (Draft, error)
	UpdateDraft(context.Context, Draft) error
	DeleteDraft(context.Context, string) error
	AcceptDraft(context.Context, string) (Scene, error)
	CreateCharacter(context.Context, Character) error
	ListCharacters(context.Context, string) ([]Character, error)
	ListCanonEvents(context.Context, string) ([]CanonEvent, error)
	ListCharacterStates(context.Context, string) ([]CharacterState, error)
}
