package auth

import (
	"net/http"
	"strings"
)

// Middleware creates auth middleware that validates OAuth tokens
func Middleware(adapter *OAuthAdapter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for OAuth endpoints and well-known
			path := r.URL.Path
			if strings.HasPrefix(path, "/oauth/") ||
				strings.HasPrefix(path, "/.well-known/") ||
				path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			// Check for Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// Return 401 with WWW-Authenticate header (June 2025 spec)
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+adapter.serverURL+`"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Validate token
			apiKey, err := adapter.ValidateToken(authHeader)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="`+adapter.serverURL+`" error="invalid_token"`)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// Add API key to request context for handlers
			r.Header.Set("X-RTM-API-Key", apiKey)

			next.ServeHTTP(w, r)
		})
	}
}
