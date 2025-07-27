package rtm

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vcto/cowpilot/internal/auth"
)

// OAuthAdapter adapts RTM's frob-based auth to OAuth flow
type OAuthAdapter struct {
	client       *Client
	sessions     map[string]*AuthSession
	sessionMutex sync.RWMutex
	serverURL    string
}

// AuthSession tracks RTM auth progress with OAuth parameters
type AuthSession struct {
	Code        string // Our fake OAuth code
	Frob        string // RTM frob
	CreatedAt   time.Time
	Token       string // Set after successful exchange
	State       string // Client's CSRF state
	RedirectURI string // Client's callback URL
	ClientID    string // OAuth client ID
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

	// Validate CSRF cookie
	csrfCookie, err := r.Cookie("csrf_token")
	if err != nil || csrfCookie.Value == "" {
		http.Error(w, "Missing CSRF cookie", http.StatusBadRequest)
		return
	}

	csrfState := r.FormValue("csrf_state")
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

	// Step 3: Store session with all OAuth parameters
	session := &AuthSession{
		Code:        code,
		Frob:        frob,
		CreatedAt:   time.Now(),
		State:       state,
		RedirectURI: redirectURI,
		ClientID:    clientID,
	}

	a.sessionMutex.Lock()
	a.sessions[code] = session
	a.sessionMutex.Unlock()

	// Step 4: Build RTM auth URL with frob
	rtmParams := map[string]string{
		"api_key": a.client.APIKey,
		"perms":   "delete", // We need delete perms for task management
		"frob":    frob,
	}
	sig := a.client.sign(rtmParams)

	rtmURL := fmt.Sprintf("https://www.rememberthemilk.com/services/auth/?api_key=%s&perms=delete&frob=%s&api_sig=%s",
		url.QueryEscape(a.client.APIKey),
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
	log.Printf("RTM DEBUG: Callback hit for code %s", code)

	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// Look up session to get redirect URI
	a.sessionMutex.RLock()
	session, exists := a.sessions[code]
	a.sessionMutex.RUnlock()

	if !exists {
		http.Error(w, "Invalid code", http.StatusBadRequest)
		return
	}

	// Verify token exists (should be set by check-auth endpoint)
	if session.Token == "" {
		log.Printf("RTM DEBUG: Callback hit but no token for code %s - auth not completed", code)
		http.Error(w, "Authorization not completed", http.StatusBadRequest)
		return
	}

	log.Printf("RTM DEBUG: Auth verified, redirecting to %s", session.RedirectURI)

	// Redirect back to original redirect_uri with our code
	u, _ := url.Parse(session.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	q.Set("state", session.State)
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

// HandleToken implements OAuth token endpoint
func (a *OAuthAdapter) HandleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
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
	session.Token = a.client.AuthToken
	a.sendTokenSuccess(w, session.Token)
	a.removeSession(code)
}

// Helper methods

func (a *OAuthAdapter) showAuthForm(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	state := r.URL.Query().Get("state")
	redirectURI := r.URL.Query().Get("redirect_uri")

	// Generate CSRF token
	csrfToken := uuid.New().String()

	// Set CSRF cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
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
            After authorizing, you may need to manually return to this window.
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

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprint(w, html); err != nil {
		log.Printf("Failed to write auth form response: %v", err)
	}
}

func (a *OAuthAdapter) showIntermediatePage(w http.ResponseWriter, rtmURL, code, clientID, state, redirectURI string) {
	checkAuthURL := fmt.Sprintf("%s/rtm/check-auth?code=%s", a.serverURL, code)
	callbackURL := fmt.Sprintf("%s/rtm/callback?code=%s", a.serverURL, code)

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Authorize with Remember The Milk</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        .container { border: 1px solid #ddd; border-radius: 8px; padding: 30px; text-align: center; }
        h1 { color: #333; }
        .step { margin: 20px 0; padding: 20px; background: #f8f9fa; border-radius: 4px; }
        .button { display: inline-block; background: #007bff; color: white; text-decoration: none; padding: 12px 24px; border-radius: 4px; margin: 10px; text-align: center; border: none; cursor: pointer; }
        .button:hover { background: #0056b3; }
        .button:disabled { background: #6c757d; cursor: not-allowed; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; color: #155724; }
        .error { background: #f8d7da; border: 1px solid #f5c6cb; color: #721c24; }
        .checking { background: #fff3cd; border: 1px solid #ffeaa7; color: #856404; }
        #status { margin: 20px 0; padding: 15px; border-radius: 4px; display: none; }
    </style>
    <script>
        let authWindow = null;
        let checkInterval = null;
        
        function openRTMAuth() {
            authWindow = window.open('%s', 'rtm_auth', 'width=800,height=600');
            
            // Check if popup was blocked
            if (!authWindow || authWindow.closed || typeof authWindow.closed == 'undefined') {
                document.getElementById('status').style.display = 'block';
                document.getElementById('status').className = 'error';
                document.getElementById('status').innerHTML = 'Popup blocked! Please allow popups and try again.';
                document.getElementById('retryBtn').style.display = 'inline-block';
                return;
            }
            
            document.getElementById('step2').style.display = 'block';
            document.getElementById('continueBtn').disabled = true;
            
            // Start checking auth status instead of just polling window
            startAuthCheck();
        }
        
        function startAuthCheck() {
            document.getElementById('status').style.display = 'block';
            document.getElementById('status').className = 'checking';
            document.getElementById('status').innerHTML = 'Waiting for authorization...';
            
            checkInterval = setInterval(checkAuthStatus, 2000);
        }
        
        function checkAuthStatus() {
            fetch('%s')
                .then(response => response.json())
                .then(data => {
                    if (data.authorized) {
                        // Success! Enable continue button
                        clearInterval(checkInterval);
                        document.getElementById('status').className = 'success';
                        document.getElementById('status').innerHTML = '[✓] Authorization successful! You can now continue.';
                        document.getElementById('continueBtn').disabled = false;
                        document.getElementById('continueBtn').style.background = '#28a745';
                        if (authWindow) authWindow.close();
                    } else if (data.error) {
                        // Error occurred
                        clearInterval(checkInterval);
                        document.getElementById('status').className = 'error';
                        document.getElementById('status').innerHTML = '[✗] ' + data.error;
                        document.getElementById('retryBtn').style.display = 'inline-block';
                    }
                    // If pending, keep checking
                })
                .catch(err => {
                    console.log('Check failed:', err);
                    // Continue checking on network errors
                });
        }
        
        function completeAuth() {
            if (checkInterval) clearInterval(checkInterval);
            window.location.href = '%s';
        }
        
        function retryAuth() {
            document.getElementById('retryBtn').style.display = 'none';
            document.getElementById('continueBtn').disabled = true;
            document.getElementById('continueBtn').style.background = '#007bff';
            openRTMAuth();
        }
        
        // Auto-open RTM auth on page load
        window.onload = function() {
            setTimeout(openRTMAuth, 500);
        };
    </script>
</head>
<body>
    <div class="container">
        <h1>Connect to Remember The Milk</h1>
        
        <div class="step">
            <h2>Step 1: Authorize Access</h2>
            <p>A new window will open for Remember The Milk authorization.</p>
            <button onclick="openRTMAuth()" class="button">Open RTM Authorization</button>
        </div>
        
        <div id="status"></div>
        
        <div class="step" id="step2" style="display: none;">
            <h2>Step 2: Complete Authorization</h2>
            <p><strong>Important:</strong> Click "Yes, I authorize access" in the RTM window, then continue below.</p>
            <button id="continueBtn" onclick="completeAuth()" class="button" disabled>Continue to App</button>
            <button id="retryBtn" onclick="retryAuth()" class="button" style="display: none; background: #dc3545;">Try Again</button>
        </div>
        
        <div style="margin-top: 30px; font-size: 14px; color: #666;">
            <p><strong>Troubleshooting:</strong> Make sure to click "Yes, I authorize access" in the RTM window. If you close the window without authorizing, use "Try Again".</p>
            <p style="font-size: 12px;">Debug info: Code %s</p>
        </div>
    </div>
</body>
</html>`, rtmURL, checkAuthURL, callbackURL, code)

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

	// Try to exchange frob for token
	err := a.client.GetToken(session.Frob)
	if err == nil {
		// Success! Store token and respond
		a.sessionMutex.Lock()
		session.Token = a.client.AuthToken
		a.sessionMutex.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
			"authorized": true,
		}); writeErr != nil {
			log.Printf("Failed to write check auth success response: %v", writeErr)
		}
		return
	}

	// Check if it's a "not authorized" error vs other errors
	if fmt.Sprintf("%v", err) == "RTM API error 101: Invalid frob - did you authenticate?" {
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

	// Other error - frob expired or other issue
	w.Header().Set("Content-Type", "application/json")
	if writeErr := json.NewEncoder(w).Encode(map[string]interface{}{
		"authorized": false,
		"error":      fmt.Sprintf("Authorization failed: %v", err),
	}); writeErr != nil {
		log.Printf("Failed to write check auth error response: %v", writeErr)
	}
}

// ValidateBearer checks if a bearer token is valid
func (a *OAuthAdapter) ValidateBearer(token string) bool {
	// For RTM, we could validate by making an API call
	// For now, just check if non-empty
	return token != ""
}
