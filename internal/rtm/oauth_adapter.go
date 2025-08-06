// File: internal/rtm/oauth_adapter.go

package rtm

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vcto/mcp-adapters/internal/auth"
)

// OAuthAdapter adapts RTM's frob-based auth to OAuth flow
type OAuthAdapter struct {
	client       RTMClientInterface
	sessions     map[string]*AuthSession
	sessionMutex sync.RWMutex
	serverURL    string
}

// AuthSession tracks RTM auth progress with OAuth parameters
type AuthSession struct {
	Code                string // Our fake OAuth code
	Frob                string // RTM frob
	CreatedAt           time.Time
	Token               string // Set after successful exchange
	State               string // Client's CSRF state
	RedirectURI         string // Client's callback URL
	ClientID            string // OAuth client ID
	CodeChallenge       string // PKCE code challenge
	CodeChallengeMethod string // PKCE method (S256)
	CodeVerifier        string // PKCE code verifier
	Resource            string // MCP resource parameter
}

// NewOAuthAdapter creates RTM OAuth adapter
func NewOAuthAdapter(apiKey, secret, serverURL string) *OAuthAdapter {
	return &OAuthAdapter{
		client:    NewClient(apiKey, secret),
		sessions:  make(map[string]*AuthSession),
		serverURL: serverURL,
	}
}

// HandleAuthorize implements OAuth authorize endpoint
func (a *OAuthAdapter) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show authorization form
		a.showAuthForm(w, r)
		return
	}

	// POST - process authorization
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	state := r.FormValue("state")
	redirectURI := r.FormValue("redirect_uri")
	codeChallenge := r.FormValue("code_challenge")
	codeChallengeMethod := r.FormValue("code_challenge_method")
	resource := r.FormValue("resource")

	// Validate CSRF - check both cookie and form value
	csrfState := r.FormValue("csrf_state")
	if csrfState == "" {
		log.Printf("RTM: Missing CSRF token in form")
		http.Error(w, "Missing CSRF token in form", http.StatusBadRequest)
		return
	}

	csrfCookie, err := r.Cookie("csrf_token")
	if err != nil || csrfCookie.Value == "" {
		log.Printf("RTM: CSRF cookie missing, error: %v", err)
		http.Error(w, "Missing CSRF cookie - please disable popup blocker and try again without refreshing", http.StatusBadRequest)
		return
	}

	log.Printf("RTM: CSRF validation - form: %s, cookie: %s", csrfState, csrfCookie.Value)
	if csrfState != csrfCookie.Value {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}

	// Step 1: Get frob from RTM
	frob, err := a.client.GetFrob()
	if err != nil {
		log.Printf("RTM: Failed to get frob: %v", err)
		a.showError(w, "Failed to start RTM authentication")
		return
	}

	// Step 2: Create fake OAuth code
	code := uuid.New().String()

	// Validate PKCE if provided
	if codeChallenge != "" {
		if codeChallengeMethod != "S256" {
			http.Error(w, "Unsupported code_challenge_method. Only S256 is supported.", http.StatusBadRequest)
			return
		}
	}

	// Validate resource parameter for MCP compliance
	if resource != "" && !strings.HasPrefix(resource, a.serverURL+"/mcp") {
		http.Error(w, "Invalid resource parameter", http.StatusBadRequest)
		return
	}

	// Step 3: Store session with all OAuth parameters
	session := &AuthSession{
		Code:                code,
		Frob:                frob,
		CreatedAt:           time.Now(),
		State:               state,
		RedirectURI:         redirectURI,
		ClientID:            clientID,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Resource:            resource,
	}

	a.sessionMutex.Lock()
	a.sessions[code] = session
	a.sessionMutex.Unlock()

	// Step 4: Build RTM auth URL with frob
	rtmParams := map[string]string{
		"api_key": a.client.GetAPIKey(),
		"perms":   "delete", // We need delete perms for task management
		"frob":    frob,
	}
	sig := a.client.Sign(rtmParams)

	rtmURL := fmt.Sprintf("https://www.rememberthemilk.com/services/auth/?api_key=%s&perms=delete&frob=%s&api_sig=%s",
		url.QueryEscape(a.client.GetAPIKey()),
		url.QueryEscape(frob),
		url.QueryEscape(sig))

	// Clear CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	// Step 5: Show intermediate page with RTM link
	a.showIntermediatePage(w, rtmURL, code, clientID, state, redirectURI)
}

