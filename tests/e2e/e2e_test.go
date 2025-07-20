package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMCPProtocolCompliance runs the MCP protocol compliance test suite
// against a live cowpilot server using mcp-inspector-cli
func TestMCPProtocolCompliance(t *testing.T) {
	// Check if MCP_SERVER_URL is set
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping E2E tests")
		return
	}

	// Check if @modelcontextprotocol/inspector is available
	cmd := exec.Command("npx", "@modelcontextprotocol/inspector", "--version")
	if err := cmd.Run(); err != nil {
		t.Skip("@modelcontextprotocol/inspector not found. Install with: npm install -g @modelcontextprotocol/inspector")
		return
	}

	// Get the path to our test script
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("Failed to get current file path")
	}
	scriptPath := filepath.Join(filepath.Dir(filename), "mcp_scenarios.sh")

	// Make sure the script is executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Fatalf("Failed to make script executable: %v", err)
	}

	// Run the test script
	testCmd := exec.Command("/bin/bash", scriptPath, serverURL)
	
	// Capture output
	var stdout, stderr bytes.Buffer
	testCmd.Stdout = &stdout
	testCmd.Stderr = &stderr

	// Run the command
	err := testCmd.Run()
	
	// Always print the output for visibility
	t.Logf("Test output:\n%s", stdout.String())
	
	// Print stderr if there was any
	if stderr.Len() > 0 {
		t.Logf("Error output:\n%s", stderr.String())
	}

	// Check if tests failed
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("E2E tests failed with exit code %d\nOutput:\n%s\nErrors:\n%s", 
				exitErr.ExitCode(), stdout.String(), stderr.String())
		} else {
			t.Fatalf("Failed to run E2E tests: %v\nOutput:\n%s\nErrors:\n%s", 
				err, stdout.String(), stderr.String())
		}
	}
}

// TestMCPServerHealth verifies the server is healthy before running protocol tests
func TestMCPServerHealth(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping health check")
		return
	}

	// Extract base URL and append /health
	healthURL := serverURL
	if healthURL[len(healthURL)-1] == '/' {
		healthURL = healthURL[:len(healthURL)-1]
	}
	healthURL += "/health"

	// Use curl to check health endpoint
	cmd := exec.Command("curl", "-s", "-f", "-o", "/dev/null", "-w", "%{http_code}", healthURL)
	output, err := cmd.Output()
	
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	statusCode := string(output)
	if statusCode != "200" {
		t.Fatalf("Health check returned status %s, expected 200", statusCode)
	}

	t.Logf("Server health check passed (status: %s)", statusCode)
}

// TestLocalServer runs E2E tests against a local server if available
func TestLocalServer(t *testing.T) {
	// Skip if we're in CI or if MCP_SERVER_URL is already set
	if os.Getenv("CI") != "" || os.Getenv("MCP_SERVER_URL") != "" {
		t.Skip("Skipping local server test in CI or when MCP_SERVER_URL is set")
		return
	}

	// Check if local server is running
	cmd := exec.Command("curl", "-s", "-f", "http://localhost:8080/health")
	if err := cmd.Run(); err != nil {
		t.Skip("Local server not running on port 8080, skipping local tests")
		return
	}

	// Set environment variable and run tests
	os.Setenv("MCP_SERVER_URL", "http://localhost:8080/")
	defer os.Unsetenv("MCP_SERVER_URL")

	t.Run("LocalProtocolCompliance", TestMCPProtocolCompliance)
}

// Example of how to run these tests:
//
// 1. Against production:
//    MCP_SERVER_URL=https://cowpilot.fly.dev/ go test -v ./tests/e2e/
//
// 2. Against local server:
//    go run cmd/cowpilot/main.go  # In another terminal
//    go test -v ./tests/e2e/
//
// 3. In CI/CD pipeline:
//    export MCP_SERVER_URL=https://cowpilot.fly.dev/
//    go test ./tests/e2e/
