package identity

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestNormalizationAndUsability(t *testing.T) {
	if got := NormalizeEmail("  User@Example.TEST "); got != "user@example.test" {
		t.Fatalf("got %q", got)
	}
	now := time.Now()
	session := Session{ID: uuid.New(), Status: SessionActive, ExpiresAt: now.Add(time.Minute)}
	if !session.IsUsable(now) {
		t.Fatal("active session should be usable")
	}
	session.Status = SessionRotated
	if session.IsUsable(now) {
		t.Fatal("rotated session should not be usable")
	}
	token := OneTimeToken{ExpiresAt: now.Add(time.Minute)}
	if !token.IsUsable(now) {
		t.Fatal("fresh token should be usable")
	}
	consumed := now
	token.ConsumedAt = &consumed
	if token.IsUsable(now) {
		t.Fatal("consumed token should not be usable")
	}
}
