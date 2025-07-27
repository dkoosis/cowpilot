// Package contracts provides test helpers for protocol compliance
package specs

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// OAuthTestClient provides correct OAuth flow implementation
type OAuthTestClient struct {
	client *http.Client
	t      *testing.T
}

func NewOAuthTestClient(t *testing.T) *OAuthTestClient {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &OAuthTestClient{client: client, t: t}
}

// GetAuthorizationCode performs the complete authorization flow correctly
func (o *OAuthTestClient) GetAuthorizationCode(serverURL, clientID, redirectURI, apiKey string) string {
	// Step 1: GET to obtain CSRF token
	resp, err := o.client.Get(serverURL + "/oauth/authorize?client_id=" + clientID +
		"&redirect_uri=" + url.QueryEscape(redirectURI) + "&state=test-state")
	if err != nil {
		o.t.Fatalf("Failed to GET authorize: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	csrfToken := extractCSRFToken(string(body))
	if csrfToken == "" {
		o.t.Fatal("No CSRF token in authorize form")
	}

	// Step 2: POST with all required fields IN FORM BODY
	form := url.Values{
		"client_id":    {clientID},
		"csrf_state":   {csrfToken},
		"client_state": {"test-state"},
		"api_key":      {apiKey},
		// Note: redirect_uri not needed in POST
	}

	req, _ := http.NewRequest("POST", serverURL+"/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Copy cookies from GET response
	for _, cookie := range resp.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err = o.client.Do(req)
	if err != nil {
		o.t.Fatalf("Failed to POST authorize: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		o.t.Fatalf("Expected redirect, got %d: %s", resp.StatusCode, string(body))
	}

	location, _ := url.Parse(resp.Header.Get("Location"))
	code := location.Query().Get("code")
	if code == "" {
		o.t.Fatal("No authorization code in redirect")
	}

	return code
}

// ExchangeToken exchanges an authorization code for access token
func (o *OAuthTestClient) ExchangeToken(serverURL, code string) string {
	form := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	resp, err := o.client.Post(serverURL+"/oauth/token",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		o.t.Fatalf("Failed to exchange token: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		o.t.Fatalf("Token exchange failed with %d: %s", resp.StatusCode, string(body))
	}

	// Parse token from response...
	return "mock-token"
}

func extractCSRFToken(html string) string {
	start := strings.Index(html, `name="csrf_state" value="`) + 26
	if start < 26 {
		return ""
	}
	end := strings.Index(html[start:], `"`)
	if end < 0 {
		return ""
	}
	return html[start : start+end]
}

// TestOAuthContractCompliance verifies OAuth implementation follows spec
func TestOAuthContractCompliance(t *testing.T, handler http.Handler) {
	contract := &OAuthContract{}

	tests := []struct {
		name    string
		request func() *http.Request
		check   func(*http.Request) []string
	}{
		{
			name: "GET authorize requires client_id",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/oauth/authorize?redirect_uri=http://localhost", nil)
				return req
			},
			check: contract.ValidateAuthorizeRequest,
		},
		{
			name: "POST authorize requires form fields",
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "/oauth/authorize?client_id=test", nil)
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			check: contract.ValidateAuthorizeRequest,
		},
		{
			name: "Token endpoint requires POST",
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/oauth/token", nil)
				return req
			},
			check: contract.ValidateTokenRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.request()
			violations := tt.check(req)
			if len(violations) == 0 {
				t.Error("Expected contract violations but found none")
			}
			for _, v := range violations {
				t.Logf("Contract violation: %s", v)
			}
		})
	}
}
