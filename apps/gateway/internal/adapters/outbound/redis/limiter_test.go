package redisadapter

import (
	"context"
	"testing"
)

func TestLimiterSkipsLocalClassesAndRejectsUnknown(t *testing.T) {
	client := &Client{}
	if allowed, err := client.Allow(context.Background(), "x", "health"); err != nil || !allowed {
		t.Fatalf("health=%v %v", allowed, err)
	}
	if allowed, err := client.Allow(context.Background(), "x", "metrics"); err != nil || !allowed {
		t.Fatalf("metrics=%v %v", allowed, err)
	}
	if _, err := client.Allow(context.Background(), "x", "unknown"); err == nil {
		t.Fatal("unknown class accepted")
	}
	if err := client.Check(context.Background()); err == nil {
		t.Fatal("nil redis accepted")
	}
}
