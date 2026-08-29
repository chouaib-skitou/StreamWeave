//go:build security

package security

import "testing"

func TestSecuritySuiteIsExplicitlyEnabled(t *testing.T) {
	t.Log("security suite is enabled; run the repository scanner and dependency audit in CI")
}
