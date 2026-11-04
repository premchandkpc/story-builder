# Story Builder v2

Story Builder v2 is a small, single-user story graph editor. A writer creates a
DAG of scenes, generates a draft, then accepts or rejects it. Only accepting a
draft changes story canon.

The previous implementation is preserved in [`legacy/`](legacy/). It is
reference material only and is not part of the v2 build.

The scope, architecture, and delivery order are in
[`docs/PLAN.md`](docs/PLAN.md).

## Quick start

```bash
make run       # API at http://localhost:8080 (in-memory unless MONGO_URI is set)
make web       # workspace at http://localhost:5173
```

For Mongo persistence, set `MONGO_URI` and optionally `MONGO_DB` before
starting the server. The current API is documented in [`docs/api.md`](docs/api.md).

## Deployment checks

Export `AUTH_TOKEN` before `docker compose up --build`; Compose fails if it is
unset. Open the web workspace with `make web`, enter the token in the left rail,
and click Connect. The API is published only on `127.0.0.1:8080`; Mongo stays on
the private Compose network. A named `mongo-data` volume persists the database.

**Existing Compose data:** before switching an existing installation to the
named volume, back up its running Mongo database and restore into the new
volume. The new named volume starts empty. Keep the original container/volume
until the restored data is verified; do not run `docker compose down -v`.

Compose starts Mongo as a single-node replica set, which is
required for draft acceptance transactions. Set `AUTH_TOKEN` in deployment and
send it as `Authorization: Bearer <token>` on every API request except health.
Without a token, the server permits only explicit loopback listeners for local
development. One shared token does not provide multi-user owner isolation.

## Characters and canon

Use **Characters** to create and revise a character bible: identity, backstory,
motivation, goals, fears, flaws, values, voice, constraints, secrets, initial
knowledge, and relationships. Changes create immutable versions. Secrets remain
author context rather than automatically becoming character knowledge.

Drafts capture generation inputs. Accept rejects drafts when their narrative
inputs changed, locks scene content, and retains the accepted draft permanently.
Layout remains editable. Graph connections mean prerequisites in one timeline.
The generator remains a deterministic local template; no paid LLM is enabled.

## Validation

```bash
go test -race ./...
go vet ./...
(cd web && npm ci && npm test && npm run build)
(cd web && npx playwright install chromium && npm run test:e2e)
MONGO_URI='mongodb://localhost:27017/?replicaSet=rs0' go test -race ./internal/repositorycontract -count=1
```

CI provisions a real Mongo replica set and runs the same invariant contract
against memory and Mongo, in addition to Go and frontend checks.
