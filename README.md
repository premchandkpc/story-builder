# Story Builder v2

Story Builder v2 is a small, single-user story graph editor. A writer creates a
DAG of scenes, generates a draft, then accepts or rejects it. Only accepting a
draft changes story canon.

The previous implementation is preserved in [`legacy/`](legacy/). It is
reference material only and is not part of the v2 build.

The scope, architecture, and delivery order are in
[`docs/PLAN.md`](docs/PLAN.md).

## Slice 1 quick start

```bash
make run       # API at http://localhost:8080 (in-memory unless MONGO_URI is set)
make web       # workspace at http://localhost:5173
```

For Mongo persistence, set `MONGO_URI` and optionally `MONGO_DB` before
starting the server. The current API is documented in [`docs/API.md`](docs/API.md).

## Deployment checks

`docker compose up --build` starts Mongo as a single-node replica set, which is
required for draft acceptance transactions. Set `AUTH_TOKEN` in deployment and
send it as `Authorization: Bearer <token>` on every API request except health.
Leaving it unset is intentionally development-only behavior.

## Full-length scenes with OpenAI

Set the API key in the server environment (never in browser code), then run:

```bash
export OPENAI_API_KEY='your-api-key'
export OPENAI_MODEL='gpt-4.1'
SCENE_WORDS=1000 make run
```

`OPENAI_MODEL` defaults to `gpt-4.1`; choose another supported Responses model
by setting that environment variable. `SCENE_WORDS` accepts 100–5000 and defaults
to a 1000-word minimum. Set `MONGO_URI` as above to persist the generated story.
Without an API key the workspace remains available and generation is disabled;
it does not fall back to another provider.

The OpenAI Responses API receives the scene brief, story outline, character
names/traits, and accepted prose from incoming parent scenes. Output streams into
an unaccepted draft. The server permits at most three model calls to reach the
minimum, each capped at `3 * SCENE_WORDS + 1024` output tokens. API usage is paid;
short output triggers a continuation rather than an unbounded retry. Requests use
`store: false`. Model identity and scene length are included in the draft cache
key, so old drafts from a different provider are not reused for generation.

The inspector shows the OpenAI model, minimum length, and actual word counts.
Output may exceed the minimum; this is not an exact-word-count editor. Accept
scenes in graph order and regenerate when earlier acceptance changes canon.

See the official [OpenAI streaming documentation](https://developers.openai.com/api/docs/guides/streaming-responses).
