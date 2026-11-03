package repositorycontract_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/premchand/story-builder/internal/memory"
	storymongo "github.com/premchand/story-builder/internal/mongo"
	"github.com/premchand/story-builder/internal/story"
)

type repositoryFactory struct {
	name string
	new  func(*testing.T) story.Repository
}

func TestRepositoryContract(t *testing.T) {
	factories := []repositoryFactory{{name: "memory", new: func(*testing.T) story.Repository { return memory.NewRepository() }}}
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		factories = append(factories, repositoryFactory{name: "mongo", new: mongoFactory(uri)})
	} else {
		t.Log("Mongo contract skipped: set MONGO_URI to a replica-set MongoDB URI")
	}
	for _, factory := range factories {
		t.Run(factory.name, func(t *testing.T) {
			t.Run("accept_is_idempotent_and_orders_parents", func(t *testing.T) {
				ctx := context.Background()
				svc := story.NewService(factory.new(t))
				st, _ := svc.CreateStory(ctx, "Map")
				parent, _ := svc.CreateScene(ctx, st.ID, "Parent", "", story.Position{})
				child, _ := svc.CreateScene(ctx, st.ID, "Child", "", story.Position{})
				if _, err := svc.CreateEdge(ctx, st.ID, parent.ID, child.ID); err != nil {
					t.Fatal(err)
				}
				childDraft, _ := svc.GenerateDraft(ctx, child.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); !errors.Is(err, story.ErrParentsPending) {
					t.Fatalf("want parents pending, got %v", err)
				}
				parentDraft, _ := svc.GenerateDraft(ctx, parent.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, parentDraft.ID); err != nil {
					t.Fatal(err)
				}
				childDraft, _ = svc.GenerateDraft(ctx, child.ID, story.TemplateGenerator{}, func(string) error { return nil })
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.AcceptDraft(ctx, childDraft.ID); err != nil {
					t.Fatal(err)
				}
				latest, err := svc.Graph(ctx, st.ID)
				if err != nil || latest.Story.CanonVersion != 2 {
					t.Fatalf("want two canonical events, got %#v, %v", latest.Story, err)
				}
			})
			t.Run("concurrent_opposite_edges_remain_acyclic", func(t *testing.T) {
				ctx := context.Background()
				svc := story.NewService(factory.new(t))
				st, _ := svc.CreateStory(ctx, "Map")
				a, _ := svc.CreateScene(ctx, st.ID, "A", "", story.Position{})
				b, _ := svc.CreateScene(ctx, st.ID, "B", "", story.Position{})
				start := make(chan struct{})
				var wg sync.WaitGroup
				for _, edge := range [][2]string{{a.ID, b.ID}, {b.ID, a.ID}} {
					wg.Add(1)
					go func(from, to string) { defer wg.Done(); <-start; _, _ = svc.CreateEdge(ctx, st.ID, from, to) }(edge[0], edge[1])
				}
				close(start)
				wg.Wait()
				graph, err := svc.Graph(ctx, st.ID)
				if err != nil || story.HasCycle(graph.Edges) || len(graph.Edges) != 1 {
					t.Fatalf("invalid graph: %#v, %v", graph.Edges, err)
				}
			})
		})
	}
}

func mongoFactory(uri string) func(*testing.T) story.Repository {
	return func(t *testing.T) story.Repository {
		t.Helper()
		ctx := context.Background()
		database := "story_builder_contract_" + uuid.NewString()
		client, repo, err := storymongo.Connect(ctx, uri, database)
		if err != nil {
			t.Fatalf("connect Mongo contract database (requires a replica set): %v", err)
		}
		t.Cleanup(func() {
			_ = client.Database(database).Drop(ctx)
			_ = client.Disconnect(ctx)
		})
		return repo
	}
}
