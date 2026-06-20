# LLM Pipeline

## Model Tiers

```go
const (
    ModelSonnet ModelTier = "claude-sonnet"  // High-quality prose generation
    ModelHaiku  ModelTier = "claude-haiku"   // Fast, cheap validation
    ModelLocal  ModelTier = "local-7b"       // Offline extraction/summary
)
```

**Default model mappings:**
- `sonnet` → `claude-sonnet-4-20250514` (Anthropic)
- `haiku` → `claude-haiku-3-5-20250224` (Anthropic)
- `local-7b` → `llama3.2:3b` (Ollama)

## LLM Router

`internal/llm/router.go` — Dispatches requests to the correct provider based on model tier:

- `claude-sonnet` / `claude-haiku` → AnthropicClient
- `local-7b` → OllamaClient

Both clients are always created at startup and wrapped in a `CircuitBreakerClient`. Router retries on failure (attempt + 2 retries = 3 total) with exponential backoff + jitter:

| Tier | Initial | Max | Factor |
|------|---------|-----|--------|
| Anthropic (sonnet/haiku) | 1s | 15s | 2× (±25% jitter) |
| Local (ollama) | 200ms | 5s | 2× (±25% jitter) |

JSON output validation is enabled on requests from services that expect JSON (Extraction, Merge, Validation, Outline). If the response is not valid JSON, the router retries the request.

## Service Interfaces

All interfaces defined in `internal/llm/types.go`, implementations in `internal/llm/services.go`.

### ProseService
- `GenerateScene(ctx, params PromptParams) (*CompletionResponse, error)`
- Calls prompt compiler for system prompt (10 layers)
- Temperature: 0.8, MaxTokens: 4096
- Model: claude-sonnet

### ExtractionService
- `ExtractState(ctx, sceneText string, roster map[string]string) (*StateDeltas, error)`
- Extracts state deltas from generated scene text
- Expects JSON response from LLM
- Temperature: 0, MaxTokens: 1024
- Model: local-7b

### SummaryService
- `UpdateSummary(ctx, previousSummary, newScene string) (string, error)`
- Updates running plot summary
- Rules: max 200 words, preserve facts, chronological, present tense
- Temperature: 0.2, MaxTokens: 1024
- Model: local-7b

### MergeService
- `MergeBranches(ctx, summaryA, summaryB, timelineNote string) (map[string]interface{}, error)`
- Merges parallel branch summaries at join nodes
- Outputs JSON: `{"merged_summary": "...", "conflicts": [...]}`
- Temperature: 0.2, MaxTokens: 1024
- Model: claude-haiku

### ValidationService
- `ValidateAgainstCanon(ctx, canonXML, charState, draft string) (map[string]interface{}, error)`
- Validates draft against canon for continuity violations
- Outputs JSON: `{"violations": [{"type": "...", "character": "...", "evidence": "...", "explanation": "...", "severity": "high|low"}]}`
- Checks: character consistency, timeline, lore, dialogue
- Temperature: 0, MaxTokens: 2048
- Model: claude-haiku

### OutlineService
- `GenerateOutline(ctx, synopsis string) (*StoryOutline, error)`
- Generates structured story outline from synopsis
- Output: `StoryOutline` with title, characters, beats, edges
- Schema: 5-12 beats, characters with goals/flaws, seq/fork/join edges, acts 1-3
- Temperature: 0.7, MaxTokens: 4096
- Model: local-7b

### TitleService
- `GenerateTitle(ctx, synopsis string) (string, error)`
- Generates a short story title (3-8 words) from synopsis
- Strips quotes and whitespace from output
- Temperature: 0.5, MaxTokens: 64
- Model: local-7b

### BibleService
- `GenerateBible(ctx, story domain.Story) (*domain.StoryBible, error)`
- Generates complete story bible (world, rules, magic, factions, cultures, tone)
- Exists as a standalone service (not part of scene generation pipeline)
- Output is valid JSON matching the bible schema
- Temperature: 0.3, MaxTokens: 8192
- Model: claude-sonnet
- One-shot generation — bible is generated once, never regenerated

## Prompt Compiler

`internal/prompt/compiler.go` — Built and wired into all 7 LLM services (Prose, Extraction, Summary, Merge, Validation, Outline, Bible).

### 10-Layer Hierarchy

```
Safety Prompt         Content filters, banned topics (always last)
Frame Prompt          Per-frame instructions
Scene Prompt          Beat intent, POV, location, emotion
Character Prompt      Current mood, memory, relationships
Scenario Prompt       Scenario-level context
Memory Prompt         Retrieved character/world memories
Chapter Prompt        Chapter goals and progression
Story Prompt          Premise, theme, tone
Culture Prompt        Region, language, social norms
Global Prompt         Genre, rating, safety rules
```

