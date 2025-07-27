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

func TestOAuthAdapter_AuthorizeFlow(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)

	t.Run("AuthorizeEndpoint_ShowsForm_When_GETRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=abc123", nil)
		w := httptest.NewRecorder()

		adapter.HandleAuthorize(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "Connect Remember The Milk") {
			t.Error("Missing page title")
		}
		if !strings.Contains(body, "csrf_state") {
			t.Error("Missing CSRF token field")
		}
		if !strings.Contains(body, "abc123") {
			t.Error("Missing client state")
		}
	})

	t.Run("AuthorizeEndpoint_RejectsInvalidCSRF_When_POSTWithBadToken", func(t *testing.T) {
		// Submit form without cookie (missing CSRF)
		form := url.Values{}
		form.Add("client_id", "test-client")
		form.Add("redirect_uri", "http://localhost/callback")
		form.Add("csrf_state", "invalid-token")
		form.Add("api_key", "test-key")

		req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		adapter.HandleAuthorize(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing CSRF cookie, got %d", w.Code)
		}
	})

	t.Run("AuthorizeEndpoint_GeneratesAuthCode_When_ValidFormSubmitted", func(t *testing.T) {
		// First GET to obtain CSRF cookie
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=test-client&redirect_uri=http://localhost/callback&state=abc123", nil)
		w := httptest.NewRecorder()
		adapter.HandleAuthorize(w, req)

		// Extract CSRF token from response body
		body := w.Body.String()

		// Use regex to extract CSRF token value (handles whitespace)
		re := regexp.MustCompile(`name="csrf_state"\s+value="([^"]+)"`)
		matches := re.FindStringSubmatch(body)
		if len(matches) < 2 {
			t.Fatalf("Could not find csrf_state field in form")
		}
		csrfToken := matches[1]

		// Extract CSRF cookie
		cookies := w.Result().Cookies()
		t.Logf("Cookies from GET response: %v", cookies)
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

		// Submit form with CSRF token
		form := url.Values{}
		form.Add("client_id", "test-client")
		form.Add("redirect_uri", "http://localhost/callback")
		form.Add("client_state", "abc123")
		form.Add("csrf_state", csrfToken)
		form.Add("api_key", "test-rtm-key")

		req = httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(csrfCookie) // Add the CSRF cookie
		w = httptest.NewRecorder()

		t.Logf("CSRF token from form: %s", csrfToken)
		t.Logf("Form values: %v", form)

		adapter.HandleAuthorize(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("Expected 302 redirect, got %d, body: %s", w.Code, w.Body.String())
		}

		location := w.Header().Get("Location")
		u, _ := url.Parse(location)

		if u.Query().Get("code") == "" {
			t.Error("Missing auth code in redirect")
		}
		if u.Query().Get("state") != "abc123" {
			t.Error("Client state not preserved")
		}
	})
}

func TestOAuthAdapter_TokenExchange(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)

	// Generate auth code
	authCode := &AuthCode{
		Code:      "test-code",
		RTMAPIKey: "test-rtm-key",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	adapter.authCodes["test-code"] = authCode

	t.Run("TokenEndpoint_IssuesAccessToken_When_ValidCodeProvided", func(t *testing.T) {
		form := url.Values{}
		form.Add("grant_type", "authorization_code")
		form.Add("code", "test-code")

		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		adapter.HandleToken(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		// Check token was issued
		body := w.Body.String()
		if !strings.Contains(body, "access_token") {
			t.Error("Missing access token")
		}

		// Check auth code was consumed
		if _, exists := adapter.authCodes["test-code"]; exists {
			t.Error("Auth code not deleted after use")
		}
	})

	t.Run("TokenEndpoint_RejectsExpiredCode_When_CodeExpired", func(t *testing.T) {
		// Add expired code
		adapter.authCodes["expired-code"] = &AuthCode{
			Code:      "expired-code",
			RTMAPIKey: "test-key",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}

		form := url.Values{}
		form.Add("grant_type", "authorization_code")
		form.Add("code", "expired-code")

		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		adapter.HandleToken(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for expired code, got %d", w.Code)
		}
	})
}

func TestOAuthAdapter_TokenValidation(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)

	// Store a test token
	adapter.tokenStore.Store("valid-token", "test-rtm-key")

	tests := []struct {
		name       string
		authHeader string
		wantErr    bool
		wantKey    string
	}{
		{
			name:       "ValidateToken_ReturnsAPIKey_When_ValidBearerToken",
			authHeader: "Bearer valid-token",
			wantErr:    false,
			wantKey:    "test-rtm-key",
		},
		{
			name:       "ValidateToken_ReturnsError_When_InvalidToken",
			authHeader: "Bearer invalid-token",
			wantErr:    true,
		},
		{
			name:       "ValidateToken_ReturnsError_When_MissingBearerPrefix",
			authHeader: "valid-token",
			wantErr:    true,
		},
		{
			name:       "ValidateToken_ReturnsError_When_EmptyHeader",
			authHeader: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiKey, err := adapter.ValidateToken(tt.authHeader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && apiKey != tt.wantKey {
				t.Errorf("ValidateToken() apiKey = %v, want %v", apiKey, tt.wantKey)
			}
		})
	}
}

func TestOAuthMiddleware(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)
	adapter.tokenStore.Store("valid-token", "test-rtm-key")

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-RTM-API-Key")
		_, _ = w.Write([]byte("API Key: " + apiKey))
	})

	// Wrap with middleware
	protected := Middleware(adapter)(testHandler)

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
		wantAPIKey bool
	}{
		{
			name:       "Middleware_AllowsAccess_When_ValidTokenProvided",
			path:       "/mcp",
			authHeader: "Bearer valid-token",
			wantStatus: http.StatusOK,
			wantAPIKey: true,
		},
		{
			name:       "Middleware_Returns401_When_NoTokenProvided",
			path:       "/mcp",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantAPIKey: false,
		},
		{
			name:       "Middleware_SkipsAuth_When_OAuthEndpoint",
			path:       "/oauth/authorize",
			authHeader: "",
			wantStatus: http.StatusOK,
			wantAPIKey: false,
		},
		{
			name:       "Middleware_SkipsAuth_When_WellKnownEndpoint",
			path:       "/.well-known/oauth-authorization-server",
			authHeader: "",
			wantStatus: http.StatusOK,
			wantAPIKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			protected.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantAPIKey {
				body := w.Body.String()
				if !strings.Contains(body, "test-rtm-key") {
					t.Error("API key not passed through")
				}
			}
		})
	}
}
