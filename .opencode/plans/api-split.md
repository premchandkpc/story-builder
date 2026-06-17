# Plan: Split `internal/api/handlers.go` and `handlers_v2.go` by entity domain

## Summary
Split 921 lines of handler methods across 2 files into 11 focused files (1 per entity domain).

## Files to modify

### `handlers.go` (was 431 lines → ~50 lines)
**Keep only:** `Handlers` struct, `NewHandlers` constructor, `readJSON` helper.

Remove all handler methods (moved to domain files below).

### Delete `handlers_v2.go` (490 lines)
All content redistributed to domain files below.

### `server.go` — no changes
Router setup, middleware, `writeJSON`, `writeError` stay as-is.

## Files to create

### `stories.go` — Story CRUD + GenerateStory
```
func (h *Handlers) CreateStory(w, r)
func (h *Handlers) GetStory(w, r)
func (h *Handlers) UpdateStory(w, r)
func (h *Handlers) ListStories(w, r)
func (h *Handlers) DeleteStory(w, r)
func (h *Handlers) GenerateStory(w, r)
```
Imports: encoding/json, log/slog, net/http, chi, domain, service, llm

### `scenes.go` — V1 scene handlers
```
func (h *Handlers) CreateScene(w, r)
func (h *Handlers) GetScene(w, r)
func (h *Handlers) UpdateScene(w, r)
func (h *Handlers) ListScenes(w, r)
func (h *Handlers) DeleteScene(w, r)
func (h *Handlers) Topology(w, r)
```
Imports: encoding/json, net/http, chi, domain

### `nodes.go` — V2 node handlers + response types
```
types: graphNode, graphEdge, topologyResponse
funcs: sceneToNode, edgeToGraphEdge, extractIDs
func (h *Handlers) ListNodes(w, r)
func (h *Handlers) GetNode(w, r)
func (h *Handlers) CreateNode(w, r)
func (h *Handlers) UpdateNode(w, r)
func (h *Handlers) DeleteNode(w, r)
func (h *Handlers) V2Topology(w, r)
```
Imports: encoding/json, net/http, time, chi, domain

### `edges.go` — V1 + V2 edge handlers
```
func (h *Handlers) CreateEdge(w, r)
func (h *Handlers) ListEdges(w, r)
func (h *Handlers) DeleteEdge(w, r)
func (h *Handlers) V2CreateEdge(w, r)
func (h *Handlers) V2ListEdges(w, r)
```
Imports: encoding/json, net/http, chi, domain

### `characters.go` — V1 + V2 character handlers
```
func (h *Handlers) CreateCharacter(w, r)
func (h *Handlers) GetCharacter(w, r)
func (h *Handlers) ListCharacters(w, r)
func (h *Handlers) V2ListCharacters(w, r)
func (h *Handlers) V2CreateCharacter(w, r)
func (h *Handlers) V2GetCharacter(w, r)
func (h *Handlers) V2UpdateCharacter(w, r)
```
Imports: encoding/json, net/http, chi, domain

### `generations.go` — V1 + V2 generation handlers
```
func (h *Handlers) GenerateScene(w, r)
func (h *Handlers) ListGenerations(w, r)
func (h *Handlers) AcceptGeneration(w, r)
func (h *Handlers) V2GenerateNode(w, r)
func (h *Handlers) V2ListNodeGenerations(w, r)
func (h *Handlers) V2AcceptGeneration(w, r)
```
Imports: encoding/json, net/http, chi

### `timeline.go` — Timeline event handlers
```
func (h *Handlers) CreateTimelineEvent(w, r)
func (h *Handlers) ListTimelineEvents(w, r)
```
Imports: encoding/json, net/http, chi, domain

### `summaries.go` — Summary handlers
```
func (h *Handlers) GetSummaryByLevel(w, r)
func (h *Handlers) GetSceneSummary(w, r)
```
Imports: encoding/json, net/http, chi

### `memories.go` — Memory handlers
```
func (h *Handlers) ListMemories(w, r)
func (h *Handlers) SearchMemories(w, r)
```
Imports: encoding/json, net/http, chi, domain

### `helpers.go` — Stub endpoints
```
func (h *Handlers) EmptyArray(w, r)
func (h *Handlers) NotImplemented(w, r)
```
Imports: net/http

## Execution steps
1. Write `handlers.go` (trimmed) — struct + NewHandlers + readJSON
2. Write `stories.go` — CreateStory..DeleteStory + GenerateStory
3. Write `scenes.go` — V1 scene handlers + Topology
4. Write `nodes.go` — V2 node handlers + types + converters
5. Write `edges.go` — V1+V2 edge handlers
6. Write `characters.go` — V1+V2 character handlers
7. Write `generations.go` — V1+V2 generation handlers
8. Write `timeline.go` — timeline event handlers
9. Write `summaries.go` — summary handlers
10. Write `memories.go` — memory handlers
11. Write `helpers.go` — EmptyArray + NotImplemented
12. Delete `handlers_v2.go`
13. Run `go build ./...` and `go vet ./...` to verify
14. Run integration tests to verify no breakage

## Verification
```sh
cd internal/api && go build .
cd ../.. && go vet ./...
go test -tags=integration -count=1 ./internal/test/integration/
```
