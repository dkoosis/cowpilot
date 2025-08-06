package integration

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/vcto/mcp-adapters/internal/auth"
)

// clientWithCookieJar is a helper to create an HTTP client that persists cookies,
// which is essential for a stateful multi-step process like an OAuth flow.
func clientWithCookieJar() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}
}

func TestGenericOAuthFlow(t *testing.T) {
	// Importance: This suite simulates the full, multi-step authentication flow for the generic
	// (non-RTM) OAuth adapter. It verifies that CSRF protection, code generation, and token
	// exchange all work together correctly. A failure here indicates a broken login process.

	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("completes successfully when all steps are correct", func(t *testing.T) {
		// Why it's important: This is the "happy path" that ensures a user can successfully
		// authenticate from start to finish.
		client := clientWithCookieJar()

		resp, err := client.Get(testServer.URL + "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=test-state")
		if err != nil {
			t.Fatalf("Failed to get auth form: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		re := regexp.MustCompile(`name="csrf_state"\s+value="([^"]+)"`)
		matches := re.FindStringSubmatch(string(body))
		if len(matches) < 2 {
			t.Fatalf("Could not find csrf_state field in form. HTML was: %s", string(body))
		}
		csrfToken := matches[1]

		form := url.Values{"csrf_state": {csrfToken}, "api_key": {"test-api-key"}, "client_id": {"test"}, "client_state": {"test-state"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp2, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth form: %v", err)
		}
		defer func() { _ = resp2.Body.Close() }()

		if resp2.StatusCode != http.StatusFound {
			body, _ := io.ReadAll(resp2.Body)
			t.Fatalf("Expected redirect after form submission, got %d: %s", resp2.StatusCode, string(body))
		}
	})

	t.Run("rejects an authorization code that has already been used", func(t *testing.T) {
		// Why it's important: This is a critical security test to prevent "replay attacks,"
		// where an attacker could reuse an intercepted authorization code to gain access.
		client := clientWithCookieJar()

		// Step 1: Get a valid authorization code
		resp, _ := client.Get(testServer.URL + "/oauth/authorize?client_id=test&redirect_uri=http://localhost/callback&state=test-state")
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		re := regexp.MustCompile(`name="csrf_state"\s+value="([^"]+)"`)
		matches := re.FindStringSubmatch(string(body))
		csrfToken := matches[1]
		form := url.Values{"csrf_state": {csrfToken}, "api_key": {"test-api-key"}, "client_id": {"test"}, "client_state": {"test-state"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, _ = client.Do(req)
		_ = resp.Body.Close()
		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")

		// Step 2: Use the code for the first time (should succeed)
		tokenForm := url.Values{"grant_type": {"authorization_code"}, "code": {authCode}}
		resp, err := client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("First token exchange failed unexpectedly: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("First token use failed with status %d, expected 200 OK", resp.StatusCode)
		}
		_ = resp.Body.Close()

		// Step 3: Use the same code again (should fail)
		resp, err = client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Second token exchange request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Code reuse should fail with 400 Bad Request, but got %d", resp.StatusCode)
		}
	})
}
