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
	"time"

	"github.com/premchand/story-builder/internal/api"
	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/prompt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/bson"
)

func TestStoryBuilderBDD(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Story Builder BDD Integration Suite")
}

func bddClean(colls ...string) {
	ctx := context.Background()
	for _, c := range colls {
		_, err := testDB.Collection(c).DeleteMany(ctx, bson.M{})
		Expect(err).NotTo(HaveOccurred())
	}
}

func bddServer() (*api.Server, *mgorepo.StoryRepo) {
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
		service.NewCharacterService(charRepo),
		genSvc, genSvc,
		service.NewTimelineService(tlRepo),
		service.NewSummaryService(sumRepo),
		service.NewMemoryService(memRepo, nil),
		service.NewLocationService(locRepo),
		bibleSvc,
		chapterSvc,
		outlineSvc,
		titleSvc,
		nil, criticSvc, agentCfgSvc,
		progressHub, nil, nil,
	)

	return api.NewServer(h, nil), storyRepo
}

func reqJSON(method, url, body string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func decode[T any](rec *httptest.ResponseRecorder) T {
	var v T
	Expect(json.NewDecoder(rec.Body).Decode(&v)).To(Succeed())
	return v
}

var _ = Describe("Graph topology", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		story     *domain.Story
	)

	BeforeEach(func() {
		bddClean("stories", "scenes", "scene_edges")
		srv, storyRepo = bddServer()
		ctx := context.Background()
		story = &domain.Story{Title: "Graph Test", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, story)).To(Succeed())
	})

	It("returns valid topological sort for linear scenes", func() {
		ctx := context.Background()
		sceneRepo := mgorepo.NewSceneRepo(testDB)
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)

		scenes := make([]*domain.Scene, 3)
		for i := range scenes {
			scenes[i] = &domain.Scene{StoryID: story.ID, Title: fmt.Sprintf("Scene %d", i+1), Status: domain.SceneStatusDraft}
			Expect(sceneRepo.Create(ctx, scenes[i])).To(Succeed())
		}
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: scenes[0].ID, ToSceneID: scenes[1].ID, Type: domain.EdgeTypeSeq})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: scenes[1].ID, ToSceneID: scenes[2].ID, Type: domain.EdgeTypeSeq})).To(Succeed())

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/stories/%s/topology", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		resp := decode[map[string]any](rec)
		Expect(resp["nodes"]).To(HaveLen(3))
		Expect(resp["edges"]).To(HaveLen(2))
	})

	It("detects a cycle in the scene graph", func() {
		ctx := context.Background()
		sceneRepo := mgorepo.NewSceneRepo(testDB)
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)

		scenes := make([]*domain.Scene, 3)
		for i := range scenes {
			scenes[i] = &domain.Scene{StoryID: story.ID, Title: fmt.Sprintf("Scene %d", i+1)}
			Expect(sceneRepo.Create(ctx, scenes[i])).To(Succeed())
		}
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: scenes[0].ID, ToSceneID: scenes[1].ID, Type: domain.EdgeTypeSeq})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: scenes[1].ID, ToSceneID: scenes[2].ID, Type: domain.EdgeTypeSeq})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: scenes[2].ID, ToSceneID: scenes[0].ID, Type: domain.EdgeTypeSeq})).To(Succeed())

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/stories/%s/topology", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		resp := decode[map[string]any](rec)
		Expect(resp["nodes"]).To(HaveLen(3))
		Expect(resp["edges"]).To(HaveLen(3))
	})

	It("validates a fork and join DAG", func() {
		ctx := context.Background()
		sceneRepo := mgorepo.NewSceneRepo(testDB)
		edgeRepo := mgorepo.NewSceneEdgeRepo(testDB)

		root := &domain.Scene{StoryID: story.ID, Title: "Root"}
		Expect(sceneRepo.Create(ctx, root)).To(Succeed())
		b1 := &domain.Scene{StoryID: story.ID, Title: "Branch A"}
		Expect(sceneRepo.Create(ctx, b1)).To(Succeed())
		b2 := &domain.Scene{StoryID: story.ID, Title: "Branch B"}
		Expect(sceneRepo.Create(ctx, b2)).To(Succeed())
		join := &domain.Scene{StoryID: story.ID, Title: "Join"}
		Expect(sceneRepo.Create(ctx, join)).To(Succeed())

		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: root.ID, ToSceneID: b1.ID, Type: domain.EdgeTypeFork})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: root.ID, ToSceneID: b2.ID, Type: domain.EdgeTypeFork})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: b1.ID, ToSceneID: join.ID, Type: domain.EdgeTypeJoin})).To(Succeed())
		Expect(edgeRepo.Create(ctx, &domain.SceneEdge{StoryID: story.ID, FromSceneID: b2.ID, ToSceneID: join.ID, Type: domain.EdgeTypeJoin})).To(Succeed())

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/stories/%s/topology", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		resp := decode[map[string]any](rec)
		Expect(resp["nodes"]).To(HaveLen(4))
		Expect(resp["edges"]).To(HaveLen(4))
	})
})

