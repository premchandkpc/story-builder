# LLM Pipeline

## Model Tiers

```go
const (
    ModelSonnet ModelTier = "claude-sonnet"  // High-quality prose generation
    ModelHaiku  ModelTier = "claude-haiku"   // Fast, cheap validation
    ModelLocal  ModelTier = "local-7b"       // Offline extraction/summary (maps to llama3.2:3b in Ollama)
)
```

**Default model mappings:**
- `claude-sonnet` → `claude-sonnet-4-20250514` (Anthropic)
- `claude-haiku` → `claude-haiku-3-5-20250224` (Anthropic)
- `local-7b` → `llama3.2:3b` (Ollama)

## LLM Router

`internal/llm/router.go` — Dispatches requests to the correct provider based on model tier:

- `claude-sonnet` / `claude-haiku` → AnthropicClient
- `local-7b` → OllamaClient

Both clients are always created at startup. Router retries on failure (1 initial + 2 retries = 3 total, exponential backoff 250ms/500ms).

## Prompt Registry

All entries in `internal/llm/types.go:140-183`:

| Template Key | Model | Temp | System Text |
|---|---|---|---|
| `scene_prose` | Sonnet | 0.8 | "You are a fiction co-writer. Write ONE scene and nothing else." |
| `state_extract` | local-7b | 0 | "You are a continuity clerk. Read the scene and call record_state_deltas." |
| `summary_update` | local-7b | 0.2 | "You maintain a running plot summary for one storyline branch." |
| `join_merge` | Haiku | 0.2 | "Two parallel storylines are converging. Merge their summaries." |
| `canon_validate` | Haiku | 0 | "You are a strict continuity editor. Check this draft against canon." |
| `outline_story` | local-7b | 0.7 | "You are a master story architect. Given a synopsis, generate a structured story outline with characters, plot beats, and narrative flow." |
| `generate_title` | local-7b | 0.5 | "You are a creative title generator. Given a synopsis, generate a short, engaging story title (3-8 words). Return ONLY the title, no quotes or punctuation." |

## Service Interfaces

All interfaces defined in `internal/llm/types.go:53-79`, implementations in `internal/llm/services.go`.

### ProseService (`types.go:53`, `services.go:17`)
- `GenerateScene(ctx, params PromptParams) (*CompletionResponse, error)`
- Builds `CompiledContext` → `BuildSceneProseSystemPrompt()` → system prompt with canon XML + state
- `BuildSceneProseUserMessage()` → user message with beat intent, POV, tone
- System prompt includes: character cards (name, traits, relationships, voice samples), location card, world rules (lore), current state, branch summary
- Hard rules: canon is law, no new characters, voice match, word count ±20%, prose only
- Temperature: 0.8, MaxTokens: 4096

### ExtractionService (`types.go:57`, `services.go:63`)
- `ExtractState(ctx, sceneText string, roster map[string]string) (*ledger.StateDeltas, error)`
- Extracts state deltas from generated scene text
- Uses `BuildStateExtractSystemPrompt(roster)` for system prompt
- Expects JSON response from LLM
- Temperature: 0, MaxTokens: 1024

### SummaryService (`types.go:61`, `services.go:94`)
- `UpdateSummary(ctx, previousSummary, newScene string) (string, error)`
- Updates running plot summary for a branch
- Uses `BuildSummaryUpdateSystemPrompt(previousSummary, newScene)`
- Rules: max 200 words, preserve facts, chronological, present tense
- Temperature: 0.2, MaxTokens: 1024

### MergeService (`types.go:65`, `services.go:118`)
- `MergeBranches(ctx, summaryA, summaryB, timelineNote string) (map[string]interface{}, error)`
- Merges parallel branch summaries at join nodes
- Uses `BuildJoinMergeSystemPrompt(summaryA, summaryB, timelineNote)`
- Outputs JSON: `{"merged_summary": "...", "conflicts": [...]}`
- Temperature: 0.2, MaxTokens: 1024

### ValidationService (`types.go:69`, `services.go:146`)
- `ValidateAgainstCanon(ctx, canonXML, charState, draft string) (map[string]interface{}, error)`
- Validates draft against canon for continuity violations
- Uses `BuildCanonValidateSystemPrompt(canonXML, charState, draft)`
- Outputs JSON: `{"violations": [{"type": "...", "character": "...", "evidence": "...", "explanation": "...", "severity": "high|low"}]}`
- Checks: knowledge, voice, traits, location, world rules
- Temperature: 0, MaxTokens: 2048

