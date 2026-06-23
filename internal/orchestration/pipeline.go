package orchestration

import (
	"context"
	"time"
)

type StepDef struct {
	Name       string
	Critical   bool
	MaxRetries int
	Timeout    time.Duration
	Model      string
	Run        func(ctx context.Context, stepCtx *StepContext) error
}

type StepContext struct {
	StoryID   string
	SceneID   string
	GenID     string
	RunID     string
	JobID     string
	StepName  string
	Artifacts map[string]any
}

type PipelineDef struct {
	Name      string
	JobType   string
	RunType   string
	Steps     []StepDef
}

func (p *PipelineDef) StepNames() []string {
	names := make([]string, len(p.Steps))
	for i, s := range p.Steps {
		names[i] = s.Name
	}
	return names
}

type StepResult struct {
	StepName string
	Status   string
	Duration time.Duration
	Error    string
}

func (r *StepResult) Failed() bool {
	return r.Status == "failed"
}

func (r *StepResult) Succeeded() bool {
	return r.Status == "done"
}
