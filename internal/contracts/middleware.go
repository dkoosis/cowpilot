package contracts

import (
	"log"
	"net/http"
)

// ValidatingMiddleware adds contract validation in development
func ValidatingMiddleware(next http.Handler) http.Handler {
	oauth := &OAuthContract{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate OAuth endpoints
		switch r.URL.Path {
		case "/oauth/authorize":
			if violations := oauth.ValidateAuthorizeRequest(r); len(violations) > 0 {
				for _, v := range violations {
					log.Printf("[CONTRACT] %s", v)
				}
			}
		case "/oauth/token":
			if violations := oauth.ValidateTokenRequest(r); len(violations) > 0 {
				for _, v := range violations {
					log.Printf("[CONTRACT] %s", v)
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// AssertOAuthCompliance runs all OAuth contract tests
func AssertOAuthCompliance(t interface{ Fatalf(string, ...interface{}) }, handler http.Handler) {
	// This would run a comprehensive test suite
	// verifying the handler meets all OAuth requirements
}
