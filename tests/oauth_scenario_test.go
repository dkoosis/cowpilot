// tests/oauth_scenario_test.go

package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/vcto/cowpilot/internal/auth"
)

func TestOauthEndToEndScenario(t *testing.T) {
	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("the o-auth flow completes successfully when all steps are executed correctly", func(t *testing.T) {
		client, _ := clientWithCookieJar()
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		resp, err := client.Get(testServer.URL + "/oauth/authorize?client_id=test&state=test-state")
		if err != nil {
			t.Fatalf("Failed to get auth form: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		csrfStart := strings.Index(string(body), `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(string(body)[csrfStart:], `"`)
		csrfToken := string(body)[csrfStart : csrfStart+csrfEnd]

		form := url.Values{"csrf_state": {csrfToken}, "api_key": {"test-api-key"}, "client_id": {"test"}, "client_state": {"test-state"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err = client.Do(req)
		if err != nil {
			t.Fatalf("Failed to submit auth form: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusFound {
			t.Fatalf("Expected redirect, got %d", resp.StatusCode)
		}
	})
}

func TestOauthErrorScenarios(t *testing.T) {
	adapter := auth.NewOAuthAdapter("http://localhost:8080", 9090)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/authorize", adapter.HandleAuthorize)
	mux.HandleFunc("/oauth/token", adapter.HandleToken)
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	t.Run("the token endpoint rejects an authorization code that has already been used", func(t *testing.T) {
		client, _ := clientWithCookieJar()
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		// Step 1: Get code
		resp, _ := client.Get(testServer.URL + "/oauth/authorize?client_id=test&state=test-state")
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		csrfStart := strings.Index(string(body), `name="csrf_state" value="`) + 26
		csrfEnd := strings.Index(string(body)[csrfStart:], `"`)
		csrfToken := string(body)[csrfStart : csrfStart+csrfEnd]

		form := url.Values{"csrf_state": {csrfToken}, "api_key": {"test-api-key"}, "client_id": {"test"}, "client_state": {"test-state"}}
		req, _ := http.NewRequest("POST", testServer.URL+"/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, _ = client.Do(req)
		_ = resp.Body.Close()

		location, _ := url.Parse(resp.Header.Get("Location"))
		authCode := location.Query().Get("code")

		// Step 2: First use (should succeed)
		tokenForm := url.Values{"grant_type": {"authorization_code"}, "code": {authCode}}
		resp, err := client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Token exchange failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("First use failed with status %d, expected 200 OK", resp.StatusCode)
		}
		_ = resp.Body.Close()

		// Step 3: Second use (should fail)
		resp, err = client.Post(testServer.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(tokenForm.Encode()))
		if err != nil {
			t.Fatalf("Second token exchange failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Code reuse should fail with 400 Bad Request, got %d", resp.StatusCode)
		}
	})
}