### OutlineService (`types.go:73`, `services.go:174`)
- `GenerateOutline(ctx, synopsis string) (*StoryOutline, error)`
- Generates structured story outline from synopsis
- Uses `BuildOutlineStorySystemPrompt(synopsis)`
- Output: `StoryOutline` with title, characters, beats, edges
- Schema: 5-12 beats, characters with goals/flaws, seq/fork/join edges, acts 1-3
- Temperature: 0.7, MaxTokens: 4096

### TitleService (`types.go:77`, `services.go:203`)
- `GenerateTitle(ctx, synopsis string) (string, error)`
- Generates a short story title (3-8 words) from synopsis
- Temperature: 0.5, MaxTokens: 64
- Strips quotes and whitespace from output

## Prompt Layering System (Future)

The current prompt registry is flat. The target architecture uses hierarchical prompt layering:

```
Global Prompt           Genre, rating, safety rules
    │
    ├── Story Prompt    Premise, theme, tone
    │
    ├── Culture Prompt  Region, language, social norms (Phase 3)
    │
    ├── Safety Prompt   Content filters, banned topics
    │
    ├── Scene Prompt    Beat intent, POV, location, emotion
    │
    ├── Character Prompt  Current mood, memory, relationships
    │
    └── Memory Prompt   Retrieved character/world memories (Phase 2)
```

Each layer supports: `override`, `merge`, `append`, `replace`, `disable`.

The **Prompt Compiler Service** (future) assembles these layers into the final system prompt sent to the LLM. See `docs/vision.md` for the full architecture.

## Pipeline Flow (per scene generation)

```
GenerateScene (Sonnet, 0.8 via Anthropic)
    │
    ▼
ExtractState (local-7b, 0 via Ollama)
    │  Extracts state deltas → persists to character_state table
    ▼
UpdateSummary (local-7b, 0.2 via Ollama)
    │  Updates scene-level summary → upserts story_summaries (level='scene')
    ▼
ValidateScene (Haiku, 0 via Anthropic)
    │  Checks draft against canon → stores in generations.validation_result
```

Note: `MergeBranches` is defined as a river worker but is **not enqueued** during the standard accept-generation pipeline. It is available for manual/branch-join use cases.

## CompiledContext

```go
type CompiledContext struct {
    CharacterCards []canon.Card            // Character: name, traits, voice samples, relationships
    LocationCard   *canon.Card             // Location: name, description, props
    BranchSummary  string                  // Story branch summary so far
    CharState      map[string]interface{}  // Per character: location, mood, knows, items
    Lore           []string                // World rules from lore table
    BeatIntent     string                  // What happens in this scene
    POV            string                  // Point-of-view character
    Tone           string                  // Scene mood
    TargetWords    int                     // Target word count
}
```

`CompiledContext.Hash()` — serializes to JSON → SHA256 → hex string. Used for generation staleness detection (optionally cached in Redis ContextCache).

Also uses `PromptParams` (types.go:18-28) which is a flattened version of `CompiledContext` used in river job args.

## LLM Clients

### AnthropicClient (`client.go:13`)
- Endpoint: `POST https://api.anthropic.com/v1/messages`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`
- Timeout: 120s
- Reads `content[].text` from response

### OllamaClient (`client.go:87`)
- Endpoint: `POST {baseURL}/v1/chat/completions` (OpenAI-compatible)
- Default baseURL: `http://localhost:11434`
- Timeout: 120s
- Reads `choices[0].message.content` from response

### Router (`router.go:10`)
- Wraps both clients
- Dispatches by model tier
- Retry: 1 initial + 2 retries = 3 attempts total, exponential backoff 250ms/500ms
- Fallback: if Anthropic unavailable for Sonnet/Haiku, returns error

### CachedLLMClient (`internal/cache/cached_llm.go`)
- Optional Redis-backed cache
- Wraps any LLMClient
- Checks cache before LLM call (keyed by prompt hash)

### RateLimitedLLMClient (`internal/cache/rate_limiter.go`)
- Optional Redis-backed rate limiter
- Wraps any LLMClient
- Enforces sliding window rate limits per provider
