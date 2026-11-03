# Code Audit — Story Builder

**15 findings** · 4 critical · 6 high · 3 medium · 2 low · 14 fixed · 1 wontfix

---

## By Severity

### Critical

| ID | Title | Status | Fix |
|---|---|---|---|
| F-01 | genInFlight leak on panic | ✓ fixed | Added `defer s.genInFlight.Delete(sceneID)` in goroutine. `generation.go:87` |
| F-02 | CharState only holds last participant | ✓ fixed | Moved `make(map[string]interface{})` before character loop. `generation.go:128` |
| F-03 | AcceptGeneration race condition | ✓ fixed | Added `acceptInFlight sync.Map` guard; atomic accept pass. `generation.go:33,180` |
| F-15 | Generate called with empty context | ✓ fixed | Added `buildPromptParams()` — fetches characters, states, locations, summary. `generation.go:94-178` |

### High

| ID | Title | Status | Fix |
|---|---|---|---|
| F-04 | GenerateStory missing character fields | ✓ fixed | Map all StoryOutlineCharacter fields. `stories.go:120-154` |
| F-05 | GenerateStory skips Location creation | ✓ fixed | New Location domain/repo/service; populate LocationRef. `stories.go` + `domain/location.go` |
| F-06 | GenerateStory missing TimelinePosition | ✓ fixed | Set `scene.TimelinePosition = i+1`. `stories.go:154` |
| F-07 | Entire Location system absent | ✓ fixed | Created `domain.Location`, `LocationRepo`, `LocationService`, CRUD handlers |
| F-08 | Topology sorts by insertion order | ✓ fixed | Sort by `TimelinePosition` before returning. `nodes.go:198` |
| F-10 | ExtractStateWorker ignores location/mood | ✓ fixed | Set `state.Location` and `state.Mood` from deltas. `extract.go:53-58` |

### Medium

| ID | Title | Status | Fix |
|---|---|---|---|
| F-09 | Dual topology endpoint confusion | ✓ fixed | Removed dead `Topology()` handler. `scenes.go:79-96` |
| F-11 | 14-param constructor | ✓ fixed | Refactored to `GenerationServiceConfig` struct. `generation.go:16-58` |
| F-12 | Missing pipeline step observability | ✓ fixed | Added `StepStatus map[string]string` to Generation; pipeline tracks each step. `scene.go:46` |

### Low

| ID | Title | Status | Fix |
|---|---|---|---|
| F-13 | go.mod minimum Go version | – wontfix | Already at `go 1.26.4` (need 1.21+). |
| F-14 | docs/schema.md Props type | ✓ fixed | Corrected `map` → `[]string`. |

---

## Status Summary

- **Fixed:** 14/15
- **Won't Fix:** 1/15 (already correct)
- **Remaining:** 0 actionable

All 14 actionable findings resolved. `go build ./...`, `go vet ./...`, and integration test compilation pass.
