# Phase 0 audit — legacy Story Builder

Audit date: 2026-11-03. This is an evidence report about the repository as it
exists; it is **not** a v2 implementation plan or a claim that any legacy
feature is production-ready.

## Method and limits

- Read the HTTP routes and handlers, generation and orchestration paths,
  repositories, graph code, agents, frontend generation UI, docs, Makefile and
  Compose configuration.
- Executed `go test ./... -count=1`: passed. This excludes the Mongo-backed
  suite because every file in `internal/test/integration` has the `integration`
  build tag (`internal/test/integration/main_test.go:1`).
- Executed `npm run build` in `web`: passed. Vite reported one 650.81 kB
  minified JavaScript chunk warning.
- `go build ./...` could compile using the installed dependencies but exited
  when Go tried to write its module download stat cache outside the writable
  sandbox. This is an environment permission failure, not source evidence of a
  compile failure. `go test ./...` did compile all non-integration packages.
- Docker is not running, so tagged Mongo integration tests and a live HTTP/LLM
  run were not executed. No end-to-end claim below relies on an unrun test.

## What works, with bounded evidence

1. The non-integration Go unit suite builds and passes, including graph,
   agents, LLM, service and validation packages.
2. The frontend type-checks and produces a production bundle.
3. The basic graph write path is present: routes register story/node/edge
   handlers (`internal/api/server.go:67-102`); the graph tests include cycle
   detection (`internal/graph/properties_test.go:59`). This is not a live-Mongo
   verification.
4. Generation requests enqueue a `pending` Mongo job
   (`internal/service/generation.go:85-112`), and the worker uses atomic
   `FindOneAndUpdate` to claim an eligible job
   (`internal/repository/mongo/jobs.go:54-87`).
5. The UI does fetch generations and renders their prose after it is available:
   `SceneDrawer` calls `api.generations.list` and renders `output`
   (`web/src/components/SceneDrawer.tsx:22-31,115-122`). This is polling/list
   display, not prose token streaming.

## Actual request/data flows

### Create story, scenes, and edges

`POST /api/v1/stories` is routed by `internal/api/server.go:67-76` to the story
handler/service/repository path. Nodes and edges are registered at lines 81-102.
The intended graph validation is in the graph/service layer and is unit-tested,
but no live Mongo execution was possible in this audit.

### Generate a scene

`POST /stories/{storyID}/nodes/{nodeID}/generate` calls
`GenerationService.Generate` (`internal/api/generations.go:11-19`). It creates
a `Generation` and `Job` (`internal/service/generation.go:85-112`). The worker
claims the job, creates a run, and executes the `generate`, `extract`, `memory`,
`timeline`, `summary`, and `validate` steps in order
(`internal/service/generation_job_worker.go:82-130`,
`internal/orchestration/worker.go:228-291`).

Generation is not pure: the prose worker writes `scene.GeneratedContent` before
any acceptance (`internal/worker/generate.go:35-42`), and later steps write
state, memory, timeline, summaries, and narrative events.

### Accept / reject

Accept loads the generation and scene, writes the scene as accepted, then
updates generation flags (`internal/service/generation.go:117-156`). Those are
separate operations; there is no Mongo transaction. There is no reject handler
or route; the frontend client likewise exposes only `generate`, `list`, `get`,
and `accept` (`web/src/api/client.ts:164-178`).

### Character state

The extraction worker accepts model deltas keyed by character name, maps them to
stable `CharID`, then appends `CharacterState` (`internal/worker/extract.go:42-66`).
The context builder reads state by the same stable ID
(`internal/service/context.go:157-197`). However, the current orchestration
discards the mapping between pipeline steps (finding F-02), so the correct
mapping code is normally not given the map it needs.

### Claim, lease, retry, and recovery

Claim is atomic and increments a `version` field
(`internal/repository/mongo/jobs.go:54-87`). A heartbeat extends a lease only by
job ID (`internal/repository/mongo/jobs.go:90-96`). Startup recovery marks
stalled jobs as failed (`internal/orchestration/worker.go:88-115`), while step
retries happen in-memory (`internal/orchestration/worker.go:294-332`). There is
no job retry state, scheduled run time, fencing token, or conditional write
using the claimant/version.

## Suspected defects: confirmed or refuted

