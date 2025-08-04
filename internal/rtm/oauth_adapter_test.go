package rtm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRTMClient for testing implements RTMClientInterface
type MockRTMClient struct {
	GetFrobFunc   func() (string, error)
	GetTokenFunc  func(frob string) error
	AuthTokenMock string
	APIKey        string
	Secret        string
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
	m.AuthTokenMock = "test-auth-token"
	return nil
}

func (m *MockRTMClient) GetAPIKey() string {
	if m.APIKey != "" {
		return m.APIKey
	}
	return "test-key"
}

func (m *MockRTMClient) GetAuthToken() string {
	return m.AuthTokenMock
}

func (m *MockRTMClient) SetAuthToken(token string) {
	m.AuthTokenMock = token
}

func (m *MockRTMClient) Sign(params map[string]string) string {
	return "test-signature"
}

func (m *MockRTMClient) GetLists() ([]List, error) {
	return []List{}, nil
}

// (Your other tests remain the same...)

func TestFullOAuthFlow(t *testing.T) {
	adapter := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")

	mockClient := &MockRTMClient{
		GetFrobFunc: func() (string, error) {
			return "test-frob-123", nil
		},
		AuthTokenMock: "test-auth-token",
	}
	adapter.SetClient(mockClient)

	// Step 1: GET /authorize to show form
	req := httptest.NewRequest("GET", "/authorize?client_id=test&state=xyz&redirect_uri=http://localhost/callback", nil)
	w := httptest.NewRecorder()

	adapter.HandleAuthorize(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Connect Remember The Milk")

	var csrfToken string
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfToken = cookie.Value
			break
		}
	}
	require.NotEmpty(t, csrfToken, "CSRF token should be set")

	// Step 2: POST /authorize with CSRF token and valid PKCE
	// Generate valid PKCE challenge/verifier pair
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	codeChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" // SHA256(codeVerifier) base64url encoded
	
	form := url.Values{}
	form.Set("client_id", "test")
	form.Set("state", "xyz")
	form.Set("redirect_uri", "http://localhost/callback")
	form.Set("csrf_state", csrfToken)
	form.Set("code_challenge", codeChallenge)
	form.Set("code_challenge_method", "S256")

	req2 := httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Cookie", fmt.Sprintf("csrf_token=%s", csrfToken))
	w2 := httptest.NewRecorder()

	adapter.HandleAuthorize(w2, req2)

	// Should show intermediate page
	assert.Equal(t, http.StatusOK, w2.Code)
	body := w2.Body.String()
	assert.Contains(t, body, "Connect to Remember The Milk")

	// Extract code from intermediate page using a robust regex
	re := regexp.MustCompile(`check-auth\?code=([a-f0-9-]+)`)
	matches := re.FindStringSubmatch(body)
	require.Len(t, matches, 2, "should find the code in the intermediate page")
	code := matches[1]

	// Step 3: Simulate successful auth by setting the token in the session
	// This simulates the user authorizing and the polling detecting it.
	adapter.sessionMutex.Lock()
	session, exists := adapter.sessions[code]
	require.True(t, exists, "session for the code should exist")
	session.Token = "test-token"
	adapter.sessionMutex.Unlock()

	// Step 4: Token exchange with valid PKCE verifier
	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("code_verifier", codeVerifier) // Use the valid verifier

	req3 := httptest.NewRequest("POST", "/token", strings.NewReader(tokenForm.Encode()))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w3 := httptest.NewRecorder()

	adapter.HandleToken(w3, req3)

	// Should return token
	assert.Equal(t, http.StatusOK, w3.Code)
	var tokenResponse map[string]interface{}
	err := json.Unmarshal(w3.Body.Bytes(), &tokenResponse)
	require.NoError(t, err)
	assert.Equal(t, "test-token", tokenResponse["access_token"])
	assert.Equal(t, "Bearer", tokenResponse["token_type"])
}

func TestCSRFCookieSettings(t *testing.T) {
	// Test with HTTPS URL - expects SameSiteNone and Secure
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
	assert.True(t, csrfCookie.Secure, "Secure flag should be set for HTTPS")
	assert.True(t, csrfCookie.HttpOnly, "HttpOnly flag should be set")
	assert.Equal(t, 1800, csrfCookie.MaxAge, "MaxAge should be 30 minutes")
	assert.Equal(t, "/", csrfCookie.Path, "Path should be /")

	// Test with HTTP URL - expects SameSiteNone but no Secure (for test environment)
	adapter2 := NewOAuthAdapter("test-key", "test-secret", "http://localhost:8080")

	req2 := httptest.NewRequest("GET", "/authorize?client_id=test&state=xyz&redirect_uri=http://localhost/callback", nil)
	w2 := httptest.NewRecorder()

	adapter2.HandleAuthorize(w2, req2)

	cookies2 := w2.Result().Cookies()
	var csrfCookie2 *http.Cookie
	for _, c := range cookies2 {
		if c.Name == "csrf_token" {
			csrfCookie2 = c
			break
		}
	}

	require.NotNil(t, csrfCookie2, "CSRF cookie should be set")
	assert.Equal(t, http.SameSiteLaxMode, csrfCookie2.SameSite, "SameSite should be Lax for HTTP test environments")
	assert.False(t, csrfCookie2.Secure, "Secure flag should not be set for HTTP")
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
				GetTokenFunc:  tt.mockGetToken,
				AuthTokenMock: "test-auth-token",
			}
			adapter.SetClient(mockClient)

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