// HandleCallback handles the callback after RTM auth verification
func (a *OAuthAdapter) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	log.Printf("RTM: Callback hit for code %s from %s", code, r.RemoteAddr)

	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// Look up session to get redirect URI
	a.sessionMutex.RLock()
	session, exists := a.sessions[code]
	a.sessionMutex.RUnlock()

	if !exists {
		log.Printf("RTM: Invalid code %s in callback", code)
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// Verify token exists (should be set by check-auth endpoint)
	if session.Token == "" {
		log.Printf("RTM: Callback hit but no token for code %s - trying immediate exchange", code)
		// Try one more time to get the token
		if err := a.client.GetToken(session.Frob); err == nil {
			a.sessionMutex.Lock()
			session.Token = a.client.GetAuthToken()
			a.sessionMutex.Unlock()
			log.Printf("RTM: Late token exchange successful for code %s", code)
		} else {
			log.Printf("RTM: Late token exchange failed: %v", err)
			http.Error(w, "Authorization not completed. Please try again.", http.StatusBadRequest)
			return
		}
	}

	log.Printf("RTM: Auth verified, redirecting to %s with code=%s state=%s",
		session.RedirectURI, code, session.State)

	// Redirect back to original redirect_uri with our code
	u, err := url.Parse(session.RedirectURI)
	if err != nil {
		log.Printf("RTM: Invalid redirect URI: %v", err)
		http.Error(w, "Invalid redirect URI", http.StatusInternalServerError)
		return
	}
	q := u.Query()
	q.Set("code", code)
	if session.State != "" {
		q.Set("state", session.State)
	}
	u.RawQuery = q.Encode()

	log.Printf("RTM: Final redirect URL: %s", u.String())
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// HandleToken implements OAuth token endpoint
func (a *OAuthAdapter) HandleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" {
		a.sendTokenError(w, "invalid_request", "Missing code parameter")
		return
	}

	// Look up session
	a.sessionMutex.RLock()
	session, exists := a.sessions[code]
	a.sessionMutex.RUnlock()

	if !exists {
		a.sendTokenError(w, "invalid_grant", "Invalid authorization code")
		return
	}

	// Validate PKCE if challenge was provided
	if session.CodeChallenge != "" {
		if codeVerifier == "" {
			a.sendTokenError(w, "invalid_request", "Missing code_verifier for PKCE")
			return
		}
		if !a.validatePKCE(session.CodeChallenge, codeVerifier) {
			a.sendTokenError(w, "invalid_grant", "Invalid code_verifier")
			return
		}
	}

	log.Printf("RTM DEBUG: Token request for code %s, session.Token='%s'", code, session.Token)

	// Check if we already have token (from polling)
	if session.Token != "" {
		log.Printf("RTM DEBUG: Token ready, returning success")
		a.sendTokenSuccess(w, session.Token)
		a.removeSession(code)
		return
	}

	// Try to exchange frob for token
	log.Printf("RTM DEBUG: Token not ready, trying immediate exchange")
	if err := a.client.GetToken(session.Frob); err != nil {
		log.Printf("RTM DEBUG: Immediate exchange failed: %v", err)
		// User might not have authorized yet
		a.sendTokenError(w, "authorization_pending", "User has not completed authorization")
		return
	}

	// Success!
	log.Printf("RTM DEBUG: Immediate exchange succeeded")
	session.Token = a.client.GetAuthToken()
	a.sendTokenSuccess(w, session.Token)
	a.removeSession(code)
}

// Helper methods

