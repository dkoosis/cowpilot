package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vcto/mcp-adapters/internal/rtm"
)

// MockRTMClient simulates RTM API behavior for testing the adapter.
type MockRTMClient struct {
	frobAuthorized map[string]bool
	mu             *sync.Mutex
}

var _ rtm.RTMClientInterface = (*MockRTMClient)(nil) // Verify interface compliance

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
	return &rtm.RTMError{Code: 101, Msg: "Invalid frob - did you authenticate?"}
}
func (m *MockRTMClient) AuthorizeFrob(frob string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.frobAuthorized[frob] = true
}
func (m *MockRTMClient) GetAPIKey() string                    { return "test-key" }
func (m *MockRTMClient) GetAuthToken() string                 { return "test-auth-token" }
func (m *MockRTMClient) SetAuthToken(token string)            {}
func (m *MockRTMClient) Sign(params map[string]string) string { return "mock-signature" }
func (m *MockRTMClient) GetLists() ([]rtm.List, error)        { return []rtm.List{}, nil }

func TestRtmOauthFlow(t *testing.T) {
	// Importance: This suite tests the complex, custom OAuth adapter for Remember The Milk.
	// RTM does not use standard OAuth, so this adapter is critical for bridging RTM's legacy
	// authentication with the modern flow expected by clients like Claude.ai.

	adapter := rtm.NewOAuthAdapter("test-key", "test-secret", "https://rtm.fly.dev")
	mockClient := &MockRTMClient{frobAuthorized: make(map[string]bool), mu: &sync.Mutex{}}
	adapter.SetClient(mockClient)

	t.Run("handles the race condition when user proceeds without authorizing on RTM", func(t *testing.T) {
		t.Logf("  > Why it's important: This simulates a common user error. The system must fail gracefully with a clear 'authorization_pending' error, rather than crashing or getting stuck.")
		form := url.Values{"client_id": {"test"}, "state": {"test"}, "redirect_uri": {"http://localhost/cb"}, "csrf_state": {"test-csrf"}}
		req := httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf"})
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)

		re := regexp.MustCompile(`check-auth\?code=([a-f0-9-]+)`)
		matches := re.FindStringSubmatch(w.Body.String())
		require.Len(t, matches, 2, "should find the code")
		code := matches[1]

		// User clicks "Continue" on our page *without* having clicked "Allow" on RTM's page.
		reqCallback := httptest.NewRequest("GET", "/rtm/callback?code="+code, nil)
		wCallback := httptest.NewRecorder()
		adapter.HandleCallback(wCallback, reqCallback)
		assert.Equal(t, http.StatusBadRequest, wCallback.Code)
		assert.Contains(t, wCallback.Body.String(), "Authorization not completed")
	})

	t.Run("succeeds when user authorizes RTM before token exchange", func(t *testing.T) {
		t.Logf("  > Why it's important: This is the 'happy path' for RTM auth, ensuring a user who correctly follows the steps can log in successfully.")
		form := url.Values{"client_id": {"test"}, "state": {"test"}, "redirect_uri": {"http://localhost/cb"}, "csrf_state": {"test-csrf"}}
		req := httptest.NewRequest("POST", "/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "test-csrf"})
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)
		re := regexp.MustCompile(`check-auth\?code=([a-f0-9-]+)`)
		matches := re.FindStringSubmatch(w.Body.String())
		require.Len(t, matches, 2, "should find the code")
		code := matches[1]
		session := adapter.GetSession(code)
		require.NotNil(t, session)

		// User now successfully authorizes on RTM's side.
		mockClient.AuthorizeFrob(session.Frob)

		// The client's polling via check-auth should now succeed.
		reqCheck := httptest.NewRequest("GET", "/rtm/check-auth?code="+code, nil)
		wCheck := httptest.NewRecorder()
		adapter.HandleCheckAuth(wCheck, reqCheck)
		var checkResponse map[string]interface{}
		err := json.Unmarshal(wCheck.Body.Bytes(), &checkResponse)
		require.NoError(t, err)
		assert.True(t, checkResponse["authorized"].(bool), "check-auth should confirm authorization")

		// The final token exchange should now succeed.
		tokenForm := url.Values{"grant_type": {"authorization_code"}, "code": {code}}
		reqToken := httptest.NewRequest("POST", "/token", strings.NewReader(tokenForm.Encode()))
		reqToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		wToken := httptest.NewRecorder()
		adapter.HandleToken(wToken, reqToken)
		assert.Equal(t, http.StatusOK, wToken.Code)
		var tokenResponse map[string]interface{}
		err = json.Unmarshal(wToken.Body.Bytes(), &tokenResponse)
		require.NoError(t, err)
		assert.NotEmpty(t, tokenResponse["access_token"], "access token should be returned")
	})
}
