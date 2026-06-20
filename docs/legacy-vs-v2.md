# Legacy vs V2: boundary and migration

## Canonical model (V2 — graph-first)

**Everything new should target this model.**

```text
Story
 ├─ Nodes (scenes)         ← the DAG node
 ├─ Edges (scene_edges)    ← directed connections
 ├─ Characters             ← character definitions
 ├─ Locations              ← story settings
 ├─ Generations            ← LLM output per node
 ├─ Memories               ← semantic memory per character
 ├─ Timeline               ← ordered events
 ├─ Summaries              ← multi-level narrative compression
 └─ Bible                  ← generated lore document
```

## Deprecated (V1 — chapter/scene tree)

Still present in the codebase but receiving no new features.

```text
Story
 ├─ Chapters               ← editorial grouping
 │   └─ Scenes             ← narrative beats
 ├─ Characters
 └─ Locations
```

## Migration rules

1. **No new API routes added under chapter paths.** New features go on nodes.
2. **Chapter CRUD endpoints remain but are deprecated.** They return `NotImplemented` for scene sub-routes.
3. **The `Scene` domain type is also the `GraphNode`.** There is no separate node type. The V2 API just re-uses scenes with the V2 transport shape.
4. **Remove legacy UI components** when they have a V2 equivalent.

## API route mapping

| Legacy path | V2 path | Status |
|---|---|---|
| `GET /stories/{id}/chapters` | — | Deprecated |
| `POST /stories/{id}/chapters` | — | Deprecated |
| `GET/PUT/DELETE /chapters/{id}` | — | Deprecated |
| `GET /chapters/{id}/scenes` | `GET /stories/{id}/nodes` | Replaced |
| `POST /chapters/{id}/scenes` | `POST /stories/{id}/nodes` | Replaced |
| — | `GET /stories/{id}/topology` | New (V2 canonical) |

## Stub / experimental features

These are NOT production code. They exist as route stubs returning `NotImplemented` or `EmptyArray`:

- Actors
- Character traits (with assign/unassign)
- Casting (actor → character link)
- Lore (world-building entries)
- Scene turns (interactive turn-by-turn generation)
- Scene structure CRUD on nodes

These will be moved to `/api/v1/experimental/...` or feature-flagged.