| Suspected defect | Result | Evidence |
|---|---|---|
| Unique index breaks character versioning | **Refuted.** | The index is unique on `(charId, version)`, not name (`internal/repository/mongo/client.go:51-58`); `Update` creates a new document and increments version (`internal/repository/mongo/characters.go:82-89`). |
| State/memory written by name but read by ID | **Refuted as stated; fragile by name at the LLM boundary.** | Extraction converts `delta.Character` name to `CharID` before write (`internal/worker/extract.go:42-64`); context reads by `CharID` (`internal/service/context.go:157-197`). Duplicate names can overwrite `charNameToID` (`internal/service/context.go:101-145`). |
| Generate mutates state before Accept | **Confirmed.** | Prose writes the scene at generate time (`internal/worker/generate.go:35-42`); extraction appends state (`internal/worker/extract.go:56-65`); memory, timeline and summary steps execute before acceptance (`internal/service/generation_job_worker.go:414-489`). |
| `Blocking:false` steps run sequentially | **Refuted in current orchestrator.** | `runBatch` explicitly batches non-blocking turns concurrently (`internal/agents/orchestrator.go:228-360`). The durable generation pipeline itself is deliberately sequential (`internal/orchestration/worker.go:228-291`). |
| Frontend does not fetch/display generated prose | **Refuted.** | It fetches generation lists and renders prose (`web/src/components/SceneDrawer.tsx:22-31,115-122`). It does **not** display streaming tokens: SSE transports only progress fields (`internal/api/progress.go:99-109`), and the generate action only refetches once immediately (`web/src/components/StoryGraph.tsx:262-285`). |
| Docs describe unimplemented behavior | **Confirmed.** | Docs claim atomic accept (`docs/api.md:206-218`, `docs/services.md:300-309`), but accept uses independent writes (`internal/service/generation.go:142-150`). Docs promise canon changes only on accept (`docs/rules.md:70`), contradicted by generation-time writes above. `docs/scene-agents.md:76-110` documents endpoints and SQL/sqlc work not registered in the live router, whose equivalents are `NotImplemented` under `/experimental` (`internal/api/server.go:183-226`). |

## Findings

