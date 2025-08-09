package rtm_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/vcto/mcp-adapters/internal/rtm"
)

// IntegrationTestServer simulates Claude Desktop making OAuth requests
type IntegrationTestServer struct {
	t           *testing.T
	rtmServer   *httptest.Server
	oauthServer *httptest.Server
	adapter     *rtm.OAuthAdapter
}

// NewIntegrationTestServer sets up the test environment
func NewIntegrationTestServer(t *testing.T) *IntegrationTestServer {
	its := &IntegrationTestServer{t: t}

	// Create OAuth adapter
	its.adapter = rtm.NewOAuthAdapter("test-api-key", "test-secret", "")

	// Create OAuth server
	mux := http.NewServeMux()
	mux.HandleFunc("/rtm/authorize", its.adapter.HandleAuthorize)
	mux.HandleFunc("/rtm/callback", its.adapter.HandleCallback)
	mux.HandleFunc("/rtm/token", its.adapter.HandleToken)
	mux.HandleFunc("/rtm/check-auth", its.adapter.HandleCheckAuth)

	its.oauthServer = httptest.NewServer(mux)
	its.adapter = rtm.NewOAuthAdapter("test-api-key", "test-secret", its.oauthServer.URL)

	// Re-register handlers with new adapter
	mux = http.NewServeMux()
	mux.HandleFunc("/rtm/authorize", its.adapter.HandleAuthorize)
	mux.HandleFunc("/rtm/callback", its.adapter.HandleCallback)
	mux.HandleFunc("/rtm/token", its.adapter.HandleToken)
	mux.HandleFunc("/rtm/check-auth", its.adapter.HandleCheckAuth)
	its.oauthServer = httptest.NewServer(mux)

	return its
}

func (its *IntegrationTestServer) Cleanup() {
	if its.oauthServer != nil {
		its.oauthServer.Close()
	}
	if its.rtmServer != nil {
		its.rtmServer.Close()
	}
}

