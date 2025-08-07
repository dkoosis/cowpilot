package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OAuth Flow Diagnostic Tool
// Tests the complete OAuth flow and logs all requests/responses

var (
	serverURL = "https://rtm.fly.dev"
	localURL  = "http://localhost:8081"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "local" {
		serverURL = localURL
		fmt.Println("Testing LOCAL server")
	} else {
		fmt.Println("Testing PRODUCTION server")
	}

	fmt.Printf("Server URL: %s\n", serverURL)
	fmt.Println("=" + string(make([]byte, 60)) + "=")

	// Test 1: Check discovery endpoints
	testDiscoveryEndpoints()

	// Test 2: Check authorization endpoint
	testAuthorizationEndpoint()

	// Test 3: Check token endpoint with mock code
	testTokenEndpoint()

	// Test 4: Check authenticated endpoint
	testAuthenticatedEndpoint()
}

func testDiscoveryEndpoints() {
	fmt.Println("\n1. TESTING DISCOVERY ENDPOINTS")
	fmt.Println("-" + string(make([]byte, 30)))

	endpoints := []string{
		"/.well-known/oauth-protected-resource",
		"/.well-known/oauth-authorization-server",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("\nGET %s%s\n", serverURL, endpoint)
		resp, err := http.Get(serverURL + endpoint)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		dumpResponse(resp)
	}
}

func testAuthorizationEndpoint() {
	fmt.Println("\n2. TESTING AUTHORIZATION ENDPOINT")
	fmt.Println("-" + string(make([]byte, 30)))

	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=test&redirect_uri=http://localhost:3000/callback&state=xyz123", serverURL)
	fmt.Printf("\nGET %s\n", authURL)

	resp, err := http.Get(authURL)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Check if we get HTML form
	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		if bytes.Contains(body, []byte("<form")) {
			fmt.Println("✓ HTML form received")
		} else {
			fmt.Printf("Response body (first 500 chars):\n%s\n", string(body[:min(500, len(body))]))
		}
	}
}

func testTokenEndpoint() {
	fmt.Println("\n3. TESTING TOKEN ENDPOINT")
	fmt.Println("-" + string(make([]byte, 30)))

	tokenURL := serverURL + "/oauth/token"
	payload := "grant_type=authorization_code&code=test-code-123&client_id=test"

	fmt.Printf("\nPOST %s\n", tokenURL)
	fmt.Printf("Body: %s\n", payload)

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", bytes.NewBufferString(payload))
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	dumpResponse(resp)
}

func testAuthenticatedEndpoint() {
	fmt.Println("\n4. TESTING AUTHENTICATED MCP ENDPOINT")
	fmt.Println("-" + string(make([]byte, 30)))

	// Test without auth
	fmt.Println("\na) Without Authorization header:")
	testMCPEndpoint("")

	// Test with invalid auth
	fmt.Println("\nb) With invalid Bearer token:")
	testMCPEndpoint("Bearer invalid-token-xyz")
}

func testMCPEndpoint(authHeader string) {
	mcpURL := serverURL + "/mcp"
	
	// Create MCP initialize request
	initRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "oauth-diagnostic",
				"version": "1.0.0",
			},
		},
		"id": 1,
	}

	jsonData, _ := json.Marshal(initRequest)
	
	req, err := http.NewRequest("POST", mcpURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("ERROR creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
		fmt.Printf("Authorization: %s\n", authHeader)
	}

	fmt.Printf("POST %s\n", mcpURL)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	dumpResponse(resp)

	// Check for WWW-Authenticate header on 401
	if resp.StatusCode == 401 {
		if authHeader := resp.Header.Get("WWW-Authenticate"); authHeader != "" {
			fmt.Printf("✓ WWW-Authenticate header present: %s\n", authHeader)
		} else {
			fmt.Println("✗ Missing WWW-Authenticate header on 401 response")
		}
	}
}

func dumpResponse(resp *http.Response) {
	fmt.Printf("Status: %s\n", resp.Status)
	
	// Dump headers
	fmt.Printf("Headers:\n")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// Read and pretty-print JSON body if present
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR reading body: %v\n", err)
		return
	}

	if len(body) > 0 {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			// It's JSON, pretty print it
			prettyJSON, _ := json.MarshalIndent(jsonData, "", "  ")
			fmt.Printf("Body (JSON):\n%s\n", string(prettyJSON))
		} else {
			// Not JSON, print as-is (truncate if too long)
			if len(body) > 1000 {
				fmt.Printf("Body (first 1000 chars):\n%s...\n", string(body[:1000]))
			} else {
				fmt.Printf("Body:\n%s\n", string(body))
			}
		}
	} else {
		fmt.Println("Body: (empty)")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
