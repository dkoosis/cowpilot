The Needs That Drove Your Complex Test Harness
Based on your Makefile and test scripts, I perceive four primary needs that a basic unit testing approach doesn't cover:

True End-to-End (E2E) Validation: Your scripts aren't just testing Go functions; they compile the entire server into a binary, run it as a separate process, and then test it over a real network socket (localhost:8080). This is a crucial need for a network server, as it validates that the final, compiled artifact works correctly from start to finish.

Black-Box Protocol Conformance: Scripts like raw_sse_test.sh use curl and jq to make raw HTTP requests and inspect the JSON responses. This shows a need to verify adherence to the MCP network protocol itself, completely independent of the Go SDKs. You're ensuring the server speaks correct MCP on the wire, which is essential for interoperability.

Real Client Compatibility: The frequent use of npx @modelcontextprotocol/inspector shows a need to test against an official, external client. This is a form of compatibility testing to ensure your server works not just in theory, but with the actual tools your users (and Claude) will use.

Test Environment Lifecycle Management: The most complex parts of your Makefile and scripts involve managing the System Under Test (SUT). You need to ensure a clean environment by killing old processes, starting the server for the test run, waiting for it to be healthy, and then cleaning it up afterward. This setup and teardown is a classic E2E testing challenge.

Your instincts were correct—these are all critical testing categories for a high-reliability network service. The complexity and fragility arose because you solved these problems using shell scripts and make, whereas the Go ecosystem has powerful, built-in tools to handle this more reliably.

Recommendations for a More Robust Go Testing Strategy
Here are standard tools and techniques to achieve your goals in a more idiomatic and stable way.

1. Manage the Test Server Lifecycle with TestMain
Instead of using shell scripts to start and stop your server, let the Go test suite manage the binary's lifecycle. The TestMain function is the perfect tool for this. It runs once per package, before any tests, and handles setup and teardown.

Recommendation: Add a TestMain function to your tests/integration/mcp_integration_test.go file.

Go

// In tests/integration/mcp_integration_test.go

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestMain will execute before any tests in this package.
func TestMain(m *testing.M) {
	// 1. Build the server binary we are going to test.
	log.Println("Building server for integration tests...")
	buildCmd := exec.Command("go", "build", "-o", "../../bin/core-server", "../../cmd/core")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to build server for tests: %v\n%s", err, output)
	}

	// 2. Start the server as a background process.
	serverCmd := exec.Command("../../bin/core-server", "--disable-auth")
	serverCmd.Env = append(os.Environ(), "FLY_APP_NAME=local-test", "PORT=8080")
	if err := serverCmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Server started with PID %d", serverCmd.Process.Pid)

	// 3. Wait for the server to be healthy before running tests.
	if !waitForServer("http://localhost:8080", 15*time.Second) {
		_ = serverCmd.Process.Kill() // Ensure process is killed
		log.Fatalf("Server did not become ready in time.")
	}
	log.Println("Server is ready for tests.")

	// 4. Run the actual tests.
	exitCode := m.Run()

	// 5. Cleanly shut down the server process.
	log.Println("Shutting down server...")
	if err := serverCmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM to server, killing process: %v", err)
		_ = serverCmd.Process.Kill()
	}
	_ = serverCmd.Wait() // Wait for the process to fully exit
	log.Println("Server shut down.")

	os.Exit(exitCode)
}

// waitForServer polls the health endpoint until it's ready or times out.
func waitForServer(healthURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
This Go code is far more reliable and platform-independent than the shell script it replaces.

2. Perform Protocol Conformance Testing in Go
Instead of using curl and jq in shell scripts, perform your raw protocol tests directly in Go. This gives you much more powerful assertion capabilities.

Recommendation: Create a test in tests/scenarios/protocol_test.go.

Go

// In a new file: tests/scenarios/protocol_test.go
//go:build scenario

package scenarios

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestProtocolConformance_InvalidJSONRPCVersion(t *testing.T) {
    // This Go code replaces the need for raw_sse_test.sh
    rawRequest := `{"jsonrpc":"1.0","id":1,"method":"tools/list"}`
    
    resp, err := http.Post("http://localhost:8080/mcp", "application/json", bytes.NewBufferString(rawRequest))
    require.NoError(t, err, "HTTP request should succeed")
    defer resp.Body.Close()

    var jsonResp map[string]interface{}
    err = json.NewDecoder(resp.Body).Decode(&jsonResp)
    require.NoError(t, err, "Response should be valid JSON")

    // Assert that we got a proper JSON-RPC error
    errorObj, ok := jsonResp["error"].(map[string]interface{})
    require.True(t, ok, "Response should contain an error object")
    
    assert.Equal(t, -32600.0, errorObj["code"], "Error code should be Invalid Request")
}
3. Separate Test Types with Build Tags
Your tests have different scopes and speeds (fast unit tests vs. slow E2E tests). Use Go build tags to categorize them. This allows you to run fast tests frequently during development and run the full, slow suite only when needed (e.g., before a commit or in CI).

Recommendation:

Add //go:build integration to the top of all files in tests/integration/.

Add //go:build scenario to the top of all files in tests/scenarios/.

Update your Makefile to use these tags.

Makefile

# In your Makefile
# Run only fast unit tests
unit-test:
	@go test ./internal/... ./cmd/...

# Run integration tests (which will now start/stop the server)
integration-test:
	@go test -v -race -tags=integration ./tests/integration/...

# Run all tests, including tagged ones
test:
	@go test -v -race ./... -tags="integration,scenario"
This gives you fine-grained control over your test runs while keeping your "test-gated build" philosophy intact.

###