func (a *OAuthAdapter) showAuthForm(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	state := r.URL.Query().Get("state")
	redirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")

	log.Printf("[OAUTH] /authorize called with: client_id=%s, state=%s, redirect_uri=%s, response_type=%s",
		clientID, state, redirectURI, responseType)
	log.Printf("[OAUTH] Full query string: %s", r.URL.RawQuery)
	log.Printf("[OAUTH] User-Agent: %s", r.Header.Get("User-Agent"))

	// Generate CSRF token
	csrfToken := uuid.New().String()

	// Conditionally set cookies based on environment
	isSecure := strings.HasPrefix(a.serverURL, "https://")

	// SameSiteNone requires Secure=true, so adjust for test environments
	sameSite := http.SameSiteNoneMode
	if !isSecure {
		sameSite = http.SameSiteLaxMode // Use Lax for HTTP/test environments
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
		MaxAge:   1800,
	})

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Connect Remember The Milk</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; }
        h1 { color: #333; }
        .warning { background: #fff3cd; border: 1px solid #ffeaa7; padding: 15px; border-radius: 4px; margin: 20px 0; }
        button { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #0056b3; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Connect Remember The Milk</h1>
        <p>This will connect your Remember The Milk account to allow task management.</p>
        <div class="warning">
        <strong>Note:</strong> You'll be redirected to Remember The Milk to authorize access.
        After authorizing, click the return link we'll provide to complete the connection.
        </div>
        <form method="POST">
            <input type="hidden" name="client_id" value="%s">
            <input type="hidden" name="state" value="%s">
            <input type="hidden" name="redirect_uri" value="%s">
            <input type="hidden" name="csrf_state" value="%s">
            <button type="submit">Connect Remember The Milk</button>
        </form>
    </div>
</body>
</html>`, clientID, state, redirectURI, csrfToken)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		log.Printf("Failed to write auth form response: %v", err)
	}
}

func (a *OAuthAdapter) showIntermediatePage(w http.ResponseWriter, rtmURL, code, _, _, _ string) {
	checkAuthURL := fmt.Sprintf("%s/rtm/check-auth?code=%s", a.serverURL, code)
	callbackURL := fmt.Sprintf("%s/rtm/callback?code=%s", a.serverURL, code)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authorize with Remember The Milk</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; text-align: center; }
        h1 { color: #333; }
        .button { display: inline-block; background: #007bff; color: white; text-decoration: none; padding: 12px 24px; border-radius: 4px; margin: 10px; cursor: pointer; font-size: 16px; border: none; }
        .button:hover { background: #0056b3; }
        .button:disabled { background: #6c757d; cursor: not-allowed; }
        .status { margin: 20px 0; padding: 15px; border-radius: 4px; }
        .checking { background: #fff3cd; border: 1px solid #ffeaa7; color: #856404; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .error { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .instructions { margin: 20px 0; color: #666; }
    </style>
    <script>
        let checkInterval = null;
        let isChecking = false;

        function startChecking() {
            if (checkInterval) return;
            isChecking = true;
            updateStatus('checking', 'Waiting for you to click "Allow" on the RTM page...');
            checkInterval = setInterval(checkAuthStatus, 2000);
            checkAuthStatus(); // Check immediately
        }

        function checkAuthStatus() {
            fetch('%s')
                .then(response => response.json())
                .then(data => {
                    if (data.authorized) {
                        clearInterval(checkInterval);
                        updateStatus('success', 'Authorization successful! Redirecting...');
                        setTimeout(() => {
                            window.location.href = '%s';
                        }, 1000);
                    } else if (data.error && !data.pending) {
                        clearInterval(checkInterval);
                        updateStatus('error', data.error);
                        document.getElementById('checkBtn').disabled = false;
                        document.getElementById('checkBtn').textContent = 'Try Again';
                    } else if (data.pending) {
                        // Still waiting - update message periodically
                        updateStatus('checking', 'Still waiting... Make sure you clicked "Allow" on the RTM page!');
                    }
                })
                .catch(err => {
                    console.error('Check failed:', err);
                });
        }

        function updateStatus(type, message) {
            const status = document.getElementById('status');
            status.className = 'status ' + type;
            status.textContent = message;
            status.style.display = 'block';
        }

        function manualCheck() {
            document.getElementById('checkBtn').disabled = true;
            startChecking();
        }

        // Start checking when returning to tab
        document.addEventListener('visibilitychange', function() {
            if (!document.hidden && !isChecking) {
                startChecking();
            }
        });
    </script>
</head>
<body>
    <div class="container">
        <h1>Connect to Remember The Milk</h1>

        <div class="instructions">
            <p><strong>Step 1:</strong> Click the button below to open Remember The Milk in a new tab</p>
            <p><strong>Step 2:</strong> On the RTM page, you'll see either:</p>
            <ul style="text-align: left; display: inline-block;">
                <li>An "OK, I'll allow it" button - click this to authorize</li>
                <li>OR a success message if you've already authorized</li>
            </ul>
            <p><strong>Step 3:</strong> Return to this tab and click "I've Authorized"</p>
        </div>

        <a href="%s" target="_blank" class="button" onclick="setTimeout(startChecking, 1000)">Open Remember The Milk →</a>

        <div style="margin: 20px 0; padding: 15px; background: #f0f8ff; border: 1px solid #4682b4; border-radius: 4px;">
            <p style="margin: 0; color: #333;">💡 <strong>What you'll see on RTM:</strong></p>
            <ul style="margin: 10px 0; padding-left: 30px; text-align: left;">
                <li>Application name: <strong>API Application</strong></li>
                <li>Permission level: <strong>delete</strong> (full access)</li>
                <li>A button saying <strong>"OK, I'll allow it"</strong> - click this!</li>
                <li>OR: "You have successfully authorized" if already done</li>
            </ul>
        </div>

        <div id="status" class="status" style="display: none;"></div>

        <div style="margin-top: 30px;">
            <button id="checkBtn" class="button" onclick="manualCheck()" style="background: #28a745;">
                I've Authorized
            </button>
        </div>
    </div>
</body>
</html>`, checkAuthURL, callbackURL, rtmURL)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		log.Printf("Failed to write intermediate page response: %v", err)
	}
}

