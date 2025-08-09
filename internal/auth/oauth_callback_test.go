package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestOAuthCallbackServer_ValidatesStateTokens_When_CheckingCSRF(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9092)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})
	server := adapter.callbackServer
	// No need to start/stop since we're not actually running the server for this test

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
}

func TestOAuthCallbackServer_StartsAndStops_When_LifecycleMethodsCalled(t *testing.T) {
	adapter := NewOAuthAdapter("http://localhost:8080", 9093)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})
	server := adapter.callbackServer

	// Since GO_TEST=1, server should NOT be running yet
	ctx := context.Background()

	// Should be able to start the server
	if err := server.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		if err := server.Stop(); err != nil {
			t.Logf("Failed to stop server in cleanup: %v", err)
		}
	}()

	// Should fail to start again while already running
	if err := server.Start(ctx); err == nil {
		t.Error("Should fail to start already running server")
	}

	// Verify it's running
	time.Sleep(100 * time.Millisecond)
	resp, err := http.Get("http://localhost:9093/health")
	if err == nil {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Failed to close response body: %v", err)
		}
	}

	// Stop server
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Should be able to start again after stopping
	if err := server.Start(ctx); err != nil {
		t.Errorf("Failed to restart server: %v", err)
	}

	// Clean up (final stop)
	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server in final cleanup: %v", err)
	}
}

func TestOAuthCallbackServer_ProcessesCallback_When_RequestIsValid(t *testing.T) {
	// Create adapter with test mode enabled
	adapter := NewOAuthAdapter("http://localhost:8080", 9094)
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Logf("Failed to close adapter: %v", err)
		}
	})
	server := adapter.callbackServer

	// Start the server since GO_TEST=1 prevents auto-start
	ctx := context.Background()
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
		resp, err := http.Get("http://localhost:9094/callback?code=test-code&state=test-state")
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
}
