# Repository review and enhancement plan

## Overview
This project already has a solid domain-oriented structure with a Go API layer, a React/Vite frontend, and a graph-first storytelling model. The main opportunity was to improve robustness around request handling and reduce repeated validation logic so the server layer becomes easier to extend.

## What was improved
1. Request validation was centralized for story and node handlers.
   - Story titles are now trimmed and rejected when blank.
   - UUID parsing for character references and optional location references now flows through shared helpers.
   - Scene-structure persistence now reports failures instead of silently ignoring them.

2. The backend build was stabilized.
   - The stale import issue in the lore service layer was removed so the Go build completes successfully again.

3. The API surface was made easier to verify locally.
   - A health endpoint was added at `/api/v1/healthz`.
   - The frontend API client now exposes a simple health check helper.

4. The repository gained regression coverage for the new validation helpers.
   - The tests cover invalid UUID lists and blank story titles.

## Files touched
- [internal/api/request_validation.go](../internal/api/request_validation.go)
- [internal/api/handlers_stories.go](../internal/api/handlers_stories.go)
- [internal/api/router.go](../internal/api/router.go)
- [internal/api/request_validation_test.go](../internal/api/request_validation_test.go)
- [web/src/api/client.ts](../web/src/api/client.ts)
- [docs/review-and-enhancements.md](review-and-enhancements.md)

## Recommended next steps
- Introduce a dedicated validation layer for all request payloads rather than handling them inline in handlers.
- Move domain-specific persistence and business rules behind service interfaces with clearer contracts.
- Expand test coverage around story, node, and scene handlers.
- Add end-to-end API smoke tests for the most critical flows.
