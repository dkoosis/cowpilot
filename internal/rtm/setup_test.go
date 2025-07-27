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

// Test helper to mock RTM client
type mockRTMClient struct {
	response string
	err      error
}

func (m *mockRTMClient) GetFrob() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// Test helper to inject mock client
type testSetupHandler struct {
	store      CredentialStore
	mockClient *mockRTMClient
}

// NewTestSetupHandler creates test handler with mock
func NewTestSetupHandler(store CredentialStore, mock *mockRTMClient) *testSetupHandler {
	return &testSetupHandler{
		store:      store,
		mockClient: mock,
	}
}

// HandleSetup implements the same interface but with mocked validation
func (h *testSetupHandler) HandleSetup(w http.ResponseWriter, r *http.Request) {
	setupHandler := &SetupHandler{store: h.store}

	if r.Method == "GET" {
		setupHandler.showSetupForm(w, r)
		return
	}

	if r.Method == "POST" {
		h.processSetupWithMock(w, r, setupHandler)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// processSetupWithMock handles POST with mocked validation
func (h *testSetupHandler) processSetupWithMock(w http.ResponseWriter, r *http.Request, setupHandler *SetupHandler) {
	if err := r.ParseForm(); err != nil {
		setupHandler.showError(w, "Invalid form data")
		return
	}

	apiKey := strings.TrimSpace(r.FormValue("api_key"))
	secret := strings.TrimSpace(r.FormValue("secret"))

	// Validate required fields
	if apiKey == "" || secret == "" {
		setupHandler.showError(w, "API key and secret are required")
		return
	}

	// Basic format validation
	if len(apiKey) < 10 || len(secret) < 10 {
		setupHandler.showError(w, "API key and secret appear to be too short")
		return
	}

	// Use mocked validation
	if err := h.validateRTMCredentials(apiKey, secret); err != nil {
		setupHandler.showError(w, fmt.Sprintf("Invalid RTM credentials: %v", err))
		return
	}

	// Store encrypted credentials
	if h.store == nil {
		setupHandler.showError(w, "Credential storage unavailable")
		return
	}

	// Use client IP as user ID for now
	userID := r.RemoteAddr
	if userID == "" {
		userID = "default_user"
	}

	if err := h.store.Store(userID, apiKey, secret); err != nil {
		setupHandler.showError(w, fmt.Sprintf("Failed to save credentials: %v", err))
		return
	}

	setupHandler.showSuccess(w, "Credentials validated and saved successfully!")
}

// validateRTMCredentials uses mock for testing
func (h *testSetupHandler) validateRTMCredentials(_, _ string) error {
	if h.mockClient != nil {
		_, err := h.mockClient.GetFrob()
		if err != nil {
			return fmt.Errorf("RTM API test failed: %w", err)
		}
		return nil
	}
	return fmt.Errorf("no mock client configured")
}

func TestSetupHandler_GET(t *testing.T) {
	handler := NewSetupHandler()
	req := httptest.NewRequest("GET", "/rtm/setup", nil)
	w := httptest.NewRecorder()

	handler.HandleSetup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	expectedElements := []string{
		"RTM Setup",
		"API Key:",
		"API Secret:",
		"Validate & Save Credentials",
		"form method=\"POST\"",
	}

	for _, element := range expectedElements {
		if !strings.Contains(body, element) {
			t.Errorf("Expected form to contain '%s'", element)
		}
	}
}

func TestSetupHandler_POST_ValidInput(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_creds.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	mock := &mockRTMClient{
		response: "test_frob_12345",
		err:      nil,
	}
	handler := NewTestSetupHandler(store, mock)

	form := url.Values{}
	form.Add("api_key", "test_api_key_12345")
	form.Add("secret", "test_secret_67890")

	req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	handler.HandleSetup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Setup Complete") {
		t.Error("Expected success page")
	}

	// Verify credentials were stored
	apiKey, secret, err := store.Retrieve("127.0.0.1:12345")
	if err != nil {
		t.Errorf("Credentials not stored: %v", err)
	}
	if apiKey != "test_api_key_12345" || secret != "test_secret_67890" {
		t.Error("Stored credentials don't match input")
	}
}

func TestSetupHandler_POST_StorageFailure(t *testing.T) {
	mock := &mockRTMClient{
		response: "test_frob_12345",
		err:      nil,
	}
	handler := NewTestSetupHandler(nil, mock) // nil store

	form := url.Values{}
	form.Add("api_key", "test_api_key_12345")
	form.Add("secret", "test_secret_67890")

	req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSetup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "storage unavailable") {
		t.Error("Expected storage error message")
	}
}

func TestSetupHandler_POST_InvalidRTMCredentials(t *testing.T) {
	testCases := []struct {
		name        string
		mockError   error
		expectedMsg string
	}{
		{
			name:        "invalid signature",
			mockError:   fmt.Errorf("RTM API error 98: Invalid signature"),
			expectedMsg: "Invalid RTM credentials",
		},
		{
			name:        "invalid api key",
			mockError:   fmt.Errorf("RTM API error 100: Invalid API Key"),
			expectedMsg: "Invalid RTM credentials",
		},
		{
			name:        "network error",
			mockError:   fmt.Errorf("HTTP request failed: connection refused"),
			expectedMsg: "Invalid RTM credentials",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockRTMClient{err: tc.mockError}
			handler := NewTestSetupHandler(nil, mock)

			form := url.Values{}
			form.Add("api_key", "invalid_key_12345")
			form.Add("secret", "invalid_secret_67890")

			req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.HandleSetup(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, tc.expectedMsg) {
				t.Errorf("Expected error message '%s', got: %s", tc.expectedMsg, body)
			}
		})
	}
}

func TestSetupHandler_POST_MissingFields(t *testing.T) {
	handler := NewSetupHandler()

	testCases := []struct {
		name   string
		apiKey string
		secret string
	}{
		{"missing api key", "", "test_secret"},
		{"missing secret", "test_api_key", ""},
		{"missing both", "", ""},
		{"whitespace only", "   ", "   "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := url.Values{}
			form.Add("api_key", tc.apiKey)
			form.Add("secret", tc.secret)

			req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.HandleSetup(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			body := w.Body.String()
			if !strings.Contains(body, "Setup Error") {
				t.Error("Expected error page")
			}
		})
	}
}

func TestSetupHandler_POST_ShortCredentials(t *testing.T) {
	handler := NewSetupHandler()

	form := url.Values{}
	form.Add("api_key", "short")    // Less than 10 chars
	form.Add("secret", "alsoshort") // Less than 10 chars

	req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSetup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "too short") {
		t.Error("Expected 'too short' error message")
	}
}

func TestSetupHandler_InvalidMethod(t *testing.T) {
	handler := NewSetupHandler()

	methods := []string{"PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/rtm/setup", nil)
			w := httptest.NewRecorder()

			handler.HandleSetup(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405 for %s, got %d", method, w.Code)
			}
		})
	}
}

func TestSetupHandler_InvalidFormData(t *testing.T) {
	handler := NewSetupHandler()

	// Send invalid form data
	req := httptest.NewRequest("POST", "/rtm/setup", strings.NewReader("invalid%form%data"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSetup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid form data") {
		t.Error("Expected 'Invalid form data' error message")
	}
}
