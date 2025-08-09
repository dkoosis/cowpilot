package rtm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// MockRTMClient implements RTMClientInterface for testing
type MockRTMClient struct {
	// Control behavior
	ShouldFailGetFrob  bool
	ShouldFailGetToken bool
	TokenExchangeDelay time.Duration
	FrobValue          string
	TokenValue         string

	// Track calls
	GetFrobCalls  int
	GetTokenCalls int
	GetTokenFrobs []string
}

func NewMockRTMClient() *MockRTMClient {
	return &MockRTMClient{
		FrobValue:  "test-frob-123",
		TokenValue: "test-token-456",
	}
}

func (m *MockRTMClient) GetFrob() (string, error) {
	m.GetFrobCalls++
	if m.ShouldFailGetFrob {
		return "", fmt.Errorf("mock frob generation failed")
	}
	return m.FrobValue, nil
}

func (m *MockRTMClient) GetToken(frob string) error {
	m.GetTokenCalls++
	m.GetTokenFrobs = append(m.GetTokenFrobs, frob)

	if m.TokenExchangeDelay > 0 {
		time.Sleep(m.TokenExchangeDelay)
	}

	if m.ShouldFailGetToken {
		return &RTMError{Code: 101, Msg: "Invalid frob"}
	}

	return nil
}

func (m *MockRTMClient) GetAPIKey() string {
	return "test-api-key"
}

func (m *MockRTMClient) GetAuthToken() string {
	if m.ShouldFailGetToken {
		return ""
	}
	return m.TokenValue
}

func (m *MockRTMClient) SetAuthToken(token string) {
	m.TokenValue = token
}

func (m *MockRTMClient) Sign(params map[string]string) string {
	return "mock-signature"
}

func (m *MockRTMClient) GetLists() ([]List, error) {
	if m.TokenValue == "" {
		return nil, fmt.Errorf("not authenticated")
	}
	return []List{{ID: "1", Name: "Inbox"}}, nil
}

// TestOAuthFlowComplete tests the complete OAuth flow
func TestOAuthFlowComplete(t *testing.T) {
	mockClient := NewMockRTMClient()
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	adapter.SetClient(mockClient)

	// Step 1: Start authorization
	req := httptest.NewRequest("GET", "/rtm/authorize?client_id=test&state=xyz&redirect_uri=http://localhost:3000/callback", nil)
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check CSRF cookie was set
	cookies := resp.Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("CSRF cookie not set")
	}

	if csrfCookie.HttpOnly != true {
		t.Error("CSRF cookie should be HttpOnly")
	}

	// Step 2: Submit authorization form
	form := url.Values{
		"client_id":    {"test"},
		"state":        {"xyz"},
		"redirect_uri": {"http://localhost:3000/callback"},
		"csrf_state":   {csrfCookie.Value},
	}

	req = httptest.NewRequest("POST", "/rtm/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)

	w = httptest.NewRecorder()
	adapter.HandleAuthorize(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify frob was requested
	if mockClient.GetFrobCalls != 1 {
		t.Errorf("Expected 1 GetFrob call, got %d", mockClient.GetFrobCalls)
	}

	// Extract code from response (would be in the HTML)
	// For testing, we'll check the session was created
	var code string
	for c, session := range adapter.sessions {
		if session.Frob == mockClient.FrobValue {
			code = c
			break
		}
	}

	if code == "" {
		t.Fatal("No session created with frob")
	}

	// Step 3: Check auth status (simulate polling)
	req = httptest.NewRequest("GET", "/rtm/check-auth?code="+code, nil)
	w = httptest.NewRecorder()

	adapter.HandleCheckAuth(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var checkResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&checkResult); err != nil {
		t.Fatalf("Failed to decode check response: %v", err)
	}

	if checkResult["authorized"] != true {
		t.Error("Expected authorized=true after token exchange")
	}

	// Step 4: Exchange code for token
	tokenForm := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	req = httptest.NewRequest("POST", "/rtm/token", strings.NewReader(tokenForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w = httptest.NewRecorder()
	adapter.HandleToken(w, req)

	resp = w.Result()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, body)
	}

	var tokenResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		t.Fatalf("Failed to decode token response: %v", err)
	}

	if tokenResp["access_token"] != mockClient.TokenValue {
		t.Errorf("Expected access_token=%s, got %v", mockClient.TokenValue, tokenResp["access_token"])
	}

	// Verify session was cleaned up
	if adapter.GetSession(code) != nil {
		t.Error("Session should be removed after token exchange")
	}
}

