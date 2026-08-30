//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chouaib-skitou/event-driven-ecommerce-platform/apps/identity/internal/adapters/outbound/postgres"
)

func TestIdentityIntegrationEnvironment(t *testing.T) {
	databaseURL := os.Getenv("IDENTITY_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set IDENTITY_INTEGRATION_DATABASE_URL to run integration tests")
	}
	database, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.Check(ctx); err != nil {
		t.Fatal(err)
	}
	migrationDir := os.Getenv("IDENTITY_INTEGRATION_MIGRATION_DIR")
	if migrationDir == "" {
		migrationDir = filepath.Join("..", "..", "..", "..", "database", "identity", "migrations")
	}
	if err := database.Migrate(ctx, migrationDir); err != nil {
		t.Fatal(err)
	}
}
