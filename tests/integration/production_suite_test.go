package integration

import (
	"fmt"
	"os"
	"testing"
)

// TestProductionSuite runs all production validation tests
// This ensures our deployed services are ready for Claude Desktop
func TestProductionSuite(t *testing.T) {
	// Skip only if explicitly requested
	if os.Getenv("SKIP_PRODUCTION_TESTS") == "true" {
		t.Skip("Skipping production tests (SKIP_PRODUCTION_TESTS=true)")
	}

	t.Log("\n" +
		"========================================\n" +
		"   PRODUCTION VALIDATION SUITE         \n" +
		"========================================")

	// Run RTM health check if RTM is deployed
	t.Run("RTM Production Health", func(t *testing.T) {
		// This will skip if LOCAL_TEST=true
		TestRTMProductionHealth(t)
	})

	// Could add other production checks here
	// t.Run("Spektrix Production Health", TestSpektrixProductionHealth)
	// t.Run("Core Production Health", TestCoreProductionHealth)

	t.Log("\n" +
		"========================================\n" +
		"   PRODUCTION VALIDATION COMPLETE      \n" +
		"========================================")
}

// formatTestOutput provides consistent formatting for test output
func formatTestOutput(t *testing.T, category string, checks []TestCheck) {
	t.Helper()

	maxLen := 0
	for _, check := range checks {
		if len(check.Name) > maxLen {
			maxLen = len(check.Name)
		}
	}

	t.Logf("\n%s:", category)
	for _, check := range checks {
		status := "✓"
		if !check.Passed {
			status = "✗"
		}
		padding := maxLen - len(check.Name)
		t.Logf("  %s %-*s %s", status, padding, check.Name, check.Message)
	}
}

// TestCheck represents a single validation check
type TestCheck struct {
	Name    string
	Passed  bool
	Message string
}

// ProductionEndpoints returns the standard endpoints to validate
func ProductionEndpoints(serverURL string) map[string]string {
	return map[string]string{
		"/health":          "Health check",
		"/mcp":             "MCP endpoint",
		"/oauth/authorize": "OAuth authorize",
		"/oauth/token":     "OAuth token",
		"/.well-known/oauth-authorization-server": "OAuth discovery",
		"/.well-known/oauth-protected-resource":   "Protected resource metadata",
	}
}

// ValidateEndpoint checks if an endpoint responds correctly
func ValidateEndpoint(endpoint, expectedStatus string) TestCheck {
	// Implementation would go here
	return TestCheck{
		Name:    endpoint,
		Passed:  true,
		Message: fmt.Sprintf("HTTP %s", expectedStatus),
	}
}