func (a *OAuthAdapter) showError(w http.ResponseWriter, message string) {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authorization Error</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .error { border: 1px solid #f5c6cb; background: #f8d7da; padding: 20px; border-radius: 4px; color: #721c24; }
    </style>
</head>
<body>
    <div class="error">
        <h2>Authorization Error</h2>
        <p>%s</p>
    </div>
</body>
</html>`, message)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		log.Printf("Failed to write error response: %v", err)
	}
}

func (a *OAuthAdapter) sendTokenSuccess(w http.ResponseWriter, token string) {
	response := auth.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   0, // RTM tokens don't expire
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write token success response: %v", err)
	}
}

func (a *OAuthAdapter) sendTokenError(w http.ResponseWriter, error, description string) {
	response := auth.TokenError{
		Error:            error,
		ErrorDescription: description,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to write token error response: %v", err)
	}
}

func (a *OAuthAdapter) removeSession(code string) {
	a.sessionMutex.Lock()
	delete(a.sessions, code)
	a.sessionMutex.Unlock()
}

// HandleCheckAuth checks if frob has been authorized
func (a *OAuthAdapter) HandleCheckAuth(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// Look up session
	a.sessionMutex.RLock()
	session, exists := a.sessions[code]
	a.sessionMutex.RUnlock()

	if !exists {
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// If we already have a token, return success immediately
	if session.Token != "" {
		w.Header().Set("Content-Type", "application/json")
		if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"authorized": true,
		}); writeErr != nil {
			log.Printf("Failed to write check auth cached response: %v", writeErr)
		}
		return
	}

	// Try to exchange frob for token
	err := a.client.GetToken(session.Frob)
	if err == nil {
		// Success! Store token and respond
		a.sessionMutex.Lock()
		session.Token = a.client.GetAuthToken()
		a.sessionMutex.Unlock()

		log.Printf("RTM: Successfully exchanged frob for token for code %s", code)

		w.Header().Set("Content-Type", "application/json")
		if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"authorized": true,
		}); writeErr != nil {
			log.Printf("Failed to write check auth success response: %v", writeErr)
		}
		return
	}

	// Check if it's a "not authorized" error vs other errors
	if rtmErr, ok := err.(*RTMError); ok {
		log.Printf("RTM: Check auth failed with code %d: %s", rtmErr.Code, rtmErr.Msg)
		if rtmErr.Code == 101 {
			// User hasn't authorized yet, return pending
			w.Header().Set("Content-Type", "application/json")
			if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
				"authorized": false,
				"pending":    true,
			}); writeErr != nil {
				log.Printf("Failed to write check auth pending response: %v", writeErr)
			}
			return
		}
	}

	// Other error - frob expired or other issue
	log.Printf("RTM: Check auth failed with error: %v", err)
	w.Header().Set("Content-Type", "application/json")
	if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
		"authorized": false,
		"error":      fmt.Sprintf("Authorization failed: %v", err),
	}); writeErr != nil {
		log.Printf("Failed to write check auth error response: %v", writeErr)
	}
}

// validatePKCE validates PKCE code_verifier against code_challenge
func (a *OAuthAdapter) validatePKCE(codeChallenge, codeVerifier string) bool {
	// Generate challenge from verifier using S256
	h := sha256.Sum256([]byte(codeVerifier))
	computedChallenge := base64.RawURLEncoding.EncodeToString(h[:])
	return computedChallenge == codeChallenge
}

// HandleRegister implements Dynamic Client Registration (RFC 7591)
func (a *OAuthAdapter) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate client credentials
	clientID := "rtm_" + generateRandomString(16)
	clientSecret := generateRandomString(32)

	response := map[string]interface{}{
		"client_id":                clientID,
		"client_secret":            clientSecret,
		"client_id_issued_at":      time.Now().Unix(),
		"client_secret_expires_at": 0, // Never expires
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode DCR response: %v", err)
	}
}

// generateRandomString creates a cryptographically secure random string
func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

// ValidateBearer checks if a bearer token is valid by testing it against RTM API
func (a *OAuthAdapter) ValidateBearer(token string) bool {
	if token == "" {
		return false
	}

	// Create a temporary client with the token to test it
	// Note: This requires the concrete client for now
	testClient := NewClient(a.client.GetAPIKey(), "")
	testClient.AuthToken = token

	// Test token by making a minimal API call
	_, err := testClient.GetLists()
	if err != nil {
		log.Printf("RTM DEBUG: Token validation failed: %v", err)
		return false
	}

	log.Printf("RTM DEBUG: Token validation successful")
	return true
}

// SetClient sets the RTM client (for testing)
func (a *OAuthAdapter) SetClient(client RTMClientInterface) {
	a.client = client
}

// GetSession retrieves a session by code (for testing)
func (a *OAuthAdapter) GetSession(code string) *AuthSession {
	a.sessionMutex.RLock()
	defer a.sessionMutex.RUnlock()
	return a.sessions[code]
}
