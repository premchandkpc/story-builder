# Architecture — Slice 1

V2 is a modular Go monolith and React single-page application. The current implementation includes draft acceptance, canon events, versioned
character definitions, and optional single-user bearer authentication. It has no
durable job queue or multi-user ownership subsystem.

```
React Flow workspace → /api/v1 (chi) → story.Service → story.Repository
                                                       ├─ memory.Repository
                                                       └─ mongo.Repository
```

`internal/story` owns `Story`, `Scene`, `Edge`, graph validation, and the
repository port. It imports neither HTTP nor MongoDB.

`internal/httpapi` owns JSON DTO handling and maps domain errors to HTTP status
codes. `internal/memory` makes the graph workspace runnable without services.
It is for local development only: a restart clears all data.

`internal/mongo` is the production persistence adapter. It creates the scene
lookup, unique edge, active-draft, and unique `(story_id, seq)` canon-event
indexes at startup. Set `MONGO_URI` and optionally
`MONGO_DB` to use it; otherwise the server deliberately reports that it is
using in-memory persistence.

## Draft boundary — Slice 2

`story.Draft` is a temporary document with a 24-hour TTL and a pinned
`base_canon_version`. Generation sends prose chunks through the `ProseGenerator`
port and only writes the draft. It cannot update a scene or a story.

Accept is transactional in the production Mongo adapter: it verifies the pinned
version, stores accepted prose, appends canon events, increments `canonVersion`,
and marks the draft accepted in one transaction. Tests must run the same
acceptance contract against Mongo replica-set storage; the in-memory adapter is
only a fast local implementation, not evidence that Mongo transactions work.
Reject deletes only READY or FAILED drafts. Accepted drafts have no expiry.

## Canon events — Slice 3

Characters use stable `characterId` values and immutable version documents.
The deterministic template proposes no character facts. Slice 3's exit condition requires an LLM
extractor with schema validation and pure Go validation. Accept appends events
as gap-free per-story `canon_events`; when no character event is proposed it
appends a `scene_accepted` event so `canonVersion` always matches the final
event sequence. Current character state is a pure replay projection
(`ProjectCharacterStates`).

## Graph invariant

`story.Service.CreateEdge` confirms that both scenes belong to the requested
story, rejects duplicates and self-edges. The persistence adapter then repeats
cycle detection within its atomic write boundary. Mongo uses a transaction and
a story-document write to force concurrent graph mutations to conflict and
retry against fresh edges. A cycle always maps to HTTP 422.

## Ordering and revision policy

Accepted scenes are locked in v1: revision means creating a new scene, not
overwriting an accepted scene or compensating old canon events. A scene may be
accepted only after every incoming parent scene is accepted. Canon sequence is
therefore a valid topological acceptance order, rather than arbitrary button
press order.

## Concurrency and pinned inputs

Scene updates change only author-owned fields and compare an optimistic
revision; acceptance increments that revision. Content revisions exclude layout
changes. Drafts pin content, graph, character-definition, and canon revisions;
acceptance validates them in the same transaction as canonical writes. Graph
edits and character-version writes also update the story document, serializing
those operations against acceptance. Incoming edges to accepted scenes are
rejected at the persistence boundary.

Character updates append immutable versions. Each generation snapshot includes
latest definitions, initial knowledge plus accepted events, and direct parent
scenes/prose. Character secrets are not copied into character knowledge. A
revision barrier detects definition, graph, or canon changes while assembling
the snapshot. Definition edits affect future context, not historical snapshots.

Generation has a two-minute deadline, a persisted three-minute ownership lease,
and a 1 MiB prose limit. Later generation attempts recover expired owners.
Conditional draft state transitions fence recovered owners from completing.
Process-local admission rejects overlaps promptly; it is not a distributed
one-generation-per-story lock. Mongo's unique scene/context index prevents
duplicate live drafts across processes.
