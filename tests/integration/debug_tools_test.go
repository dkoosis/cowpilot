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

func TestDebugTools(t *testing.T) {
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		t.Skip("MCP_SERVER_URL not set, skipping debug tools test")
		return
	}

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Test 1: Raw tools/list
	fmt.Println("=== TEST 1: Raw tools/list ===")
	resp, err := http.Post(serverURL, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	// FIXED: Check error on close
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	var buf bytes.Buffer
	// FIXED: Check error on read
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}
	fmt.Printf("Response: %s\n", buf.String())

	// Test 2: Initialize first
	fmt.Println("\n=== TEST 2: Initialize first ===")
	initReq := `{
        "jsonrpc":"2.0",
        "id":1,
        "method":"initialize",
        "params":{
            "protocolVersion":"2024-11-05",
            "capabilities":{},
            "clientInfo":{"name":"test","version":"1.0"}
        }
    }`
	resp2, err := http.Post(serverURL, "application/json", strings.NewReader(initReq))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	// FIXED: Check error on close
	defer func() {
		if err := resp2.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	buf.Reset()
	// FIXED: Check error on read
	if _, err := io.Copy(&buf, resp2.Body); err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}
	fmt.Printf("Initialize response: %s\n", buf.String())

	// Test 3: Tools/list after initialize
	fmt.Println("\n=== TEST 3: tools/list after initialize ===")
	resp3, err := http.Post(serverURL, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":2}`))
	if err != nil {
		t.Fatalf("Error: %v", err)
	}
	// FIXED: Check error on close
	defer func() {
		if err := resp3.Body.Close(); err != nil {
			fmt.Printf("Error closing response body: %v\n", err)
		}
	}()

	buf.Reset()
	// FIXED: Check error on read
	if _, err := io.Copy(&buf, resp3.Body); err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}
	result := buf.String()
	fmt.Printf("Tools list response: %s\n", result)

	// Parse and check
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(result), &jsonResp); err == nil {
		if res, ok := jsonResp["result"].(map[string]interface{}); ok {
			if tools, ok := res["tools"].([]interface{}); ok {
				fmt.Printf("\nFound %d tools\n", len(tools))
				for i, tool := range tools {
					if t, ok := tool.(map[string]interface{}); ok {
						fmt.Printf("Tool %d: %s\n", i+1, t["name"])
					}
				}
			}
		}
	} else {
		fmt.Printf("Error unmarshalling JSON: %v\n", err)
	}
}
