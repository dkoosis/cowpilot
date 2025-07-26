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
	go func() {
		fmt.Printf("OAuth callback server starting on %s\n", addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Callback server error: %v\n", err)

			s.mu.Lock()
			if s.resultChan != nil {
				s.resultChan <- err
			}
			s.mu.Unlock()
		}
	}()

	return nil
}

// createCallbackHandler creates the main callback handler
func (s *OAuthCallbackServer) createCallbackHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log request for debugging
		fmt.Printf("OAuth callback received: method=%s path=%s query=%s\n",
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
			s.errorPage(w, http.StatusBadRequest, "Authorization Failed",
				fmt.Sprintf("OAuth error: %s - %s", errorParam, errorDesc))

			s.mu.Lock()
			if s.resultChan != nil {
				s.resultChan <- fmt.Errorf("oauth error: %s", errorParam)
			}
			s.mu.Unlock()
			return
		}

		// Validate required parameters
		if code == "" || state == "" {
			s.errorPage(w, http.StatusBadRequest, "Invalid Request",
				"Missing required parameters: code and state.")

			s.mu.Lock()
			if s.resultChan != nil {
				s.resultChan <- fmt.Errorf("missing code or state")
			}
			s.mu.Unlock()
			return
		}

		// Success page
		s.successPage(w, code, state)

		// Signal success
		s.mu.Lock()
		s.activeCallback = true
		if s.resultChan != nil {
			s.resultChan <- nil
		}
		s.mu.Unlock()
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
func (s *OAuthCallbackServer) successPage(w http.ResponseWriter, code, state string) {
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
</head>
<body>
    <div class="container">
        <div class="checkmark">✓</div>
        <h1>Authorization Successful!</h1>
        <p>You have successfully authorized the application.</p>
        <p>You can now close this window and return to Claude.</p>
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
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	fmt.Println("Stopping OAuth callback server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.server.Shutdown(ctx); err != nil {
		fmt.Printf("Error shutting down callback server: %v\n", err)
		return err
	}

	s.server = nil

	// Close result channel
	if s.resultChan != nil {
		close(s.resultChan)
		s.resultChan = nil
	}

	return nil
}

// WaitForCallback waits for callback completion with timeout
func (s *OAuthCallbackServer) WaitForCallback(timeout time.Duration) error {
	select {
	case err := <-s.resultChan:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("callback timeout after %v", timeout)
	}
}
