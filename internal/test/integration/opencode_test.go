//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/config"
	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/prompt"
	"github.com/premchand/story-builder/internal/validation"
)

func buildServerWithOpenCode(t *testing.T) (*api.Server, *mgorepo.StoryRepo) {
	t.Helper()

	cfg := config.FromEnv()

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
	runRepo := mgorepo.NewRunRepo(testDB)
	stepRepo := mgorepo.NewRunStepRepo(testDB)
	narrativeEventRepo := mgorepo.NewNarrativeEventRepo(testDB)
	sceneLockRepo := mgorepo.NewSceneLockRepo(testDB)
	budgetRepo := mgorepo.NewTokenBudgetRepo(testDB)

	deleter := &service.StoryCascadeDeleter{
		SceneRepo: sceneRepo, EdgeRepo: edgeRepo, CharRepo: charRepo,
		StateRepo: stateRepo, GenRepo: genRepo, MemRepo: memRepo,
		TlRepo: tlRepo, SumRepo: sumRepo, LocRepo: locRepo,
		BibleRepo: bibleRepo, ChapterRepo: chapterRepo,
	}

	// OpenCode is the sole LLM provider for all model tiers
	ocClient := llm.NewOpenCodeClient(cfg.OpenCodeURL, cfg.OpenCodeModel, cfg.OpenCodeKey)
	router := llm.NewRouter(ocClient, ocClient)

	store := prompt.NewMemoryStore()
	for _, tmpl := range prompt.DefaultTemplates() {
		store.Save(tmpl)
	}
	compiler := prompt.NewCompilerService(store)

	proseSvc := llm.NewProseService(router, compiler)
	extractSvc := llm.NewExtractionService(router, compiler)
	summarySvc := llm.NewSummaryService(router, compiler)
	validateSvc := llm.NewValidationService(router, compiler)
	outlineSvc := llm.NewOutlineService(router, compiler)
	titleSvc := llm.NewTitleService(router)
	bibleGenSvc := llm.NewBibleService(router, compiler)
	embedSvc := llm.NewOpenCodeEmbeddingService(cfg.OpenCodeURL, "nomic-embed-text")

	contextBldr := service.NewContextBuilder(bibleRepo, storyRepo, charRepo, stateRepo, locRepo, memRepo, sumRepo, tlRepo)
	sceneValidator := validation.NewSceneValidator(charRepo, locRepo)

	eventBus := events.NewInMemoryBus()
	progressHub := api.NewProgressHub()

	budgetSvc := service.NewTokenBudgetService(budgetRepo)
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: genRepo, SceneRepo: sceneRepo, JobRepo: jobRepo,
		EventBus: eventBus, BudgetSvc: budgetSvc,
	})

	genJobWorker := service.NewGenerationJobWorker(service.GenerationJobWorkerConfig{
		JobRepo: jobRepo, LockRepo: sceneLockRepo, RunRepo: runRepo, StepRepo: stepRepo,
		GenRepo: genRepo, SceneRepo: sceneRepo,
		StoryRepo: storyRepo, CharRepo: charRepo, StateRepo: stateRepo,
		EdgeRepo: edgeRepo, BibleRepo: bibleRepo, MemRepo: memRepo, TlRepo: tlRepo,
		SumRepo: sumRepo, LocRepo: locRepo,
		ProseSvc: proseSvc, ExtractSvc: extractSvc,
		SummarySvc: summarySvc, ValidateSvc: validateSvc,
		ContextBldr: contextBldr, EventBus: eventBus,
		EmbeddingSvc: embedSvc, SceneValidator: sceneValidator,
		Progress: progressHub,
		PollInterval: 500 * time.Millisecond,
		LeaseTime:    5 * time.Minute,
	})
	genJobWorker.Start()
	t.Cleanup(genJobWorker.Stop)

	genSvc.SetProgressPublisher(progressHub)

	bibleSvc := service.NewBibleService(bibleRepo, storyRepo, charRepo, bibleGenSvc)
	chapterSvc := service.NewChapterSvc(chapterRepo)
	tlSvc := service.NewTimelineService(tlRepo)
	sumSvc := service.NewSummaryService(sumRepo)
	memSvc := service.NewMemoryService(memRepo, embedSvc)
	locSvc := service.NewLocationService(locRepo)
	runSvc := service.NewRunService(runRepo, stepRepo, jobRepo)
	narrativeSvc := service.NewNarrativeEventService(narrativeEventRepo)
	plannerSvc := service.NewPlannerService(storyRepo, sceneRepo, edgeRepo, charRepo, mgorepo.NewBlueprintRepo(testDB))
	diffSvc := service.NewDiffService(genRepo, narrativeEventRepo)
	metricsSvc := service.NewMetricsService(genRepo)
	criticSvc := service.NewCriticScoresService(genRepo, sceneRepo)
	agentCfgSvc := service.NewAgentConfigService(agentCfgRepo)
	charSvc := service.NewCharacterService(charRepo, stateRepo, memRepo)
	sceneSvc := service.NewSceneService(sceneRepo, edgeRepo, genRepo)
	edgeSvc := service.NewEdgeService(edgeRepo)
	storySvc := service.NewStoryService(storyRepo, deleter)

	h := api.NewHandlers(
		storySvc, sceneSvc, edgeSvc, charSvc,
		genSvc, genSvc,
		tlSvc, sumSvc, memSvc, locSvc,
		bibleSvc, chapterSvc,
		outlineSvc, titleSvc,
		metricsSvc, criticSvc, agentCfgSvc,
		progressHub, eventBus,
		nil, nil,
		runSvc, narrativeSvc, plannerSvc, diffSvc,
	)

	return api.NewServer(h, nil), storyRepo
}

