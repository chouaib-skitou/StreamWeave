package health

import (
	"context"
	"errors"
	"testing"
)

type dependency struct {
	name string
	err  error
}

func (d dependency) Name() string                { return d.name }
func (d dependency) Check(context.Context) error { return d.err }

func TestHealthStatuses(t *testing.T) {
	service := NewService(dependency{name: "database"}, dependency{name: "kafka", err: errors.New("down")})
	if got := service.Live(); got.Status != "ok" || got.Dependencies != nil {
		t.Fatalf("live=%+v", got)
	}
	got := service.Ready(context.Background())
	if got.Status != "not_ready" || got.Dependencies["database"] != "ok" || got.Dependencies["kafka"] != "unavailable" {
		t.Fatalf("ready=%+v", got)
	}
	if got := NewService().Ready(context.Background()); got.Status != "ready" {
		t.Fatalf("empty ready=%+v", got)
	}
}
