package rtm

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// mockRTMValidator provides a mock validator for testing without real API calls.
func mockRTMValidator(shouldSucceed bool) func(apiKey, secret string) error {
	return func(apiKey, secret string) error {
		if shouldSucceed {
			return nil
		}
		return fmt.Errorf("mock validation failed: invalid credentials")
	}
}

// testSetupHandler wraps the real handler to inject a mock validator.
type testSetupHandler struct {
	*SetupHandler
}

// newTestSetupHandler creates a new setup handler with a mock validator.
func newTestSetupHandler(store CredentialStore, validator func(apiKey, secret string) error) *testSetupHandler {
	return &testSetupHandler{
		SetupHandler: &SetupHandler{
			store:     store,
			validator: validator,
		},
	}
}

func TestSetupHandlerGET(t *testing.T) {
	t.Logf("Importance: Verifies that the user-facing setup form is rendered correctly, which is the first step in the credential configuration process.")

	t.Run("renders the credential setup form", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures that users are presented with the necessary HTML form to input their API credentials.")
		handler := NewSetupHandler()
		req := httptest.NewRequest("GET", "/rtm/setup", nil)
		w := httptest.NewRecorder()

		handler.HandleSetup(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}

		body := w.Body.String()
		expectedElements := []string{"RTM Setup", "API Key:", "API Secret:", "form method=\"POST\""}
		for _, element := range expectedElements {
			if !strings.Contains(body, element) {
				t.Errorf("Expected form to contain '%s', but it was missing", element)
			}
		}
	})
}

func TestSetupHandlerPOST(t *testing.T) {
	t.Logf("Importance: This suite tests the server's logic for processing and validating the submitted RTM credentials, a critical step for enabling the integration.")

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_creds.db")
	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create credential store for testing: %v", err)
	}
	defer func() { _ = store.Close() }()

	t.Run("succeeds with valid credentials and stores them", func(t *testing.T) {
		t.Logf("  > Why it's important: This is the 'happy path' test, ensuring that correct, validated credentials are securely stored and the user gets a success message.")
		handler := newTestSetupHandler(store, mockRTMValidator(true)) // Mock validator that succeeds
		form := url.Values{"api_key": {"valid_api_key_12345"}, "secret": {"valid_secret_67890"}}
		req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "test_user_valid"
		w := httptest.NewRecorder()

		handler.HandleSetup(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Setup Complete") {
			t.Error("Expected success page on valid submission")
		}

		// Verify credentials were stored
		apiKey, secret, err := store.Retrieve("test_user_valid")
		if err != nil || apiKey != "valid_api_key_12345" || secret != "valid_secret_67890" {
			t.Error("Credentials were not stored correctly after successful validation")
		}
	})

	t.Run("fails with invalid RTM credentials", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the system rejects incorrect API credentials by showing a clear error, preventing users from proceeding with a broken configuration.")
		handler := newTestSetupHandler(store, mockRTMValidator(false)) // Mock validator that fails
		form := url.Values{"api_key": {"invalid_api_key_12345"}, "secret": {"invalid_secret_67890"}}
		req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.HandleSetup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid RTM credentials") {
			t.Error("Expected error message for invalid credentials")
		}
	})

	t.Run("fails when required fields are missing", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies basic input validation, ensuring the server doesn't attempt to process incomplete form submissions.")
		handler := NewSetupHandler()
		form := url.Values{"api_key": {"only_key"}, "secret": {""}}
		req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.HandleSetup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "API key and secret are required") {
			t.Error("Expected error message for missing fields")
		}
	})

	t.Run("fails when credential storage is unavailable", func(t *testing.T) {
		t.Logf("  > Why it's important: Tests a failure condition where the backend storage is broken, ensuring the system fails gracefully instead of appearing to succeed.")
		handler := newTestSetupHandler(nil, mockRTMValidator(true)) // Nil store
		form := url.Values{"api_key": {"valid_key_but_no_store"}, "secret": {"valid_secret_but_no_store"}}
		req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		handler.HandleSetup(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Credential storage unavailable") {
			t.Error("Expected error message for storage failure")
		}
	})
}
