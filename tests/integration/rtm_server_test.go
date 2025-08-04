package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRTMServer(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping RTM server test")
		return
	}

	// Only run this test against the deployed RTM server
	if !strings.Contains(serverURL, "cowpilot.fly.dev") {
		t.Skip("RTM server test only runs against cowpilot.fly.dev instance")
		return
	}

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Test 1: Initialize the MCP connection
	fmt.Println("=== TEST 1: Initialize MCP connection ===")
	initReq := `{
        "jsonrpc":"2.0",
        "id":1,
        "method":"initialize",
        "params":{
            "protocolVersion":"2024-11-05",
            "capabilities":{},
            "clientInfo":{"name":"integration-test","version":"1.0"}
        }
    }`

	resp, err := http.Post(serverURL, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("Error initializing: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		t.Fatalf("Error reading initialize response: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Fatalf("Initialize failed with status %d: %s", resp.StatusCode, buf.String())
	}

	fmt.Printf("Initialize response: %s\n", buf.String())

	// Test 2: List tools (should show RTM tools)
	fmt.Println("\n=== TEST 2: List RTM tools ===")
	resp2, err := http.Post(serverURL, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":2}`))
	if err != nil {
		t.Fatalf("Error listing tools: %v", err)
	}
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	buf.Reset()
	if _, err := io.Copy(&buf, resp2.Body); err != nil {
		t.Fatalf("Error reading tools response: %v", err)
	}

	result := buf.String()
	fmt.Printf("Tools list response: %s\n", result)

	// Parse and verify RTM tools
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &jsonResp); err != nil {
		t.Fatalf("Error parsing tools response: %v", err)
	}

	res, ok := jsonResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid tools response format")
	}

	tools, ok := res["tools"].([]interface{})
	if !ok {
		t.Fatalf("No tools array in response")
	}

	fmt.Printf("\nFound %d tools\n", len(tools))

	// Check for expected RTM tools
	expectedRTMTools := []string{
		"rtm_auth_url", "rtm_lists", "rtm_search",
		"rtm_quick_add", "rtm_update", "rtm_complete", "rtm_manage_list",
	}

	foundTools := make(map[string]bool)
	for _, tool := range tools {
		// FIX: Renamed the inner variable from 't' to 'toolMap' to avoid shadowing the *testing.T variable.
		if toolMap, ok := tool.(map[string]interface{}); ok {
			if name, ok := toolMap["name"].(string); ok {
				foundTools[name] = true
				fmt.Printf("- %s\n", name)
			}
		}
	}

	// Verify all expected RTM tools are present
	for _, expectedTool := range expectedRTMTools {
		if !foundTools[expectedTool] {
			t.Errorf("Missing expected RTM tool: %s", expectedTool)
		}
	}

	// Verify no demo tools are present (clean RTM server)
	demoTools := []string{"hello", "echo", "add", "get_time", "base64_encode"}
	for _, demoTool := range demoTools {
		if foundTools[demoTool] {
			t.Errorf("Unexpected demo tool found: %s (should be RTM-only deployment)", demoTool)
		}
	}

	// Test 3: List resources (should show RTM resources)
	fmt.Println("\n=== TEST 3: List RTM resources ===")
	resp3, err := http.Post(serverURL, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","method":"resources/list","id":3}`))
	if err != nil {
		t.Fatalf("Error listing resources: %v", err)
	}
	defer func() {
		if err := resp3.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	buf.Reset()
	if _, err := io.Copy(&buf, resp3.Body); err != nil {
		t.Fatalf("Error reading resources response: %v", err)
	}

	resourceResult := buf.String()
	fmt.Printf("Resources list response: %s\n", resourceResult)

	// Parse and verify RTM resources
	var resourceResp map[string]interface{}
	if err := json.Unmarshal([]byte(resourceResult), &resourceResp); err != nil {
		t.Fatalf("Error parsing resources response: %v", err)
	}

	resourceRes, ok := resourceResp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Invalid resources response format")
	}

	resources, ok := resourceRes["resources"].([]interface{})
	if !ok {
		t.Fatalf("No resources array in response")
	}

	fmt.Printf("Found %d resources\n", len(resources))

	// Check for expected RTM resources
	expectedRTMResources := []string{
		"rtm://today", "rtm://inbox", "rtm://overdue",
		"rtm://week", "rtm://lists",
	}

	foundResources := make(map[string]bool)
	for _, resource := range resources {
		if r, ok := resource.(map[string]interface{}); ok {
			if uri, ok := r["uri"].(string); ok {
				foundResources[uri] = true
				fmt.Printf("- %s\n", uri)
			}
		}
	}

	// Verify expected RTM resources are present
	for _, expectedResource := range expectedRTMResources {
		if !foundResources[expectedResource] {
			t.Errorf("Missing expected RTM resource: %s", expectedResource)
		}
	}

	fmt.Println("\n✅ RTM server integration test passed")
}
