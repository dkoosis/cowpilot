package auth

import (
	"testing"
	"time"
)

func TestCSRFTokens(t *testing.T) {
	server := &OAuthCallbackServer{
		stateTokens: make(map[string]*StateToken),
	}
	
	t.Run("GenerateStateToken_CreatesUniqueTokens_When_CalledMultipleTimes", func(t *testing.T) {
		tokens := make(map[string]bool)
		clientID := "test-client"
		
		// Generate multiple tokens
		for i := 0; i < 10; i++ {
			token := server.GenerateStateToken(clientID)
			if tokens[token] {
				t.Error("Duplicate token generated")
			}
			tokens[token] = true
		}
	})
	
	t.Run("ValidateStateToken_RejectsToken_When_Expired", func(t *testing.T) {
		clientID := "test-client"
		
		// Generate token
		token := server.GenerateStateToken(clientID)
		
		// Set expiry to past
		server.mu.Lock()
		server.stateTokens[token].ExpiresAt = time.Now().Add(-1 * time.Hour)
		server.mu.Unlock()
		
		// Validation should fail
		err := server.ValidateStateToken(token, clientID)
		if err == nil {
			t.Error("Expected expired token to fail validation")
		}
	})
	
	t.Run("GenerateStateToken_RemovesExpiredTokens_When_CleanupTriggered", func(t *testing.T) {
		// Add expired tokens
		server.mu.Lock()
		server.stateTokens["expired1"] = &StateToken{
			State:     "expired1",
			ClientID:  "test",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		server.stateTokens["valid1"] = &StateToken{
			State:     "valid1",
			ClientID:  "test",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		server.mu.Unlock()
		
		// Generate new token (triggers cleanup)
		server.GenerateStateToken("test")
		
		// Check expired token was removed
		server.mu.Lock()
		defer server.mu.Unlock()
		
		if _, exists := server.stateTokens["expired1"]; exists {
			t.Error("Expired token not cleaned up")
		}
		if _, exists := server.stateTokens["valid1"]; !exists {
			t.Error("Valid token was incorrectly removed")
		}
	})
	
	t.Run("ValidateStateToken_RejectsToken_When_ClientIDMismatch", func(t *testing.T) {
		token := server.GenerateStateToken("client1")
		
		// Try to validate with different client
		err := server.ValidateStateToken(token, "client2")
		if err == nil {
			t.Error("Token validated with wrong client ID")
		}
	})
	
	t.Run("ValidateStateToken_RejectsToken_When_UsedTwice", func(t *testing.T) {
		clientID := "test-client"
		token := server.GenerateStateToken(clientID)
		
		// First validation should succeed
		err := server.ValidateStateToken(token, clientID)
		if err != nil {
			t.Errorf("First validation failed: %v", err)
		}
		
		// Second validation should fail
		err = server.ValidateStateToken(token, clientID)
		if err == nil {
			t.Error("Token reuse should fail")
		}
	})
}

func TestTokenStore(t *testing.T) {
	store := NewTokenStore()
	
	t.Run("TokenStore_RetrievesAPIKey_When_ValidTokenProvided", func(t *testing.T) {
		store.Store("token1", "api-key-1")
		
		apiKey, exists := store.Get("token1")
		if !exists {
			t.Error("Token not found")
		}
		if apiKey != "api-key-1" {
			t.Errorf("Wrong API key: got %s, want api-key-1", apiKey)
		}
	})
	
	t.Run("TokenStore_ReturnsNotFound_When_InvalidToken", func(t *testing.T) {
		_, exists := store.Get("non-existent")
		if exists {
			t.Error("Non-existent token should not be found")
		}
	})
	
	t.Run("TokenStore_RejectsToken_When_Expired", func(t *testing.T) {
		// Manually add expired token
		store.mu.Lock()
		store.tokens["expired"] = &Token{
			Value:     "expired",
			RTMAPIKey: "test-key",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		store.mu.Unlock()
		
		_, exists := store.Get("expired")
		if exists {
			t.Error("Expired token should not be retrievable")
		}
	})
	
	t.Run("TokenStore_RemovesToken_When_Deleted", func(t *testing.T) {
		store.Store("to-delete", "api-key")
		store.Delete("to-delete")
		
		_, exists := store.Get("to-delete")
		if exists {
			t.Error("Deleted token should not exist")
		}
	})
}
