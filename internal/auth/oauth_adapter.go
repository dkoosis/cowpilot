package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
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

	// Only start the callback server in production (not during tests)
	// Tests should call StartCallbackServer() explicitly if needed
	if os.Getenv("GO_TEST") != "1" {
		if err := adapter.callbackServer.Start(context.Background()); err != nil {
			fmt.Printf("Warning: Failed to start OAuth callback server: %v\n", err)
		}
	}

	return adapter
}

// StartCallbackServer starts the OAuth callback server (for testing or manual control)
func (a *OAuthAdapter) StartCallbackServer(ctx context.Context) error {
	if a.callbackServer == nil {
		return fmt.Errorf("callback server not initialized")
	}
	return a.callbackServer.Start(ctx)
}

// StopCallbackServer stops the OAuth callback server (for testing or cleanup)
func (a *OAuthAdapter) StopCallbackServer() error {
	if a.callbackServer == nil {
		return nil
	}
	return a.callbackServer.Stop()
}

// Close cleans up all resources (for testing)
func (a *OAuthAdapter) Close() error {
	// Stop callback server if running
	if a.callbackServer != nil {
		if err := a.callbackServer.Stop(); err != nil {
			return err
		}
	}
	// Close token store
	if a.tokenStore != nil {
		return a.tokenStore.Close()
	}
	return nil
}

// HandleProtectedResourceMetadata handles /.well-known/oauth-protected-resource
func (a *OAuthAdapter) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := map[string]interface{}{
		"resource":              a.serverURL + "/mcp",
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
// CRITICAL: This must immediately redirect back to Claude after authorization
// No intermediate pages or "Open RTM" buttons!
func (a *OAuthAdapter) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	// Log incoming request for debugging
	fmt.Printf("[OAuth] Authorize request: method=%s, client_id=%s, redirect_uri=%s\n",
		r.Method, r.URL.Query().Get("client_id"), r.URL.Query().Get("redirect_uri"))

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

		// IMPORTANT: Form submits directly back to this same URL
		// No intermediate pages!
		html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Connect Remember The Milk</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
			display: flex;
			justify-content: center;
			align-items: center;
			min-height: 100vh;
			margin: 0;
			background-color: #f5f5f5;
		}
		.container {
			background: white;
			padding: 2rem;
			border-radius: 8px;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			max-width: 500px;
			text-align: center;
		}
		h1 {
			color: #333;
			margin-bottom: 1rem;
		}
		form {
			margin-top: 2rem;
		}
		label {
			display: block;
			margin-bottom: 1rem;
			text-align: left;
		}
		input[type="password"] {
			width: 100%%;
			padding: 0.5rem;
			font-size: 1rem;
			border: 1px solid #ddd;
			border-radius: 4px;
			margin-top: 0.5rem;
		}
		button {
			background-color: #007bff;
			color: white;
			border: none;
			padding: 0.75rem 2rem;
			font-size: 1rem;
			border-radius: 4px;
			cursor: pointer;
			margin-top: 1rem;
		}
		button:hover {
			background-color: #0056b3;
		}
		.info {
			margin-top: 2rem;
			padding: 1rem;
			background-color: #f8f9fa;
			border-radius: 4px;
			text-align: left;
			font-size: 0.9rem;
			color: #666;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>🐄 Connect Remember The Milk</h1>
		<p>Enter your RTM API Key to authorize Claude to access your tasks.</p>
		<form method="POST">
			<input type="hidden" name="client_id" value="%s">
			<input type="hidden" name="redirect_uri" value="%s">
			<input type="hidden" name="client_state" value="%s">
			<input type="hidden" name="csrf_state" value="%s">
			<input type="hidden" name="resource" value="%s">
			<label>
				RTM API Key:
				<input type="password" name="api_key" required autofocus>
			</label>
			<button type="submit">Connect</button>
		</form>
		<div class="info">
			<strong>Note:</strong> Your API key will be securely stored and used only to access your RTM tasks on your behalf.
		</div>
	</div>
</body>
</html>`, clientID, redirectURI, clientState, csrfState, resource)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(html)); err != nil {
			// Log error but response already started
			fmt.Printf("Failed to write HTML response: %v\n", err)
		}
		return
	}

	// Handle form submission (POST)
	apiKey := r.FormValue("api_key")
	csrfState = r.FormValue("csrf_state")
	clientState = r.FormValue("client_state")
	formRedirectURI := r.FormValue("redirect_uri")

	fmt.Printf("[OAuth] Form submission: has_api_key=%v, csrf_state=%s, client_state=%s\n",
		apiKey != "", csrfState, clientState)

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

	fmt.Printf("[OAuth] Generated auth code: %s (expires in 10 min)\n", code)

	// Clear CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete cookie
	})

	// CRITICAL: Immediately redirect back to Claude with the authorization code
	// No intermediate pages, no "success" page, no "Open RTM" button!
	// Claude is waiting for this redirect to complete the OAuth flow.
	u, _ := url.Parse(formRedirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", clientState) // Return client's original state
	u.RawQuery = q.Encode()

	fmt.Printf("[OAuth] Immediately redirecting back to Claude: %s\n", u.String())

	// Use 302 Found for the redirect (standard OAuth practice)
	http.Redirect(w, r, u.String(), http.StatusFound)

	// DO NOT show any success page or intermediate page here!
	// The redirect above sends the user back to Claude immediately.
}

// HandleToken handles /oauth/token
func (a *OAuthAdapter) HandleToken(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[OAuth] Token request: method=%s\n", r.Method)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		fmt.Printf("[OAuth] ERROR: Failed to parse form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	// resource := r.FormValue("resource") // June 2025 spec - TODO: use for validation

	fmt.Printf("[OAuth] Token request: grant_type=%s, code=%s\n", grantType, code)

	if grantType != "authorization_code" {
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	// Validate auth code
	authCode, exists := a.authCodes[code]
	if !exists || time.Now().After(authCode.ExpiresAt) {
		fmt.Printf("[OAuth] ERROR: Invalid or expired code: %s (exists=%v)\n", code, exists)
		http.Error(w, "Invalid or expired code", http.StatusBadRequest)
		return
	}

	fmt.Printf("[OAuth] Code validated successfully\n")

	// Generate bearer token
	token := uuid.New().String()
	a.tokenStore.Store(token, authCode.RTMAPIKey)

	fmt.Printf("[OAuth] Generated bearer token: %s...\n", token[:8])

	// Clean up auth code (one-time use)
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
	fmt.Printf("[OAuth] ValidateToken called with header: %s...\n",
		authHeader[:min(20, len(authHeader))])

	if !strings.HasPrefix(authHeader, "Bearer ") {
		fmt.Printf("[OAuth] ERROR: Invalid auth header format\n")
		return "", fmt.Errorf("invalid auth header")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	apiKey, exists := a.tokenStore.Get(token)
	if !exists {
		fmt.Printf("[OAuth] ERROR: Token not found in store\n")
		return "", fmt.Errorf("invalid token")
	}

	fmt.Printf("[OAuth] Token validated successfully\n")
	return apiKey, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
