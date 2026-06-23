# Phase 1: Durable Scene/Job Orchestration

## Problem

Current `GenerationJobWorker` (`internal/service/generation_job_worker.go:1-591`) is monolithic. It mixes job claiming, step execution, event publishing, error handling, and status tracking in one file. The `genInFlight sync.Map` dedup is in-memory only — lost on restart. Steps are tracked via `Generation.StepStatus` map on the `generations` collection, not as independent `RunStep` documents. No heartbeat mechanism — stuck jobs recovered only at worker start.

## Target

A reusable `internal/orchestration/` package that decouples job lifecycle from step execution.

## Package Layout

```
internal/orchestration/
  job_queue.go          Claim/release/heartbeat/recover via domain.Job
  run_recorder.go       Create/update StoryRun + RunStep documents
  pipeline.go           Ordered step defs (critical, retries, timeout, model)
  worker.go             Generic poll+lease loop (replace generation_job_worker.go)
```

## Detailed Design

### 1. `job_queue.go` — Durable Job Lifecycle

```go
package orchestration

import (
    "context"
    "time"
    "github.com/premchand/story-builder/internal/domain"
    "github.com/premchand/story-builder/internal/repository"
)

type JobQueueConfig struct {
    JobRepo     repository.JobRepository
    LeaseTime   time.Duration // default 5min
    HeartbeatInterval time.Duration // default 30s
    MaxRetries  int            // default 3
    DeadLetterAfter int        // moves to dead-letter after N failures
}

type JobQueue struct {
    cfg     JobQueueConfig
    stopped chan struct{}
    workerID string
}
```

**Claim semantics** — use Mongo `FindOneAndUpdate` with a version field for optimistic locking:

```go
// In mongo/job_repository.go
func (r *JobRepository) Claim(ctx context.Context, jobType string, leaseTime time.Duration, workerID string) (*domain.Job, error) {
    filter := bson.M{
        "type":           jobType,
        "status":         domain.JobStatusPending,
        "leaseUntil":     nil, // never leased
        "$or": []bson.M{
            {"nextRunAt": nil},
            {"nextRunAt": bson.M{"$lte": time.Now()}},
        },
    }
    update := bson.M{
        "$set": bson.M{
            "status":      domain.JobStatusRunning,
            "leaseUntil":  time.Now().Add(leaseTime),
            "heartbeatAt": time.Now(),
            "updatedAt":   time.Now(),
            "workerId":    workerID,
        },
        "$inc": bson.M{"attempts": 1, "version": 1},
    }
    var job domain.Job
    err := r.coll.FindOneAndUpdate(ctx, filter, update,
        options.FindOneAndUpdate().SetReturnDocument(options.After),
    ).Decode(&job)
    if err == mongo.ErrNoDocuments {
        return nil, nil // no pending job
    }
    return &job, err
}
```

**Re-claim on crash** — expired leases become claimable:

```go
// Expired lease filter — same Claim method, different initial filter
"$or": []bson.M{
    {"leaseUntil": nil},
    {"leaseUntil": bson.M{"$lte": time.Now()}},
},
```

**Heartbeat** — per-worker goroutine extends lease while running:

```go
func (q *JobQueue) heartbeat(ctx context.Context, jobID string) {
    ticker := time.NewTicker(q.cfg.HeartbeatInterval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            q.cfg.JobRepo.Heartbeat(ctx, jobID)
        }
    }
}
```

**Dead-letter** — after N failures, move to dead-letter state:

```go
const JobStatusDeadLetter = "dead_letter"
```

Add `deadLetterReason` field to `domain.Job`.

### 2. `run_recorder.go` — StoryRun + RunStep Creation

```go
type RunRecorder struct {
    RunRepo    repository.RunRepository
    StepRepo   repository.RunStepRepository
}

func (r *RunRecorder) CreateRun(ctx context.Context, storyID, sceneID, runType string) (*domain.StoryRun, error) {
    run := &domain.StoryRun{
        ID:        generateID("run_"),
        StoryID:   storyID,
        SceneID:   sceneID,
        RunType:   runType,
        Status:    domain.RunStatusRunning,
        StartedAt: timePtr(time.Now()),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    return run, r.RunRepo.Create(ctx, run)
}

func (r *RunRecorder) RecordStep(ctx context.Context, runID, stepName string, opts StepRecordOptions) error {
    step := &domain.RunStep{
        ID:         generateID("step_"),
        RunID:      runID,
        StepName:   stepName,
        Status:     opts.Status,
        StartedAt:  opts.StartedAt,
        FinishedAt: opts.FinishedAt,
        PromptHash: opts.PromptHash,
        Model:      opts.Model,
        TokensIn:   opts.TokensIn,
        TokensOut:  opts.TokensOut,
        Error:      opts.Error,
        Artifacts:  opts.Artifacts,
        CreatedAt:  time.Now(),
    }
    return r.StepRepo.Create(ctx, step)
}
```

