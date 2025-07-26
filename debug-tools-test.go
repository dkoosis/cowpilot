package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"bytes"
)

func main() {
	// Wait for server to be ready
	time.Sleep(2 * time.Second)
	
	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080/"
	}
	
	// Test 1: Raw tools/list
	fmt.Println("=== TEST 1: Raw tools/list ===")
	resp, err := http.Post(serverURL, "application/json", 
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":1}`))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp2.Body.Close()
	
	buf.Reset()
	buf.ReadFrom(resp2.Body)
	fmt.Printf("Initialize response: %s\n", buf.String())
	
	// Test 3: Tools/list after initialize
	fmt.Println("\n=== TEST 3: tools/list after initialize ===")
	resp3, err := http.Post(serverURL, "application/json", 
		strings.NewReader(`{"jsonrpc":"2.0","method":"tools/list","id":2}`))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	defer resp3.Body.Close()
	
	buf.Reset()
	buf.ReadFrom(resp3.Body)
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
	}
}
