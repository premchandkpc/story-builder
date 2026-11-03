# Story Builder v2: limited implementation plan

## Product boundary

V2 has one job: a writer draws a directed acyclic graph of scenes, generates a
prose draft for one scene, and explicitly accepts or rejects it. A rejected
draft changes nothing. An accepted draft is the only operation that changes
canon and character state.

This is a single-user product for the first release. It has no collaboration,
organizations, roles, social features, multiple universes, branch merging,
export formats, semantic memory, vector database, autonomous agents, Java
sidecar, Elasticsearch, Redis, MCP server, cost dashboard, or durable job
queue.

## Why this plan is smaller

The legacy system tried to make generation, extraction, state, validation,
memory, timeline, observability, and ten agents progress at once. That made
Generate a state-changing operation and made acceptance impossible to reason
about. V2 starts with one owner, one Mongo database, one Go binary, one React
application, one prose call, and explicit acceptance.

## Design rules

1. **Generate is pure with respect to canon.** It may create/update a draft and
   run record only.
2. **Accept is atomic and idempotent.** It performs one Mongo transaction,
   verifies the draft still matches the story's canon version, appends canonical
   events, updates projections, and records the accepted scene.
3. **Reject deletes only the draft.** No state, event, summary, or scene write.
4. **Stable IDs are identity.** Names are display-only. Character definitions
   are immutable versions identified by `(character_id, version)`.
5. **Canon facts and character state are append-only events.** Current state is
   a deterministic projection; replay must reproduce it.
6. **LLM output is untrusted.** The gateway validates structured extraction,
   then pure Go validators run before a draft becomes ready. Nothing becomes
   canon without a user Accept.
7. **No speculative abstraction.** Interfaces exist only at external boundaries:
   Mongo repository, LLM provider, clock, and ID generator.

## Initial technical shape

```
web/ (React, TypeScript, React Flow, TanStack Query)
    │ HTTP + SSE
cmd/server/ (one Go binary, chi)
    │
internal/
  story/      scenes, DAG rules, accept/reject use cases
  canon/      append-only events and deterministic projections
  draft/      draft lifecycle and generation orchestration
  llm/        provider gateway, prompts, schema validation
  mongo/      Mongo adapters and index reconciliation
  httpapi/    request/response DTOs and handlers
```

MongoDB is the only required service. A replica set is required before the
Accept transaction is introduced. Redis is not planned for the initial release.

## Deliberately simple generation protocol

The first working vertical slice uses a request-scoped SSE generation stream:

1. `POST /api/v1/scenes/{id}/generate` creates or returns a deduplicated draft
   at the current canon version and streams prose chunks in the response.
2. The server buffers the completed prose and performs extraction/validation.
3. The completed draft is persisted as `READY` or `NEEDS_REVIEW`.
4. `POST /api/v1/drafts/{id}/accept` accepts it atomically, or returns `409`
   when its base canon version is stale.
5. `POST /api/v1/drafts/{id}/reject` deletes it.

This intentionally avoids a durable queue in the first release. If a client
disconnects, the request context cancels the LLM call and the incomplete draft
is marked failed. Persistent job leases are a later, separately justified
addition—not a prerequisite for a correct accept/reject loop.

## Data model for the first release

| Collection | Required fields | Notes |
|---|---|---|
| `stories` | `id`, `title`, `canon_version`, `created_at` | Plain CRUD. |
| `scenes` | `id`, `story_id`, `title`, `brief`, `position`, `accepted_draft_id` | Plain CRUD plus accepted draft reference. |
| `edges` | `story_id`, `from_scene_id`, `to_scene_id` | Validated DAG. |
| `characters` | `character_id`, `version`, `story_id`, `name`, `traits` | Immutable definitions. |
| `drafts` | `id`, `story_id`, `scene_id`, `status`, `prose`, `proposed_events`, `base_canon_version`, `context_hash`, `expires_at` | TTL index; never canonical. |
| `canon_events` | `story_id`, `seq`, `type`, `payload`, `scene_id`, `draft_id`, `at` | Unique `(story_id, seq)`; append-only. |
| `character_state` | `story_id`, `character_id`, `as_of_canon_version`, state fields | Rebuildable projection. |

## Delivery slices

### Slice 1 — graph workspace

- Go server with health/readiness endpoint and Mongo lifecycle.
- Story, scene, and edge CRUD; cycle detection and topological sort.
- React Flow workspace for creating, connecting, moving, and refreshing scenes.
- Unit tests for graph invariants; Mongo integration test for persistence.

Exit condition: a graph survives refresh and a cycle gets `422`.

### Slice 2 — draft loop

- Fake LLM and one provider adapter behind one gateway.
- Request-scoped prose SSE, drafts with TTL, context-hash deduplication.
- Draft view with streamed prose, explicit Accept and Reject actions.
- Atomic, idempotent accept and stale-draft detection.

Exit condition: reject leaves the relevant collections unchanged; a repeated
accept is a no-op; stale accept returns `409`; prose streams visibly in the UI.

### Slice 3 — characters and canon

- Immutable character versions.
- Structured extraction into proposed events; schema validation plus pure Go
  validators.
- Event append and character-state projection inside Accept.
- Replay test proving events reproduce the stored projection.

Exit condition: renaming a character cannot break state identity, and replay
matches the live projection exactly.

### Slice 4 — hardening only when warranted

- Authentication/owner isolation before external deployment.
- Request limits, logging, metrics, OpenTelemetry, Docker, CI.
- Add a durable job queue only if request-scoped SSE has a demonstrated product
  limitation; it must use fencing and idempotency from day one.

## Explicit exclusions

- No agent orchestration. A single prose prompt is the starting point.
- No memory system or vector search.
- No summaries, timeline explorer, critic, world model, or arc model.
- No Java service, Elasticsearch, Redis, Kafka, Postgres, or MCP server.
- No migration of legacy data. Legacy data has no trustworthy acceptance
  boundary, so v2 starts with an empty database.

## Verification standard

Every slice must run its own tests before the next begins. The core end-to-end
tests are:

1. Generate then reject: byte-for-byte unchanged canon/event/state data.
2. Generate then accept twice: one canonical effect.
3. Generate two drafts from the same scene, accept one, then reject stale
   acceptance of the other.
4. Add edges concurrently and never permit a cycle.
5. Replay canon events and match the current character-state projection.