### 3. `pipeline.go` — Step Definitions

```go
type StepDef struct {
    Name        string                        // "generate", "extract", etc.
    Critical    bool                          // if true, pipeline fails on error
    MaxRetries  int                           // default 3 for critical, 0 for non-critical
    Timeout     time.Duration
    Model       string                        // model tier hint
    Run         func(ctx context.Context, stepCtx *StepContext) error
}

type StepContext struct {
    StoryID    string
    SceneID    string
    GenID      string
    RunID      string
    JobID      string
    StepName   string
    Artifacts  map[string]any // writable — producers store intermediate data
}

type PipelineDef struct {
    Steps []StepDef
}

// Execute runs steps in order, recording RunSteps via RunRecorder
func (p *PipelineDef) Execute(ctx context.Context, recorder *RunRecorder, jobQueue *JobQueue, genCtx *GenerationContext) error
```

**Current pipeline as PipelineDef:**

```go
var GenerateScenePipeline = PipelineDef{
    Steps: []StepDef{
        {
            Name: "generate", Critical: true, MaxRetries: 3,
            Timeout: 4 * time.Minute,
            Run: func(ctx context.Context, sc *StepContext) error {
                return generateScene(ctx, sc.GenID, sc.StoryID, sc.SceneID)
            },
        },
        {
            Name: "extract", Critical: true, MaxRetries: 3,
            Timeout: 2 * time.Minute,
            Run: func(ctx context.Context, sc *StepContext) error {
                return extractState(ctx, sc.GenID, sc.StoryID, sc.SceneID)
            },
        },
        {
            Name: "memory", Critical: false, MaxRetries: 0,
            Timeout: 1 * time.Minute,
            Run: func(ctx context.Context, sc *StepContext) error {
                return updateMemories(ctx, sc.GenID, sc.StoryID, sc.SceneID)
            },
        },
        // ... timeline, summary, validate
    },
}
```

### 4. `worker.go` — Generic Worker Loop

```go
type WorkerConfig struct {
    Queue          *JobQueue
    Pipeline       *PipelineDef
    Recorder       *RunRecorder
    GenRepo        repository.GenerationRepository
    SceneRepo      repository.SceneRepository
    PollInterval   time.Duration // default 2s
    MaxConcurrency int           // default 3
    JobTypes       []string      // which job types this worker handles
}

type Worker struct {
    cfg WorkerConfig
    sem chan struct{} // concurrency limiter
    wg  sync.WaitGroup
    stop chan struct{}
}
```

**Loop:**
1. On start: `recoverStuckJobs()` (existing behavior)
2. Poll loop: `Queue.Claim()` → acquire semaphore → `go processJob()`
3. `processJob()`: create Run → execute pipeline steps → update Run → release semaphore
4. On step failure: check retries, update RunStep error, either retry or fail pipeline

**Stuck job recovery:**
```go
func (w *Worker) recoverStuckJobs() {
    stuck, _ := w.cfg.Queue.JobRepo.ListStuck(ctx, w.cfg.Queue.LeaseTime*2)
    for _, job := range stuck {
        // Check heartbeat age
        if job.HeartbeatAt != nil && time.Since(*job.HeartbeatAt) < w.cfg.Queue.LeaseTime {
            continue // still alive
        }
        // Reset to pending for re-claim
        job.Status = domain.JobStatusPending
        job.LeaseUntil = nil
        job.WorkerID = ""
        _ = w.cfg.Queue.JobRepo.Update(ctx, job)
    }
}
```

### 5. Migration from Current Worker

**Step 1:** Create `internal/orchestration/` with the types above. No behavior changes yet.

