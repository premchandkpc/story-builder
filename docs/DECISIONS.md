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

## 2026-11-04 — Protect canon across all authoring operations

Edges mean prerequisites in one shared timeline. New incoming edges to an
accepted scene are rejected. Accepted narrative fields are locked; layout is
still editable. Scene updates use field updates plus an optimistic revision
check, so an older editor write cannot erase acceptance. Content revisions are
separate from layout revisions. Acceptance transactionally validates scene,
graph, character, and canon revisions, and checks expiry independently of TTL.
Accepted drafts are retained without expiry; they cannot be rejected.

## 2026-11-04 — Version character definitions and pin generation input

Identity, backstory, motivations, voice, constraints, secrets, initial knowledge,
and relationships are immutable definition versions. Edits append using an
expected version. Latest definitions feed future generation; stored draft
snapshots preserve historical inputs. Initial knowledge is an author-owned
baseline; secrets do not automatically become character knowledge. The local
template proposes no character events instead of fabricating emotions or facts.
An LLM extractor and validated event proposal/review remain future work.

## 2026-11-04 — Recover request-scoped generation without a queue

Overlapping requests in a process fail promptly rather than waiting on a mutex.
A two-minute request deadline and three-minute persisted lease allow abandoned
generations to be marked failed on a later generation attempt. Completion is a
conditional GENERATING-to-READY transition; failed or recovered owners cannot
promote an old draft. Every post-creation failure attempts bounded cleanup.

## 2026-11-04 — Authenticated local deployment

Compose requires AUTH_TOKEN, exposes only the API on loopback, keeps Mongo
private, and persists its data in a named volume. Non-loopback API listeners
refuse startup without a token. The browser supplies the token on JSON and SSE
requests and retains it only for the browser session. Multi-user authentication
and owner isolation are still unimplemented; this is a single-user application.
