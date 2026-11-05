# Local scenario test report

Tested on 4 November 2026 against the v2 application. `legacy/` is excluded as documented in the README.

## Automated checks — passed

- `MONGO_URI='mongodb://localhost:27017/?replicaSet=rs0&directConnection=true' go test -race -count=1 -cover ./...`
- `go vet ./...` and `go build ./...`
- `cd web && npm test` — five tests
- `cd web && npm run build`
- `git diff --check`

The HTTP scenario suite runs against both memory and isolated MongoDB databases. It covers:

- Empty story lists/graphs; missing story/scene/draft IDs; blank titles and character names.
- Short scenes, Tamil/Chinese/emoji input, and 120,000-byte scene briefs without truncation.
- A 20-scene connected story with three characters; acceptance, event sequencing, and replayed character knowledge across all chapters.
- Draft generation deduplication; rejection leaving canon unchanged; acceptance idempotency.
- Parent ordering, stale canon, edited scene text, changed character definitions, expired drafts, failed stream retries, and cached-stream errors.
- Cycles, duplicate edges, self-edges, cross-story edges, and missing endpoints.
- Authentication, public health endpoint, unknown request fields, oversized bodies, and multiple JSON values.

Frontend tests cover empty 204 rejection responses, HTTP errors, fragmented UTF-8 SSE with long prose, streamed errors, and incomplete streams. HTTP handler statement coverage was 90.9%; this is not whole-application coverage.

## Actual running application

The HTTP smoke workflow passed against the restarted server on port 18080 through the frontend proxy on port 5173: story/scene creation, position updates, character creation, generation, rejection, parent ordering, stale drafts, repeat acceptance, and persisted graph reads.

Chrome UI checks completed during this session and the preceding UI pass: story creation, scene creation/editing, streamed generation, rejection, acceptance, and switching stories clearing draft controls. The later native browser capture became unstable; full visual regression, responsive layout, drag/connection gestures, have not been fully verified in the browser. A subsequent Chrome check verified accepted prose is visible and generation is disabled for accepted scenes.

## Fixes

- Empty MongoDB query results return JSON arrays instead of null.
- Character reads for nonexistent stories return 404.
- Request decoding rejects trailing JSON values.
- Acceptance checks expiry and the current scene/character context in both repositories.
- Deduplicated generation propagates client stream errors.
- The frontend handles empty 204 responses and empty scene lists.
- Switching selections cancels generation and clears draft state; late draft/save responses cannot select or clear another scene's draft.
- Accepted prose remains available in the scene inspector, and accepted scenes cannot trigger generation from the UI.

Draft context hashing changed. Existing unaccepted drafts made by an older running server must be regenerated; accepted prose is unaffected.

## Additional review fixes

- Scene updates change editable fields only, preserving canonical prose and accepted-draft references even when the update uses a pre-acceptance snapshot.
- Repeat acceptance stays idempotent after graph changes.
- A failed final draft write releases the active generation slot so retries work; repository contract tests inject this failure against both adapters.
- Accepted drafts cannot be rejected through the API.
- Mutations invalidate the graph identified by their response, avoiding stale caches when the user switches stories during a request.
- React Flow receives node change events for controlled dragging and resets its viewport when changing stories.
- Unsaved scene notes cannot be used accidentally for generation, and position-only refetches preserve the editor's unsaved text.

These checks also passed: `go test -race` with MongoDB, `go vet`, Go build, frontend tests/build, and whitespace validation.

## Product limits

Generation is a deterministic local template, not an AI provider. Long-input and multi-scene storage/streaming tests do not demonstrate long-form AI writing quality. There are no short/long output-length controls. Characters can be created/read through the API; the current UI has no character editor. These features require implementation before they can be tested as UI/product flows.

## Repeat verification

Repeated the complete v2 Go suite with MongoDB and race detection, vet/build,
five frontend tests, TypeScript/Vite build, whitespace validation, and the actual
HTTP runtime workflow. All passed; no new automated failures were found.
Chrome again displayed accepted prose with regeneration disabled and created a
fresh story. The subsequent fresh-scene UI check was interrupted by native
browser errors (`cgWindowNotFound` after an application-change notification),
so a complete fresh browser walkthrough is not claimed for this rerun.

## Long-scene generation

Added an opt-in local Ollama adapter (`OLLAMA_MODEL`, `OLLAMA_URL`,
`SCENE_WORDS`). Its tests cover a 1,000-word minimum reached through a
continuation, narrative context, length-specific cache identity, provider errors,
truncated streams, cancellation, disconnected clients, and bounded short-output
failure. Backend race tests with MongoDB, vet/build, and frontend tests/build
passed. The live endpoint reports local mode and a 1,000-word minimum; the first
real Llama 3.2 example scene completed at 1,438 words. This supersedes the earlier
report's statement that generation is template-only. Character editing remains
API-only.

The complete real local-model example finished: ten scenes, 13,838 words total, per-scene counts [1438, 1376, 1584, 1269, 1449, 1202, 1482, 1292, 1476, 1270]. All ten exceeded 1,000 words. The running MongoDB-backed graph contains ten scenes, nine edges, three characters, and canon version zero (drafts await explicit acceptance). A standalone Markdown copy was saved to `/tmp/the-cartographer-of-lost-dawn.md`.

## OpenAI replacement

The local Ollama adapter was replaced by `OpenAIGenerator` using the OpenAI Responses API. Server configuration now uses `OPENAI_API_KEY` and `OPENAI_MODEL` (default `gpt-4.1`). The UI shows the selected OpenAI model and disables generation when the key is missing. The 1,000-word minimum, bounded continuations, token cap, server-side secret handling, and explicit acceptance remain in effect. Provider-prefixed cache identities prevent old-provider prose being reused as new OpenAI generation.

Mocked OpenAI tests cover authenticated requests, typed SSE, context, continuations, incomplete output, refusal, cancellation, missing keys, 401/429 errors, and long-scene HTTP acceptance. MongoDB-backed race tests, Go vet/build, frontend tests/build, and whitespace checks passed. The live app reports OpenAI mode and an unconfigured credential state. A supplied credential was checked without writing it to disk or printing it and OpenAI returned HTTP 401; real OpenAI story generation was therefore not performed. Historical local-model example results above are not OpenAI results.

A subsequent valid credential authenticated successfully, but the real OpenAI
smoke generation failed with `credit_balance_exhausted`. The provider reported
no remaining API credits. The app now recognizes both nested error events and
response failure events and shows a billing/credits diagnostic without relaying
raw provider messages. New OpenAI-generated large stories remain blocked on
available API credits; authentication success alone does not demonstrate
successful generation.
