# LLM Pipeline

5 prompts per node, each with a specific model/temperature routing.

## Prompt registry

| Key              | Model       | Temp | System prompt role     |
|------------------|-------------|------|------------------------|
| `scene_prose`    | Sonnet      | 0.8  | Fiction co-writer      |
| `state_extract`  | local 7B    | 0    | Continuity clerk       |
| `summary_update` | local 7B    | 0.2  | Plot summary keeper    |
| `join_merge`     | Haiku       | 0.2  | Branch converger       |
| `canon_validate` | Haiku       | 0    | Continuity editor      |

## Pipeline flow (per node)

```
1. GenerateSceneWorker
   Input: CompiledContext (canon cards, char state, lore, beat intent)
   Prompt: BuildSceneProseSystemPrompt + BuildSceneProseUserMessage
   Model: Sonnet, temp=0.8
   Output: scene prose text
   Next: enqueue ExtractState job

2. ExtractStateWorker
   Input: scene text
   Prompt: BuildStateExtractSystemPrompt
       (instructs LLM to call `record_state_deltas` tool)
   Model: local 7B, temp=0
   Output: StateDeltas JSON
   Next: ledger.ApplyDeltas(), enqueue UpdateSummary job

3. UpdateSummaryWorker
   Input: previous summary + new scene text
   Prompt: BuildSummaryUpdateSystemPrompt
   Model: local 7B, temp=0.2
   Output: updated summary text

4. MergeBranchesWorker (only if fork node's branches all accepted)
   Input: two branch summaries + timeline note
   Prompt: BuildJoinMergeSystemPrompt (returns JSON)
   Model: Haiku, temp=0.2
   Output: merged summary JSON

5. ValidateSceneWorker (optional toggle per node)
   Input: compiled canon XML, char state, draft scene
   Prompt: BuildCanonValidateSystemPrompt (returns JSON)
   Model: Haiku, temp=0
   Output: validation result JSON
```

## CompiledContext

Fetched from DB before generation:

```go
type CompiledContext struct {
    CharacterCards []canon.Card         // pinned character versions
    LocationCard   *canon.Card          // pinned location version
    BranchSummary  string               // accumulated summary for this branch
    CharState      map[string]ledger.CharacterState  // per-char state
    Lore           []string             // relevant lore entries
    BeatIntent     string               // what this scene should accomplish
    POV            string               // whose perspective
    Tone           string               // mood/tone
    TargetWords    int                  // target length
}
```

## Staleness detection

`CompiledContext.Hash()` = SHA256 of canonical JSON.

Before generation, compared against stored `context_hash` from the last generation. If different, previous output is stale.

## Prompt builders (`internal/compiler/prompts.go`)

- `BuildSceneProseSystemPrompt`: wraps canon cards, char state, lore in XML tags + writing rules
- `BuildSceneProseUserMessage`: "Write the scene where: {beat_intent}"
- `BuildStateExtractSystemPrompt`: instructs LLM to emit `record_state_deltas` function call
- `BuildSummaryUpdateSystemPrompt`: takes prev_summary + new_scene, returns updated summary
- `BuildJoinMergeSystemPrompt`: takes two branch summaries + timeline note, returns merged JSON
- `BuildCanonValidateSystemPrompt`: continuity checker, returns JSON with issues found

## Model tiers

```go
const (
    ModelSonnet ModelTier = "claude-sonnet"  // Anthropic Claude Sonnet
    ModelHaiku  ModelTier = "claude-haiku"   // Anthropic Claude Haiku
    ModelLocal  ModelTier = "local-7b"       // Ollama local 7B model
)
```

LLM clients not yet wired. Server returns "not implemented" for generate endpoints.
