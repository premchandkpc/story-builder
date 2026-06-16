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

Both clients are always created at startup. Router retries on failure (1 initial + 2 retries = 3 total, exponential backoff 250ms/500ms).

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

## Prompt Compiler

`internal/prompt/compiler.go` — Built and wired into all 6 LLM services (Prose, Extraction, Summary, Merge, Validation, Outline).

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

### Templates (7 built-in)

| Name | Model | Temp | System |
|---|---|---|---|
| `scene_prose` | claude-sonnet | 0.8 | "You generate story prose" |
| `state_extract` | local-7b | 0.0 | "You extract character state deltas" |
| `summary_update` | local-7b | 0.2 | "You maintain a running plot summary" |
| `canon_validate` | claude-haiku | 0.0 | "You check continuity violations" |
| `join_merge` | claude-haiku | 0.2 | "You merge parallel branch summaries" |
| `outline_story` | local-7b | 0.7 | "You generate structured story outlines" |
| `generate_title` | local-7b | 0.5 | "You generate short story titles" |

## Pipeline Flow (per scene generation)

```
GenerateScene (Sonnet, 0.8 via Anthropic)
    │
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

```go
type CompiledContext struct {
    Characters     []Character
    Location       *Location
    BranchSummary  string
    CharState      map[string]CharacterState
    Memories       []Memory
    BeatIntent     string
    POV            string
    Tone           string
    TargetWords    int
}
```

`CompiledContext.Hash()` — serializes to JSON → SHA256 → hex string. Used for generation staleness detection.

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
- Retry: 1 initial + 2 retries = 3 attempts total, exponential backoff 250ms/500ms
- Fallback: if Anthropic unavailable for Sonnet/Haiku, returns error

### CachedLLMClient (optional, Redis-backed)
- Wraps any LLMClient
- Checks cache before LLM call (keyed by prompt hash)

### RateLimitedLLMClient (optional, Redis-backed)
- Wraps any LLMClient
- Enforces sliding window rate limits per provider
