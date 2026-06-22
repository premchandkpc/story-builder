//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/prompt"
)

func buildServer(t *testing.T) (*api.Server, *mgorepo.StoryRepo) {
	t.Helper()

	storyRepo := mgorepo.NewStoryRepo(testDB)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	stateRepo := mgorepo.NewCharacterStateRepo(testDB)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	memRepo := mgorepo.NewMemoryRepo(testDB)
	tlRepo := mgorepo.NewTimelineRepo(testDB)
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	locRepo := mgorepo.NewLocationRepo(testDB)
	bibleRepo := mgorepo.NewBibleRepo(testDB)
	chapterRepo := mgorepo.NewChapterRepo(testDB)
	jobRepo := mgorepo.NewJobRepo(testDB)
	agentCfgRepo := mgorepo.NewAgentConfigRepo(testDB)

	deleter := &service.StoryCascadeDeleter{
		SceneRepo: sceneRepo, EdgeRepo: edgeRepo, CharRepo: charRepo,
		StateRepo: stateRepo, GenRepo: genRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		BibleRepo: bibleRepo, ChapterRepo: chapterRepo,
	}

	mockLLM := &stubLLMClient{}

	store := prompt.NewMemoryStore()
	for _, tmpl := range prompt.DefaultTemplates() {
		store.Save(tmpl)
	}
	compiler := prompt.NewCompilerService(store)

	outlineSvc := llm.NewOutlineService(mockLLM, compiler)
	titleSvc := llm.NewTitleService(mockLLM)
	bibleGenSvc := llm.NewBibleService(mockLLM, compiler)
	bibleSvc := service.NewBibleService(bibleRepo, storyRepo, charRepo, bibleGenSvc)
	chapterSvc := service.NewChapterSvc(chapterRepo)
	progressHub := api.NewProgressHub()
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo,
		JobRepo: jobRepo, EventBus: events.NewInMemoryBus(),
	})
	genSvc.SetProgressPublisher(progressHub)
	agentCfgSvc := service.NewAgentConfigService(agentCfgRepo)

	criticSvc := service.NewCriticScoresService(genRepo, sceneRepo)

	h := api.NewHandlers(
		service.NewStoryService(storyRepo, deleter),
		service.NewSceneService(sceneRepo, edgeRepo, genRepo),
		service.NewEdgeService(edgeRepo),
		service.NewCharacterService(charRepo, stateRepo, memRepo),
		genSvc, genSvc,
		service.NewTimelineService(tlRepo),
		service.NewSummaryService(sumRepo),
		service.NewMemoryService(memRepo, &mockEmbeddingService{}),
		service.NewLocationService(locRepo),
		bibleSvc,
		chapterSvc,
		outlineSvc,
		titleSvc,
		nil,
		criticSvc,
		agentCfgSvc,
		progressHub,
		nil,
		nil,
		nil,
	)

	return api.NewServer(h, nil), storyRepo
}

type stubLLMClient struct{}

func (s *stubLLMClient) Complete(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{
		Content: `{"title":"Generated Story","synopsis":"A test story","characters":[{"name":"Hero","persona":"protagonist","backstory":"Born in fire.","moral_alignment":"good"}],"beats":[{"title":"Chapter 1","beat_intent":"Hero begins journey","character_names":["Hero"],"pov":"Hero","tone":"hopeful","target_words":500,"act":1}],"edges":[{"from":"Chapter 1","to":"Chapter 2","type":"seq"}]}`,
		Model:   "mock-sonnet",
	}, nil
}

func TestAPI_Healthz(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")

	srv, _ := buildServer(t)
	req := httptest.NewRequest("GET", "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected ok, got %v", body)
	}
}

