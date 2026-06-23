# Phase 3: Test Harness

## Problem

Current test situation (`internal/test/integration/`, 8 files) is light for a system with this many moving parts. No deterministic pipeline tests. No property tests for graph invariants. No contract tests for agents. Most tests use live Mongo + live LLM or stubs, but there's no standard mock layer for the LLM.

## Target

Three-layer test strategy:
1. **Golden pipeline tests** (P0) — deterministic fixture-driven generation tests
2. **Property/invariant tests** (P1) — graph + narrative invariants via property-based testing
3. **Agent contract tests** (P2) — input/output contract verification per agent

---

## Layer 1: Golden Pipeline Tests

### Test Fixture Format

**Directory:** `test/golden/fixtures/<scenario>/`

Each scenario is a directory with JSON files:

```
test/golden/fixtures/simple-dialogue/
  story.json             — Story + StoryBlueprint
  characters.json        — []domain.Character
  states.json            — []domain.CharacterState (initial)
  bible.json             — domain.StoryBible
  scenes.json            — []domain.Scene (graph nodes)
  edges.json             — []domain.SceneEdge
  timeline.json          — []domain.TimelineEvent (initial)
  memories.json          — []domain.CharacterMemory (initial)
  summaries.json         — []domain.Summary (initial)
  locations.json         — []domain.Location
  mocked_outputs/
    generate.json        — Mocked LLM output for GenerateScene
    extract.json         — Mocked LLM output for ExtractState
  expected/
    generation_status    — "success" | "partial_success"
    step_status.json     — map[step]status
    scene_text_tokens    — min words in generated content
    character_states_count — expected number of extracted states
    timeline_exists      — true/false
    summary_exists       — true/false
```

### Fixture Scenarios

**Phase 1 (3 scenarios):**

| Scenario | Flow | What it tests |
|----------|------|---------------|
| `simple-dialogue` | 2 characters, dialogue flow, no branching | Basic pipeline, extract, memories, timeline |
| `fork-choice-merge` | 3 scenes: A → (B or C) → D, generate on D | Graph traversal, branch summary, merge |
| `monologue-action` | 1 character, action flow, 1 turn | Monologue, state extraction, runtime |

**Phase 2 (2 more scenarios):**

| Scenario | Flow | What it tests |
|----------|------|---------------|
| `dead-character-validation` | Generated scene where char should die | Post-generation validator, event rejection |
| `empty-memory-context` | New character with no memories | Memory retrieval with empty context |

### Test Runner

```go
// test/golden/pipeline_test.go
package golden

import (
    "testing"
    "github.com/stretchr/testify/suite"
)

type PipelineTestSuite struct {
    suite.Suite
    db     *mongo.Database
    svc    *service.GenerationService
    // ... other deps
}

func (s *PipelineTestSuite) SetupSuite() {
    // Start mongo-container or connect to test Mongo
    // Create indexes
    // Wire services with mocked LLM
}

func (s *PipelineTestSuite) SetupTest() {
    // Load fixture JSON files
    // Insert fixture data into Mongo
}

func (s *PipelineTestSuite) TearDownTest() {
    // Drop fixture collections
}

func (s *PipelineTestSuite) TestSimpleDialoguePipeline() {
    // Load fixture
    // Call Generate on the configured scene
    // Assert expected outcomes
}
```

### Mocked LLM Layer

```go
// internal/llm/mock.go
type MockLLM struct {
    Responses map[string]string // key: "generate", "extract", etc.
    Calls     []string
}

func (m *MockLLM) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
    key := req.Metadata["step"]
    if resp, ok := m.Responses[key]; ok {
        m.Calls = append(m.Calls, key)
        return &CompletionResponse{Content: resp}, nil
    }
    return &CompletionResponse{Content: "{}"}, nil
}
```

### What Pipeline Tests Assert

1. **Generation status**: `generation.Status` matches expected
2. **Step status**: each step in `generation.StepStatus` matches `step_status.json`
3. **Scene content**: `scene.GeneratedContent` is non-empty, within token range
4. **Character states**: at least one `character_state` document for the scene per participant
5. **Timeline**: `timeline_events` has a matching entry
6. **Summary**: `summaries` has a scene-level summary
7. **Run records**: `story_runs` has a completed entry, `run_steps` have correct status
8. **Graph integrity**: no cycles introduced (call `ValidateDAG`)
9. **Memory dedup**: no duplicate `character_memories` for same (char, scene, type)

