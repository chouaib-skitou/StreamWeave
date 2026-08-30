package health

import (
	"context"
	"errors"
	"testing"
)

type fakeDependency struct {
	name string
	err  error
}

func (f fakeDependency) Name() string                { return f.name }
func (f fakeDependency) Check(context.Context) error { return f.err }

func TestReadyReportsEveryDependency(t *testing.T) {
	service := NewService(fakeDependency{name: "postgresql"}, fakeDependency{name: "redis", err: errors.New("down")})
	result := service.Ready(context.Background())
	if result.Status != "not_ready" || result.Dependencies["postgresql"] != "ok" || result.Dependencies["redis"] != "unavailable" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestStartupTransitionsAfterBootstrap(t *testing.T) {
	service := NewService()
	if service.Startup().Status != "starting" {
		t.Fatal("expected starting status")
	}
	service.MarkStarted()
	if service.Startup().Status != "ok" {
		t.Fatal("expected started status")
	}
}
