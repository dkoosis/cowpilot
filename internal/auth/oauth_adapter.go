package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// OAuthAdapter provides OAuth2 facade for RTM API key authentication
type OAuthAdapter struct {
	serverURL      string
	tokenStore     TokenStoreInterface
	authCodes      map[string]*AuthCode // Temporary auth codes
	callbackServer *OAuthCallbackServer
	callbackPort   int
}

type AuthCode struct {
	Code      string
	RTMAPIKey string
	ExpiresAt time.Time
}

// NewOAuthAdapter creates a new OAuth adapter
func NewOAuthAdapter(serverURL string, callbackPort int) *OAuthAdapter {
	adapter := &OAuthAdapter{
		serverURL:    serverURL,
		tokenStore:   CreateTokenStore(),
		authCodes:    make(map[string]*AuthCode),
		callbackPort: callbackPort,
	}
	adapter.callbackServer = NewOAuthCallbackServer(adapter, callbackPort)
	return adapter
}

// HandleProtectedResourceMetadata handles /.well-known/oauth-protected-resource
func (a *OAuthAdapter) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"resource":              a.serverURL,
		"authorization_servers": []string{a.serverURL},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleAuthServerMetadata handles /.well-known/oauth-authorization-server
func (a *OAuthAdapter) HandleAuthServerMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"issuer":                           a.serverURL,
		"authorization_endpoint":           a.serverURL + "/oauth/authorize",
		"token_endpoint":                   a.serverURL + "/oauth/token",
		"registration_endpoint":            a.serverURL + "/oauth/register",
		"response_types_supported":         []string{"code"},
		"grant_types_supported":            []string{"authorization_code"},
		"code_challenge_methods_supported": []string{"S256"},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleAuthorize handles /oauth/authorize
func (a *OAuthAdapter) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Extract parameters
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	clientState := r.URL.Query().Get("state") // Client's state parameter
	resource := r.URL.Query().Get("resource") // June 2025 spec

	// Generate CSRF token (stateless - just a UUID)
	csrfState := uuid.New().String()

	// For RTM adapter, show API key input form
	if r.Method == "GET" {
		// Set CSRF cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfState,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   600, // 10 minutes
		})

		html := fmt.Sprintf(`
		<html>
		<head><title>Connect Remember The Milk</title></head>
		<body>
			<h1>Connect Remember The Milk</h1>
			<form method="POST">
				<input type="hidden" name="client_id" value="%s">
				<input type="hidden" name="redirect_uri" value="%s">
				<input type="hidden" name="client_state" value="%s">
				<input type="hidden" name="csrf_state" value="%s">
				<input type="hidden" name="resource" value="%s">
				<label>RTM API Key: <input type="password" name="api_key" required></label>
				<button type="submit">Connect</button>
			</form>
		</body>
		</html>`, clientID, redirectURI, clientState, csrfState, resource)

		w.Header().Set("Content-Type", "text/html")
		if _, err := w.Write([]byte(html)); err != nil {
			// Log error but response already started
			fmt.Printf("Failed to write HTML response: %v\n", err)
		}
		return
	}

	// Handle form submission
	apiKey := r.FormValue("api_key")
	csrfState = r.FormValue("csrf_state")     // Use = not :=
	clientState = r.FormValue("client_state") // Use = not :=
	formRedirectURI := r.FormValue("redirect_uri")

	// Validate CSRF token from cookie
	cookie, err := r.Cookie("csrf_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Missing CSRF cookie", http.StatusBadRequest)
		return
	}

	// Verify the form token matches the cookie
	if csrfState != cookie.Value {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	if apiKey == "" {
		http.Error(w, "API key required", http.StatusBadRequest)
		return
	}

	// Generate auth code
	code := uuid.New().String()
	a.authCodes[code] = &AuthCode{
		Code:      code,
		RTMAPIKey: apiKey,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	// Clear CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	})

	// Redirect back with code
	u, _ := url.Parse(formRedirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", clientState) // Return client's original state
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

// HandleToken handles /oauth/token
func (a *OAuthAdapter) HandleToken(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	// resource := r.FormValue("resource") // June 2025 spec - TODO: use for validation

	if grantType != "authorization_code" {
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	// Validate auth code
	authCode, exists := a.authCodes[code]
	if !exists || time.Now().After(authCode.ExpiresAt) {
		http.Error(w, "Invalid or expired code", http.StatusBadRequest)
		return
	}

	// Generate bearer token
	token := uuid.New().String()
	a.tokenStore.Store(token, authCode.RTMAPIKey)

	// Clean up auth code
	delete(a.authCodes, code)

	// Return token response
	response := map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// HandleRegister handles /oauth/register (DCR)
func (a *OAuthAdapter) HandleRegister(w http.ResponseWriter, r *http.Request) {
	// Simple DCR implementation - accept any client
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	clientID := uuid.New().String()
	response := map[string]interface{}{
		"client_id":     clientID,
		"client_name":   req["client_name"],
		"redirect_uris": req["redirect_uris"],
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// ValidateToken checks if bearer token is valid and returns RTM API key
func (a *OAuthAdapter) ValidateToken(authHeader string) (string, error) {
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid auth header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	apiKey, exists := a.tokenStore.Get(token)
	if !exists {
		return "", fmt.Errorf("invalid token")
	}

	return apiKey, nil
}
