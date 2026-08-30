package health

import (
	"context"
	"sync"
)

type Dependency interface {
	Name() string
	Check(context.Context) error
}

type Result struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type Service struct {
	dependencies []Dependency
	mu           sync.RWMutex
	started      bool
}

func NewService(dependencies ...Dependency) *Service {
	return &Service{dependencies: dependencies}
}

func (s *Service) Live() Result {
	return Result{Status: "ok"}
}

func (s *Service) Startup() Result {
	s.mu.RLock()
	started := s.started
	s.mu.RUnlock()
	if !started {
		return Result{Status: "starting"}
	}
	return Result{Status: "ok"}
}

func (s *Service) MarkStarted() {
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
}

func (s *Service) Ready(ctx context.Context) Result {
	result := Result{Status: "ready", Dependencies: make(map[string]string, len(s.dependencies))}
	for _, dependency := range s.dependencies {
		if err := dependency.Check(ctx); err != nil {
			result.Status = "not_ready"
			result.Dependencies[dependency.Name()] = "unavailable"
			continue
		}
		result.Dependencies[dependency.Name()] = "ok"
	}
	return result
}