| ID | Priority | File:line | Evidence | Production impact | Fix | Test that proves the fix |
|---|---|---|---|---|---|---|
| F-01 | P0 | `internal/worker/generate.go:35-42`; `internal/service/generation_job_worker.go:327-411,414-489` | Generate writes scene prose, state, events, memory, timeline and summary before Accept. | Reject cannot leave zero trace; later context is polluted by unaccepted prose. | Replace generations with expiring drafts. Generate writes only draft/run artifacts; one accept transaction writes canon/events/projections. | Generate then reject; byte-compare canon/state/memory/timeline collections with their pre-generate snapshot. |
| F-02 | P0 | `internal/orchestration/worker.go:237-245`; `internal/service/generation_job_worker.go:269,318-330`; `internal/worker/extract.go:42-46` | A new `StepContext.Artifacts` map is created for every step. `charNameToID` set in generate is absent in extract, so every extracted delta is skipped. | Standard pipeline silently loses character-state updates. | Use a single run-scoped immutable artifact structure, or reload stable IDs in extraction; fail rather than skip when expected mapping is absent. | Fake extractor returns a known participant delta; assert exactly one state/event keyed by that character ID. |
| F-03 | P0 | `internal/api/server.go:62-271` | All business routes are public; no authentication or ownership middleware appears before routes. | Any caller can read/change any story by ID; no tenant isolation. | Add auth principal middleware and enforce owner filters in every story-scoped repository query. | Two users: user B receives 403/404 for every user A resource route. |
| F-04 | P0 | `internal/service/generation.go:117-156`; `internal/repository/mongo/generations.go:97-109` | Accept checks only that a generation exists, not that it belongs to the requested scene; writes scene then flags generations in separate calls; errors setting accepted are logged and ignored. | Cross-scene accept can leave an invalid scene reference; partial writes and duplicate effects are possible. | Accept only a ready draft belonging to the scene, inside one Mongo transaction, with a unique/idempotent acceptance record. | Concurrent accept, cross-scene generation ID, repeated accept, and forced second-write failure all leave a single coherent result. |
| F-05 | P0 | `internal/api/server.go:81-93`; `web/src/api/client.ts:164-178` | No reject route or client method exists. | The required product loop cannot be completed. | Add `POST /drafts/{id}/reject` that deletes only the draft. | Reject absent/ready draft; prove draft deletion and no canon-side mutation. |
| F-06 | P0 | `internal/worker/generate.go:35-42`; `internal/domain/scene.go:120-143` | Standard prose worker writes content but never moves scene from `draft` to `generated`; accept permits only `generated -> accepted`. | The ordinary durable path can produce output that cannot be accepted. | In v2 remove scene mutation during generate; draft readiness replaces this state coupling. | Generate a draft from a draft scene, accept it, assert accepted prose/canon once. |
| F-07 | P0 | `internal/repository/mongo/jobs.go:90-96`; `internal/orchestration/worker.go:88-115,335-338` | Heartbeats and final writes do not include claimant/version fencing; recovery marks stalled work failed rather than retryable. | An expired worker can commit after a new worker claims the job; crashes strand jobs. | Add `PENDING/RUNNING/RETRY_WAIT/SUCCEEDED/FAILED`, fencing tokens in every handler write, expiry reclaim and idempotency keys. | Lease-expiry takeover test proves old worker write is rejected and exactly one logical result remains. |
| F-08 | P1 | `internal/service/generation.go:57-59,85-112`; `internal/repository/mongo/generations.go:56-65` | Process-local `sync.Map` is removed immediately after enqueue; hash lookup only sees accepted generations and has no TTL/unique index. | Duplicate requests or restarts can buy multiple LLM calls and produce competing writes. | Persist an idempotency key and context hash on draft creation with a unique partial index. | Same key/hash from concurrent clients results in one draft and one fake-provider invocation. |
| F-09 | P1 | `internal/api/progress.go:14-18,53-64,99-109`; `web/src/components/StoryGraph.tsx:262-285` | In-memory, lossy SSE hub emits status only; client does not connect to it when generating. | No prose streaming, progress is lost on refresh and slow clients lose events. | Persist run steps; SSE emits buffered prose chunks and run state can be refetched. | Disconnect/reconnect mid-stream; UI recovers text/progress from persisted run/draft. |
| F-10 | P1 | `internal/repository/mongo/client.go:142-151` | Index reconciliation suppresses `IndexKeySpecsConflict` rather than reconciling the existing index to required options. | Incorrect legacy indexes can survive silently. | Compare desired/existing index specs and fail startup with a migration instruction on an incompatible spec. | Seed incompatible index, assert startup fails clearly; seed compatible index, assert idempotent start. |
| F-11 | P1 | `internal/repository/mongo/narrative_events.go:23-94,116-128`; `internal/repository/mongo/client.go:128-133` | Events are ordered by timestamp, no unique `(storyId, seq)` index exists, and `LatestVersion` reads a field this append path does not allocate. | No gap-free canonical order or reliable replay boundary. | Use transactionally allocated per-story sequence numbers and append-only canon events. | Concurrent accepts yield contiguous unique sequence numbers; replay equals projection. |
| F-12 | P1 | `internal/repository/mongo/memories.go:45-73`; `docker-compose.yml` service list | Memory search attempts Mongo Atlas `$vectorSearch`; local Compose Mongo has no Atlas vector index. | Feature silently falls back; semantic-memory behavior cannot be assumed. | Cut semantic memory from v1. | Remove the path and its dependencies; core loop tests remain green with no memory service. |
| F-13 | P1 | `docs/rules.md:67-70`; `internal/agents/orchestrator.go:507-512`; `internal/service/generation_job_worker.go:363-403` | Docs say turns serialize and canon waits for accept; implementation parallelizes non-blocking agents and appends narrative events during generation. | Operators and contributors are guided toward unsafe assumptions. | Replace legacy docs in the rewrite with a short, tested contract. | Documentation review is tied to API/acceptance integration tests. |
| F-14 | P2 | `web/src/components/SceneDrawer.tsx:29-31` | List API returns newest first, but UI chooses the final array item as `lastGen`. | Unaccepted prose display may select the oldest generation. | Select first newest generation or explicit current draft. | Fixture with three ordered generations shows newest non-accepted prose. |

## Agent triage

