# ADR 0001: Baseline guardrails for Phase 0

## Context
The repository had no formal linting, CI, or reusable test entry points, so regressions could slip in during the restructuring work.

## Decision
Add a golangci-lint configuration, Make targets for lint/test/build/simulate, an HTTP smoke characterization suite, and a CI workflow with unit tests on pull requests and integration tests on main.

## Consequences
The repository now has a repeatable local and CI validation path.
New changes are checked by lint, unit tests, and integration coverage.
Future phases can rely on a stable safety net before refactoring.
