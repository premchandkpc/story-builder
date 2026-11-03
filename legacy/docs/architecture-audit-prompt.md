# MASTER ARCHITECTURE AUDIT & RESTRUCTURING PROMPT

> **Historical — references the old Postgres + Kafka + Qdrant stack. See ADR 0003 for current architecture (MongoDB + Redis only).

Act as:

* Principal Software Architect
* Staff Go Engineer
* Distributed Systems Architect
* MongoDB Architect
* AI Agent Architect
* Narrative Engine Architect
* Production Reliability Engineer
* Test Architecture Lead

Review the ENTIRE repository.

Do NOT provide a shallow review.

Perform a deep architecture audit.

Analyze:

* every folder
* every package
* every service
* every API
* every gRPC contract
* every repository
* every database model
* every workflow
* every event
* every integration point
* every dependency
* every test
* every configuration

Provide a COMPLETE redesign plan.

---

## PHASE 1: REPOSITORY AUDIT

Review:

- cmd/
- internal/
- proto/
- web/
- docs/
- migrations/
- config/
- scripts/
- tests/

For every directory provide:

- Purpose
- Problems
- Code Smells
- Missing Components
- Suggested Structure
- Ownership
- Coupling Issues
- Scalability Risks
- Maintainability Risks

---

## PHASE 2: DOMAIN AUDIT

Review current domain model.

Analyze:

- Story
- Chapter
- Scenario
- Scene
- Character
- Relationship
- Memory
- Timeline
- Prompt
- Localization
- Rendering
- Asset
- Generation

For each domain identify:

- Incorrect Responsibilities
- Missing Entities
- Missing Aggregates
- Missing Value Objects
- Missing Runtime State
- Missing Event Types
- Missing Workflows
- Missing Ownership
- Missing Boundaries

---

## PHASE 3: MONGODB MIGRATION AUDIT

Identify every current SQL table.

For each table determine:

- Remain SQL
- Move To Mongo
- Move To Redis
- Move To Qdrant
- Move To Event Store

Explain why.

Create final matrix.

Example:

- stories → Mongo
- chapters → Mongo
- scene_runtime → Mongo
- character_runtime → Mongo
- timeline → Mongo
- prompt_layers → Mongo
- story_bible → Mongo
- render_metadata → Mongo
- character_memory → Qdrant
- story_memory → Qdrant
- generation_progress → Redis
- users → SQL
- billing → SQL
- audit_logs → SQL

---

## PHASE 4: COLLECTION DESIGN

Generate complete Mongo collections:

- stories
- story_versions
- story_bibles
- chapters
- chapter_plans
- scenes
- scene_plans
- scene_runtime
- scene_results
- scene_templates
- characters
- character_runtime
- character_scene_state
- relationships
- relationship_history
- timeline_events
- timeline_snapshots
- prompt_layers
- prompt_versions
- prompt_experiments
- localization_overlays
- runtime_snapshots
- agent_decisions
- director_decisions
- canon_violations
- workflow_state
- generation_jobs
- generation_metrics
- generation_costs
- event_store
- assets
- render_metadata

For every collection provide:

- Purpose
- Schema
- Indexes
- TTL Indexes
- Shard Keys
- Query Patterns
- Read Patterns
- Write Patterns
- Expected Growth

---

## PHASE 5: SERVICE DECOMPOSITION

Review all services.

Identify:

- God Services
- Overloaded Services
- Wrong Responsibilities
- Missing Services

Create final service map.

Required services:

- story-service
- chapter-service
- scene-service
- character-service
- memory-service
- relationship-service
- timeline-service
- prompt-service
- context-builder-service
- narrative-planner-service
- generation-service
- validator-service
- localization-service
- render-service
- analytics-service
- search-service
- workflow-service

For each service provide:

- Responsibilities
- Owned Collections
- Owned APIs
- Owned Events
- Owned gRPC Contracts
- Dependencies
- Scaling Strategy
- Failure Modes

---

## PHASE 6: API AUDIT

Review every REST endpoint.

Identify:

- Missing APIs
- Incorrect APIs
- Overloaded APIs
- Missing Idempotency
- Missing Pagination
- Missing Filtering
- Missing Search
- Missing Versioning

Generate final API specification.

Include:

- Story APIs
- Chapter APIs
- Scene APIs
- Character APIs
- Memory APIs
- Relationship APIs
- Timeline APIs
- Prompt APIs
- Generation APIs
- Localization APIs
- Render APIs
- Analytics APIs
- Workflow APIs
- Admin APIs

---

## PHASE 7: gRPC AUDIT

Review all protobuf definitions.

Identify:

- Missing Contracts
- Bad Contracts
- Large Payload Problems
- Chatty Service Calls
- Circular Dependencies

