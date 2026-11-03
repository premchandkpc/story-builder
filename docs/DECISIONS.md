# Decisions

## 2026-11-03 — Preserve, do not repair, the legacy system

The former codebase is stored in `legacy/` with its original files intact.
Repairing it would preserve its generation-time canon mutations and unnecessary
system breadth; v2 starts with an empty implementation root.

## 2026-11-03 — No durable queue in the first core loop

Request-scoped SSE keeps cancellation and ownership simple while the product is
proven. A queue will be added only for a demonstrated need and must include
idempotency and fencing, rather than inheriting the legacy lease behavior.
