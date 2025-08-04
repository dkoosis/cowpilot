package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vcto/mcp-adapters/internal/rtm"
)

// MockRTMClient simulates RTM API behavior and now explicitly implements the interface
type MockRTMClient struct {
	frobAuthorized map[string]bool
	mu             *sync.Mutex
}

// Ensure MockRTMClient satisfies the RTMClientInterface
var _ rtm.RTMClientInterface = (*MockRTMClient)(nil)

func (m *MockRTMClient) GetFrob() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	frob := fmt.Sprintf("mock-frob-%d", time.Now().UnixNano())
	m.frobAuthorized[frob] = false
	return frob, nil
}

func (m *MockRTMClient) GetToken(frob string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if authorized, exists := m.frobAuthorized[frob]; exists && authorized {
		return nil
	}

	return &rtm.RTMError{
		Code: 101,
		Msg:  "Invalid frob - did you authenticate?",
	}
}

func (m *MockRTMClient) AuthorizeFrob(frob string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.frobAuthorized[frob] = true
}

func (m *MockRTMClient) GetAPIKey() string {
	return "test-key"
}

func (m *MockRTMClient) GetAuthToken() string {
	return "test-auth-token"
}

func (m *MockRTMClient) SetAuthToken(token string) {
	// No-op for mock
}

func (m *MockRTMClient) Sign(params map[string]string) string {
	return "mock-signature"
}

func (m *MockRTMClient) GetLists() ([]rtm.List, error) {
	return []rtm.List{}, nil
}

// TestRTMOAuthRaceCondition simulates the race condition where user never completes auth
func TestRTMOAuthRaceCondition(t *testing.T) {
	// Create adapter
	adapter := rtm.NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Create mock client that simulates RTM API behavior
	mockClient := &MockRTMClient{
		frobAuthorized: make(map[string]bool),
		mu:             &sync.Mutex{},
	}
	adapter.SetClient(mockClient)

	// Step 1: Start OAuth flow
	form := url.Values{}
	form.Set("client_id", "test-client")
	form.Set("state", "test-state")
	form.Set("redirect_uri", "http://localhost/callback")
	form.Set("csrf_state", "test-csrf")

	req := httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf"})
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)

	// Extract code from response
	body := w.Body.String()
	codeStart := strings.Index(body, "code=")
	require.NotEqual(t, -1, codeStart)
	codeEnd := strings.Index(body[codeStart:], "&")
	if codeEnd == -1 {
		codeEnd = strings.Index(body[codeStart:], "\"")
	}
	if codeEnd == -1 {
		codeEnd = strings.Index(body[codeStart:], "'")
	}
	if codeEnd == -1 {
		codeEnd = len(body[codeStart:])
	}
	code := body[codeStart+5 : codeStart+codeEnd]

	// Step 2: Simulate concurrent polling and user actions
	var wg sync.WaitGroup
	authCompleted := false
	pollResults := make([]bool, 0)
	pollErrors := make([]error, 0)

	// Start polling goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				req := httptest.NewRequest("GET", "/rtm/check-auth?code="+code, nil)
				w := httptest.NewRecorder()

				adapter.HandleCheckAuth(w, req)

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				if err != nil {
					pollErrors = append(pollErrors, err)
					continue
				}

				if auth, ok := response["authorized"].(bool); ok {
					pollResults = append(pollResults, auth)
					if auth {
						authCompleted = true
						return
					}
				}
			}
		}
	}()

	// Simulate user never clicking "Allow" on RTM
	// (In real scenario, user opens RTM page but doesn't complete auth)
	time.Sleep(3 * time.Second)

	// User clicks "Continue" without authorizing
	req = httptest.NewRequest("GET", "/rtm/callback?code="+code, nil)
	w = httptest.NewRecorder()

	adapter.HandleCallback(w, req)

	// Should fail because auth not completed
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization not completed")

	// Try token exchange - should get pending error
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)

	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	adapter.HandleToken(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var tokenError map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &tokenError)
	require.NoError(t, err)
	assert.Equal(t, "authorization_pending", tokenError["error"])

	// Wait for polling to complete
	wg.Wait()

	// Verify polling never succeeded
	assert.False(t, authCompleted)
	assert.NotEmpty(t, pollResults)
	for _, result := range pollResults {
		assert.False(t, result, "Polling should never succeed when user doesn't authorize")
	}
}

// TestRTMOAuthSuccessfulFlow tests the happy path
func TestRTMOAuthSuccessfulFlow(t *testing.T) {
	adapter := rtm.NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Create mock client
	mockClient := &MockRTMClient{
		frobAuthorized: make(map[string]bool),
		mu:             &sync.Mutex{},
	}
	adapter.SetClient(mockClient)

	// Start OAuth flow
	form := url.Values{}
	form.Set("client_id", "test-client")
	form.Set("state", "test-state")
	form.Set("redirect_uri", "http://localhost/callback")
	form.Set("csrf_state", "test-csrf")

	req := httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf"})
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)

	// Extract code and frob
	body := w.Body.String()
	codeStart := strings.Index(body, "code=")
	require.NotEqual(t, -1, codeStart)
	codeEnd := strings.Index(body[codeStart:], "&")
	if codeEnd == -1 {
		codeEnd = strings.Index(body[codeStart:], "\"")
	}
	if codeEnd == -1 {
		codeEnd = strings.Index(body[codeStart:], "'")
	}
	if codeEnd == -1 {
		codeEnd = len(body[codeStart:])
	}
	code := body[codeStart+5 : codeStart+codeEnd]

	// Get frob from session
	session := adapter.GetSession(code)
	require.NotNil(t, session)
	frob := session.Frob

	// Simulate user authorizing
	mockClient.AuthorizeFrob(frob)

	// Check auth status
	req = httptest.NewRequest("GET", "/rtm/check-auth?code="+code, nil)
	w = httptest.NewRecorder()

	adapter.HandleCheckAuth(w, req)

	var checkResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &checkResponse)
	require.NoError(t, err)
	assert.True(t, checkResponse["authorized"].(bool))

	// Token exchange should succeed
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)

	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	adapter.HandleToken(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var tokenResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &tokenResponse)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenResponse["access_token"])
}
