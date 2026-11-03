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
	CreateEdge(context.Context, Edge) error
	ListEdges(context.Context, string) ([]Edge, error)
}
