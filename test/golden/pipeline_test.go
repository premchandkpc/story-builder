//go:build golden

package golden

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/events"
	"github.com/premchand/story-builder/internal/llm"
	mgorepo "github.com/premchand/story-builder/internal/repository/mongo"
	"github.com/premchand/story-builder/internal/service"
	"github.com/premchand/story-builder/internal/validation"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GoldenFixture struct {
	Name      string
	Dir       string
	Story     *domain.Story
	Scenes    []*domain.Scene
	Chars     []*domain.Character
	Edges     []*domain.SceneEdge
	Locations []*domain.Location
	Expected  struct {
		GenerationStatus   string            `json:"generationStatus"`
		SceneTextMinTokens int               `json:"sceneTextMinTokens"`
		CharStatesCount    int               `json:"characterStatesCount"`
		TimelineExists     bool              `json:"timelineExists"`
		SummaryExists      bool              `json:"summaryExists"`
		StepStatus         map[string]string `json:"stepStatus"`
	} `json:"expected"`
	MockedOutputs struct {
		Generate string `json:"generate"`
		Extract  string `json:"extract"`
	}
}

func loadFixture(t *testing.T, dir string) *GoldenFixture {
	t.Helper()
	f := &GoldenFixture{Name: filepath.Base(dir), Dir: dir}

	readJSON := func(name string, v any) {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		s := string(data)
		if f.Story != nil {
			s = strings.ReplaceAll(s, "STORY_ID", f.Story.ID)
		}
		if err := json.Unmarshal([]byte(s), v); err != nil {
			t.Fatalf("unmarshal %s: %v", name, err)
		}
	}

	readJSON("story.json", &f.Story)
	readJSON("scenes.json", &f.Scenes)
	readJSON("characters.json", &f.Chars)
	readJSON("edges.json", &f.Edges)
	readJSON("locations.json", &f.Locations)
	readJSON("mocked_outputs/generate.json", &f.MockedOutputs.Generate)
	readJSON("mocked_outputs/extract.json", &f.MockedOutputs.Extract)
	readJSON("expected.json", &f.Expected)
	f.Story.ID = ""
	return f
}

type GoldenTestRunner struct {
	t       *testing.T
	fixture *GoldenFixture
	db      *mongo.Database
}

func newRunner(t *testing.T, fixture *GoldenFixture) *GoldenTestRunner {
	ctx := context.Background()
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("mongo connect: %v", err)
	}
	t.Cleanup(func() { client.Disconnect(ctx) })
	db := client.Database("golden_" + strings.ReplaceAll(fixture.Name, "-", "_"))
	return &GoldenTestRunner{t: t, fixture: fixture, db: db}
}