func TestAPI_StoriesCRUD(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")

	srv, _ := buildServer(t)
	base := "/api/v1/stories"

	t.Run("create story", func(t *testing.T) {
		payload := `{"title":"API Test Story"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var story domain.Story
		json.NewDecoder(rec.Body).Decode(&story)
		if story.Title != "API Test Story" {
			t.Fatalf("title: got %q", story.Title)
		}
		if story.ID == "" {
			t.Fatal("id is empty")
		}
		if story.Status != domain.StoryStatusDraft {
			t.Fatalf("status: got %q", story.Status)
		}
	})

	t.Run("create story with empty title returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("create story with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("list empty stories returns empty array", func(t *testing.T) {
		cleanCollections(t, "stories")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var stories []*domain.Story
		json.NewDecoder(rec.Body).Decode(&stories)
		if len(stories) != 0 {
			t.Fatalf("expected empty array, got %d", len(stories))
		}
	})

	t.Run("get existing story", func(t *testing.T) {
		var created domain.Story
		reqCreate := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Get Me"}`))
		reqCreate.Header.Set("Content-Type", "application/json")
		recCreate := httptest.NewRecorder()
		srv.Router.ServeHTTP(recCreate, reqCreate)
		json.NewDecoder(recCreate.Body).Decode(&created)

		getReq := httptest.NewRequest("GET", base+"/"+created.ID, nil)
		recGet := httptest.NewRecorder()
		srv.Router.ServeHTTP(recGet, getReq)

		if recGet.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recGet.Code, recGet.Body.String())
		}
		var got domain.Story
		json.NewDecoder(recGet.Body).Decode(&got)
		if got.Title != "Get Me" {
			t.Fatalf("title: got %q", got.Title)
		}
	})

	t.Run("get missing story returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/nonexistent", nil))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})

	t.Run("update story", func(t *testing.T) {
		var created domain.Story
		reqCreate := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Original"}`))
		reqCreate.Header.Set("Content-Type", "application/json")
		recCreate := httptest.NewRecorder()
		srv.Router.ServeHTTP(recCreate, reqCreate)
		json.NewDecoder(recCreate.Body).Decode(&created)

		updatePayload := `{"title":"Updated","status":"active"}`
		reqUpdate := httptest.NewRequest("PUT", base+"/"+created.ID, bytes.NewBufferString(updatePayload))
		reqUpdate.Header.Set("Content-Type", "application/json")
		recUpdate := httptest.NewRecorder()
		srv.Router.ServeHTTP(recUpdate, reqUpdate)

		if recUpdate.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", recUpdate.Code, recUpdate.Body.String())
		}
		var updated domain.Story
		json.NewDecoder(recUpdate.Body).Decode(&updated)
		if updated.Title != "Updated" {
			t.Fatalf("title: got %q", updated.Title)
		}
		if updated.Status != "active" {
			t.Fatalf("status: got %q", updated.Status)
		}
	})

	t.Run("delete story returns 204", func(t *testing.T) {
		var created domain.Story
		reqCreate := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Delete Me"}`))
		reqCreate.Header.Set("Content-Type", "application/json")
		recCreate := httptest.NewRecorder()
		srv.Router.ServeHTTP(recCreate, reqCreate)
		json.NewDecoder(recCreate.Body).Decode(&created)

		recDelete := httptest.NewRecorder()
		srv.Router.ServeHTTP(recDelete, httptest.NewRequest("DELETE", base+"/"+created.ID, nil))

		if recDelete.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", recDelete.Code)
		}

		recGet := httptest.NewRecorder()
		srv.Router.ServeHTTP(recGet, httptest.NewRequest("GET", base+"/"+created.ID, nil))
		if recGet.Code != http.StatusNotFound {
			t.Fatalf("expected 404 after delete, got %d", recGet.Code)
		}
	})
}

