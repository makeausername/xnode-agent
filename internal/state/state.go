package state

import "sync"

type Status string

const (
	Uninitialized Status = "uninitialized"
	Enrolling     Status = "enrolling"
	Enrolled      Status = "enrolled"
	Configured    Status = "configured"
	Running       Status = "running"
	Degraded      Status = "degraded"
	Failed        Status = "failed"
	Recovering    Status = "recovering"
	Stopping      Status = "stopping"
)

type Manager struct {
	mu     sync.RWMutex
	status Status
}

func NewManager(initial Status) *Manager {
	return &Manager{status: initial}
}

func (m *Manager) Get() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) Set(status Status) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
}
