package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/premchand/story-builder/internal/domain"
	"github.com/premchand/story-builder/internal/repository"
)

type WorkerConfig struct {
	JobRepo        repository.JobRepository
	GenRepo        repository.GenerationRepository
	Recorder       *RunRecorder
	Pipelines      []*PipelineDef
	PollInterval   time.Duration
	LeaseTime      time.Duration
	HeartbeatInterval time.Duration
	MaxConcurrency int
	WorkerID       string
}

type Worker struct {
	cfg    WorkerConfig
	sem    chan struct{}
	wg     sync.WaitGroup
	stopCh chan struct{}
	inFlight sync.Map
}

func NewWorker(cfg WorkerConfig) *Worker {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.LeaseTime == 0 {
		cfg.LeaseTime = 5 * time.Minute
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	if cfg.MaxConcurrency == 0 {
		cfg.MaxConcurrency = 3
	}
	if cfg.WorkerID == "" {
		cfg.WorkerID = fmt.Sprintf("worker_%d", time.Now().UnixMilli())
	}
	return &Worker{
		cfg:    cfg,
		sem:    make(chan struct{}, cfg.MaxConcurrency),
		stopCh: make(chan struct{}),
	}
}

func (w *Worker) Start() {
	w.recoverStuckJobs()
	w.wg.Add(1)
	go w.loop()
	slog.Info("orchestration worker started",
		"workerId", w.cfg.WorkerID,
		"pollInterval", w.cfg.PollInterval,
		"maxConcurrency", w.cfg.MaxConcurrency,
		"pipelines", len(w.cfg.Pipelines),
	)
}

func (w *Worker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	slog.Info("orchestration worker stopped", "workerId", w.cfg.WorkerID)
}

func (w *Worker) loop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.stopCh:
			return
		case <-time.After(w.cfg.PollInterval):
			w.pollOnce()
		}
	}
}

func (w *Worker) recoverStuckJobs() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stuck, err := w.cfg.JobRepo.ListStuck(ctx, w.cfg.LeaseTime*2)
	if err != nil {
		slog.Warn("failed to list stuck jobs", "error", err)
		return
	}
	for _, job := range stuck {
		if job.HeartbeatAt != nil && time.Since(*job.HeartbeatAt) < w.cfg.LeaseTime {
			continue
		}
		job.Status = domain.JobStatusFailed
		job.Error = "stuck (worker restart or crash)"
		if err := w.cfg.JobRepo.Update(ctx, job); err != nil {
			slog.Error("failed to mark stuck job as failed", "jobId", job.ID, "error", err)
		} else {
			slog.Warn("marked stuck job as failed", "jobId", job.ID, "genId", job.GenID)
		}
		if job.GenID != "" && w.cfg.GenRepo != nil {
			if gen, err := w.cfg.GenRepo.Get(ctx, job.GenID); err == nil && gen != nil && gen.Status == domain.GenStatusRunning {
				gen.Status = domain.GenStatusPending
				_ = w.cfg.GenRepo.Update(ctx, gen)
				slog.Warn("reset generation status to pending", "genId", job.GenID)
			}
		}
	}
}

func (w *Worker) pollOnce() {
	for _, pipe := range w.cfg.Pipelines {
		select {
		case <-w.stopCh:
			return
		case w.sem <- struct{}{}:
			go w.tryProcess(pipe)
		default:
			return
		}
	}
}

func (w *Worker) tryProcess(pipe *PipelineDef) {
	defer func() { <-w.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	job, err := w.cfg.JobRepo.PickPending(ctx, pipe.JobType, w.cfg.LeaseTime, w.cfg.WorkerID)
	if err != nil {
		slog.Error("failed to pick pending job", "pipeline", pipe.Name, "error", err)
		return
	}
	if job == nil {
		return
	}

	if _, loaded := w.inFlight.LoadOrStore(job.SceneID, true); loaded {
		slog.Warn("already processing scene, skipping job", "sceneId", job.SceneID, "jobId", job.ID)
		w.failJob(ctx, job, "already processing scene")
		return
	}
	defer w.inFlight.Delete(job.SceneID)

	pCtx, pCancel := context.WithTimeout(context.Background(), w.cfg.LeaseTime)
	defer pCancel()

	hbCtx, hbCancel := context.WithCancel(context.Background())
	defer hbCancel()
	go w.heartbeatLoop(hbCtx, job.ID)

	w.executePipeline(pCtx, pipe, job)

	job.Status = domain.JobStatusDone
	_ = w.cfg.JobRepo.Update(context.Background(), job)
}

func (w *Worker) heartbeatLoop(ctx context.Context, jobID string) {
	ticker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.cfg.JobRepo.Heartbeat(ctx, jobID, w.cfg.LeaseTime); err != nil {
				slog.Warn("heartbeat failed", "jobId", jobID, "error", err)
			}
		}
	}
}