func TestAPI_Scenes(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Scene Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/nodes"

	t.Run("create node", func(t *testing.T) {
		payload := fmt.Sprintf(`{"beat_intent":"Hero fights villain","pov":"Hero","tone":"intense","target_words":750}`)
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var node map[string]any
		json.NewDecoder(rec.Body).Decode(&node)
		if node["beat_intent"] != "Hero fights villain" {
			t.Fatalf("beat_intent: got %q", node["beat_intent"])
		}
		if node["story_id"] != s.ID {
			t.Fatalf("storyID: got %q", node["story_id"])
		}
	})

	t.Run("create node with invalid JSON returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`invalid`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAPI_Characters(t *testing.T) {
	cleanCollections(t, "stories", "characters")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Char API Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/characters"

	t.Run("create character", func(t *testing.T) {
		payload := `{"name":"Gandalf","persona":"wizard","traits":["wise","powerful"]}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var char domain.Character
		json.NewDecoder(rec.Body).Decode(&char)
		if char.Name != "Gandalf" {
			t.Fatalf("name: got %q", char.Name)
		}
		if char.StoryID != s.ID {
			t.Fatalf("storyID: got %q", char.StoryID)
		}
	})

	t.Run("list characters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var chars []*domain.Character
		json.NewDecoder(rec.Body).Decode(&chars)
		if len(chars) == 0 {
			t.Fatal("expected at least 1 character")
		}
	})
}

func TestAPI_GenerateStory(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")

	srv, _ := buildServer(t)

	t.Run("generate story from synopsis", func(t *testing.T) {
		payload := `{"synopsis":"A hero rises to defeat darkness."}`
		req := httptest.NewRequest("POST", "/api/v1/stories/generate", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["story_id"] == "" {
			t.Fatal("expected story_id in response")
		}
		if resp["status"] != "outlined" {
			t.Fatalf("status: got %q", resp["status"])
		}

		getRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(getRec, httptest.NewRequest("GET", "/api/v1/stories/"+resp["story_id"], nil))
		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200 for get, got %d", getRec.Code)
		}
		var story domain.Story
		json.NewDecoder(getRec.Body).Decode(&story)
		if story.Title == "" {
			t.Fatal("story title should not be empty")
		}
	})

	t.Run("generate story with empty synopsis returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/generate", bytes.NewBufferString(`{"synopsis":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestAPI_Edges(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Edge Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc1 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc2)

	base := "/api/v1/stories/" + s.ID + "/edges"

	t.Run("create edge via API", func(t *testing.T) {
		payload := `{"from_node":"` + sc1.ID + `","to_node":"` + sc2.ID + `","edge_type":"seq"}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list edges via API", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var edges []*domain.SceneEdge
		json.NewDecoder(rec.Body).Decode(&edges)
		if len(edges) == 0 {
			t.Fatal("expected at least 1 edge")
		}
	})
}

func TestAPI_Timeline(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "TL Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/timeline"

	t.Run("create timeline event", func(t *testing.T) {
		payload := `{"title":"Opening","description":"Story begins","order":1}`
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list timeline events", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var events []*domain.TimelineEvent
		json.NewDecoder(rec.Body).Decode(&events)
		if len(events) == 0 {
			t.Fatal("expected at least 1 event")
		}
	})
}

func TestAPI_Summaries(t *testing.T) {
	cleanCollections(t, "stories", "summaries")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	s := &domain.Story{Title: "Sum Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sumRepo.Upsert(ctx, &domain.Summary{
		StoryID: s.ID, Level: domain.SummaryLevelStory,
		Content: "Full story summary.", WordCount: 3,
	})

	t.Run("get summary by level", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(
			rec,
			httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/summaries/level?level=story", nil),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var sum domain.Summary
		json.NewDecoder(rec.Body).Decode(&sum)
		if sum.Content != "Full story summary." {
			t.Fatalf("content: got %q", sum.Content)
		}
	})

	t.Run("get missing summary returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(
			rec,
			httptest.NewRequest("GET", "/api/v1/stories/nonexistent/summaries/level?level=story", nil),
		)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
	})
}

func TestAPI_Memories(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_memories")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	memRepo := mgorepo.NewMemoryRepo(testDB)
	s := &domain.Story{Title: "Mem Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	c := &domain.Character{StoryID: s.ID, Name: "Hero"}
	charRepo.Create(ctx, c)
	memRepo.Create(ctx, &domain.CharacterMemory{
		StoryID: s.ID, CharacterID: c.CharID,
		Content: "Remember this.", Importance: 0.8,
	})

	t.Run("list character memories", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(
			rec,
			httptest.NewRequest("GET", "/api/v1/characters/"+c.CharID+"/memories", nil),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var mems []*domain.CharacterMemory
		json.NewDecoder(rec.Body).Decode(&mems)
		if len(mems) == 0 {
			t.Fatal("expected at least 1 memory")
		}
	})

	t.Run("list memories for unknown character returns empty array", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(
			rec,
			httptest.NewRequest("GET", "/api/v1/characters/unknown/memories", nil),
		)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var mems []*domain.CharacterMemory
		json.NewDecoder(rec.Body).Decode(&mems)
		if mems == nil {
			t.Fatal("expected empty array, not nil")
		}
	})
}

func TestAPI_Topology(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")

	srv, storyRepo := buildServer(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Topo Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)

	sc1 := &domain.Scene{StoryID: s.ID, Title: "A"}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID, Title: "B"}
	sceneRepo.Create(ctx, sc2)
	edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc2.ID, Type: "seq"})

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(
		rec,
		httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/topology", nil),
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)

	nodes, _ := resp["nodes"].([]any)
	edges, _ := resp["edges"].([]any)

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
}
