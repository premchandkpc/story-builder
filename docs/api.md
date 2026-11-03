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

Generation, drafts, Accept, and Reject are intentionally absent until Slice 2.
