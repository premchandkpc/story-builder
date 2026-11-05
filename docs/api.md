# API — Slice 1

All responses are JSON. Error bodies have `{ "code", "message" }`.

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/healthz` | Liveness check. |
| GET | `/api/v1/stories` | List story maps. |
| POST | `/api/v1/stories` | Create a story with `{ "title" }`. |
| GET | `/api/v1/stories/{storyID}/graph` | Read story, scenes, and edges together. |
| POST | `/api/v1/stories/{storyID}/scenes` | Create `{ "title", "brief?", "position": { "x", "y" } }`. |
| PATCH | `/api/v1/scenes/{sceneID}` | Update any supplied `title`, `brief`, or `position`. |
| POST | `/api/v1/stories/{storyID}/edges` | Create `{ "from_scene_id", "to_scene_id" }`. Returns 422 if cyclic. |
| POST | `/api/v1/scenes/{sceneID}/generate` | Streams a new or deduplicated draft as SSE. |
| POST | `/api/v1/drafts/{draftID}/accept` | Accept a ready draft; returns 409 for a stale draft. |
| POST | `/api/v1/drafts/{draftID}/reject` | Delete a draft; returns 204. |
| GET/POST | `/api/v1/stories/{storyID}/characters` | List or create immutable character version 1 definitions. |
| GET | `/api/v1/stories/{storyID}/character-states` | Replay canonical events into current character state. |

Generation uses OpenAI's Responses API. The API key remains on the server.

### Full-length OpenAI generation

`GET /api/v1/generation` returns mode (`openai` or the test-only `template`),
`minimumWords`, `configured`, and `model`. It never returns credentials.
Set `OPENAI_API_KEY` and optionally `OPENAI_MODEL` (default `gpt-4.1`).
`SCENE_WORDS` chooses a 100–5000 word minimum (default 1000).
Without the key, `configured` is false and the UI disables generation.

Generation uses the existing SSE draft lifecycle, allowing up to three capped
OpenAI calls to reach the minimum. Failure, refusal, truncation, interrupted
streams, and insufficient prose do not produce ready drafts. OpenAI settings
use `store: false` and an output cap of `3 * SCENE_WORDS + 1024` per call.
No prose becomes canonical before explicit acceptance. Ollama is not used.
