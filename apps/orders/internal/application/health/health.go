package health

import "context"

type Dependency interface {
	Name() string
	Check(context.Context) error
}

type Status struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type Service struct {
	dependencies []Dependency
}

func NewService(dependencies ...Dependency) *Service {
	return &Service{dependencies: append([]Dependency(nil), dependencies...)}
}

func (s *Service) Live() Status { return Status{Status: "ok"} }

func (s *Service) Ready(ctx context.Context) Status {
	result := Status{Status: "ready", Dependencies: make(map[string]string, len(s.dependencies))}
	for _, dependency := range s.dependencies {
		if dependency == nil {
			continue
		}
		if err := dependency.Check(ctx); err != nil {
			result.Status = "not_ready"
			result.Dependencies[dependency.Name()] = "unavailable"
			continue
		}
		result.Dependencies[dependency.Name()] = "ok"
	}
	return result
}