// TestClaudeDesktopFlow simulates the complete Claude Desktop OAuth flow
func TestClaudeDesktopFlow(t *testing.T) {
	its := NewIntegrationTestServer(t)
	defer its.Cleanup()

	// Use a mock RTM client
	mockClient := &rtm.MockRTMClient{
		FrobValue:  "test-frob-123",
		TokenValue: "rtm-token-456",
	}
	its.adapter.SetClient(mockClient)

	// Step 1: Claude Desktop initiates OAuth
	claudeClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	// Build authorization URL (what Claude would send)
	authURL := fmt.Sprintf("%s/rtm/authorize?client_id=claude-desktop&response_type=code&redirect_uri=%s&state=claude-state-123&code_challenge=test-challenge&code_challenge_method=S256",
		its.oauthServer.URL,
		url.QueryEscape("http://localhost:59263/callback"))

	resp, err := claudeClient.Get(authURL)
	if err != nil {
		t.Fatalf("Failed to start OAuth: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200, got %d. Body: %s", resp.StatusCode, body)
	}

	// Extract CSRF cookie
	var csrfToken string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "csrf_token" {
			csrfToken = cookie.Value
			break
		}
	}

	if csrfToken == "" {
		t.Fatal("No CSRF token cookie set")
	}

	// Step 2: User clicks "Connect" button (form submission)
	form := url.Values{
		"client_id":             {"claude-desktop"},
		"state":                 {"claude-state-123"},
		"redirect_uri":          {"http://localhost:59263/callback"},
		"csrf_state":            {csrfToken},
		"code_challenge":        {"test-challenge"},
		"code_challenge_method": {"S256"},
	}

	req, _ := http.NewRequest("POST", its.oauthServer.URL+"/rtm/authorize",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", fmt.Sprintf("csrf_token=%s", csrfToken))

	resp, err = claudeClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to submit auth form: %v", err)
	}
	defer resp.Body.Close()

	// Should get intermediate page with code
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Connect to Remember The Milk") {
		t.Error("Expected intermediate page")
	}

	// Extract code from HTML (in real flow, JavaScript would do this)
	// For testing, we'll get it from the session
	var code string
	for c, session := range its.adapter.sessions {
		if session.State == "claude-state-123" {
			code = c
			break
		}
	}

	if code == "" {
		t.Fatal("No authorization code generated")
	}

	// Step 3: Simulate user authorizing on RTM
	// (In real flow, user would click "OK, I'll allow it" on RTM)
	// We simulate this by having the mock client succeed

	// Step 4: Check authorization status (what the intermediate page JS does)
	checkURL := fmt.Sprintf("%s/rtm/check-auth?code=%s", its.oauthServer.URL, code)
	resp, err = http.Get(checkURL)
	if err != nil {
		t.Fatalf("Failed to check auth: %v", err)
	}
	defer resp.Body.Close()

	var checkResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&checkResult); err != nil {
		t.Fatalf("Failed to decode check result: %v", err)
	}

	if checkResult["authorized"] != true {
		t.Errorf("Expected authorized=true, got %v", checkResult)
	}

	// Step 5: Callback redirect (what happens after auth is confirmed)
	callbackURL := fmt.Sprintf("%s/rtm/callback?code=%s", its.oauthServer.URL, code)
	resp, err = claudeClient.Get(callbackURL)
	if err != nil {
		t.Fatalf("Failed to hit callback: %v", err)
	}
	defer resp.Body.Close()

	// Should redirect to Claude's redirect_uri with code
	if resp.StatusCode != http.StatusFound {
		t.Errorf("Expected redirect, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if !strings.Contains(location, "code="+code) {
		t.Errorf("Expected code in redirect, got %s", location)
	}

	if !strings.Contains(location, "state=claude-state-123") {
		t.Errorf("Expected state in redirect, got %s", location)
	}

	// Step 6: Claude exchanges code for token
	tokenForm := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {"test-verifier"}, // Would be the actual PKCE verifier
	}

	resp, err = http.PostForm(its.oauthServer.URL+"/rtm/token", tokenForm)
	if err != nil {
		t.Fatalf("Failed to exchange token: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Token exchange failed: %d - %s", resp.StatusCode, body)
	}

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("Failed to decode token response: %v", err)
	}

	if tokenResp["access_token"] != "rtm-token-456" {
		t.Errorf("Expected RTM token, got %v", tokenResp["access_token"])
	}

	if tokenResp["token_type"] != "Bearer" {
		t.Errorf("Expected Bearer token type, got %v", tokenResp["token_type"])
	}

	// Verify session was cleaned up
	if its.adapter.GetSession(code) != nil {
		t.Error("Session should be removed after successful token exchange")
	}
}

// TestErrorScenarios tests various error conditions
func TestErrorScenarios(t *testing.T) {
	its := NewIntegrationTestServer(t)
	defer its.Cleanup()

	mockClient := &rtm.MockRTMClient{
		FrobValue:  "test-frob",
		TokenValue: "test-token",
	}
	its.adapter.SetClient(mockClient)

	t.Run("Missing CSRF Token", func(t *testing.T) {
		form := url.Values{
			"client_id":    {"test"},
			"state":        {"test"},
			"redirect_uri": {"http://localhost/callback"},
			// Missing csrf_state
		}

		resp, err := http.PostForm(its.oauthServer.URL+"/rtm/authorize", form)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid Code", func(t *testing.T) {
		resp, err := http.Get(its.oauthServer.URL + "/rtm/check-auth?code=invalid")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid code, got %d", resp.StatusCode)
		}
	})

	t.Run("Authorization Timeout", func(t *testing.T) {
		// Create expired session
		session := &rtm.AuthSession{
			Code:      "expired-code",
			Frob:      "expired-frob",
			CreatedAt: time.Now().Add(-61 * time.Minute),
		}
		its.adapter.sessions["expired-code"] = session

		// Try to exchange token
		form := url.Values{
			"grant_type": {"authorization_code"},
			"code":       {"expired-code"},
		}

		resp, err := http.PostForm(its.oauthServer.URL+"/rtm/token", form)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400 for expired code, got %d", resp.StatusCode)
		}

		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)

		if errorResp["error"] != "authorization_pending" {
			t.Errorf("Expected authorization_pending error, got %v", errorResp["error"])
		}
	})

	t.Run("User Denies Authorization", func(t *testing.T) {
		mockClient.ShouldFailGetToken = true
		defer func() { mockClient.ShouldFailGetToken = false }()

		// Create session
		session := &rtm.AuthSession{
			Code:      "denied-code",
			Frob:      "denied-frob",
			CreatedAt: time.Now(),
		}
		its.adapter.sessions["denied-code"] = session

		// Check auth status
		resp, err := http.Get(its.oauthServer.URL + "/rtm/check-auth?code=denied-code")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["authorized"] == true {
			t.Error("Should not be authorized when user denies")
		}

		if result["pending"] != true {
			t.Error("Should show as pending when not yet authorized")
		}
	})
}