---

## Layer 2: Property/Invariant Tests

### Tooling

Use `github.com/leanovate/gopter` (lightweight, no external deps) or `testing/quick` for simple cases.

### Graph Properties

```go
// internal/graph/properties_test.go
func TestGraph_ValidateDAG_Acyclic(t *testing.T) {
    parameters := gopter.DefaultTestParameters()
    parameters.MinSuccessfulTests = 100
    props := gopter.NewProperties(parameters)

    props.Property("scenes form a DAG", prop.ForAll(
        func(edges []edgeGen, sceneIDs []string) bool {
            // Property: after building graph from generated edges,
            // ValidateDAG returns nil (no cycles)
            dag := buildGraph(sceneIDs, edges)
            return graph.ValidateDAG(dag.Scenes, dag.Edges) == nil
        },
        edgeGenGen(), // generator that avoids creating cycles
    ))
}

// Additional properties:
// - TopologicalSort returns all nodes in valid order
// - Deleting a node removes all its incident edges
// - FindDeadEnds correctly identifies sinks
// - FindUnreachableScenes works after node deletion
```

### Narrative Properties

```go
// internal/event/properties_test.go
func TestNarrative_Invariants(t *testing.T) {
    props.Property("dead character cannot emit actions", prop.ForAll(
        func(events []domain.NarrativeEvent, state StoryState) bool {
            for _, e := range events {
                if e.SubjectType == "character" && isActionEvent(e.EventType) {
                    charState := state.Characters[e.SubjectID]
                    if charState != nil && charState.Health <= 0 {
                        return false // violation
                    }
                }
            }
            return true
        },
        eventGenGen(),
    ))

    props.Property("timeline order is monotonic", prop.ForAll(
        func(events []domain.TimelineEvent) bool {
            for i := 1; i < len(events); i++ {
                if events[i].Order <= events[i-1].Order {
                    return false
                }
            }
            return true
        },
        timelineGenGen(),
    ))
}
```

### What to Property-Test (Priority Order)

1. **Graph acyclicity** — 100 random DAGs, no cycles after operations
2. **Topological sort completeness** — sort returns all nodes
3. **Scene status transitions** — `CanTransitionTo` allows valid paths, rejects invalid
4. **Relationship bounds** — trust/respect/fear/affection ∈ [0, 100] after any delta
5. **Event ID uniqueness** — no duplicate IDs in a single generation run
6. **Edge type semantics** — fork nodes have >1 outgoing edges, join nodes have >1 incoming

---

## Layer 3: Agent Contract Tests

### Contracts per Agent

```go
// internal/agents/contract_test.go

// Input contract — valid empty, valid full, malformed
type AgentContractTest struct {
    Name        string
    Spec        AgentSpec
    ValidInput  AgentInput
    EmptyInput  AgentInput
    Malformed   AgentInput // missing required fields
}

func TestDirectorContract(t *testing.T) {
    spec, _ := registry.Get("director")
    test := AgentContractTest{
        Name: "director",
        Spec: spec,
        ValidInput: AgentInput{
            Ctx: &AgentContext{
                Scene: &domain.Scene{FlowType: "dialogue", MaxTurns: 5, BeatIntent: "Hero confronts villain"},
                Characters: []*domain.Character{{Name: "Hero"}, {Name: "Villain"}},
            },
        },
        EmptyInput: AgentInput{
            Ctx: &AgentContext{
                Scene: &domain.Scene{FlowType: "silent"},
            },
        },
    }
    test.Run(t)
}

func (ct *AgentContractTest) Run(t *testing.T) {
    t.Run(ct.Name+"/valid_input", func(t *testing.T) {
        output, err := ct.Spec.Runner(context.Background(), ct.ValidInput)
        assert.NoError(t, err)
        assert.NotEmpty(t, output.Content)
    })
    t.Run(ct.Name+"/malformed_json_handling", func(t *testing.T) {
        // Mock LLM to return unparseable JSON
        // Assert agent returns structured error, not panic
        output, err := ct.Spec.Runner(context.Background(), ct.Malformed)
        assert.NoError(t, err) // should degrade gracefully
        assert.Contains(t, output.Error, "parse") // or similar
    })
    t.Run(ct.Name+"/empty_context", func(t *testing.T) {
        output, err := ct.Spec.Runner(context.Background(), ct.EmptyInput)
        assert.NoError(t, err)
        // Should produce something, even if minimal
    })
    t.Run(ct.Name+"/timeout", func(t *testing.T) {
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
        defer cancel()
        time.Sleep(5 * time.Millisecond) // ensure timeout
        _, err := ct.Spec.Runner(ctx, ct.ValidInput)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "deadline")
    })
}
```