// TestCSRFProtection tests CSRF token validation
func TestCSRFProtection(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	adapter.SetClient(NewMockRTMClient())

	// Try to submit form without CSRF token
	form := url.Values{
		"client_id":    {"test"},
		"state":        {"xyz"},
		"redirect_uri": {"http://localhost:3000/callback"},
		"csrf_state":   {"wrong-token"},
	}

	req := httptest.NewRequest("POST", "/rtm/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// No cookie set

	w := httptest.NewRecorder()
	adapter.HandleAuthorize(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing CSRF cookie, got %d", resp.StatusCode)
	}
}

// TestPKCEValidation tests PKCE code challenge validation
func TestPKCEValidation(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	mockClient := NewMockRTMClient()
	adapter.SetClient(mockClient)

	// Create session with PKCE
	codeVerifier := "test-verifier-123456789012345678901234567890123"
	codeChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" // SHA256 of verifier

	session := &AuthSession{
		Code:                "test-code",
		Frob:                "test-frob",
		CreatedAt:           time.Now(),
		Token:               "test-token",
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
	}

	adapter.sessions["test-code"] = session

	// Try with wrong verifier
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"test-code"},
		"code_verifier": {"wrong-verifier"},
	}

	req := httptest.NewRequest("POST", "/rtm/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	adapter.HandleToken(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid PKCE, got %d", w.Code)
	}

	// Try with correct verifier
	form.Set("code_verifier", codeVerifier)
	req = httptest.NewRequest("POST", "/rtm/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w = httptest.NewRecorder()
	adapter.HandleToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 for valid PKCE, got %d", w.Code)
	}
}

// TestAuthorizationTimeout tests handling of authorization timeout
func TestAuthorizationTimeout(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	mockClient := NewMockRTMClient()
	mockClient.ShouldFailGetToken = true // Simulate user not authorizing
	adapter.SetClient(mockClient)

	// Create an old session
	session := &AuthSession{
		Code:      "old-code",
		Frob:      "old-frob",
		CreatedAt: time.Now().Add(-56 * time.Minute), // Just past expiry
	}

	adapter.sessions["old-code"] = session

	// Try to check auth
	req := httptest.NewRequest("GET", "/rtm/check-auth?code=old-code", nil)
	w := httptest.NewRecorder()

	adapter.HandleCheckAuth(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	// Should indicate pending, not error (frob might still be valid for a few more minutes)
	if result["authorized"] == true {
		t.Error("Should not be authorized with expired session")
	}
}

// TestPollingMechanism tests the polling mechanism for token exchange
func TestPollingMechanism(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	mockClient := NewMockRTMClient()
	mockClient.ShouldFailGetToken = true // Start with failure
	adapter.SetClient(mockClient)

	// Create session
	session := &AuthSession{
		Code:      "poll-code",
		Frob:      "poll-frob",
		CreatedAt: time.Now(),
	}
	adapter.sessions["poll-code"] = session

	// First check - should be pending
	req := httptest.NewRequest("GET", "/rtm/check-auth?code=poll-code", nil)
	w := httptest.NewRecorder()
	adapter.HandleCheckAuth(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["pending"] != true {
		t.Error("Should be pending initially")
	}

	// Simulate user authorizing
	mockClient.ShouldFailGetToken = false

	// Second check - should be authorized
	req = httptest.NewRequest("GET", "/rtm/check-auth?code=poll-code", nil)
	w = httptest.NewRecorder()
	adapter.HandleCheckAuth(w, req)

	json.NewDecoder(w.Body).Decode(&result)

	if result["authorized"] != true {
		t.Error("Should be authorized after user action")
	}

	// Verify token was stored in session
	if adapter.sessions["poll-code"].Token == "" {
		t.Error("Token should be stored in session")
	}
}

// TestSessionCleanup tests that sessions are properly cleaned up
func TestSessionCleanup(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		session := &AuthSession{
			Code:      fmt.Sprintf("code-%d", i),
			Frob:      fmt.Sprintf("frob-%d", i),
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
		}
		adapter.sessions[session.Code] = session
	}

	if len(adapter.sessions) != 5 {
		t.Errorf("Expected 5 sessions, got %d", len(adapter.sessions))
	}

	// Remove a specific session
	adapter.removeSession("code-2")

	if len(adapter.sessions) != 4 {
		t.Errorf("Expected 4 sessions after removal, got %d", len(adapter.sessions))
	}

	if adapter.GetSession("code-2") != nil {
		t.Error("Session code-2 should be removed")
	}
}

// TestValidateBearer tests bearer token validation
func TestValidateBearer(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")
	mockClient := NewMockRTMClient()
	mockClient.TokenValue = "valid-token"
	adapter.SetClient(mockClient)

	// Test with valid token
	if !adapter.ValidateBearer("valid-token") {
		t.Error("Should validate valid token")
	}

	// Test with empty token
	if adapter.ValidateBearer("") {
		t.Error("Should not validate empty token")
	}

	// Test with invalid token (will fail GetLists)
	mockClient.TokenValue = ""
	if adapter.ValidateBearer("invalid-token") {
		t.Error("Should not validate invalid token")
	}
}
