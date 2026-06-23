package agents

import "sync"

type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]AgentSpec
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]AgentSpec),
	}
}

func (r *AgentRegistry) Register(spec AgentSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[spec.Name] = spec
}

func (r *AgentRegistry) Get(name string) (AgentSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.agents[name]
	return spec, ok
}

func (r *AgentRegistry) List() []AgentSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	specs := make([]AgentSpec, 0, len(r.agents))
	for _, spec := range r.agents {
		specs = append(specs, spec)
	}
	return specs
}