var _ = Describe("Memory management", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		story     *domain.Story
		char      *domain.Character
	)

	BeforeEach(func() {
		bddClean("stories", "characters", "character_memories")
		srv, storyRepo = bddServer()
		ctx := context.Background()

		story = &domain.Story{Title: "Memory Test", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, story)).To(Succeed())

		charRepo := mgorepo.NewCharacterRepo(testDB)
		char = &domain.Character{StoryID: story.ID, Name: "Hero", Persona: "protagonist"}
		Expect(charRepo.Create(ctx, char)).To(Succeed())

		memRepo := mgorepo.NewMemoryRepo(testDB)
		Expect(memRepo.Create(ctx, &domain.CharacterMemory{
			StoryID: story.ID, CharacterID: char.CharID,
			Content:    "The hero found the ancient sword in the cave.",
			Importance: 0.9,
			Type:       domain.MemoryTypeEvent,
		})).To(Succeed())
		Expect(memRepo.Create(ctx, &domain.CharacterMemory{
			StoryID: story.ID, CharacterID: char.CharID,
			Content:    "A dragon appeared on the mountain peak.",
			Importance: 0.7,
			Type:       domain.MemoryTypeObservation,
		})).To(Succeed())
	})

	It("lists memories for a character", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/characters/%s/memories", char.CharID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		mems := decode[[]*domain.CharacterMemory](rec)
		Expect(mems).To(HaveLen(2))
	})

	It("returns empty array for unknown character", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/characters/unknown/memories", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		mems := decode[[]*domain.CharacterMemory](rec)
		Expect(mems).To(BeEmpty())
	})
})

var _ = Describe("Generation pipeline", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		story     *domain.Story
		scene     *domain.Scene
	)

	BeforeEach(func() {
		bddClean("stories", "scenes", "generations")
		srv, storyRepo = bddServer()
		ctx := context.Background()

		story = &domain.Story{Title: "Pipeline Test", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, story)).To(Succeed())

		sceneRepo := mgorepo.NewSceneRepo(testDB)
		scene = &domain.Scene{StoryID: story.ID, Title: "Pipeline Scene", Status: domain.SceneStatusDraft}
		Expect(sceneRepo.Create(ctx, scene)).To(Succeed())
	})

	It("tracks pipeline step status on a generation", func() {
		genRepo := mgorepo.NewGenerationRepo(testDB)
		gen := &domain.Generation{
			StoryID: story.ID, SceneID: scene.ID,
			Output: "Generated prose for the scene.",
			Model:  "mock-sonnet", Status: domain.GenStatusPartialSuccess,
			StepStatus: map[string]string{
				domain.StepGenerate: domain.StepDone,
				domain.StepExtract:  domain.StepDone,
				domain.StepMemory:   domain.StepDone,
				domain.StepTimeline: domain.StepRunning,
				domain.StepValidate: domain.StepPending,
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		Expect(genRepo.Create(context.Background(), gen)).To(Succeed())

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/nodes/%s/generations", story.ID, scene.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		gens := decode[[]*domain.Generation](rec)
		Expect(gens).To(HaveLen(1))
		Expect(gens[0].StepStatus[domain.StepGenerate]).To(Equal(domain.StepDone))
		Expect(gens[0].StepStatus[domain.StepTimeline]).To(Equal(domain.StepRunning))
		Expect(gens[0].StepStatus[domain.StepValidate]).To(Equal(domain.StepPending))
	})

	It("persists critic score and summary", func() {
		genRepo := mgorepo.NewGenerationRepo(testDB)
		Expect(genRepo.Create(context.Background(), &domain.Generation{
			StoryID: story.ID, SceneID: scene.ID,
			Output: "Excellent prose.", Model: "claude-sonnet",
			Status: domain.GenStatusSuccess,
			CriticScore: 0.92, CriticSummary: "Well-structured narrative with strong character voice.",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})).To(Succeed())

		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/stories/%s/critic-scores", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		scores := decode[[]domain.CriticScoreEntry](rec)
		Expect(scores).To(HaveLen(1))
		Expect(scores[0].Score).To(BeNumerically("==", 0.92))
		Expect(scores[0].Summary).To(ContainSubstring("strong character voice"))
	})

	It("returns empty critic scores when none exist", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", fmt.Sprintf("/api/v1/stories/%s/critic-scores", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		scores := decode[[]domain.CriticScoreEntry](rec)
		Expect(scores).To(BeEmpty())
	})
})

