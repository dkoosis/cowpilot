package auth

import (
	"testing"
	"time"
)

func TestCSRFTokenLifecycle(t *testing.T) {
	// Importance: This suite verifies our Cross-Site Request Forgery (CSRF) protection.
	// Without these guarantees, an attacker could potentially trick a user's browser into
	// performing unauthorized actions, making this a critical security test.

	server := &OAuthCallbackServer{
		stateTokens: make(map[string]*StateToken),
	}

	t.Run("generates unique tokens to prevent session conflicts", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures concurrent logins don't cross-contaminate, a critical security boundary.")
		tokens := make(map[string]bool)
		clientID := "test-client"

		for i := 0; i < 10; i++ {
			token := server.GenerateStateToken(clientID)
			if tokens[token] {
				t.Error("Duplicate token generated, which could lead to session hijacking")
			}
			tokens[token] = true
		}
	})

	t.Run("rejects an expired token to prevent replay attacks", func(t *testing.T) {
		t.Logf("  > Why it's important: A fundamental security check to ensure old, potentially compromised tokens are worthless.")
		clientID := "test-client"
		token := server.GenerateStateToken(clientID)

		// Force the token to be expired
		server.mu.Lock()
		server.stateTokens[token].ExpiresAt = time.Now().Add(-1 * time.Hour)
		server.mu.Unlock()

		err := server.ValidateStateToken(token, clientID)
		if err == nil {
			t.Error("An expired token was incorrectly validated, creating a security hole")
		}
	})

	t.Run("cleans up expired tokens to manage memory", func(t *testing.T) {
		t.Logf("  > Why it's important: Prevents a memory leak where the server would accumulate expired tokens indefinitely.")
		// Add one expired and one valid token
		server.mu.Lock()
		server.stateTokens["expired1"] = &StateToken{ExpiresAt: time.Now().Add(-1 * time.Hour)}
		server.stateTokens["valid1"] = &StateToken{ExpiresAt: time.Now().Add(1 * time.Hour)}
		server.mu.Unlock()

		// This action triggers the internal cleanup
		server.GenerateStateToken("another-client")

		server.mu.Lock()
		defer server.mu.Unlock()
		if _, exists := server.stateTokens["expired1"]; exists {
			t.Error("Expired token was not cleaned up, risking a memory leak")
		}
		if _, exists := server.stateTokens["valid1"]; !exists {
			t.Error("A valid token was incorrectly removed during cleanup")
		}
	})

	t.Run("rejects a token when the client ID does not match", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures a token generated for one client cannot be used by another, preventing session mix-ups.")
		token := server.GenerateStateToken("client-A")

		err := server.ValidateStateToken(token, "client-B")
		if err == nil {
			t.Error("Token validated with the wrong client ID, which is a security risk")
		}
	})

	t.Run("rejects a token after it has been used once", func(t *testing.T) {
		t.Logf("  > Why it's important: One-time-use tokens are a key defense against replay attacks.")
		clientID := "test-client"
		token := server.GenerateStateToken(clientID)

		// First use should succeed
		err := server.ValidateStateToken(token, clientID)
		if err != nil {
			t.Fatalf("First validation failed unexpectedly: %v", err)
		}

		// Second use should fail
		err = server.ValidateStateToken(token, clientID)
		if err == nil {
			t.Error("Token was successfully reused, which should not be possible")
		}
	})
}

func TestTokenStore(t *testing.T) {
	// Importance: This suite validates the in-memory store that links temporary authorization codes
	// to the user's permanent API key. If this component fails, no user would ever be able to
	// successfully complete the login flow.

	store := NewTokenStore()

	t.Run("retrieves an API key when a valid token is provided", func(t *testing.T) {
		t.Logf("  > Why it's important: The primary function of the token store; verifies the core lookup logic.")
		store.Store("token1", "api-key-1")

		apiKey, exists := store.Get("token1")
		if !exists || apiKey != "api-key-1" {
			t.Errorf("Failed to retrieve the correct API key for a valid token")
		}
	})

	t.Run("returns not found for an invalid token", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the store doesn't return false positives for non-existent tokens.")
		_, exists := store.Get("non-existent-token")
		if exists {
			t.Error("Store found a token that should not exist")
		}
	})

	t.Run("rejects an expired token", func(t *testing.T) {
		t.Logf("  > Why it's important: A security check to ensure that even if a token leaks, its lifetime is limited.")
		store.mu.Lock()
		store.tokens["expired-token"] = &Token{
			Value: "expired-token", ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		store.mu.Unlock()

		_, exists := store.Get("expired-token")
		if exists {
			t.Error("Store incorrectly returned an expired token")
		}
	})
}