### Contracts to Write

| Agent | Contract Tests |
|-------|---------------|
| Director | Valid scene plan, empty participants, malformed LLM output |
| Narrator | Valid prose stitching, empty turns, single turn |
| Editor | Valid prose trim, already-clean prose, empty input |
| CanonGuard | Consistent states, contradictory states, empty canon |
| Critic | Valid scoring, zero-length scene, already-scored scene |
| StateExtractor | Valid state delta, no changes, malformed JSON |
| World | Valid lore check, empty bible, unknown faction |
| Arc | Valid arc tracking, no arcs defined, completed arcs |
| Memory | Valid memory extraction, empty memories, too many memories |

---

## Test Infrastructure

### MongoTestContainer

Use `github.com/testcontainers/testcontainers-go` for integration tests with real Mongo:

```go
// internal/test/mongo.go
func SetupTestDB(t *testing.T) *mongo.Database {
    ctx := context.Background()
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image: "mongo:7",
            ExposedPorts: []string{"27017/tcp"},
        },
        Started: true,
    })
    require.NoError(t, err)
    t.Cleanup(func() { container.Terminate(ctx) })

    uri, _ := container.Endpoint(ctx, "")
    client, _ := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+uri))
    return client.Database("test_" + randomString(8))
}
```

### Test Data Builders

```go
// internal/test/builder.go
type StoryBuilder struct {
    t      *testing.T
    story  *domain.Story
    scenes []*domain.Scene
    edges  []*domain.SceneEdge
    chars  []*domain.Character
}

func NewStory(t *testing.T) *StoryBuilder {
    return &StoryBuilder{
        t: t,
        story: &domain.Story{ID: uuid(), Title: "Test Story"},
    }
}

func (b *StoryBuilder) WithScene(id, title string, opts ...SceneOpt) *StoryBuilder {
    scene := &domain.Scene{
        ID: id, StoryID: b.story.ID, Title: title,
        Status: domain.SceneStatusDraft, FlowType: "dialogue",
    }
    for _, opt := range opts {
        opt(scene)
    }
    b.scenes = append(b.scenes, scene)
    return b
}

func (b *StoryBuilder) WithEdge(from, to, edgeType string) *StoryBuilder {
    b.edges = append(b.edges, &domain.SceneEdge{
        ID: uuid(), StoryID: b.story.ID,
        FromSceneID: from, ToSceneID: to, Type: edgeType,
    })
    return b
}

func (b *StoryBuilder) Insert(ctx context.Context, db *mongo.Database) {
    // Insert all docs
}
```

---

## Existing Tests to Improve

| Existing file | Problem | Fix |
|---------------|---------|-----|
| `internal/graph/traversal_test.go` | 5 tests, hardcoded | Add property tests, edge cases |
| `internal/validation/validate_test.go` | Validates scene only | Add event validation tests |
| `internal/test/integration/pipeline_test.go` | No golden fixtures | Convert to fixture-driven |
| `internal/events/bus_test.go` | Basic pub/sub | Add wildcard, error handling, concurrent tests |
| `internal/agents/agent_helpers_test.go` | Stub tests | Add contract tests |

---

## File Changes Summary

| File | Change |
|------|--------|
| `test/golden/` (new) | Pipeline test runner + fixtures |
| `test/golden/fixtures/simple-dialogue/` (new) | Fixture JSON files |
| `test/golden/fixtures/fork-choice-merge/` (new) | Fixture JSON files |
| `test/golden/fixtures/monologue-action/` (new) | Fixture JSON files |
| `internal/graph/properties_test.go` (new) | Property-based graph tests |
| `internal/event/properties_test.go` (new) | Property-based narrative tests |
| `internal/agents/contract_test.go` (new) | Agent contract tests |
| `internal/llm/mock.go` (new) | Mock LLM for golden tests |
| `internal/test/mongo.go` (new) | Testcontainers helper |
| `internal/test/builder.go` (new) | Test data builders |
| `internal/test/integration/pipeline_test.go` | Refactor to use golden fixtures |
