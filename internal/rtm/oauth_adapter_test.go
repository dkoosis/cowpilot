package rtm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRTMClient for testing
type MockRTMClient struct {
	*Client
	GetFrobFunc   func() (string, error)
	GetTokenFunc  func(frob string) error
	AuthTokenMock string
}

func (m *MockRTMClient) GetFrob() (string, error) {
	if m.GetFrobFunc != nil {
		return m.GetFrobFunc()
	}
	return "test-frob-123", nil
}

func (m *MockRTMClient) GetToken(frob string) error {
	if m.GetTokenFunc != nil {
		return m.GetTokenFunc(frob)
	}
	m.AuthToken = m.AuthTokenMock
	return nil
}

func TestCSRFCookieSettings(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Test GET /authorize sets CSRF cookie correctly
	req := httptest.NewRequest("GET", "/authorize?client_id=test&state=xyz&redirect_uri=http://localhost/callback", nil)
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)

	// Check CSRF cookie
	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			csrfCookie = c
			break
		}
	}

	require.NotNil(t, csrfCookie, "CSRF cookie should be set")
	assert.Equal(t, http.SameSiteNoneMode, csrfCookie.SameSite, "SameSite should be None for cross-site OAuth")
	assert.True(t, csrfCookie.Secure, "Secure flag should be set")
	assert.True(t, csrfCookie.HttpOnly, "HttpOnly flag should be set")
	assert.Equal(t, 1800, csrfCookie.MaxAge, "MaxAge should be 30 minutes")
	assert.Equal(t, "/", csrfCookie.Path, "Path should be /")
}

func TestAuthSessionLifecycle(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Create a session
	session := &AuthSession{
		Code:                uuid.New().String(),
		Frob:                "test-frob",
		CreatedAt:           time.Now(),
		State:               "test-state",
		RedirectURI:         "http://localhost/callback",
		ClientID:            "test-client",
		CodeChallenge:       "test-challenge",
		CodeChallengeMethod: "S256",
	}

	// Store session
	adapter.sessionMutex.Lock()
	adapter.sessions[session.Code] = session
	adapter.sessionMutex.Unlock()

	// Verify exists
	adapter.sessionMutex.RLock()
	stored, exists := adapter.sessions[session.Code]
	adapter.sessionMutex.RUnlock()

	assert.True(t, exists, "Session should exist")
	assert.Equal(t, session.Frob, stored.Frob)
	assert.Equal(t, session.State, stored.State)

	// Remove session
	adapter.removeSession(session.Code)

	// Verify removed
	adapter.sessionMutex.RLock()
	_, exists = adapter.sessions[session.Code]
	adapter.sessionMutex.RUnlock()

	assert.False(t, exists, "Session should be removed")
}

func TestCheckAuthStates(t *testing.T) {
	tests := []struct {
		name            string
		setupSession    func() *AuthSession
		mockGetToken    func(frob string) error
		expectedStatus  int
		expectedAuth    bool
		expectedPending bool
		expectedError   string
	}{
		{
			name: "successful authorization",
			setupSession: func() *AuthSession {
				return &AuthSession{
					Code: "test-code",
					Frob: "test-frob",
				}
			},
			mockGetToken: func(frob string) error {
				return nil // Success
			},
			expectedStatus:  http.StatusOK,
			expectedAuth:    true,
			expectedPending: false,
		},
		{
			name: "pending authorization",
			setupSession: func() *AuthSession {
				return &AuthSession{
					Code: "test-code",
					Frob: "test-frob",
				}
			},
			mockGetToken: func(frob string) error {
				return &RTMError{Code: 101, Msg: "Invalid frob - did you authenticate?"}
			},
			expectedStatus:  http.StatusOK,
			expectedAuth:    false,
			expectedPending: true,
		},
		{
			name: "already has token",
			setupSession: func() *AuthSession {
				return &AuthSession{
					Code:  "test-code",
					Frob:  "test-frob",
					Token: "existing-token",
				}
			},
			mockGetToken:    nil, // Won't be called
			expectedStatus:  http.StatusOK,
			expectedAuth:    true,
			expectedPending: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

			// Mock the client
			mockClient := &MockRTMClient{
				Client:        adapter.client,
				GetTokenFunc:  tt.mockGetToken,
				AuthTokenMock: "test-auth-token",
			}
			adapter.client = mockClient.Client

			// Setup session
			session := tt.setupSession()
			adapter.sessionMutex.Lock()
			adapter.sessions[session.Code] = session
			adapter.sessionMutex.Unlock()

			// Test check-auth endpoint
			req := httptest.NewRequest("GET", "/rtm/check-auth?code="+session.Code, nil)
			w := httptest.NewRecorder()

			adapter.HandleCheckAuth(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedAuth, response["authorized"])
			if tt.expectedPending {
				assert.Equal(t, tt.expectedPending, response["pending"])
			}
			if tt.expectedError != "" {
				assert.Contains(t, response["error"], tt.expectedError)
			}
		})
	}
}