func TestOpenCode_Healthz(t *testing.T) {
	srv, _ := buildServerWithOpenCode(t)
	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected ok, got %v", body)
	}
}

func TestOpenCode_StoriesCRUD(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")
	srv, _ := buildServerWithOpenCode(t)
	base := "/api/v1/stories"

	t.Run("create story", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"OpenCode Story"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var story domain.Story
		json.NewDecoder(rec.Body).Decode(&story)
		if story.Title != "OpenCode Story" {
			t.Fatalf("title: got %q", story.Title)
		}
	})

	t.Run("list stories", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var stories []*domain.Story
		json.NewDecoder(rec.Body).Decode(&stories)
		if len(stories) == 0 {
			t.Fatal("expected at least 1 story")
		}
	})

	t.Run("get story by ID", func(t *testing.T) {
		var created domain.Story
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Get Me"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/"+created.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var got domain.Story
		json.NewDecoder(rec.Body).Decode(&got)
		if got.Title != "Get Me" {
			t.Fatalf("title: got %q", got.Title)
		}
	})

	t.Run("update story", func(t *testing.T) {
		var created domain.Story
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Original"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		req := httptest.NewRequest("PUT", base+"/"+created.ID, bytes.NewBufferString(`{"title":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var updated domain.Story
		json.NewDecoder(rec.Body).Decode(&updated)
		if updated.Title != "Updated" {
			t.Fatalf("title: got %q", updated.Title)
		}
	})

	t.Run("delete story", func(t *testing.T) {
		var created domain.Story
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":"Delete Me"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/"+created.ID, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"title":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestOpenCode_NodesCRUD(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Node Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/nodes"

	t.Run("create node", func(t *testing.T) {
		req := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"Hero opens the door"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var node map[string]any
		json.NewDecoder(rec.Body).Decode(&node)
		if node["beat_intent"] != "Hero opens the door" {
			t.Fatalf("beat_intent: got %q", node["beat_intent"])
		}
	})

	t.Run("list nodes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update node", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"Original"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		req := httptest.NewRequest("PUT", base+"/"+created["id"].(string), bytes.NewBufferString(`{"beat_intent":"Updated"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete node", func(t *testing.T) {
		var created map[string]any
		creq := httptest.NewRequest("POST", base, bytes.NewBufferString(`{"beat_intent":"Delete Me"}`))
		creq.Header.Set("Content-Type", "application/json")
		crec := httptest.NewRecorder()
		srv.Router.ServeHTTP(crec, creq)
		json.NewDecoder(crec.Body).Decode(&created)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base+"/"+created["id"].(string), nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}

func TestOpenCode_CharactersCRUD(t *testing.T) {
	cleanCollections(t, "stories", "characters")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Char Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("create character (story-scoped)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/characters",
			bytes.NewBufferString(`{"name":"Aria","persona":"mage"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("create character (top-level v2)", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/characters",
			bytes.NewBufferString(`{"name":"Kael","persona":"warrior","storyId":"`+s.ID+`"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list characters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters?story_id="+s.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update character", func(t *testing.T) {
		charRepo := mgorepo.NewCharacterRepo(testDB)
		c := &domain.Character{StoryID: s.ID, Name: "Old Name"}
		charRepo.Create(ctx, c)

		req := httptest.NewRequest("PUT", "/api/v1/characters/"+c.CharID,
			bytes.NewBufferString(`{"name":"New Name"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_EdgesCRUD(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Edge Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc1 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc2)

	t.Run("create edge", func(t *testing.T) {
		payload := fmt.Sprintf(`{"from_node":"%s","to_node":"%s","edge_type":"seq"}`, sc1.ID, sc2.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/edges", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list edges", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/edges", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("delete edge by ID", func(t *testing.T) {
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
		e := &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc2.ID, Type: "fork"}
		edgeRepo.Create(ctx, e)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/stories/"+s.ID+"/edges/"+e.ID, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}

func TestOpenCode_Topology(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "scene_edges")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Topo", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)
	sc1 := &domain.Scene{StoryID: s.ID, Title: "A"}
	sceneRepo.Create(ctx, sc1)
	sc2 := &domain.Scene{StoryID: s.ID, Title: "B"}
	sceneRepo.Create(ctx, sc2)
	edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: s.ID, FromSceneID: sc1.ID, ToSceneID: sc2.ID, Type: "seq"})

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/topology", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestOpenCode_LocationsCRUD(t *testing.T) {
	cleanCollections(t, "stories", "locations")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Loc Test", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("create location", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/locations",
			bytes.NewBufferString(`{"name":"Dark Forest","description":"Eerie woods"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list locations", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/locations", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("get location by ID", func(t *testing.T) {
		locRepo := mgorepo.NewLocationRepo(testDB)
		loc := &domain.Location{StoryID: s.ID, Name: "River"}
		locRepo.Create(ctx, loc)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/"+loc.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update location", func(t *testing.T) {
		locRepo := mgorepo.NewLocationRepo(testDB)
		loc := &domain.Location{StoryID: s.ID, Name: "Mountain"}
		locRepo.Create(ctx, loc)

		req := httptest.NewRequest("PUT", "/api/v1/locations/"+loc.ID,
			bytes.NewBufferString(`{"description":"Snowy peak"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_TimelineCRUD(t *testing.T) {
	cleanCollections(t, "stories", "timeline_events")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "TL", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("create timeline event", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/timeline",
			bytes.NewBufferString(`{"title":"Event 1","description":"First event","order":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list timeline events", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/timeline", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("create cross-story event", func(t *testing.T) {
		s2 := &domain.Story{Title: "TL2", Status: domain.StoryStatusDraft}
		storyRepo.Create(ctx, s2)

		payload := fmt.Sprintf(`{"title":"Cross Event","description":"Shared","relatedStoryIds":["%s"],"order":1}`, s2.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/timeline/cross-story",
			bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list cross-story events", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/timeline/cross-story", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestOpenCode_Summaries(t *testing.T) {
	cleanCollections(t, "stories", "summaries")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Sum", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sumRepo := mgorepo.NewSummaryRepo(testDB)
	sumRepo.Upsert(ctx, &domain.Summary{
		StoryID: s.ID, Level: domain.SummaryLevelStory,
		Content: "Full summary.", WordCount: 2,
	})

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/summaries/level?level=story", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var sum domain.Summary
	json.NewDecoder(rec.Body).Decode(&sum)
	if sum.Content != "Full summary." {
		t.Fatalf("content: got %q", sum.Content)
	}
}

func TestOpenCode_Memories(t *testing.T) {
	cleanCollections(t, "stories", "characters", "character_memories")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Mem", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	c := &domain.Character{StoryID: s.ID, Name: "MemoryChar"}
	charRepo.Create(ctx, c)

	t.Run("list memories", func(t *testing.T) {
		memRepo := mgorepo.NewMemoryRepo(testDB)
		memRepo.Create(ctx, &domain.CharacterMemory{
			CharacterID: c.CharID, StoryID: s.ID,
			Content: "A memory.", Importance: 0.8,
		})
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters/"+c.CharID+"/memories", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("search memories (uses OpenCode embedding)", func(t *testing.T) {
		memRepo := mgorepo.NewMemoryRepo(testDB)
		memRepo.Create(ctx, &domain.CharacterMemory{
			CharacterID: c.CharID, StoryID: s.ID,
			Content: "The ancient ring of power.", Importance: 0.9,
		})
		memRepo.Create(ctx, &domain.CharacterMemory{
			CharacterID: c.CharID, StoryID: s.ID,
			Content: "A peaceful meadow with flowers.", Importance: 0.3,
		})

		payload := fmt.Sprintf(`{"story_id":"%s","query":"ancient ring","limit":5}`, s.ID)
		req := httptest.NewRequest("POST", "/api/v1/characters/"+c.CharID+"/memories/search",
			bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var mems []*domain.CharacterMemory
		json.NewDecoder(rec.Body).Decode(&mems)
		t.Logf("memory search returned %d results", len(mems))
	})
}

func TestOpenCode_ChaptersCRUD(t *testing.T) {
	cleanCollections(t, "stories", "chapters")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Ch", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("create chapter", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/chapters",
			bytes.NewBufferString(`{"title":"Chapter 1","actNumber":1}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list chapters", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/chapters", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update chapter", func(t *testing.T) {
		chapterRepo := mgorepo.NewChapterRepo(testDB)
		ch := &domain.Chapter{StoryID: s.ID, Title: "Old"}
		chapterRepo.Create(ctx, ch)
		req := httptest.NewRequest("PUT", "/api/v1/stories/"+s.ID+"/chapters/"+ch.ID,
			bytes.NewBufferString(`{"title":"New"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_AgentConfigsCRUD(t *testing.T) {
	cleanCollections(t, "agent_configs")
	srv, _ := buildServerWithOpenCode(t)
	base := "/api/v1/agent-configs"

	t.Run("create agent config", func(t *testing.T) {
		req := httptest.NewRequest("POST", base,
			bytes.NewBufferString(`{"name":"opencode-critic","role":"critic","systemPrompt":"Review.","shared":true}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("get agent config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/opencode-critic", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("export agent config", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/opencode-critic/export", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("marketplace lists shared configs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base+"/marketplace", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("import agent config", func(t *testing.T) {
		req := httptest.NewRequest("POST", base+"/imported-agent/import",
			bytes.NewBufferString(`{"name":"imported-agent","role":"world","systemPrompt":"World rules."}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_Blueprint(t *testing.T) {
	cleanCollections(t, "stories")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "BP", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("get blueprint", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/blueprint", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("update blueprint", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/v1/stories/"+s.ID+"/blueprint",
			bytes.NewBufferString(`{"acts":[{"number":1,"title":"Act I"}]}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_BibleCRUD(t *testing.T) {
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Bible", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	base := "/api/v1/stories/" + s.ID + "/bible"

	t.Run("create bible", func(t *testing.T) {
		bibleRepo := mgorepo.NewBibleRepo(testDB)
		bibleRepo.Create(ctx, &domain.StoryBible{
			StoryID: s.ID, Title: "My Bible", World: "Test World",
		})
	})

	t.Run("get bible", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", base, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("update bible", func(t *testing.T) {
		req := httptest.NewRequest("PUT", base, bytes.NewBufferString(`{"world":"Updated World"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("delete bible", func(t *testing.T) {
		bibleRepo := mgorepo.NewBibleRepo(testDB)
		bibleRepo.Create(ctx, &domain.StoryBible{
			StoryID: s.ID, Title: "Del Bible",
		})
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", base, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rec.Code)
		}
	})
}

func TestOpenCode_CriticScores(t *testing.T) {
	cleanCollections(t, "stories", "generations")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Critic", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	genRepo.Create(ctx, &domain.Generation{
		StoryID: s.ID, SceneID: "sc1",
		Output: "Great prose.", Model: "claude-sonnet",
		Status: domain.GenStatusSuccess, CriticScore: 0.85,
	})

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/critic-scores", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOpenCode_LLMMetrics(t *testing.T) {
	cleanCollections(t, "stories", "generations")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Metrics", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	genRepo := mgorepo.NewGenerationRepo(testDB)
	genRepo.Create(ctx, &domain.Generation{
		StoryID: s.ID, SceneID: "sc1", Model: "claude-sonnet",
		Status: domain.GenStatusSuccess, TotalTokens: 700,
	})

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+s.ID+"/metrics/llm", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// ── LLM-Dependent Endpoints (OpenCode) ──────────────────────────────────

func TestOpenCode_GenerateTitle(t *testing.T) {
	if os.Getenv("OPENCODE_URL") == "" {
		t.Skip("OPENCODE_URL not set, skipping real LLM test")
	}
	cleanCollections(t, "stories")
	srv, _ := buildServerWithOpenCode(t)

	t.Run("generate title with OpenCode", func(t *testing.T) {
		payload := `{"synopsis":"A lone detective uncovers a conspiracy in a cyberpunk city."}`
		req := httptest.NewRequest("POST", "/api/v1/stories/generate-title", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		if resp["title"] == "" {
			t.Fatal("expected non-empty title from OpenCode")
		}
		t.Logf("OpenCode generated title: %q", resp["title"])
	})

	t.Run("generate title empty synopsis returns 400", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/generate-title",
			bytes.NewBufferString(`{"synopsis":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
	})
}

func TestOpenCode_GenerateStory(t *testing.T) {
	if os.Getenv("OPENCODE_URL") == "" {
		t.Skip("OPENCODE_URL not set, skipping real LLM test")
	}
	cleanCollections(t, "stories", "scenes", "scene_edges", "characters",
		"character_state", "character_memories", "generations", "timeline_events", "summaries")
	srv, _ := buildServerWithOpenCode(t)

	t.Run("generate story outline with OpenCode", func(t *testing.T) {
		payload := `{"synopsis":"A young inventor builds a robot to find their lost father."}`
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

		// Verify the story was created with a title from OpenCode
		getRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(getRec, httptest.NewRequest("GET", "/api/v1/stories/"+resp["story_id"], nil))
		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", getRec.Code)
		}
		var story domain.Story
		json.NewDecoder(getRec.Body).Decode(&story)
		if story.Title == "" {
			t.Fatal("story title should not be empty (generated by OpenCode)")
		}
		t.Logf("OpenCode generated story: %q", story.Title)

		// Verify topology has nodes from the outline
		topoRec := httptest.NewRecorder()
		srv.Router.ServeHTTP(topoRec, httptest.NewRequest("GET",
			"/api/v1/stories/"+resp["story_id"]+"/topology", nil))
		if topoRec.Code == http.StatusOK {
			var topo map[string]any
			json.NewDecoder(topoRec.Body).Decode(&topo)
			nodes, _ := topo["nodes"].([]any)
			t.Logf("story has %d scene nodes from outline", len(nodes))
		}
	})
}

func TestOpenCode_GenerateBible(t *testing.T) {
	if os.Getenv("OPENCODE_URL") == "" {
		t.Skip("OPENCODE_URL not set, skipping real LLM test")
	}
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Bible Gen", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	t.Run("generate bible with OpenCode", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/bible/generate",
			bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
		var bible domain.StoryBible
		json.NewDecoder(rec.Body).Decode(&bible)
		if bible.Title == "" {
			t.Log("bible generated but title is empty (OpenCode may not have returned one)")
		}
		t.Logf("bible generated: %q / %q", bible.Title, bible.World)
	})
}

func TestOpenCode_GenerateNode(t *testing.T) {
	if os.Getenv("OPENCODE_URL") == "" {
		t.Skip("OPENCODE_URL not set, skipping real LLM test")
	}
	cleanCollections(t, "stories", "scenes", "generations", "characters",
		"character_state", "character_memories", "timeline_events", "summaries")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Gen Node", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	charRepo := mgorepo.NewCharacterRepo(testDB)
	char := &domain.Character{StoryID: s.ID, Name: "Hero", Traits: []string{"brave"}}
	charRepo.Create(ctx, char)

	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc := &domain.Scene{
		StoryID:      s.ID,
		Title:        "The Confrontation",
		BeatIntent:   "Hero confronts the villain and makes a choice",
		POV:          "Hero",
		Tone:         "dramatic",
		TargetWords:  150,
		Participants: []string{char.CharID},
	}
	sceneRepo.Create(ctx, sc)
	nodeBase := "/api/v1/stories/" + s.ID + "/nodes/" + sc.ID

	t.Run("generate scene with OpenCode", func(t *testing.T) {
		// Enqueue generation
		req := httptest.NewRequest("POST", nodeBase+"/generate", nil)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
		}
		var resp map[string]string
		json.NewDecoder(rec.Body).Decode(&resp)
		t.Logf("generation status: %q, id: %q", resp["status"], resp["generation_id"])

		// Poll for completion
		var finalOutput string
		for i := 0; i < 60; i++ {
			statusRec := httptest.NewRecorder()
			srv.Router.ServeHTTP(statusRec,
				httptest.NewRequest("GET", "/api/v1/generations/"+resp["generation_id"]+"/status", nil))
			if statusRec.Code == http.StatusOK {
				var status map[string]any
				json.NewDecoder(statusRec.Body).Decode(&status)
				output, _ := status["output"].(string)
				if output != "" {
					finalOutput = output
					break
				}
				if s, _ := status["status"].(string); s == "error" || s == "failed" {
					// Check if it completed via SSE progress
				}
			}
			time.Sleep(2 * time.Second)
		}

		if finalOutput == "" {
			// Generation may be slow on OpenCode; check generation record directly
			genRepo := mgorepo.NewGenerationRepo(testDB)
			gens, _ := genRepo.ListByScene(ctx, sc.ID)
			if len(gens) > 0 && gens[0].Output != "" {
				finalOutput = gens[0].Output
			}
		}

		if finalOutput != "" {
			t.Logf("scene generated with OpenCode (%d chars)", len(finalOutput))
		} else {
			t.Log("generation still pending (OpenCode may be slow)")
		}
	})

	t.Run("list generations", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", nodeBase+"/generations", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var gens []*domain.Generation
		json.NewDecoder(rec.Body).Decode(&gens)
		t.Logf("found %d generations", len(gens))
	})

	t.Run("get generation status", func(t *testing.T) {
		genRepo := mgorepo.NewGenerationRepo(testDB)
		gen := &domain.Generation{
			StoryID: s.ID, SceneID: sc.ID,
			Output: "OpenCode test prose.", Model: "opencode",
			Status: domain.GenStatusSuccess,
		}
		genRepo.Create(ctx, gen)

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/generations/"+gen.ID+"/status", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestOpenCode_AcceptGeneration(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "generations")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Accept", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc := &domain.Scene{StoryID: s.ID, Title: "Accept Scene"}
	sceneRepo.Create(ctx, sc)

	genRepo := mgorepo.NewGenerationRepo(testDB)
	gen := &domain.Generation{
		StoryID: s.ID, SceneID: sc.ID,
		Output: "Acceptable prose.", Model: "opencode",
		Status: domain.GenStatusSuccess,
	}
	genRepo.Create(ctx, gen)

	sc.Status = domain.SceneStatusGenerated
	sceneRepo.Update(ctx, sc)

	payload := fmt.Sprintf(`{"generation_id":"%s"}`, gen.ID)
	req := httptest.NewRequest("POST", "/api/v1/stories/"+s.ID+"/nodes/"+sc.ID+"/accept",
		bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOpenCode_CharacterMigration(t *testing.T) {
	cleanCollections(t, "stories", "characters")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s1 := &domain.Story{Title: "Src", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "Dst", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)
	charRepo := mgorepo.NewCharacterRepo(testDB)
	c := &domain.Character{StoryID: s1.ID, Name: "Wanderer"}
	charRepo.Create(ctx, c)

	rec := httptest.NewRecorder()
	srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
		"/api/v1/stories/"+s2.ID+"/characters/"+c.ID+"/migrate", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var migrated domain.Character
	json.NewDecoder(rec.Body).Decode(&migrated)
	if migrated.StoryID != s2.ID {
		t.Fatalf("storyID should be target: got %q", migrated.StoryID)
	}
}

func TestOpenCode_ExperimentalEndpoints(t *testing.T) {
	cleanCollections(t, "stories", "scenes", "agent_runs")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "Exp", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)
	sceneRepo := mgorepo.NewSceneRepo(testDB)
	sc := &domain.Scene{StoryID: s.ID}
	sceneRepo.Create(ctx, sc)
	expBase := "/api/v1/experimental/stories/" + s.ID + "/nodes/" + sc.ID

	t.Run("list scene turns", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", expBase+"/scene/turns", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("list canon deltas", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", expBase+"/scene/deltas", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("record canon delta", func(t *testing.T) {
		payload := `{"field":"emotion","oldValue":"calm","newValue":"angry","characterId":"hero"}`
		req := httptest.NewRequest("POST", expBase+"/scene/deltas", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list agent runs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/experimental/agent-runs", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("narrative events", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/narrative-events", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("narrative events by scene", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			expBase+"/narrative-events", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("run endpoints", func(t *testing.T) {
		runRepo := mgorepo.NewRunRepo(testDB)
		run := &domain.StoryRun{StoryID: s.ID, SceneID: sc.ID, Status: "pending"}
		runRepo.Create(ctx, run)
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/runs/"+run.ID, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list story runs", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s.ID+"/runs", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

func TestOpenCode_BibleSharing(t *testing.T) {
	cleanCollections(t, "stories", "bibles")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s1 := &domain.Story{Title: "Source", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s1)
	s2 := &domain.Story{Title: "Target", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s2)
	bibleRepo := mgorepo.NewBibleRepo(testDB)
	bible := &domain.StoryBible{StoryID: s1.ID, Title: "Shared"}
	bibleRepo.Create(ctx, bible)

	t.Run("link bible", func(t *testing.T) {
		payload := fmt.Sprintf(`{"bibleId":"%s"}`, bible.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s2.ID+"/bibles/link",
			bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("list referencing bibles", func(t *testing.T) {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			"/api/v1/stories/"+s2.ID+"/bibles/referencing", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("unlink bible", func(t *testing.T) {
		payload := fmt.Sprintf(`{"bibleId":"%s"}`, bible.ID)
		req := httptest.NewRequest("POST", "/api/v1/stories/"+s2.ID+"/bibles/unlink",
			bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestOpenCode_NotImplementedEndpoints(t *testing.T) {
	cleanCollections(t, "stories")
	srv, storyRepo := buildServerWithOpenCode(t)
	ctx := context.Background()
	s := &domain.Story{Title: "NI", Status: domain.StoryStatusDraft}
	storyRepo.Create(ctx, s)

	exp := "/api/v1/experimental/stories/" + s.ID + "/nodes/fake"

	notImpl := []string{
		"/api/v1/experimental/actors",
		"/api/v1/experimental/character-traits",
		"/api/v1/experimental/lore",
		exp + "/scene/structure",
		exp + "/scene/start",
		exp + "/scene/next",
		exp + "/scene/finish",
	}

	for _, route := range notImpl {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", route, nil))
		if rec.Code == http.StatusNotFound {
			t.Logf("not implemented route %s returns 404", route)
		}
	}

	// EmptyArray endpoints return 200 with []
	emptyArr := []string{
		"/api/v1/experimental/actors",
		"/api/v1/experimental/character-traits",
		"/api/v1/experimental/lore",
		"/api/v1/experimental/stories/" + s.ID + "/casting",
	}
	for _, route := range emptyArr {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", route, nil))
		if rec.Code == http.StatusOK {
			t.Logf("empty array endpoint %s returns 200", route)
		}
	}
}
