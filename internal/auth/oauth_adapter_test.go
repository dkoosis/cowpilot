package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestOAuthAdapterAuthorizeFlow(t *testing.T) {
	t.Logf("Importance: This suite tests the first half of the OAuth2 flow (/authorize endpoint), covering user-facing form rendering and CSRF protection.")
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})

	t.Run("shows an HTML form on a GET request", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that the user is presented with the necessary UI to initiate the authentication process.")
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=abc123", nil)
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Connect Remember The Milk") || !strings.Contains(body, "csrf_state") {
			t.Error("Response body did not contain expected form elements")
		}
	})

	t.Run("rejects a POST request with an invalid CSRF token", func(t *testing.T) {
		t.Logf("  > Why it's important: A critical security test to ensure the server is protected against Cross-Site Request Forgery attacks.")
		form := url.Values{"csrf_state": {"invalid-token"}, "api_key": {"test-key"}}
		req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for missing CSRF cookie, got %d", w.Code)
		}
	})

	t.Run("generates an authorization code on a valid form submission", func(t *testing.T) {
		t.Logf("  > Why it's important: This is the successful path for the first leg of OAuth, ensuring a valid user submission results in an auth code.")
		// Step 1: GET to get a valid CSRF token and cookie
		reqGet := httptest.NewRequest("GET", "/oauth/authorize", nil)
		wGet := httptest.NewRecorder()
		adapter.HandleAuthorize(wGet, reqGet)
		csrfCookie := wGet.Result().Cookies()[0]
		re := regexp.MustCompile(`name="csrf_state"\s+value="([^"]+)"`)
		matches := re.FindStringSubmatch(wGet.Body.String())
		if len(matches) < 2 {
			t.Fatalf("Could not extract CSRF token from form")
		}
		csrfToken := matches[1]

		// Step 2: POST with the valid token and cookie
		form := url.Values{"csrf_state": {csrfToken}, "client_state": {"abc123"}, "api_key": {"test-key"}, "redirect_uri": {"http://localhost/cb"}}
		reqPost := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqPost.AddCookie(csrfCookie)
		wPost := httptest.NewRecorder()
		adapter.HandleAuthorize(wPost, reqPost)

		if wPost.Code != http.StatusFound {
			t.Errorf("Expected 302 Found redirect, got %d", wPost.Code)
		}
		location, err := url.Parse(wPost.Header().Get("Location"))
		if err != nil {
			t.Fatalf("Invalid redirect location: %v", err)
		}
		if location.Query().Get("code") == "" {
			t.Error("Redirect URL is missing the authorization code")
		}
		if location.Query().Get("state") != "abc123" {
			t.Error("Redirect URL is missing the client state")
		}
	})
}

func TestOAuthAdapterTokenFlow(t *testing.T) {
	t.Logf("Importance: This suite tests the second half of the OAuth2 flow (/token endpoint), ensuring authorization codes can be securely exchanged for access tokens.")
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})

	t.Run("issues an access token for a valid authorization code", func(t *testing.T) {
		t.Logf("  > Why it's important: The successful completion of the OAuth flow, verifying that a valid auth code can be exchanged for the actual access token.")
		authCode := &AuthCode{Code: "test-code", RTMAPIKey: "test-rtm-key", ExpiresAt: time.Now().Add(5 * time.Minute)}
		adapter.authCodes["test-code"] = authCode

		form := url.Values{"grant_type": {"authorization_code"}, "code": {"test-code"}}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		adapter.HandleToken(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "access_token") {
			t.Error("Response body is missing access_token")
		}
		if _, exists := adapter.authCodes["test-code"]; exists {
			t.Error("Authorization code was not consumed after use")
		}
	})

	t.Run("rejects an expired authorization code", func(t *testing.T) {
		t.Logf("  > Why it's important: A security test to ensure that old or stolen authorization codes have a limited lifetime and cannot be used indefinitely.")
		expiredCode := &AuthCode{Code: "expired-code", RTMAPIKey: "test-key", ExpiresAt: time.Now().Add(-1 * time.Hour)}
		adapter.authCodes["expired-code"] = expiredCode

		form := url.Values{"grant_type": {"authorization_code"}, "code": {"expired-code"}}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		adapter.HandleToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request for expired code, got %d", w.Code)
		}
	})
}

func TestOAuthAdapterMiddleware(t *testing.T) {
	t.Logf("Importance: This suite tests the authentication middleware that protects server resources, ensuring it correctly validates tokens and protects endpoints.")
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	protected := Middleware(adapter)(testHandler)

	t.Run("allows access with a valid token", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that legitimate, authenticated users can access protected resources.")
		adapter.tokenStore.Store("valid-token", "test-rtm-key")
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}
	})

	t.Run("rejects access without a token", func(t *testing.T) {
		t.Logf("  > Why it's important: The most basic security check, ensuring unauthenticated requests are denied.")
		req := httptest.NewRequest("GET", "/mcp", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("rejects access with an invalid token", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures that guessed, malformed, or revoked tokens do not grant access.")
		req := httptest.NewRequest("GET", "/mcp", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 Unauthorized, got %d", w.Code)
		}
	})

	t.Run("skips authentication for well-known discovery endpoints", func(t *testing.T) {
		t.Logf("  > Why it's important: OAuth discovery endpoints must be public so clients can learn how to authenticate. This test ensures they are not incorrectly protected.")
		req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
		w := httptest.NewRecorder()
		protected.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK for well-known endpoint, got %d", w.Code)
		}
	})
}
