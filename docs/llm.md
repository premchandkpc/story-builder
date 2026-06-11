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

## Prompt Registry

| Template Key | Model | Temp | System Text |
|---|---|---|---|
| `scene_prose` | Sonnet | 0.8 | "You are a fiction co-writer. Write ONE scene and nothing else." |
| `state_extract` | local-7b | 0 | "You are a continuity clerk. Read the scene and call record_state_deltas." |
| `summary_update` | local-7b | 0.2 | "You maintain a running plot summary for one storyline branch." |
| `join_merge` | Haiku | 0.2 | "Two parallel storylines are converging. Merge their summaries." |
| `canon_validate` | Haiku | 0 | "You are a strict continuity editor. Check this draft against canon." |
| `outline_story` | Sonnet | 0.7 | "You are a master story architect. Given a synopsis, generate a structured story outline..." |

## Service Interfaces

### ProseService (`llm/services.go:11`)
- `GenerateScene(params PromptParams) (*CompletionResponse, error)`
- Builds `CompiledContext` → `BuildSceneProseSystemPrompt()` → system prompt with canon XML + state
- `BuildSceneProseUserMessage()` → user message with beat intent, POV, tone
- System prompt includes: character cards (name, traits, relationships, voice samples), location card, world rules (lore), current state (mood, knows/does-not-know), branch summary
- Hard rules: canon is law, no new characters, voice match, word count ±20%, prose only
- Temperature: 0.8, MaxTokens: 4096

### ExtractionService (`llm/services.go:57`)
- `ExtractState(sceneText string) (map[string]interface{}, error)`
- Extracts state deltas from generated scene text
- Expects JSON response from LLM
- System prompt: "extract ONLY what is explicit in the text"
- Temperature: 0, MaxTokens: 1024

### SummaryService (`llm/services.go:85`)
- `UpdateSummary(previousSummary, newScene string) (string, error)`
- Updates running plot summary for a branch
- Rules: max 200 words, preserve facts, chronological, present tense
- Temperature: 0.2, MaxTokens: 1024

### MergeService (`llm/services.go:109`)
- `MergeBranches(summaryA, summaryB, timelineNote string) (map[string]interface{}, error)`
- Merges parallel branch summaries at join nodes
- Outputs JSON: `{"merged_summary": "...", "conflicts": [...]}`
- Temperature: 0.2, MaxTokens: 1024

### ValidationService (`llm/services.go:137`)
- `ValidateAgainstCanon(canonXML, charState, draft string) (map[string]interface{}, error)`
- Validates draft against canon for continuity violations
- Outputs JSON: `{"violations": [{"type": "...", "character": "...", "evidence": "...", "explanation": "...", "severity": "high|low"}]}`
- Checks: knowledge, voice, traits, location, world rules
- Temperature: 0, MaxTokens: 2048

### OutlineService (`llm/services.go:165`)
- `GenerateOutline(synopsis string) (*StoryOutline, error)`
- Generates structured story outline from synopsis
- Output: `StoryOutline` with title, characters, beats, edges
- Schema: 5-12 beats, characters with goals/flaws, seq/fork/join edges, acts 1-3
- Temperature: 0.7, MaxTokens: 4096

## Pipeline Flow (per node generation)

```
GenerateScene (Sonnet, 0.8)
    │
    ▼
ExtractState (local-7b, 0)
    │  Extracts state deltas from generated text
    │  → Upserts character_state rows
    ▼
UpdateSummary (local-7b, 0.2)
    │  Updates scene-level summary
    │  → Upserts story_summaries (level='scene')
    ▼
MergeBranches (Haiku, 0.2)
    │  At join nodes: merges branch summaries
    │  → Upserts story_summaries (level='story')
    ▼
ValidateScene (Haiku, 0)
    │  Checks draft against canon
    │  → Reports violations (non-blocking)
```

## CompiledContext

```go
type CompiledContext struct {
    CharacterCards []canon.Card            // Character: name, traits, voice samples, relationships
    LocationCard   *canon.Card             // Location: name, description, props
    BranchSummary  string                  // Story branch summary so far
    CharState      map[string]ledger.CharacterState  // Per character: location, mood, knows, items
    Lore           []string                // World rules from lore table
    BeatIntent     string                  // What happens in this scene
    POV            string                  // Point-of-view character
    Tone           string                  // Scene mood
    TargetWords    int                     // Target word count
}
```

`CompiledContext.Hash()` — serializes to JSON → SHA256 → hex string. Used for generation staleness detection.

## LLM Clients

### AnthropicClient (`llm/client.go:12`)
- Endpoint: `POST https://api.anthropic.com/v1/messages`
- Headers: `x-api-key`, `anthropic-version: 2023-06-01`
- Timeout: 120s
- Reads `content[].text` from response

### OllamaClient (`llm/client.go:84`)
- Endpoint: `POST {baseURL}/v1/chat/completions` (OpenAI-compatible)
- Default baseURL: `http://localhost:11434`
- Timeout: 120s
- Reads `choices[0].message.content` from response
