package integration

import (
	"os"
	"strings"
	"testing"
)

// TestDebugTools - Disabled for deployed RTM server
func TestDebugTools(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping debug tools test")
		return
	}

	// ALWAYS skip for deployed RTM server - debug tools not available
	if strings.Contains(serverURL, "mcp-adapters.fly.dev") || strings.Contains(serverURL, "https://") {
		t.Skip("Debug tools test not applicable to deployed RTM server - RTM server has no debug/demo tools")
		return
	}

	// Only run debug tools test against local everything server
	t.Skip("Debug tools test temporarily disabled - use local everything server for debug testing")
}
