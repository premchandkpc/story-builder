# Narrative OS — Aspirational Vision

> **Note:** This document describes the long-term vision for a Narrative Operating System. The current architecture (Story Builder V3) is intentionally simpler: Go + MongoDB + Redis + React Flow. We build narrative intelligence first, infrastructure later. See `docs/adr/0003-simplify-stack-2026.md` for the current infrastructure decisions.

The vision of Universe → World → Timeline → Story → Scenario → Scene → Frame remains the north star for the domain model. But it will be implemented on MongoDB + Redis until measured bottlenecks prove otherwise.

## What Stays

- **Character split:** Definition (immutable) / State (per-scene, event-sourced) / Memory (vector)
- **Prompt layering:** 10-layer compiler with 5 merge strategies ✅ Built
- **DAG engine:** Topological sort, branch detection, cycle prevention ✅ Built
- **Validation engine:** Character/Timeline/Lore/Dialogue validators ✅ Built
- **Timeline engine:** Event-sourced timeline with branch/fork/merge ✅ Built
- **Scene graph:** seq/fork/join/choice/parallel edges ✅ Built

## What Changes

- **Databases:** MongoDB replaces Postgres + pgvector + Qdrant + Neo4j. Redis stays as cache.
- **Queue:** Goroutine workers replace River + Kafka.
- **Evolution path:** Infrastructure is frozen. All effort goes into narrative quality.
- **Culture/Emotion engines:** Built on MongoDB documents, not new databases.

The 12-phase roadmap from ADR 0002 is superseded. Phases 2-5 (prompt compiler, timeline, character state, validation) are built. The remaining phases (culture, emotion, rendering, agents) will use the same MongoDB + Redis stack.

The canonical data model vision (Universe → World → Timeline → Story → Scenario → Scene → Frame) is preserved as a domain evolution, not an infrastructure evolution.
