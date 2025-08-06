package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

var (
	serverURL   string
	serverCmd   *exec.Cmd
	projectRoot string
)

// TestMain manages the server lifecycle for all integration tests.
// This replaces the fragile shell script approach with robust Go-native process management.
func TestMain(m *testing.M) {
	// Find project root
	projectRoot = findProjectRoot()
	if projectRoot == "" {
		log.Fatal("Could not find project root (looking for go.mod)")
	}

	// 1. Build the server binary
	log.Println("Building server for integration tests...")
	binaryPath := filepath.Join(projectRoot, "bin", "core-server")
	buildCmd := exec.Command("go", "build", "-o", binaryPath, filepath.Join(projectRoot, "cmd", "core"))
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to build server: %v\n%s", err, output)
	}

	// 2. Start the server as a background process
	// NOTE: OAuth is disabled for general protocol tests.
	// Claude OAuth compliance tests run separately with auth enabled.
	serverCmd = exec.Command(binaryPath, "--disable-auth")
	serverCmd.Env = append(os.Environ(),
		"FLY_APP_NAME=local-test",
		"PORT=8080",
		"MCP_LOG_LEVEL=WARN",
	)

	// Capture server output for debugging
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	if err := serverCmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	log.Printf("Server started with PID %d", serverCmd.Process.Pid)

	// 3. Wait for the server to be healthy
	serverURL = "http://localhost:8080"
	if !waitForServer(serverURL, 15*time.Second) {
		_ = serverCmd.Process.Kill()
		log.Fatalf("Server did not become ready in time")
	}
	log.Println("Server is ready for tests")

	// 4. Run the actual tests
	exitCode := m.Run()

	// 5. Cleanly shut down the server
	log.Println("Shutting down server...")
	if err := serverCmd.Process.Signal(syscall.SIGTERM); err != nil {
		log.Printf("Failed to send SIGTERM, killing process: %v", err)
		_ = serverCmd.Process.Kill()
	}
	_ = serverCmd.Wait()
	log.Println("Server shut down")

	os.Exit(exitCode)
}

// waitForServer polls the health endpoint until ready or timeout
func waitForServer(baseURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 1 * time.Second}

	for time.Now().Before(deadline) {
		// Check health endpoint
		resp, err := client.Get(baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()

			// Also verify MCP endpoint is responding (should return 401 when OAuth is enabled)
			mcpReq := []byte(`{"jsonrpc":"2.0","method":"tools/list","id":1}`)
			mcpResp, err := client.Post(baseURL+"/mcp", "application/json", bytes.NewReader(mcpReq))
			if err == nil {
				defer func() { _ = mcpResp.Body.Close() }()
				// Server is ready if it returns 401 (needs auth) or 200 (no auth)
				if mcpResp.StatusCode == http.StatusOK || mcpResp.StatusCode == http.StatusUnauthorized {
					return true
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

// findProjectRoot walks up the directory tree to find go.mod
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// TestMCPInitialize tests the MCP initialization handshake
func TestMCPInitialize(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "integration-test",
				"version": "1.0",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get 200 OK when OAuth is disabled for testing
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", result["jsonrpc"])
	}

	if _, ok := result["result"]; !ok {
		t.Error("Response missing result field")
	}
}

// TestMCPToolsList tests the tools/list method
func TestMCPToolsList(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get 200 OK when OAuth is disabled for testing
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to decode response: %v\nBody: %s", err, string(body))
	}

	if result["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc 2.0, got %v", result["jsonrpc"])
	}

	if resultData, ok := result["result"].(map[string]interface{}); ok {
		if tools, ok := resultData["tools"].([]interface{}); ok {
			if len(tools) == 0 {
				t.Error("No tools returned")
			} else {
				t.Logf("Found %d tools", len(tools))
			}
		} else {
			t.Error("Result missing tools array")
		}
	} else {
		t.Error("Response missing result field")
	}
}

// TestMCPResourcesList tests the resources/list method
func TestMCPResourcesList(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get 200 OK when OAuth is disabled for testing
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 (OAuth disabled), got %d", resp.StatusCode)
	}
}

// TestHealthEndpoint verifies the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	resp, err := http.Get(serverURL + "/health")
	if err != nil {
		t.Fatalf("Failed to check health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	expected := `{"status":"healthy"}`
	if string(body) != expected {
		t.Errorf("Expected %s, got %s", expected, string(body))
	}
}

// TestMCPError tests error handling for malformed requests
func TestMCPError(t *testing.T) {
	client := &http.Client{Timeout: 5 * time.Second}

	// Send request with invalid JSON-RPC version
	reqBody := map[string]interface{}{
		"jsonrpc": "1.0", // Invalid version
		"id":      999,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := client.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should get 200 with error in JSON-RPC response (mcp-go returns errors in body, not HTTP status)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check for error in JSON-RPC response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to decode response: %v\nBody: %s", err, string(body))
	}

	// Should have an error field for invalid JSON-RPC version
	if _, hasError := result["error"]; !hasError {
		t.Error("Expected error in response for invalid JSON-RPC version")
	}
}

// TestConcurrentRequests verifies the server handles concurrent requests
func TestConcurrentRequests(t *testing.T) {
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			client := &http.Client{Timeout: 5 * time.Second}
			reqBody := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      fmt.Sprintf("concurrent-%d", id),
				"method":  "tools/list",
				"params":  map[string]interface{}{},
			}

			jsonData, _ := json.Marshal(reqBody)
			resp, err := client.Post(serverURL+"/mcp", "application/json", bytes.NewReader(jsonData))
			if err != nil {
				t.Errorf("Request %d failed: %v", id, err)
				return
			}
			_ = resp.Body.Close()

			// Should get 200 OK when OAuth is disabled for testing
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Request %d got status %d (expected 200)", id, resp.StatusCode)
			}
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