**Step 2:** Extract `runPipeline` body into a `PipelineDef` struct. Keep the same `GenerateSceneWorker` etc. but call them through `StepDef.Run`.

**Step 3:** Replace `GenerationJobWorker` with `orchestration.Worker`. The `GenerationJobWorker` struct remains as a thin wrapper that configures the orchestration worker with the existing pipeline steps.

**Step 4:** Add heartbeat goroutine per claimed job. This is the behavioral change — jobs become observable as "running" vs "stuck" in real time.

**Step 5:** Wire `StoryRun` + `RunStep` creation. Currently `GenerationJobWorker` calls `setStepStatus(genID, step, status)`. Replace with `RunRecorder.RecordStep()`.

### 6. API Changes

Add to `internal/repository/interfaces.go`:

```go
type JobRepository interface {
    // ... existing ...
    Heartbeat(ctx context.Context, id string) error
    Claim(ctx context.Context, jobType string, leaseTime time.Duration, workerID string) (*domain.Job, error)
    ListByStatus(ctx context.Context, status string) ([]*domain.Job, error)
    IncrementAttempt(ctx context.Context, id string) error
}
```

Add to `internal/repository/mongo/jobs.go`:

```go
func (r *JobRepository) Heartbeat(ctx context.Context, id string) error {
    _, err := r.coll.UpdateOne(ctx,
        bson.M{"_id": id},
        bson.M{"$set": bson.M{"heartbeatAt": time.Now(), "updatedAt": time.Now()}},
    )
    return err
}
```

Add `WorkerID` and `HeartbeatAt` fields to `domain.Job`:

```go
type Job struct {
    // ... existing ...
    WorkerID     string     `bson:"workerId,omitempty"`
    HeartbeatAt  *time.Time `bson:"heartbeatAt,omitempty"`
    Version      int        `bson:"version"`
}
```

### 7. Collections / Indexes

Add `scene_locks` collection:

```go
// In EnsureIndexes
"scene_locks": {
    {Keys: bson.D{{Key: "sceneId", Value: 1}}, Options: options.Index().SetUnique(true)},
    {Keys: bson.D{{Key: "ttl", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
},
```

Add index on `jobs.heartbeatAt`:

```go
"jobs": {
    // ... existing ...
    {Keys: bson.D{{Key: "status", Value: 1}, {Key: "heartbeatAt", Value: 1}}},
},
```

### 8. Testing

In `internal/orchestration/worker_test.go`:

```go
func TestWorker_ClaimsJob(t *testing.T) {
    // Setup: insert pending job
    // Run: start worker, poll once
    // Assert: job claimed, status=running, leaseUntil set
}

func TestWorker_Heartbeats(t *testing.T) {
    // Setup: insert pending job
    // Run: claim job, wait 2 heartbeats
    // Assert: heartbeatAt updated
}

func TestWorker_RecoversStuckJobs(t *testing.T) {
    // Setup: insert job with expired lease, no heartbeat
    // Run: recoverStuckJobs
    // Assert: job reset to pending
}

func TestWorker_RecordsRunAndSteps(t *testing.T) {
    // Setup: insert pending job + mock pipeline steps
    // Run: worker processes job
    // Assert: StoryRun created, RunSteps created with correct status, tokens, etc.
}
```

### 9. File Changes Summary

| File | Change |
|------|--------|
| `internal/domain/job.go` | Add `WorkerID`, `HeartbeatAt`, `Version` fields |
| `internal/domain/job.go` | Add `JobStatusDeadLetter` constant |
| `internal/repository/interfaces.go` | Add `Heartbeat`, `Claim`, `ListByStatus`, `IncrementAttempt` to JobRepository |
| `internal/repository/mongo/jobs.go` | Implement new methods |
| `internal/repository/mongo/client.go` | Add `scene_locks` indexes |
| `internal/orchestration/` (new) | `job_queue.go`, `run_recorder.go`, `pipeline.go`, `worker.go` |
| `internal/service/generation_job_worker.go` | Refactor to use orchestration.Worker |
| `internal/service/generation.go` | Use scene locks instead of `genInFlight` |
| `cmd/server/init.go` | Wire orchestration.Worker instead of GenerationJobWorker |
| `web/src/components/RunInspector.tsx` | Add cancel button, status filter tabs |
| `web/src/api/hooks.ts` | Add `useCancelRun` mutation, `useRunHeartbeat` |
