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
