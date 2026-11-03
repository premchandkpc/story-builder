# Decisions

## 2026-11-03 — Preserve, do not repair, the legacy system

The former codebase is stored in `legacy/` with its original files intact.
Repairing it would preserve its generation-time canon mutations and unnecessary
system breadth; v2 starts with an empty implementation root.

## 2026-11-03 — No durable queue in the first core loop

Request-scoped SSE keeps cancellation and ownership simple while the product is
proven. A queue will be added only for a demonstrated need and must include
idempotency and fencing, rather than inheriting the legacy lease behavior.

## 2026-11-03 — Lock accepted scenes and accept in graph order

V1 does not rewrite append-only canon. Once a scene has an accepted draft it
cannot be regenerated; a revision is represented by a new scene. A scene may
be accepted only when all incoming graph parents are accepted, so canonical
sequence is topologically meaningful.

## 2026-11-03 — Request-scoped streaming trade-offs

The browser uses `fetch` plus a stream reader for the POST generation endpoint;
`EventSource` cannot issue this POST. A refresh or disconnect cancels the
request, marks its incomplete draft failed, and wastes any tokens already used.
This is accepted for v1. Generation is limited to one active request per story
in the process. The current deterministic generator has fixed output; any paid
provider adapter must receive a per-call output-token cap before it is enabled.

## 2026-11-03 — Local-only until authentication

The server binds to `127.0.0.1` by default. It must not be externally deployed
until Slice 4 has authentication and owner isolation.
