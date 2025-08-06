package rtm

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCredentialStoreLifecycle(t *testing.T) {
	t.Logf("Importance: This suite verifies the complete lifecycle and security of the credential store, which is critical for protecting user API keys. A failure here could lead to data exposure or non-functional authentication.")

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")
	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create credential store for testing: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	t.Run("stores and retrieves credentials for a new user", func(t *testing.T) {
		t.Logf("  > Why it's important: This is the primary function of the store; verifies the core create and read operations.")
		userID := "user_123"
		apiKey := "key_123"
		secret := "secret_123"

		err := store.Store(userID, apiKey, secret)
		if err != nil {
			t.Fatalf("Failed to store credentials: %v", err)
		}

		retrievedKey, retrievedSecret, err := store.Retrieve(userID)
		if err != nil {
			t.Fatalf("Failed to retrieve credentials: %v", err)
		}

		if retrievedKey != apiKey || retrievedSecret != secret {
			t.Errorf("Retrieved credentials do not match stored credentials")
		}
	})

	t.Run("updates credentials for an existing user", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures users can update their credentials without creating duplicate or conflicting entries.")
		userID := "user_456"
		err := store.Store(userID, "old_key", "old_secret")
		if err != nil {
			t.Fatalf("Failed to store initial credentials: %v", err)
		}

		newKey := "new_key_789"
		newSecret := "new_secret_789"
		err = store.Store(userID, newKey, newSecret)
		if err != nil {
			t.Fatalf("Failed to update credentials: %v", err)
		}

		retrievedKey, retrievedSecret, err := store.Retrieve(userID)
		if err != nil {
			t.Fatalf("Failed to retrieve updated credentials: %v", err)
		}

		if retrievedKey != newKey || retrievedSecret != newSecret {
			t.Error("Updated credentials were not stored correctly")
		}
	})

	t.Run("deletes credentials for a user", func(t *testing.T) {
		t.Logf("  > Why it's important: Verifies that user data can be correctly and completely removed.")
		userID := "user_to_delete"
		err := store.Store(userID, "key_to_delete", "secret_to_delete")
		if err != nil {
			t.Fatalf("Failed to store credentials for deletion test: %v", err)
		}

		err = store.Delete(userID)
		if err != nil {
			t.Fatalf("Failed to delete credentials: %v", err)
		}

		_, _, err = store.Retrieve(userID)
		if err == nil {
			t.Error("Expected an error when retrieving deleted credentials, but none was returned")
		}
	})

	t.Run("isolates credentials between different users", func(t *testing.T) {
		t.Logf("  > Why it's important: A critical multi-tenancy test to ensure one user's data cannot be accessed or affected by another's.")
		err := store.Store("user_A", "key_A", "secret_A")
		if err != nil {
			t.Fatalf("Failed to store user_A credentials: %v", err)
		}
		err = store.Store("user_B", "key_B", "secret_B")
		if err != nil {
			t.Fatalf("Failed to store user_B credentials: %v", err)
		}

		keyA, secretA, err := store.Retrieve("user_A")
		if err != nil || keyA != "key_A" || secretA != "secret_A" {
			t.Error("User_A credentials incorrect or inaccessible")
		}

		keyB, secretB, err := store.Retrieve("user_B")
		if err != nil || keyB != "key_B" || secretB != "secret_B" {
			t.Error("User_B credentials incorrect or inaccessible")
		}
	})

	t.Run("encrypts credentials at rest", func(t *testing.T) {
		t.Logf("  > Why it's important: This is the core security guarantee of the credential store. It verifies that raw API keys are never stored in plaintext on disk.")
		userID := "secure_user"
		apiKey := "super_secret_api_key"
		secret := "super_secret_value"

		err := store.Store(userID, apiKey, secret)
		if err != nil {
			t.Fatalf("Failed to store credentials for encryption test: %v", err)
		}

		// Directly inspect the database file to check for plaintext
		sqliteStore, ok := store.(*SQLiteCredentialStore)
		if !ok {
			t.Fatal("Store is not of expected type SQLiteCredentialStore")
		}
		var encryptedKey, encryptedSecret string
		query := `SELECT encrypted_api_key, encrypted_secret FROM rtm_credentials WHERE user_id = ?`
		err = sqliteStore.db.QueryRow(query, userID).Scan(&encryptedKey, &encryptedSecret)
		if err != nil {
			t.Fatalf("Failed to read raw data from database: %v", err)
		}

		if encryptedKey == apiKey || strings.Contains(encryptedKey, apiKey) {
			t.Error("API key appears to be stored in plaintext")
		}
		if encryptedSecret == secret || strings.Contains(encryptedSecret, secret) {
			t.Error("Secret appears to be stored in plaintext")
		}

		// Verify decryption works
		retrievedKey, retrievedSecret, err := store.Retrieve(userID)
		if err != nil {
			t.Fatalf("Failed to retrieve credentials for decryption check: %v", err)
		}
		if retrievedKey != apiKey || retrievedSecret != secret {
			t.Error("Decryption failed to restore original credentials")
		}
	})

	t.Run("returns an error for a non-existent user", func(t *testing.T) {
		t.Logf("  > Why it's important: Ensures the store correctly reports when a user is not found, preventing false successes.")
		_, _, err := store.Retrieve("nonexistent_user")
		if err == nil {
			t.Error("Expected an error when retrieving a non-existent user, but got nil")
		}
	})
}
