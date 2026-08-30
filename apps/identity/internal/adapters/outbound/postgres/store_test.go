package postgres

import (
	"database/sql"
	"errors"
	"testing"

	domain "github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/domain/identity"
)

func TestDatabaseErrorMapping(t *testing.T) {
	if !errors.Is(mapDatabaseError(sql.ErrNoRows), domain.ErrNotFound) {
		t.Fatal("sql no rows was not mapped")
	}
	original := errors.New("database unavailable")
	if !errors.Is(mapDatabaseError(original), original) {
		t.Fatal("database error was not preserved")
	}
	if mapDatabaseError(nil) != nil {
		t.Fatal("nil error changed")
	}
}

func TestNullableHelpers(t *testing.T) {
	if nullableTime(sql.NullTime{}) != nil {
		t.Fatal("invalid time mapped")
	}
	if nullableString(nil).Valid {
		t.Fatal("nil string marked valid")
	}
}