var _ = Describe("Character migration", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		source    *domain.Story
		target    *domain.Story
		character *domain.Character
	)

	BeforeEach(func() {
		bddClean("stories", "characters")
		srv, storyRepo = bddServer()
		ctx := context.Background()

		source = &domain.Story{Title: "Source", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, source)).To(Succeed())
		target = &domain.Story{Title: "Target", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, target)).To(Succeed())

		charRepo := mgorepo.NewCharacterRepo(testDB)
		character = &domain.Character{
			StoryID: source.ID, Name: "Wanderer", Persona: "traveler",
			Backstory: "Lost in time.", Goals: []string{"Find home"},
			Traits: []string{"curious", "resilient"},
		}
		Expect(charRepo.Create(ctx, character)).To(Succeed())
	})

	It("creates a migrated copy with trace fields", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/stories/%s/characters/%s/migrate", target.ID, character.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))

		migrated := decode[domain.Character](rec)

		By("preserving character identity fields")
		Expect(migrated.Name).To(Equal("Wanderer"))
		Expect(migrated.Persona).To(Equal("traveler"))
		Expect(migrated.Backstory).To(Equal("Lost in time."))
		Expect(migrated.Goals).To(ConsistOf("Find home"))
		Expect(migrated.Traits).To(ConsistOf("curious", "resilient"))

		By("assigning a new unique CharID in the target story")
		Expect(migrated.StoryID).To(Equal(target.ID))
		Expect(migrated.ID).ToNot(Equal(character.ID))
		Expect(migrated.CharID).ToNot(Equal(character.CharID))
		Expect(migrated.CharID).ToNot(BeEmpty())

		By("recording the migration origin")
		Expect(migrated.MigratedFrom).To(Equal(source.ID))
		Expect(migrated.MigratedAt).ToNot(BeNil())

		By("leaving the source character intact")
		charRepo := mgorepo.NewCharacterRepo(testDB)
		original, err := charRepo.Get(context.Background(), character.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(original).ToNot(BeNil())
		Expect(original.StoryID).To(Equal(source.ID))
	})

	It("returns an error for nonexistent characters", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("POST",
			fmt.Sprintf("/api/v1/stories/%s/characters/nonexistent/migrate", target.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusInternalServerError))
	})
})

var _ = Describe("Bible sharing", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		s1, s2    *domain.Story
		bible     *domain.StoryBible
	)

	BeforeEach(func() {
		bddClean("stories", "bibles")
		srv, storyRepo = bddServer()
		ctx := context.Background()

		s1 = &domain.Story{Title: "Primary", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, s1)).To(Succeed())
		s2 = &domain.Story{Title: "Secondary", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, s2)).To(Succeed())

		bibleRepo := mgorepo.NewBibleRepo(testDB)
		bible = &domain.StoryBible{
			ID: s1.ID + "_bible", StoryID: s1.ID,
			Title: "Shared World", World: "Middle-earth",
		}
		Expect(bibleRepo.Create(ctx, bible)).To(Succeed())
	})

	It("links and references a bible from another story", func() {
		linkBody := fmt.Sprintf(`{"bibleId":"%s"}`, bible.ID)

		By("linking the bible to the secondary story")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST",
			fmt.Sprintf("/api/v1/stories/%s/bibles/link", s2.ID), linkBody))
		Expect(rec.Code).To(Equal(http.StatusOK))

		By("listing referencing bibles for the linked story")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/bibles/referencing", s2.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		bibles := decode[[]*domain.StoryBible](rec)
		Expect(bibles).To(HaveLen(1))
		Expect(bibles[0].Title).To(Equal("Shared World"))

		By("unlinking the bible")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST",
			fmt.Sprintf("/api/v1/stories/%s/bibles/unlink", s2.ID), linkBody))
		Expect(rec.Code).To(Equal(http.StatusOK))

		By("confirming referencing list is now empty")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/bibles/referencing", s2.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		bibles = decode[[]*domain.StoryBible](rec)
		Expect(bibles).To(BeEmpty())
	})
})

