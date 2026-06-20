# Story Builder remediation plan

## Week 1 — correctness lockdown

| # | Fix | Priority | Files |
|---|-----|----------|-------|
| 1 | SceneService.Update: merge onto existing before save | P0 | `internal/service/story.go` |
| 2 | AcceptGeneration: scene-level `AcceptedGenerationID` source of truth | P0 | `internal/domain/scene.go`, `internal/service/generation.go` |
| 3 | CharacterService.Update: move merge/versioning into service layer | P1 | `internal/service/story.go`, `internal/api/characters.go` |
| 4 | Audit all ReplaceOne repos — standardize update semantics | P1 | `internal/repository/mongo/scenes.go`, `internal/repository/mongo/chapters.go`, `internal/repository/mongo/locations.go`, `internal/repository/mongo/generations.go`, `internal/repository/mongo/bible.go` |

## Week 2 — simplify architecture truth

| # | Fix | Priority | Files |
|---|-----|----------|-------|
| 5 | Write `docs/current-state.md` — explicit active vs experimental | P1 | `docs/current-state.md` |
| 6 | Hide stub endpoints from primary router | P1 | `internal/api/server.go` |
| 7 | Graph node as canonical scene unit — enforce in remaining paths | P1 | `internal/api/`, `internal/service/` |

## Week 3 — harden generation

| # | Fix | Priority | Files |
|---|-----|----------|-------|
| 8 | Generation observability: prompt hash, context hash, step durations, model in every generation record | P1 | `internal/service/generation_job_worker.go`, `internal/domain/job.go` |
| 9 | Stuck job recovery tests | P2 | `internal/service/generation_test.go` |

## Week 4 — frontend correctness

| # | Fix | Priority | Files |
|---|-----|----------|-------|
| 10 | Build accept-generation UI flow | P1 | `web/src/components/`, `web/src/api/` |
| 11 | Split `api/types.ts` and `api/client.ts` by feature | P2 | `web/src/api/` |
| 12 | Backend stats endpoint for story list | P2 | `internal/api/stories.go`, `internal/service/` |
