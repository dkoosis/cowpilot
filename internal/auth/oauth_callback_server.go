package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OAuthCallbackServer handles OAuth callback with robustness patterns from cowgnition
type OAuthCallbackServer struct {
	adapter        *OAuthAdapter
	server         *http.Server
	callbackPort   int
	resultChan     chan error
	mu             sync.Mutex
	stateTokens    map[string]*StateToken // CSRF protection
	activeCallback bool
}

// StateToken for CSRF protection
type StateToken struct {
	State     string
	ClientID  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewOAuthCallbackServer creates a callback server
func NewOAuthCallbackServer(adapter *OAuthAdapter, port int) *OAuthCallbackServer {
	return &OAuthCallbackServer{
		adapter:      adapter,
		callbackPort: port,
		stateTokens:  make(map[string]*StateToken),
	}
}

// GenerateStateToken creates a CSRF protection token
func (s *OAuthCallbackServer) GenerateStateToken(clientID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := uuid.New().String()
	s.stateTokens[state] = &StateToken{
		State:     state,
		ClientID:  clientID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	// Clean expired tokens
	s.cleanExpiredTokens()

	return state
}

// ValidateStateToken checks CSRF token validity
func (s *OAuthCallbackServer) ValidateStateToken(state, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.stateTokens[state]
	if !exists {
		return fmt.Errorf("invalid state token")
	}

	if time.Now().After(token.ExpiresAt) {
		delete(s.stateTokens, state)
		return fmt.Errorf("state token expired")
	}

	if token.ClientID != clientID {
		return fmt.Errorf("state token client mismatch")
	}

	// Token is valid, remove it (one-time use)
	delete(s.stateTokens, state)
	return nil
}

// cleanExpiredTokens removes expired CSRF tokens
func (s *OAuthCallbackServer) cleanExpiredTokens() {
	now := time.Now()
	for state, token := range s.stateTokens {
		if now.After(token.ExpiresAt) {
			delete(s.stateTokens, state)
		}
	}
}

// Start begins the callback server
func (s *OAuthCallbackServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return fmt.Errorf("callback server already running")
	}

	s.resultChan = make(chan error, 1)

	// Create mux with multiple callback paths
	mux := http.NewServeMux()
	handler := s.createCallbackHandler()

	// Support multiple callback paths for flexibility
	mux.HandleFunc("/oauth/callback", handler)
	mux.HandleFunc("/auth/callback", handler)
	mux.HandleFunc("/callback", handler)
	mux.HandleFunc("/", handler) // Catch-all

	// Create server with security timeouts
	addr := fmt.Sprintf(":%d", s.callbackPort)
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// Start server in goroutine
	// Capture server and channel references to avoid race conditions
	localServer := s.server
	localResultChan := s.resultChan
	
	go func() {
		fmt.Printf("OAuth callback server starting on %s\n", addr)
		if err := localServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Callback server error: %v\n", err)

			s.mu.Lock()
			if localResultChan != nil {
				localResultChan <- err
			}
			s.mu.Unlock()
		}
	}()

	return nil
}

// createCallbackHandler creates the main callback handler
func (s *OAuthCallbackServer) createCallbackHandler() http.HandlerFunc {
	// Don't lock here - Start() already holds the lock when calling this
	// Just use the resultChan that was already created
	localResultChan := s.resultChan
	
	return func(w http.ResponseWriter, r *http.Request) {
		// Log request for debugging
		fmt.Printf("[OAuth Callback] Received: method=%s path=%s query=%s\n",
			r.Method, r.URL.Path, r.URL.RawQuery)

		// Only accept GET requests
		if r.Method != http.MethodGet {
			s.errorPage(w, http.StatusMethodNotAllowed, "Method Not Allowed",
				"This endpoint only accepts GET requests.")
			return
		}

		// Extract parameters
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorParam := r.URL.Query().Get("error")
		errorDesc := r.URL.Query().Get("error_description")

		// Handle OAuth errors
		if errorParam != "" {
			fmt.Printf("[OAuth Callback] ERROR: OAuth error received: %s - %s\n", errorParam, errorDesc)
			s.errorPage(w, http.StatusBadRequest, "Authorization Failed",
				fmt.Sprintf("OAuth error: %s - %s", errorParam, errorDesc))

			if localResultChan != nil {
				localResultChan <- fmt.Errorf("oauth error: %s", errorParam)
			}
			return
		}

		// Validate required parameters
		if code == "" || state == "" {
			fmt.Printf("[OAuth Callback] ERROR: Missing parameters - code: %v, state: %v\n", 
				code != "", state != "")
			s.errorPage(w, http.StatusBadRequest, "Invalid Request",
				"Missing required parameters: code and state.")

			if localResultChan != nil {
				localResultChan <- fmt.Errorf("missing code or state")
			}
			return
		}

		// Success page
		fmt.Printf("[OAuth Callback] SUCCESS: code=%s, state=%s\n", code, state)
		s.successPage(w)

		// Signal success
		s.mu.Lock()
		s.activeCallback = true
		s.mu.Unlock()
		
		if localResultChan != nil {
			fmt.Printf("[OAuth Callback] Signaling success through result channel\n")
			localResultChan <- nil
		}
	}
}

