# API endpoint status matrix

> Base path: `/api/v1`

## Stories

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/healthz` | ✅ Stable | Always returns `{"status":"ok"}` |
| POST | `/stories` | ✅ Stable | Create story |
| GET | `/stories` | ✅ Stable | List all stories |
| POST | `/stories/generate` | 🟡 Beta | Full story generation from synopsis |
| POST | `/stories/generate-title` | 🟡 Beta | LLM title suggestion |
| GET | `/stories/{id}` | ✅ Stable | Get story |
| PUT | `/stories/{id}` | ✅ Stable | Update story |
| DELETE | `/stories/{id}` | ✅ Stable | Delete story + cascade |

## Graph (V2 canonical)

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/stories/{id}/topology` | ✅ Stable | Nodes + edges + topological order |
| GET | `/stories/{id}/nodes` | ✅ Stable | List all nodes |
| POST | `/stories/{id}/nodes` | ✅ Stable | Create node |
| GET | `/stories/{id}/nodes/{nid}` | ✅ Stable | Get node |
| PUT | `/stories/{id}/nodes/{nid}` | ✅ Stable | Update node |
| DELETE | `/stories/{id}/nodes/{nid}` | ✅ Stable | Delete node |
| POST | `/stories/{id}/edges` | ✅ Stable | Create edge |
| GET | `/stories/{id}/edges` | ✅ Stable | List edges |
| DELETE | `/stories/{id}/edges` | 🟡 Beta | Delete edge by from/to |
| DELETE | `/stories/{id}/edges/{eid}` | 🔴 Planned | Delete edge by ID |

## Generations

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| POST | `/stories/{id}/nodes/{nid}/generate` | 🟡 Beta | Triggers async generation |
| GET | `/stories/{id}/nodes/{nid}/generations` | 🟡 Beta | List generations for node |
| POST | `/stories/{id}/nodes/{nid}/accept` | 🟡 Beta | Accept a generation |
| GET | `/generations/{gid}/status` | 🟡 Beta | Poll generation status |
| GET | `/generations/{gid}/progress` | 🟡 Beta | SSE progress stream |

## Characters

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/characters` | 🟡 Beta | List all characters (V2) |
| POST | `/characters` | 🟡 Beta | Create character (V2) |
| GET | `/characters/{id}` | 🟡 Beta | Get character |
| PUT | `/characters/{id}` | 🟡 Beta | Update character |
| GET | `/characters/{id}/memories` | 🔴 Partial | List character memories |
| POST | `/characters/{id}/memories/search` | 🔴 Partial | Search memories |
| GET | `/characters/{id}/traits` | 🔴 Stub | Returns `[]` |
| POST | `/characters/{id}/traits/assign` | ⛔ Stub | `NotImplemented` |
| DELETE | `/characters/{id}/traits/{tid}` | ⛔ Stub | `NotImplemented` |

## Story-scoped characters

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| POST | `/stories/{id}/characters` | ✅ Stable | Create character in story |
| GET | `/stories/{id}/characters` | ✅ Stable | List story characters |

## Locations

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| POST | `/stories/{id}/locations` | ✅ Stable | Create location |
| GET | `/stories/{id}/locations` | ✅ Stable | List story locations |
| GET | `/locations/{id}` | ✅ Stable | Get location |
| PUT | `/locations/{id}` | ✅ Stable | Update location |

## Bible

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/stories/{id}/bible` | 🟡 Beta | Get story bible |
| POST | `/stories/{id}/bible/generate` | 🟡 Beta | Generate bible via LLM |
| PUT | `/stories/{id}/bible` | 🟡 Beta | Update bible |
| DELETE | `/stories/{id}/bible` | 🟡 Beta | Delete bible |

## Timeline

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| POST | `/stories/{id}/timeline` | 🟡 Beta | Create timeline event |
| GET | `/stories/{id}/timeline` | 🟡 Beta | List timeline events |

## Summaries

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/stories/{id}/summaries/level` | 🟡 Beta | Get summary by level |
| GET | `/stories/{id}/summaries/nodes/{nid}` | 🟡 Beta | Get scene/node summary |
| GET | `/stories/{id}/summaries/count` | ⛔ Stub | `NotImplemented` |
| GET | `/stories/{id}/summaries/elevate` | ⛔ Stub | `NotImplemented` |

## Blueprint

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/stories/{id}/blueprint` | 🟡 Beta | Get story blueprint |
| PUT | `/stories/{id}/blueprint` | 🟡 Beta | Update story blueprint |

## Chapters (deprecated)

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET | `/stories/{id}/chapters` | 🔴 Deprecated | List chapters |
| POST | `/stories/{id}/chapters` | 🔴 Deprecated | Create chapter |
| GET | `/stories/{id}/chapters/{cid}` | 🔴 Deprecated | Get chapter |
| PUT | `/stories/{id}/chapters/{cid}` | 🔴 Deprecated | Update chapter |
| DELETE | `/stories/{id}/chapters/{cid}` | 🔴 Deprecated | Delete chapter |
| GET | `/stories/{id}/chapters/{cid}/scenes` | ⛔ Stub | Unimplemented |

## Scene turns (experimental)

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| PUT | `/stories/{id}/nodes/{nid}/scene/structure` | ⛔ Stub | Unimplemented |
| GET | `/stories/{id}/nodes/{nid}/scene/structure` | ⛔ Stub | Unimplemented |
| POST | `/stories/{id}/nodes/{nid}/scene/start` | ⛔ Stub | Unimplemented |
| POST | `/stories/{id}/nodes/{nid}/scene/next` | ⛔ Stub | Unimplemented |
| POST | `/stories/{id}/nodes/{nid}/scene/finish` | ⛔ Stub | Unimplemented |
| GET | `/stories/{id}/nodes/{nid}/scene/turns` | ⛔ Stub | Unimplemented |

## Stub resources (all experimental)

| Method | Path | Status | Notes |
|--------|------|--------|-------|
| GET/POST | `/actors` | ⛔ Stub | Not implemented |
| GET/PUT | `/actors/{id}` | ⛔ Stub | Not implemented |
| GET/POST | `/character-traits` | ⛔ Stub | Not implemented |
| GET | `/character-traits/{id}` | ⛔ Stub | Not implemented |
| POST | `/character-traits` | ⛔ Stub | Not implemented |
| GET/POST | `/lore` | ⛔ Stub | Not implemented |
| POST | `/lore/search` | ⛔ Stub | Not implemented |
| POST | `/stories/{id}/casting` | ⛔ Stub | Not implemented |
| GET | `/stories/{id}/casting` | ⛔ Stub | Returns `[]` |
| GET | `/casting/actor/{id}` | ⛔ Stub | Not implemented |
| GET | `/casting/character/{id}` | ⛔ Stub | Not implemented |

## Legend

| Symbol | Meaning |
|--------|---------|
| ✅ | Stable, tested, no known issues |
| 🟡 | Beta, functional but may change |
| 🔴 | Partial or deprecated |
| ⛔ | Stub (route exists but returns `NotImplemented` or `[]`) |
