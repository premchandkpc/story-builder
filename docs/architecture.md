# Architecture — Slice 1

V2 is a modular Go monolith and React single-page application. Slice 1 has no
canon, queue, or authentication subsystem.

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
lookup and unique edge indexes at startup. Set `MONGO_URI` and optionally
`MONGO_DB` to use it; otherwise the server deliberately reports that it is
using in-memory persistence.

## Draft boundary — Slice 2

`story.Draft` is a temporary document with a 24-hour TTL and a pinned
`base_canon_version`. Generation sends prose chunks through the `ProseGenerator`
port and only writes the draft. It cannot update a scene or a story.

Accept is implemented by the repository adapter: the in-memory adapter holds
one mutex; Mongo runs a transaction that verifies the pinned version, stores
the accepted prose, increments `canonVersion`, and marks the draft accepted.
Reject deletes the draft. Canon events and character projections remain Slice 3.

## Canon events — Slice 3

Characters use stable `characterId` values and immutable version documents.
The deterministic local extractor creates proposed `character_state` events on
the draft. Accept appends them as gap-free per-story `canon_events`; current
character state is a pure replay projection (`ProjectCharacterStates`).

## Graph invariant

`story.Service.CreateEdge` confirms that both scenes belong to the requested
story, rejects duplicates and self-edges, then runs depth-first cycle detection
against the candidate graph before writing. A cycle always maps to HTTP 422.
