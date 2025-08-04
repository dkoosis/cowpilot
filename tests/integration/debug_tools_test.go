package integration

import (
	"strings"
	"testing"
)

// TestDebugTools - Disabled for deployed RTM server
func TestDebugTools(t *testing.T) {
	// This test now uses the serverURL from TestMain.
	if strings.Contains(serverURL, "mcp-adapters.fly.dev") || strings.Contains(serverURL, "https://") {
		t.Skip("Debug tools test not applicable to deployed RTM server - RTM server has no debug/demo tools")
		return
	}

	// Only run debug tools test against local everything server
	t.Skip("Debug tools test temporarily disabled - use local everything server for debug testing")
}