func (w *Worker) executePipeline(ctx context.Context, pipe *PipelineDef, job *domain.Job) {
	slog.Info("pipeline executing",
		"pipeline", pipe.Name,
		"storyId", job.StoryID,
		"jobId", job.ID,
		"sceneId", job.SceneID,
		"genId", job.GenID,
	)

	run, err := w.cfg.Recorder.CreateRun(ctx, job.StoryID, job.SceneID, job.GenID, pipe.RunType)
	if err != nil {
		slog.Error("failed to create run", "jobId", job.ID, "error", err)
		w.failJob(ctx, job, fmt.Sprintf("create run: %v", err))
		return
	}

	job.RunID = run.ID
	_ = w.cfg.JobRepo.Update(ctx, job)

	criticalFailed := false
	anyFailed := false

	for _, step := range pipe.Steps {
		select {
		case <-ctx.Done():
			slog.Warn("pipeline cancelled by context", "jobId", job.ID, "step", step.Name)
			_ = w.cfg.Recorder.CompleteRun(ctx, run.ID, domain.RunStatusFailed, step.Name, "pipeline cancelled")
			return
		default:
		}

		stepCtx := &StepContext{
			StoryID:   job.StoryID,
			SceneID:   job.SceneID,
			GenID:     job.GenID,
			RunID:     run.ID,
			JobID:     job.ID,
			StepName:  step.Name,
			Artifacts: make(map[string]any),
		}

		_ = w.cfg.Recorder.UpdateRunStep(ctx, run.ID, step.Name, domain.StepStatusRunning)
		stepStart := time.Now()
		result := w.runStep(ctx, step, stepCtx)
		recordOpts := StepRecordOptions{
			Status:     result.Status,
			StartedAt:  timePtr(stepStart),
			FinishedAt: timePtr(stepStart.Add(result.Duration)),
			Error:      result.Error,
		}
		_ = w.cfg.Recorder.RecordStep(ctx, run.ID, step.Name, recordOpts)

		if result.Failed() {
			anyFailed = true
			if step.Critical {
				criticalFailed = true
				slog.Error("critical step failed, aborting pipeline",
					"pipeline", pipe.Name, "storyId", job.StoryID,
					"jobId", job.ID, "step", step.Name,
					"duration", result.Duration, "error", result.Error)
				_ = w.cfg.Recorder.CompleteRun(ctx, run.ID, domain.RunStatusFailed, step.Name, result.Error)
				w.failJob(ctx, job, result.Error)
				return
			}
			slog.Warn("non-critical step failed, continuing",
				"pipeline", pipe.Name, "storyId", job.StoryID,
				"jobId", job.ID, "step", step.Name,
				"duration", result.Duration, "error", result.Error)
		}
	}

	slog.Info("pipeline finished",
		"pipeline", pipe.Name, "storyId", job.StoryID,
		"jobId", job.ID, "sceneId", job.SceneID,
		"criticalFailed", criticalFailed, "anyFailed", anyFailed,
	)

	switch {
	case criticalFailed:
		_ = w.cfg.Recorder.CompleteRun(ctx, run.ID, domain.RunStatusFailed, "", "critical step failed")
		w.failJob(ctx, job, "critical step failed")
	case anyFailed:
		_ = w.cfg.Recorder.CompleteRun(ctx, run.ID, domain.RunStatusPartial, "", "")
	default:
		_ = w.cfg.Recorder.CompleteRun(ctx, run.ID, domain.RunStatusCompleted, "", "")
	}
}

func (w *Worker) runStep(ctx context.Context, step StepDef, stepCtx *StepContext) StepResult {
	result := StepResult{StepName: step.Name}
	start := time.Now()

	maxRetries := step.MaxRetries
	if maxRetries == 0 && step.Critical {
		maxRetries = 3
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Info("retrying step", "step", step.Name, "attempt", attempt)
		}

		stepTimeout := step.Timeout
		if stepTimeout == 0 {
			stepTimeout = 5 * time.Minute
		}
		sCtx, sCancel := context.WithTimeout(ctx, stepTimeout)

		lastErr = step.Run(sCtx, stepCtx)
		sCancel()

		if lastErr == nil {
			result.Status = "done"
			result.Duration = time.Since(start)
			return result
		}

		if ctx.Err() != nil {
			break
		}
	}

	result.Status = "failed"
	result.Error = lastErr.Error()
	result.Duration = time.Since(start)
	return result
}

func (w *Worker) failJob(ctx context.Context, job *domain.Job, errMsg string) {
	job.Status = domain.JobStatusFailed
	job.Error = errMsg
	_ = w.cfg.JobRepo.Update(ctx, job)
}

func timePtr(t time.Time) *time.Time {
	return &t
}
