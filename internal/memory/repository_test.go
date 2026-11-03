package memory_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/memory"
	"github.com/premchand/story-builder/internal/story"
)

func TestConcurrentOppositeEdgesLeaveADAG(t *testing.T) {
	ctx := context.Background()
	svc := story.NewService(memory.NewRepository())
	st, err := svc.CreateStory(ctx, "Map")
	if err != nil {
		t.Fatal(err)
	}
	a, err := svc.CreateScene(ctx, st.ID, "A", "", story.Position{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateScene(ctx, st.ID, "B", "", story.Position{})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, edge := range [][2]string{{a.ID, b.ID}, {b.ID, a.ID}} {
		wg.Add(1)
		go func(from, to string) {
			defer wg.Done()
			<-start
			_, _ = svc.CreateEdge(ctx, st.ID, from, to)
		}(edge[0], edge[1])
	}
	close(start)
	wg.Wait()

	graph, err := svc.Graph(ctx, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	if story.HasCycle(graph.Edges) || len(graph.Edges) != 1 {
		t.Fatalf("concurrent inserts produced invalid graph: %#v", graph.Edges)
	}
}

func TestFindDraftReturnsOnlyLiveReadyDraft(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepository()
	now := time.Now().UTC()
	for _, draft := range []story.Draft{
		{ID: "failed", SceneID: "scene", ContextHash: "hash", Status: story.DraftFailed, ExpiresAt: now.Add(time.Hour)},
		{ID: "expired", SceneID: "scene", ContextHash: "hash", Status: story.DraftReady, ExpiresAt: now.Add(-time.Hour)},
		{ID: "ready", SceneID: "scene", ContextHash: "hash", Status: story.DraftReady, ExpiresAt: now.Add(time.Hour)},
	} {
		if err := repo.CreateDraft(ctx, draft); err != nil {
			t.Fatal(err)
		}
	}
	found, err := repo.FindDraft(ctx, "scene", "hash")
	if err != nil || found.ID != "ready" {
		t.Fatalf("want live ready draft, got %#v, %v", found, err)
	}
}