Each layer supports 5 merge strategies: `override`, `merge`, `append`, `replace`, `disable`.

### Templates (8 built-in)

| Name | Model | Temp | System |
|---|---|---|---|
| `scene_prose` | claude-sonnet | 0.8 | "You generate story prose" |
| `state_extract` | local-7b | 0.0 | "You extract character state deltas" |
| `summary_update` | local-7b | 0.2 | "You maintain a running plot summary" |
| `canon_validate` | claude-haiku | 0.0 | "You check continuity violations" |
| `join_merge` | claude-haiku | 0.2 | "You merge parallel branch summaries" |
| `outline_story` | local-7b | 0.7 | "You generate structured story outlines" |
| `generate_title` | local-7b | 0.5 | "You generate short story titles" |
| `generate_bible` | claude-sonnet | 0.3 | "You are a world-building assistant" |

## Pipeline Flow (per scene generation)

```
GenerateScene (Sonnet, 0.8 via Anthropic)
    │  ContextHash = CompiledContext.Hash() (SHA256)
    │  PromptSnapshot = system + user prompt (for observability)
    │  Stores in generation document
    ▼
ExtractState (local-7b, 0 via Ollama)
    │  Extracts state deltas
    │  Appends to character_state collection in Mongo
    ▼
MemoryUpdate (local-7b)
    │  Creates character memories from state changes
    │  Stores with embeddings in character_memories
    ▼
TimelineUpdate
    │  Records timeline event
    ▼
SummaryUpdate (local-7b, 0.2 via Ollama)
    │  Updates scene-level summary
    ▼
ValidateScene (Haiku, 0 via Anthropic)
    │  Checks draft against canon
    │  Stores validation result in generation document
```

All steps are in-process goroutines (no River, no Kafka). Pipeline runs asynchronously after initial scene generation.

## Memory Retrieval

Before scene generation, the context builder:

```
Character
    ↓
Fetch Relevant Memories from Mongo (vector search)
    ↓
Rank by importance
    ↓
Build Context
    ↓
Generate Scene
```

MongoDB Atlas Search handles vector similarity. No Qdrant, no pgvector.

## CompiledContext

`internal/llm/context.go` — Internal struct used by ProseService to build prompts. Assembled from `PromptParams` at generation time.

```go
type CompiledContext struct {
    CharacterCards []CharacterCard
    LocationCard   *CharacterCard
    BranchSummary  string
    CharState      map[string]CharacterState
    Memories       map[string][]string
    Lore           []string
    BeatIntent     string
    POV            string
    Tone           string
    TargetWords    int
}
```

Methods:

| Method | Output | Purpose |
|--------|--------|---------|
| `Hash()` | hex string | JSON→SHA256→hex; generation staleness detection |
| `BuildCanonXML()` | XML string | Serializes character cards to canon format for validation |
| `BuildCharStateXML()` | XML string | Serializes character states for extraction |
| `BuildSceneProseSystemPrompt()` | string | Builds the system prompt template |
| `BuildSceneProseUserMessage()` | string | Builds the user message with scene context |
| `BuildScenePromptSnapshot()` | string | Snapshot for generation metadata |

The higher-level `ContextBuilder.Build()` in `internal/service/context.go` returns `BuiltContext`:

```go
type BuiltContext struct {
    Params         llm.PromptParams
    CanonXML       string
    CharStateXML   string
    BranchSummary  string
    CharacterNames []string
}
```

## LLM Clients

### AnthropicClient
- Endpoint: `POST https://api.anthropic.com/v1/messages`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`
- Timeout: 120s
- Reads `content[].text` from response

### OllamaClient
- Endpoint: `POST {baseURL}/v1/chat/completions` (OpenAI-compatible)
- Default baseURL: `http://localhost:11434`
- Timeout: 120s
- Reads `choices[0].message.content` from response

### Router
- Wraps both clients
- Dispatches by model tier
- Retry: 1 initial + 2 retries = 3 attempts total, exponential backoff + jitter
  - Anthropic: 1s base, 15s max, 2× (±25% jitter)
  - Local: 200ms base, 5s max, 2× (±25% jitter)
- JSON response validation: if `ValidateJSON` flag is set, validates JSON before returning; retries on invalid
- Circuit breaker: 5 consecutive failures → open for 30s → half-open probe → closed on success
- Fallback: if Anthropic unavailable for Sonnet/Haiku, returns error

### CachedLLMClient (optional, Redis-backed)
- Wraps any LLMClient
- Checks cache before LLM call (keyed by prompt hash)

### RateLimitedLLMClient (optional, Redis-backed)
- Wraps any LLMClient
- Enforces sliding window rate limits per provider