var _ = Describe("Cross-story timeline", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		s1, s2    *domain.Story
	)

	BeforeEach(func() {
		bddClean("stories", "timeline_events")
		srv, storyRepo = bddServer()
		ctx := context.Background()
		s1 = &domain.Story{Title: "Alpha", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, s1)).To(Succeed())
		s2 = &domain.Story{Title: "Beta", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, s2)).To(Succeed())
	})

	It("creates events visible to related stories", func() {
		By("creating a cross-story event from s1 referencing s2")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST",
			fmt.Sprintf("/api/v1/stories/%s/timeline/cross-story", s1.ID),
			fmt.Sprintf(`{"title":"Shared Event","description":"Visible in both","relatedStoryIds":["%s"],"order":1}`, s2.ID)))
		Expect(rec.Code).To(Equal(http.StatusCreated))

		evt := decode[domain.TimelineEvent](rec)
		Expect(evt.Title).To(Equal("Shared Event"))

		By("s2 sees the event via cross-story endpoint")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/timeline/cross-story", s2.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		events := decode[[]*domain.TimelineEvent](rec)
		Expect(events).To(HaveLen(1))

		By("an unrelated story sees nothing")
		s3 := &domain.Story{Title: "Gamma", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(context.Background(), s3)).To(Succeed())
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/timeline/cross-story", s3.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		events = decode[[]*domain.TimelineEvent](rec)
		Expect(events).To(BeEmpty())
	})
})

var _ = Describe("Agent configuration marketplace", func() {
	var srv *api.Server

	BeforeEach(func() {
		bddClean("agent_configs")
		srv, _ = bddServer()
	})

	It("creates, lists, exports and imports agent configurations", func() {
		By("creating a shared agent config")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/agent-configs",
			`{"name":"story-critic","role":"critic","systemPrompt":"Review output for quality.","shared":true,"tags":["review"]}`))
		Expect(rec.Code).To(Equal(http.StatusCreated))
		created := decode[domain.AgentConfig](rec)
		Expect(created.Name).To(Equal("story-critic"))

		By("listing all agent configs")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/agent-configs", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		configs := decode[[]*domain.AgentConfig](rec)
		Expect(configs).To(HaveLen(1))

		By("exporting the agent config")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/agent-configs/story-critic/export", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		exported := decode[domain.AgentConfig](rec)
		Expect(exported.SystemPrompt).To(Equal("Review output for quality."))

		By("importing a new agent config")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/agent-configs/imported-agent/import",
			`{"name":"imported-agent","role":"world","systemPrompt":"World builder.","shared":false}`))
		Expect(rec.Code).To(Equal(http.StatusCreated))

		By("marketplace only returns shared configs")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/agent-configs/marketplace", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		marketplace := decode[[]*domain.AgentConfig](rec)
		Expect(marketplace).To(HaveLen(1))
		for _, c := range marketplace {
			Expect(c.Shared).To(BeTrue())
		}
	})

	It("validates empty name on creation", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/agent-configs", `{"name":"","role":"critic","systemPrompt":"test"}`))
		Expect(rec.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns 404 for nonexistent config", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/agent-configs/nonexistent", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})

	It("deletes an agent config", func() {
		srv.Router.ServeHTTP(httptest.NewRecorder(), reqJSON("POST", "/api/v1/agent-configs",
			`{"name":"temporary","role":"critic","systemPrompt":"temp"}`))
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/agent-configs/temporary", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/agent-configs/temporary", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})