// errorPage renders a user-friendly error page
func (s *OAuthCallbackServer) errorPage(w http.ResponseWriter, status int, title, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
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
            color: #dc3545;
            margin-bottom: 1rem;
        }
        p {
            color: #666;
            line-height: 1.6;
        }
        .error-details {
            background: #f8f9fa;
            padding: 1rem;
            border-radius: 4px;
            margin: 1rem 0;
            font-family: monospace;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <p>%s</p>
        <p>You can close this window and try again.</p>
    </div>
</body>
</html>`, title, title, message)

	if _, err := w.Write([]byte(html)); err != nil {
		// Log error but response already started
		fmt.Printf("Failed to write error page: %v\n", err)
	}
}

// successPage renders a success page
func (s *OAuthCallbackServer) successPage(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Authorization Successful</title>
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
            color: #28a745;
            margin-bottom: 1rem;
        }
        p {
            color: #666;
            line-height: 1.6;
        }
        .checkmark {
            font-size: 3rem;
            color: #28a745;
        }
    </style>
    <script>
        // Auto-close the window after a short delay
        setTimeout(function() {
            // Try to close the window
            window.close();
            
            // If window.close() doesn't work (e.g., not opened by script),
            // try to notify parent window if this is in an iframe or popup
            if (window.opener && window.opener !== window) {
                try {
                    window.opener.postMessage({ type: 'oauth-success' }, '*');
                } catch (e) {
                    console.log('Could not notify parent window');
                }
            }
            
            // Also try parent frame communication
            if (window.parent && window.parent !== window) {
                try {
                    window.parent.postMessage({ type: 'oauth-success' }, '*');
                } catch (e) {
                    console.log('Could not notify parent frame');
                }
            }
        }, 2000); // 2 second delay to let user see the success message
    </script>
</head>
<body>
    <div class="container">
        <div class="checkmark">✓</div>
        <h1>Authorization Successful!</h1>
        <p>You have successfully connected to Remember The Milk.</p>
        <p>This window will close automatically in a moment...</p>
        <p style="font-size: 0.9em; color: #999; margin-top: 1rem;">If this window doesn't close automatically, you can close it manually and return to Claude.</p>
    </div>
</body>
</html>`

	if _, err := w.Write([]byte(html)); err != nil {
		// Log error but response already started
		fmt.Printf("Failed to write success page: %v\n", err)
	}
}

// Stop gracefully shuts down the callback server
func (s *OAuthCallbackServer) Stop() error {
	s.mu.Lock()
	server := s.server
	s.server = nil
	// Don't nil out resultChan yet - WaitForCallback might still need it
	s.mu.Unlock()

	if server == nil {
		return nil
	}

	fmt.Println("Stopping OAuth callback server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down callback server: %v\n", err)
		return err
	}

	// Now close result channel after server is fully stopped
	s.mu.Lock()
	if s.resultChan != nil {
		close(s.resultChan)
		s.resultChan = nil
	}
	s.mu.Unlock()

	return nil
}

// WaitForCallback waits for callback completion with timeout
func (s *OAuthCallbackServer) WaitForCallback(timeout time.Duration) error {
	// Capture channel reference to avoid race conditions
	s.mu.Lock()
	localResultChan := s.resultChan
	s.mu.Unlock()
	
	if localResultChan == nil {
		return fmt.Errorf("result channel not initialized")
	}
	
	select {
	case err := <-localResultChan:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("callback timeout after %v", timeout)
	}
}
