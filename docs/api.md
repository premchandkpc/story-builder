# API

JSON responses use error bodies `{ "code", "message" }`. Rejection returns an
empty 204; generation uses SSE. With `AUTH_TOKEN` configured, send
`Authorization: Bearer <token>` on every request except health.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/healthz` | Liveness check. |
| GET/POST | `/api/v1/stories` | List or create a story with `{ "title" }`. |
| GET | `/api/v1/stories/{storyID}/graph` | Read story, scenes, and edges. |
| POST | `/api/v1/stories/{storyID}/scenes` | Create `{ "title", "brief?", "position": { "x", "y" } }`. |
| PATCH | `/api/v1/scenes/{sceneID}` | Update supplied `title`, `brief`, or `position`; include `expectedRevision` from the last scene read. A stale revision returns 409. |
| POST | `/api/v1/stories/{storyID}/edges` | Create `{ "from_scene_id", "to_scene_id" }`. Edges mean **must happen before**, not alternative timelines. Cycles return 422. Incoming edges to accepted scenes return 409. |
| POST | `/api/v1/scenes/{sceneID}/generate` | Stream a new or cached draft: `chunk`, then `complete`; `error` is terminal. |
| POST | `/api/v1/drafts/{draftID}/accept` | Atomically accept a ready, unexpired draft with current scene, character, graph, and canon revisions. Stale input returns 409. Retries of accepted drafts are idempotent. |
| POST | `/api/v1/drafts/{draftID}/reject` | Delete READY or FAILED drafts; returns 204. Accepted or actively generating drafts cannot be deleted. |
| GET/POST | `/api/v1/stories/{storyID}/characters` | List latest definitions or create immutable version 1. |
| PUT | `/api/v1/stories/{storyID}/characters/{characterID}` | Append a complete new definition with `expectedVersion`; concurrent edits return 409. |
| GET | `/api/v1/stories/{storyID}/character-states` | Initial knowledge plus replay of accepted character events. Author secrets are excluded from knowledge. |

Accepted scene titles and briefs are locked. Layout remains editable. Revisions
are new scenes; this API does not rewrite existing canon or provide branch-local
canon. The shared bearer token is for a single-user installation, not multi-user
identity or owner isolation.

## Character definition example

```json
{
  "name": "Mira",
  "traits": ["observant", "guarded"],
  "role": "investigator",
  "pronouns": "she/her",
  "description": "A meticulous investigator with ink-stained hands.",
  "backstory": "Her brother vanished while charting the old city.",
  "motivation": "Find her brother without betraying his trust.",
  "goals": ["Find the archive key"],
  "fears": ["Learning that he left voluntarily"],
  "flaws": ["Confuses secrecy with protection"],
  "values": ["Loyalty", "Evidence"],
  "voice": "Short precise sentences; asks questions instead of making accusations.",
  "constraints": ["Cannot read minds", "Does not know who holds the key"],
  "secrets": ["Her brother has the key"],
  "initialKnowledge": ["The archive door is locked"],
  "relationships": []
}
```

Relationships contain `{ "characterId", "kind", "description" }`. Targets must
be other characters in the same story; duplicate targets are rejected. On PUT,
include `expectedVersion` (e.g. 1 to create version 2). Lists permit up to 32
nonblank items of at most 2,000 bytes each; name is 1–200 bytes; descriptive text
fields permit 8,000 bytes. Definitions are author facts, separate from inferred
runtime emotions and acquired knowledge. Changing a definition invalidates
unaccepted drafts while their original snapshots remain preserved.

Generation uses a deterministic local template, with no paid LLM or fact
extractor enabled. Providers receive a pinned `GenerationContext` containing
scene, direct parents and prose, character definitions and state, and revision
identifiers. `Version()` must change with model/prompt/configuration semantics.
Providers must honor cancellation and enforce an output-token cap before any
paid adapter is enabled. The service also bounds requests to two minutes and
stored output to 1 MiB. Snapshots can contain author secrets: future providers
must distinguish author context from each character's knowledge.