var _ = Describe("Location CRUD", func() {
	var (
		srv       *api.Server
		storyRepo *mgorepo.StoryRepo
		story     *domain.Story
	)

	BeforeEach(func() {
		bddClean("stories", "locations")
		srv, storyRepo = bddServer()
		ctx := context.Background()
		story = &domain.Story{Title: "Loc Test", Status: domain.StoryStatusDraft}
		Expect(storyRepo.Create(ctx, story)).To(Succeed())
	})

	It("creates and retrieves a location via the API", func() {
		By("creating a location")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST",
			fmt.Sprintf("/api/v1/stories/%s/locations", story.ID),
			`{"name":"Mirkwood","type":"forest","description":"Dark and dangerous"}`))
		Expect(rec.Code).To(Equal(http.StatusCreated))
		created := decode[domain.Location](rec)
		Expect(created.Name).To(Equal("Mirkwood"))

		By("listing locations for the story")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/stories/%s/locations", story.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		locs := decode[[]*domain.Location](rec)
		Expect(locs).To(HaveLen(1))

		By("getting the location by ID via the global endpoint")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET",
			fmt.Sprintf("/api/v1/locations/%s", created.ID), nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		got := decode[domain.Location](rec)
		Expect(got.ID).To(Equal(created.ID))
	})

	It("rejects duplicate location names within a story", func() {
		locRepo := mgorepo.NewLocationRepo(testDB)
		Expect(locRepo.Create(context.Background(), &domain.Location{StoryID: story.ID, Name: "Unique"})).To(Succeed())
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST",
			fmt.Sprintf("/api/v1/stories/%s/locations", story.ID), `{"name":"Unique"}`))
		Expect(rec.Code).To(Equal(http.StatusConflict))
	})

	It("returns 404 for missing location", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/locations/nonexistent", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})

var _ = Describe("Story lifecycle", func() {
	var srv *api.Server

	BeforeEach(func() {
		bddClean("stories", "scenes", "scene_edges", "characters",
			"character_state", "character_memories", "generations", "timeline_events", "summaries")
		srv, _ = bddServer()
	})

	It("creates, reads, updates, and deletes a story", func() {
		By("creating a story")
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/stories", `{"title":"BDD Story"}`))
		Expect(rec.Code).To(Equal(http.StatusCreated))
		story := decode[domain.Story](rec)
		Expect(story.Title).To(Equal("BDD Story"))
		Expect(story.ID).ToNot(BeEmpty())
		Expect(story.Status).To(Equal(domain.StoryStatusDraft))

		By("reading the story by ID")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+story.ID, nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		got := decode[domain.Story](rec)
		Expect(got.Title).To(Equal("BDD Story"))

		By("updating the story")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("PUT", "/api/v1/stories/"+story.ID,
			`{"title":"Updated BDD Story","status":"active"}`))
		Expect(rec.Code).To(Equal(http.StatusOK))
		updated := decode[domain.Story](rec)
		Expect(updated.Title).To(Equal("Updated BDD Story"))
		Expect(updated.Status).To(Equal("active"))

		By("deleting the story")
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("DELETE", "/api/v1/stories/"+story.ID, nil))
		Expect(rec.Code).To(Equal(http.StatusNoContent))
		rec = httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/stories/"+story.ID, nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 400 for empty title", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/stories", `{"title":""}`))
		Expect(rec.Code).To(Equal(http.StatusBadRequest))
	})

	It("generates a title from synopsis", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, reqJSON("POST", "/api/v1/stories/generate-title",
			`{"synopsis":"A hero rises to defeat the darkness."}`))
		Expect(rec.Code).To(Equal(http.StatusOK))
		resp := decode[map[string]string](rec)
		Expect(resp["title"]).ToNot(BeEmpty())
	})

	It("reports health via the healthz endpoint", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/healthz", nil))
		Expect(rec.Code).To(Equal(http.StatusOK))
		body := decode[map[string]string](rec)
		Expect(body["status"]).To(Equal("ok"))
	})

	It("returns 404 for unknown routes", func() {
		rec := httptest.NewRecorder()
		srv.Router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/nonexistent-route", nil))
		Expect(rec.Code).To(Equal(http.StatusNotFound))
	})
})
