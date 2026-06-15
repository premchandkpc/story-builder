package workflow

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[uuid.UUID]*Job
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[uuid.UUID]*Job),
	}
}

func (m *MemoryStore) Create(job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job.ID = uuid.New()
	job.CreatedAt = time.Now()
	m.jobs[job.ID] = job
	return nil
}

func (m *MemoryStore) Get(id uuid.UUID) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return job, nil
}

func (m *MemoryStore) Update(job *Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.jobs[job.ID] = job
	return nil
}

func (m *MemoryStore) List(storyID uuid.UUID, status JobStatus) ([]Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Job
	for _, job := range m.jobs {
		if job.StoryID == storyID && (status == "" || job.Status == status) {
			result = append(result, *job)
		}
	}
	return result, nil
}

func (m *MemoryStore) Cancel(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return ErrNotFound
	}
	job.Status = StatusCancelled
	return nil
}

func (m *MemoryStore) Retry(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return ErrNotFound
	}
	job.Status = StatusPending
	job.CurrentStep = StepPlanner
	job.Attempts = 0
	job.Error = ""
	return nil
}