No measured quality evaluation demonstrates that multiple agents outperform one
well-built prompt. The v1 default is therefore to remove agent orchestration.
“Merge” keeps the concrete capability in the single generation pipeline, not the
agent abstraction.

| Old agent | Concrete input → output | Validated? | Verdict | Reason / latency-cost effect |
|---|---|---|---|---|
| Director | Scene/context/turns → JSON-like plan and turn ordering (`internal/agents/director.go`) | Parsing/fallback is local; plan is otherwise trusted by orchestration. | CUT | A single prose prompt can follow a brief; saves an LLM round trip. |
| Character ×N | Character card, state, memories, prior turns → per-character dialogue/action (`internal/agents/character_agent.go:123-201`) | No schema/canon gate on output. | CUT | Per-participant Sonnet calls multiply latency and cost with no demonstrated gain. |
| Narrator | Agent turns and scene context → prose (`internal/agents/narrator.go`) | No independent canonical validation of prose. | MERGE → prose call | Keep one prose generation call; removes upstream turn calls. |
| Editor | Narrator prose → revised prose (`internal/agents/editor.go`) | Output is used as prose, not structurally validated. | CUT | Cosmetic second pass is not required for the accept/reject loop. |
| CanonGuard | Prose/canon/state → validation result (`internal/agents/canon_guard.go`) | Some JSON/rule checks, but no transactional acceptance gate. | MERGE → pure validators | Retain deterministic validators after structured extraction; avoid a dedicated agent call in v1. |
| Critic | Generated prose → scores/critique (`internal/agents/critic.go`) | Scores are informational only. | CUT | No effect on correctness or acceptance. |
| StateExtract | Prose + roster → state deltas (`internal/agents/state_extractor.go`) | LLM structured extraction, but current write path is unsafe and loses mapping. | MERGE → extraction step | Necessary only as proposed events after prose; must be schema-validated and deferred to accept. |
| World | Context/turns → world-consistency observations (`internal/agents/world.go`) | Advisory output. | CUT | No v1 correctness proof; removes optional call. |
| Arc | Context/turns → arc/plot observations (`internal/agents/arc.go`) | Advisory output. | CUT | Out of core loop; removes optional call. |
| Memory | Characters, memories, turns → memory suggestions (`internal/agents/memory_agent.go`) | Suggestions are not a canon gate. | CUT | Semantic memory is explicitly out of v1; removes vector and LLM costs. |

## Smallest v1 scope

1. One Go modular monolith and Mongo replica set; no Java sidecar, Elasticsearch,
   MCP, semantic memory, multi-story features, or multi-agent system.
2. Authenticated owner-scoped CRUD for story, scenes, DAG edges, and immutable
   character versions.
3. `POST generate` creates/deduplicates a draft from a pinned canon snapshot.
   One prose call streams chunks to the UI; one structured extraction call
   produces proposed canon events.
4. Pure validators mark the draft ready or needs-review. Generate cannot mutate
   canon, state, scene prose, timeline, summaries, or memories.
5. Accept validates staleness and performs one idempotent Mongo transaction that
   appends gap-free canon events, updates projections, and marks the scene
   accepted. Reject deletes the draft only.
6. Persist runs/steps and then add a fenced, Mongo-backed durable queue only when
   moving generation off the request path in Slice 4.

## Verified facts vs. assumptions

### Verified

- The exact source paths and line evidence in this report were inspected.
- The non-integration Go test suite and frontend production build passed in this
  workspace.
- Docker was unavailable; tagged integration tests did not run.
- There is no rejection route, auth middleware, Mongo transaction in accept, or
  prose-token SSE payload in the inspected source.

### Assumptions requiring a v2 test before treating them as facts

- The existing basic CRUD and graph endpoints work against a live MongoDB;
  source and tagged tests support intent but were not executed against Mongo.
- Any agent improves prose quality or canon consistency; no evaluation set or
  measured comparison was found.
- The Java analysis service, Elasticsearch path, headroom proxy, and real LLM
  providers operate correctly in deployment; they were not started or called.
- The exact runtime consequence of F-06 depends on whether an uninspected
  alternate agent path is selected; the normal worker path shown cannot advance
  a draft scene to generated.

## Stop point

Phase 0 is complete. No v2 application code was written. Wait for an explicit
`go` before creating a separate empty v2 workspace and beginning Slice 1.