// TestConcurrentRequests tests handling of concurrent authorization attempts
func TestConcurrentRequests(t *testing.T) {
	its := NewIntegrationTestServer(t)
	defer its.Cleanup()

	mockClient := &rtm.MockRTMClient{
		FrobValue:  "test-frob",
		TokenValue: "test-token",
	}
	its.adapter.SetClient(mockClient)

	// Simulate multiple concurrent authorization attempts
	results := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// Each goroutine tries to start authorization
			resp, err := http.Get(fmt.Sprintf("%s/rtm/authorize?client_id=client-%d&state=state-%d&redirect_uri=http://localhost/cb",
				its.oauthServer.URL, id, id))

			if err != nil {
				results <- false
				return
			}
			resp.Body.Close()

			results <- resp.StatusCode == http.StatusOK
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-results {
			successCount++
		}
	}

	if successCount != 10 {
		t.Errorf("Expected all 10 requests to succeed, got %d", successCount)
	}

	// Verify we have 10 different sessions
	sessionCount := len(its.adapter.sessions)
	if sessionCount < 10 {
		t.Errorf("Expected at least 10 sessions, got %d", sessionCount)
	}
}

// BenchmarkOAuthFlow benchmarks the complete OAuth flow
func BenchmarkOAuthFlow(b *testing.B) {
	its := &IntegrationTestServer{}
	its.adapter = rtm.NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/rtm/authorize", its.adapter.HandleAuthorize)
	mux.HandleFunc("/rtm/callback", its.adapter.HandleCallback)
	mux.HandleFunc("/rtm/token", its.adapter.HandleToken)
	mux.HandleFunc("/rtm/check-auth", its.adapter.HandleCheckAuth)
	its.oauthServer = httptest.NewServer(mux)
	defer its.oauthServer.Close()

	mockClient := &rtm.MockRTMClient{
		FrobValue:  "bench-frob",
		TokenValue: "bench-token",
	}
	its.adapter.SetClient(mockClient)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Get auth page
		resp, _ := http.Get(its.oauthServer.URL + "/rtm/authorize?client_id=bench&state=bench&redirect_uri=http://localhost/cb")

		// Extract CSRF
		var csrf string
		for _, c := range resp.Cookies() {
			if c.Name == "csrf_token" {
				csrf = c.Value
				break
			}
		}
		resp.Body.Close()

		// Submit form
		form := url.Values{
			"client_id":    {"bench"},
			"state":        {"bench"},
			"redirect_uri": {"http://localhost/cb"},
			"csrf_state":   {csrf},
		}

		req, _ := http.NewRequest("POST", its.oauthServer.URL+"/rtm/authorize",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", "csrf_token="+csrf)

		client := &http.Client{}
		resp, _ = client.Do(req)
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		// Clean up session for next iteration
		for code := range its.adapter.sessions {
			delete(its.adapter.sessions, code)
		}
	}
}