func TestPKCEValidation(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Test valid PKCE
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	valid := adapter.validatePKCE(codeChallenge, codeVerifier)
	assert.True(t, valid, "Valid PKCE should pass")

	// Test invalid PKCE
	invalidVerifier := "wrong-verifier"
	valid = adapter.validatePKCE(codeChallenge, invalidVerifier)
	assert.False(t, valid, "Invalid PKCE should fail")
}

func TestFullOAuthFlow(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Mock RTM client
	mockClient := &Client{
		APIKey:    "test-key",
		Secret:    "test-secret",
		AuthToken: "",
	}
	adapter.client = mockClient

	// Step 1: GET /authorize to show form
	req := httptest.NewRequest("GET", "/authorize?client_id=test&state=xyz&redirect_uri=http://localhost/callback", nil)
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Connect Remember The Milk")

	// Get CSRF token from cookie
	var csrfToken string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfToken = cookie.Value
			break
		}
	}
	require.NotEmpty(t, csrfToken, "CSRF token should be set")

	// Step 2: POST /authorize with CSRF token
	form := url.Values{}
	form.Set("client_id", "test")
	form.Set("state", "xyz")
	form.Set("redirect_uri", "http://localhost/callback")
	form.Set("csrf_state", csrfToken)
	form.Set("code_challenge", "test-challenge")
	form.Set("code_challenge_method", "S256")

	req = httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfToken})
	w = httptest.NewRecorder()

	// Mock GetFrob
	adapter.client = &Client{
		APIKey: "test-key",
		Secret: "test-secret",
	}

	adapter.HandleAuthorize(w, req)

	// Should show intermediate page
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Connect to Remember The Milk")
	assert.Contains(t, body, "Authorize at RTM")

	// Extract code from intermediate page
	codeStart := strings.Index(body, "code=")
	require.NotEqual(t, -1, codeStart, "Code should be in page")
	codeEnd := strings.Index(body[codeStart:], "&")
	if codeEnd == -1 {
		codeEnd = strings.Index(body[codeStart:], "\"")
	}
	code := body[codeStart+5 : codeStart+codeEnd]

	// Step 3: Simulate polling check-auth
	req = httptest.NewRequest("GET", "/rtm/check-auth?code="+code, nil)
	w = httptest.NewRecorder()

	adapter.HandleCheckAuth(w, req)

	// Should be pending
	var checkResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &checkResponse)
	require.NoError(t, err)
	assert.False(t, checkResponse["authorized"].(bool))
	assert.True(t, checkResponse["pending"].(bool))

	// Step 4: Simulate successful auth by setting token
	adapter.sessionMutex.Lock()
	session := adapter.sessions[code]
	session.Token = "test-token"
	adapter.sessionMutex.Unlock()

	// Step 5: Token exchange
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("code_verifier", "test-verifier")

	req = httptest.NewRequest("POST", "/token", strings.NewReader(tokenForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	adapter.HandleToken(w, req)

	// Should return token
	assert.Equal(t, http.StatusOK, w.Code)
	var tokenResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &tokenResponse)
	require.NoError(t, err)
	assert.Equal(t, "test-token", tokenResponse["access_token"])
	assert.Equal(t, "Bearer", tokenResponse["token_type"])
}

func TestConcurrentSessions(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Create multiple sessions concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			session := &AuthSession{
				Code:        uuid.New().String(),
				Frob:        "frob-" + string(rune(id)),
				CreatedAt:   time.Now(),
				State:       "state-" + string(rune(id)),
				RedirectURI: "http://localhost/callback",
			}

			adapter.sessionMutex.Lock()
			adapter.sessions[session.Code] = session
			adapter.sessionMutex.Unlock()

			// Simulate some work
			time.Sleep(10 * time.Millisecond)

			// Read session
			adapter.sessionMutex.RLock()
			_, exists := adapter.sessions[session.Code]
			adapter.sessionMutex.RUnlock()

			assert.True(t, exists)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all sessions exist
	adapter.sessionMutex.RLock()
	sessionCount := len(adapter.sessions)
	adapter.sessionMutex.RUnlock()

	assert.Equal(t, 10, sessionCount, "All sessions should be stored")
}

func TestAuthorizationPendingRetry(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")

	// Create session without token
	session := &AuthSession{
		Code:        "test-code",
		Frob:        "test-frob",
		CreatedAt:   time.Now(),
		RedirectURI: "http://localhost/callback",
		State:       "test-state",
	}

	adapter.sessionMutex.Lock()
	adapter.sessions[session.Code] = session
	adapter.sessionMutex.Unlock()

	// First token request - should return pending
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", session.Code)

	req := httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	adapter.HandleToken(w, req)

	// Should return authorization_pending error
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	assert.Equal(t, "authorization_pending", errorResponse["error"])

	// Simulate successful auth
	adapter.sessionMutex.Lock()
	adapter.sessions[session.Code].Token = "success-token"
	adapter.sessionMutex.Unlock()

	// Second token request - should succeed
	req = httptest.NewRequest("POST", "/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()

	adapter.HandleToken(w, req)

	// Should return token
	assert.Equal(t, http.StatusOK, w.Code)
	var tokenResponse map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &tokenResponse)
	require.NoError(t, err)
	assert.Equal(t, "success-token", tokenResponse["access_token"])
}
