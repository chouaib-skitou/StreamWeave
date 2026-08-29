//go:build integration

package integration

import (
	"os"
	"testing"
)

func TestIdentityIntegrationEnvironment(t *testing.T) {
	if os.Getenv("IDENTITY_INTEGRATION_DATABASE_URL") == "" {
		t.Skip("set IDENTITY_INTEGRATION_DATABASE_URL to run integration tests")
	}
	// The Compose smoke test owns full dependency orchestration; this guard keeps
	// the integration target explicit and safe when dependencies are not running.
}