Generate new protobuf architecture.

Required services:

- StoryService
- SceneService
- CharacterService
- MemoryService
- RelationshipService
- TimelineService
- ContextBuilderService
- NarrativePlannerService
- GenerationService
- ValidatorService
- RenderService
- AnalyticsService

---

## PHASE 8: WORKFLOW AUDIT

Review generation flow.

- Current Flow
- Problems
- Race Conditions
- Data Consistency Issues
- Missing Retries
- Missing Rollbacks
- Missing Sagas

Generate new workflow.

Scene Generation Workflow:

1. Plan Scene
2. Build Context
3. Retrieve Memories
4. Retrieve Relationships
5. Compile Prompt
6. Generate Scene
7. Validate Scene
8. Store Scene
9. Extract Memories
10. Update Relationships
11. Update Timeline
12. Publish Events
13. Update Metrics
14. Create Snapshot

---

## PHASE 9: MEMORY SYSTEM AUDIT

Review current memory implementation.

Identify:

- Missing Memory Types
- Missing Retrieval Logic
- Missing Ranking
- Missing Summarization
- Missing Compression
- Missing Long-Term Memory
- Missing Episodic Memory
- Missing Semantic Memory

Generate full memory architecture.

---

## PHASE 10: CONTEXT BUILDER AUDIT

Review context assembly.

Identify:

- Missing Data Sources
- Missing Retrieval
- Prompt Bloat
- Context Waste
- Token Waste

Generate Context Builder architecture.

Input:

- Story
- Chapter
- Scene
- Character Runtime
- Relationships
- Timeline
- Memories
- Localization
- Bible

Output:

- Optimized Context

---

## PHASE 11: EVENT AUDIT

Review eventing architecture.

Identify:

- Missing Events
- Missing DLQs
- Missing Replay
- Missing Idempotency
- Missing Event Versioning

Generate final event catalog.

---

## PHASE 12: TEST ARCHITECTURE

Review test coverage.

Identify:

- Missing Unit Tests
- Missing Integration Tests
- Missing E2E Tests
- Missing Contract Tests
- Missing Load Tests
- Missing Chaos Tests

Generate complete strategy.

**Unit Tests**

- Table Driven Tests
- Mocks
- Fakes
- Property Based Tests

**Integration Tests**

- Testcontainers
- Mongo
- Redis
- Kafka
- Qdrant
- gRPC
- REST

**Contract Tests**

- OpenAPI Validation
- Proto Validation
- Consumer Driven Contracts

**BDD Tests**

- Cucumber
- Gherkin
- Story Generation Scenarios
- Memory Retrieval Scenarios
- Relationship Update Scenarios

**Playwright**

- Story Editor
- Character Editor
- Timeline Editor
- Localization Editor
- Scene Graph Editor

**Load Tests**

- k6
- Locust
- Scene Generation
- Memory Retrieval
- Search

**Chaos Tests**

- Kafka Failure
- Mongo Failure
- Redis Failure
- Qdrant Failure
- Generation Failure

---

## PHASE 13: OBSERVABILITY AUDIT

Review logging.

Review metrics.

Review tracing.

Generate:

- OpenTelemetry
- Prometheus
- Grafana
- Tempo
- Loki

Trace every request with:

- story_id
- scene_id
- character_id
- generation_id
- workflow_id

---

## PHASE 14: SECURITY AUDIT

Review:

- Auth
- Authorization
- Tenant Isolation
- Secrets
- API Keys
- Prompt Injection Risks
- LLM Risks
- Data Leakage

Generate security redesign.

---

## PHASE 15: CODE REVIEW

Review every package.

Identify:

- Dead Code
- Duplicate Logic
- Wrong Abstractions
- Cyclic Dependencies
- Long Methods
- Large Interfaces
- Leaky Repositories
- Tight Coupling
- Poor Naming
- Missing Documentation
- Missing Comments
- Missing ADRs
- Missing Design Docs

For each issue provide:

- File
- Line
- Problem
- Impact
- Refactoring Plan
- Example Code

---

## PHASE 16: FINAL OUTPUT

Generate:

1. New Folder Structure
2. New Domain Structure
3. New Service Map
4. New Mongo Collections
5. New APIs
6. New gRPC Contracts
7. New Event Catalog
8. New Workflow Diagrams
9. New Testing Strategy
10. New Deployment Architecture
11. New Observability Architecture
12. New Security Architecture
13. Migration Plan
14. Risk Assessment
15. Technical Debt Report
16. Prioritized Refactoring Roadmap

Prioritize:

- Narrative Planner
- Context Builder
- Memory System
- Relationship Engine
- Timeline Engine
- Character Runtime

over:

- Rendering
- Camera
- Animation

Because story consistency is more important than visual metadata.
