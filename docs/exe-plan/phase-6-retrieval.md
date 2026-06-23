# Phase 6: Retrieval Layer for Memory/Canon

## Problem

Current `ContextBuilder.Build()` (`internal/service/context.go`) fetches top-10 memories per character sorted by importance. No reranking, no recency bias, no distinction between "hot" and "cold" context. All memories go into the prompt as one blob. This causes token bloat and context degradation.

## Target

A retrieval service that fetches, scores, and selects context segments hierarchically:

```
Hard constraints (always included)
  ├── Active participants
  ├── Recent timeline events (last 3)
  ├── Unresolved arc beats
  └── Current scene goal
Semantic memories (top-K by score)
  ├── Vector similarity × importance × recency
  └── Reranked by purpose
Cold context (snippets)
  ├── World bible (compressed)
  ├── Distant relationships
  └── Historical events
```

## RetrievalService

```go
// internal/retrieval/service.go
type RetrievalService struct {
    MemoryRepo  repository.MemoryRepository
    TimelineRepo repository.TimelineRepository
    BibleRepo    repository.BibleRepository
    Embedding    llm.EmbeddingService
}

type RetrievalResult struct {
    HardConstraints HardContext
    HotContext      HotContext
    ColdContext     ColdContext
    TokenBudget     TokenBudgetSummary
}

type HotContext struct {
    Memories    []ScoredMemory
    TokenCount  int
}

type ScoredMemory struct {
    Memory      domain.CharacterMemory
    Relevance   float64 // 0.0-1.0 cosine similarity
    Importance  float64 // from memory record
    Recency     float64 // 0.0-1.0 normalized age
    FinalScore  float64 // weighted combination
}

func (r *RetrievalService) RetrieveForScene(ctx context.Context, storyID, sceneID string, charIDs []string, sceneGoal string) (*RetrievalResult, error) {
    // 1. Fetch hard constraints
    hardCtx, err := r.fetchHardConstraints(ctx, storyID, sceneID)

    // 2. Fetch + score semantic memories
    hotCtx, err := r.fetchHotContext(ctx, storyID, charIDs, sceneGoal, hardCtx)

    // 3. Fetch cold snippets
    coldCtx, err := r.fetchColdContext(ctx, storyID, charIDs)

    // 4. Enforce token budget
    result := r.enforceBudget(hardCtx, hotCtx, coldCtx)

    return result, nil
}
```

## Scoring Pipeline

```go
func (r *RetrievalService) scoreMemory(mem *domain.CharacterMemory, query []float64, latestTime time.Time) *ScoredMemory {
    // Cosine similarity
    relevance := cosineSimilarity(query, mem.Embedding)

    // Importance (0.0-1.0, normalized)
    importance := mem.Importance

    // Recency (1.0 = now, 0.0 = very old)
    age := latestTime.Sub(mem.CreatedAt)
    maxAge := 30 * 24 * time.Hour // 30 day window
    recency := 1.0 - float64(age)/float64(maxAge)
    if recency < 0 { recency = 0 }

    // Weighted combination
    finalScore := relevance*0.5 + importance*0.3 + recency*0.2

    return &ScoredMemory{
        Memory:     *mem,
        Relevance:  relevance,
        Importance: importance,
        Recency:    recency,
        FinalScore: finalScore,
    }
}
```

## Configurable Per-Scene

```go
type RetrievalConfig struct {
    MaxHotTokens        int     // default 4000
    MaxColdTokens       int     // default 2000
    RelevanceWeight     float64 // default 0.5
    ImportanceWeight    float64 // default 0.3
    RecencyWeight       float64 // default 0.2
    MinFinalScore       float64 // default 0.1 — discard below this
    MaxMemoriesPerChar  int     // default 5
}
```

## Integration with ContextBuilder

Replace current memory fetch in `internal/service/context.go`:

```go
// Before: simple top-K by importance
memories, _ := s.memRepo.ListByCharacter(ctx, charID)
sort.Slice(memories, func(i, j int) bool {
    return memories[i].Importance > memories[j].Importance
})
if len(memories) > 10 {
    memories = memories[:10]
}

// After: scored retrieval
result, _ := s.retrievalSvc.RetrieveForScene(ctx, storyID, sceneID, charIDs, scene.BeatIntent)
hotHunks := result.HotContext.Memories
coldStrs := result.ColdContext.Snippets
```

## Collections / Indexes

No new collections — existing `character_memories` with embedding index:

```go
"character_memories": {
    {Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "createdAt", Value: -1}}},
    {Keys: bson.D{{Key: "storyId", Value: 1}, {Key: "characterId", Value: 1}, {Key: "importance", Value: -1}}},
    // Vector index (Atlas Search) exists
},
```

## File Changes Summary

| File | Change |
|------|--------|
| `internal/retrieval/` (new) | `service.go`, `scorer.go`, `config.go` |
| `internal/service/context.go` | Replace top-K memory fetch with retrieval service |
| `internal/llm/embedding.go` | Ensure cosine similarity exported |