func (r *GoldenTestRunner) run() {
	ctx := context.Background()
	f := r.fixture
	db := r.db

	r.t.Cleanup(func() { r.dropCollections() })

	repos := struct {
		story    *mgorepo.StoryRepo
		scene    *mgorepo.SceneRepo
		edge     *mgorepo.SceneEdgeRepo
		char     *mgorepo.CharacterRepo
		state    *mgorepo.CharacterStateRepo
		gen      *mgorepo.GenerationRepo
		mem      *mgorepo.MemoryRepo
		tl       *mgorepo.TimelineRepo
		sum      *mgorepo.SummaryRepo
		bible    *mgorepo.BibleRepo
		loc      *mgorepo.LocationRepo
		job      *mgorepo.JobRepo
		run      *mgorepo.RunRepo
		step     *mgorepo.RunStepRepo
		event    *mgorepo.NarrativeEventRepo
		charView *mgorepo.CharacterViewRepo
	}{
		story:    mgorepo.NewStoryRepo(db),
		scene:    mgorepo.NewSceneRepo(db),
		edge:     mgorepo.NewSceneEdgeRepo(db),
		char:     mgorepo.NewCharacterRepo(db),
		state:    mgorepo.NewCharacterStateRepo(db),
		gen:      mgorepo.NewGenerationRepo(db),
		mem:      mgorepo.NewMemoryRepo(db),
		tl:       mgorepo.NewTimelineRepo(db),
		sum:      mgorepo.NewSummaryRepo(db),
		bible:    mgorepo.NewBibleRepo(db),
		loc:      mgorepo.NewLocationRepo(db),
		job:      mgorepo.NewJobRepo(db),
		run:      mgorepo.NewRunRepo(db),
		step:     mgorepo.NewRunStepRepo(db),
		event:    mgorepo.NewNarrativeEventRepo(db),
		charView: mgorepo.NewCharacterViewRepo(db),
	}

	if err := repos.story.Create(ctx, f.Story); err != nil {
		r.t.Fatalf("create story: %v", err)
	}

	for _, s := range f.Scenes {
		s.StoryID = f.Story.ID
		if err := repos.scene.Create(ctx, s); err != nil {
			r.t.Fatalf("create scene: %v", err)
		}
	}

	for _, c := range f.Chars {
		c.StoryID = f.Story.ID
		if err := repos.char.Create(ctx, c); err != nil {
			r.t.Fatalf("create char: %v", err)
		}
	}

	for _, e := range f.Edges {
		e.StoryID = f.Story.ID
		if err := repos.edge.Create(ctx, e); err != nil {
			r.t.Fatalf("create edge: %v", err)
		}
	}

	bible := &domain.StoryBible{StoryID: f.Story.ID}
	if err := repos.bible.Create(ctx, bible); err != nil {
		r.t.Fatalf("create bible: %v", err)
	}

	mockProse := &mockProseGolden{
		output: f.MockedOutputs.Generate,
	}
	mockExtract := &mockExtractGolden{
		output: f.MockedOutputs.Extract,
	}
	mockSummary := &mockSummaryGolden{}
	mockValidate := &mockValidateGolden{}
	mockEmbed := &mockEmbedGolden{}

	eventBus := events.NewInMemoryBus()
	genSvc := service.NewGenerationService(service.GenerationServiceConfig{
		GenRepo: repos.gen, SceneRepo: repos.scene,
		JobRepo: repos.job, EventBus: eventBus,
	})
	contextBldr := service.NewContextBuilder(repos.bible, repos.story, repos.char, repos.state, repos.loc, repos.mem, repos.sum, repos.tl)

	worker := service.NewGenerationJobWorker(service.GenerationJobWorkerConfig{
		JobRepo:        repos.job,
		RunRepo:        repos.run,
		StepRepo:       repos.step,
		GenRepo:        repos.gen,
		SceneRepo:      repos.scene,
		StoryRepo:      repos.story,
		CharRepo:       repos.char,
		StateRepo:      repos.state,
		EdgeRepo:       repos.edge,
		BibleRepo:      repos.bible,
		MemRepo:        repos.mem,
		TlRepo:         repos.tl,
		SumRepo:        repos.sum,
		LocRepo:        repos.loc,
		ProseSvc:       mockProse,
		ExtractSvc:     mockExtract,
		SummarySvc:     mockSummary,
		ValidateSvc:    mockValidate,
		ContextBldr:    contextBldr,
		EventBus:       eventBus,
		EmbeddingSvc:   mockEmbed,
		SceneValidator: validation.NewSceneValidator(repos.char, repos.loc),
		Progress:       nil,
		PollInterval:   100 * time.Millisecond,
		LeaseTime:      5 * time.Minute,
	})
	worker.Start()
	r.t.Cleanup(worker.Stop)

	targetScene := f.Scenes[0]
	gen, err := genSvc.Generate(ctx, targetScene.ID)
	if err != nil {
		r.t.Fatalf("generate: %v", err)
	}

	var completedGen *domain.Generation
	for i := 0; i < 100; i++ {
		updated, _ := repos.gen.Get(ctx, gen.ID)
		if updated != nil && (updated.Status == domain.GenStatusSuccess || updated.Status == domain.GenStatusFailed) {
			completedGen = updated
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if completedGen == nil {
		r.t.Fatal("generation did not complete within timeout")
	}

	if completedGen.Status != f.Expected.GenerationStatus {
		r.t.Errorf("generation status: got %q, want %q", completedGen.Status, f.Expected.GenerationStatus)
	}

	for step, expectedStatus := range f.Expected.StepStatus {
		got := completedGen.StepStatus[step]
		if got != expectedStatus {
			r.t.Errorf("step %q status: got %q, want %q", step, got, expectedStatus)
		}
	}

	scene, _ := repos.scene.Get(ctx, targetScene.ID)
	if scene != nil && len(scene.GeneratedContent) < f.Expected.SceneTextMinTokens {
		r.t.Errorf("scene text too short: got %d chars, want >= %d", len(scene.GeneratedContent), f.Expected.SceneTextMinTokens)
	}

	if f.Expected.CharStatesCount > 0 {
		states, _ := repos.state.ListByScene(ctx, targetScene.ID)
		if len(states) < f.Expected.CharStatesCount {
			r.t.Errorf("character states: got %d, want >= %d", len(states), f.Expected.CharStatesCount)
		}
	}

	if f.Expected.TimelineExists {
		events, _ := repos.tl.ListByStory(ctx, f.Story.ID)
		if len(events) == 0 {
			r.t.Error("expected timeline event to exist")
		}
	}

	if f.Expected.SummaryExists {
		sum, _ := repos.sum.GetByLevel(ctx, f.Story.ID, domain.SummaryLevelScene)
		if sum == nil || sum.Content == "" {
			sum2, _ := repos.sum.GetByLevel(ctx, f.Story.ID, domain.SummaryLevelStory)
			if sum2 == nil || sum2.Content == "" {
				r.t.Error("expected summary to exist")
			}
		}
	}
}

func (r *GoldenTestRunner) dropCollections() {
	for _, name := range []string{
		"stories", "scenes", "scene_edges", "characters", "character_state",
		"generations", "character_memories", "timeline_events", "summaries",
		"story_bibles", "locations", "jobs", "story_runs", "run_steps",
		"narrative_events", "character_views",
	} {
		r.db.Collection(name).Drop(context.Background())
	}
}

type mockProseGolden struct {
	output string
}

func (m *mockProseGolden) GenerateScene(_ context.Context, _ llm.PromptParams) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: m.output, Model: "mock-sonnet"}, nil
}

type mockExtractGolden struct {
	output string
}

func (m *mockExtractGolden) ExtractState(_ context.Context, _ string, _ map[string]string) (*llm.StateDeltas, error) {
	var result llm.StateDeltas
	if err := json.Unmarshal([]byte(m.output), &result); err != nil {
		return &llm.StateDeltas{Deltas: []llm.StateDelta{}}, nil
	}
	return &result, nil
}

type mockSummaryGolden struct{}

func (m *mockSummaryGolden) UpdateSummary(_ context.Context, prev, scene string) (string, error) {
	return prev + "\n" + scene, nil
}

type mockValidateGolden struct{}

func (m *mockValidateGolden) ValidateAgainstCanon(_ context.Context, _, _, _ string) (map[string]any, error) {
	return map[string]any{"violations": []any{}}, nil
}

type mockEmbedGolden struct{}

func (m *mockEmbedGolden) GenerateEmbedding(_ context.Context, _ string) ([]float64, error) {
	return make([]float64, 128), nil
}

func (m *mockEmbedGolden) Model() string { return "mock-embed" }

func TestGolden_SimpleDialogue(t *testing.T) {
	fixture := loadFixture(t, "fixtures/simple-dialogue")
	runner := newRunner(t, fixture)
	runner.run()
}
