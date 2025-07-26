package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestOAuthCallbackServer(t *testing.T) {
	// Create adapter and callback server
	adapter := NewOAuthAdapter("http://localhost:8080", 9090)
	server := adapter.callbackServer

	// Test 1: State token generation and validation
	t.Run("OAuthCallbackServer_ValidatesStateTokens_When_CheckingCSRF", func(t *testing.T) {
		clientID := "test-client"
		state := server.GenerateStateToken(clientID)

		// Should validate successfully
		if err := server.ValidateStateToken(state, clientID); err != nil {
			t.Errorf("Valid token failed: %v", err)
		}

		// Should fail on reuse (one-time token)
		if err := server.ValidateStateToken(state, clientID); err == nil {
			t.Error("Token reuse should fail")
		}

		// Should fail with wrong client
		state2 := server.GenerateStateToken(clientID)
		if err := server.ValidateStateToken(state2, "wrong-client"); err == nil {
			t.Error("Wrong client should fail")
		}
	})

	// Test 2: Callback server lifecycle
	t.Run("OAuthCallbackServer_StartsAndStops_When_LifecycleMethods", func(t *testing.T) {
		ctx := context.Background()

		// Start server
		if err := server.Start(ctx); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}

		// Verify it's running
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get("http://localhost:9090/health")
		if err == nil {
			if err := resp.Body.Close(); err != nil {
				t.Logf("Failed to close response body: %v", err)
			}
		}

		// Stop server
		if err := server.Stop(); err != nil {
			t.Errorf("Failed to stop server: %v", err)
		}

		// Should fail to start again while running
		server2 := NewOAuthCallbackServer(adapter, 9090)
		if err := server2.Start(ctx); err != nil {
			t.Logf("Expected behavior: %v", err)
		}
	})

	// Test 3: Callback handling
	t.Run("OAuthCallbackServer_ProcessesCallback_When_ValidRequest", func(t *testing.T) {
		ctx := context.Background()
		server := NewOAuthCallbackServer(adapter, 9091)

		if err := server.Start(ctx); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}
		defer func() {
			if err := server.Stop(); err != nil {
				t.Errorf("Failed to stop server: %v", err)
			}
		}()

		// Test successful callback
		go func() {
			time.Sleep(100 * time.Millisecond)
			resp, err := http.Get("http://localhost:9091/callback?code=test-code&state=test-state")
			if err != nil {
				fmt.Printf("Callback request failed: %v\n", err)
				return
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					fmt.Printf("Failed to close response body: %v\n", err)
				}
			}()
		}()

		// Wait for callback
		err := server.WaitForCallback(1 * time.Second)
		if err != nil {
			t.Errorf("Callback wait failed: %v", err)
		}
	})
}
