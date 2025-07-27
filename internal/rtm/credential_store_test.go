package rtm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialStore_StoreAndRetrieve(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	// Test data
	userID := "test_user_123"
	apiKey := "test_api_key_12345"
	secret := "test_secret_67890"

	// Store credentials
	err = store.Store(userID, apiKey, secret)
	if err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}

	// Retrieve credentials
	retrievedKey, retrievedSecret, err := store.Retrieve(userID)
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}

	// Verify
	if retrievedKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, retrievedKey)
	}
	if retrievedSecret != secret {
		t.Errorf("Expected secret %s, got %s", secret, retrievedSecret)
	}
}

func TestCredentialStore_Update(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	userID := "test_user_123"

	// Store initial credentials
	err = store.Store(userID, "old_key", "old_secret")
	if err != nil {
		t.Fatalf("Failed to store initial credentials: %v", err)
	}

	// Update credentials
	newKey := "new_api_key_12345"
	newSecret := "new_secret_67890"
	err = store.Store(userID, newKey, newSecret)
	if err != nil {
		t.Fatalf("Failed to update credentials: %v", err)
	}

	// Retrieve updated credentials
	retrievedKey, retrievedSecret, err := store.Retrieve(userID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated credentials: %v", err)
	}

	if retrievedKey != newKey {
		t.Errorf("Expected updated API key %s, got %s", newKey, retrievedKey)
	}
	if retrievedSecret != newSecret {
		t.Errorf("Expected updated secret %s, got %s", newSecret, retrievedSecret)
	}
}

func TestCredentialStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	userID := "test_user_123"

	// Store credentials
	err = store.Store(userID, "test_key", "test_secret")
	if err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}

	// Delete credentials
	err = store.Delete(userID)
	if err != nil {
		t.Fatalf("Failed to delete credentials: %v", err)
	}

	// Try to retrieve deleted credentials
	_, _, err = store.Retrieve(userID)
	if err == nil {
		t.Error("Expected error when retrieving deleted credentials")
	}
}

func TestCredentialStore_UserIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	// Store credentials for different users
	err = store.Store("user1", "key1", "secret1")
	if err != nil {
		t.Fatalf("Failed to store user1 credentials: %v", err)
	}

	err = store.Store("user2", "key2", "secret2")
	if err != nil {
		t.Fatalf("Failed to store user2 credentials: %v", err)
	}

	// Retrieve and verify isolation
	key1, secret1, err := store.Retrieve("user1")
	if err != nil {
		t.Fatalf("Failed to retrieve user1 credentials: %v", err)
	}

	key2, secret2, err := store.Retrieve("user2")
	if err != nil {
		t.Fatalf("Failed to retrieve user2 credentials: %v", err)
	}

	if key1 != "key1" || secret1 != "secret1" {
		t.Error("User1 credentials corrupted")
	}
	if key2 != "key2" || secret2 != "secret2" {
		t.Error("User2 credentials corrupted")
	}

	// Delete user1, ensure user2 unaffected
	err = store.Delete("user1")
	if err != nil {
		t.Fatalf("Failed to delete user1: %v", err)
	}

	_, _, err = store.Retrieve("user1")
	if err == nil {
		t.Error("User1 credentials should be deleted")
	}

	_, _, err = store.Retrieve("user2")
	if err != nil {
		t.Error("User2 credentials should still exist")
	}
}

func TestCredentialStore_EncryptionWorks(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	userID := "test_user_123"
	apiKey := "secret_api_key_12345"
	secret := "very_secret_value_67890"

	// Store credentials
	err = store.Store(userID, apiKey, secret)
	if err != nil {
		t.Fatalf("Failed to store credentials: %v", err)
	}

	// Check that raw database doesn't contain plaintext
	sqliteStore := store.(*SQLiteCredentialStore)
	var encryptedKey, encryptedSecret string
	query := `SELECT encrypted_api_key, encrypted_secret FROM rtm_credentials WHERE user_id = ?`
	err = sqliteStore.db.QueryRow(query, userID).Scan(&encryptedKey, &encryptedSecret)
	if err != nil {
		t.Fatalf("Failed to read raw database: %v", err)
	}

	// Verify data is encrypted (doesn't contain plaintext)
	if encryptedKey == apiKey {
		t.Error("API key stored in plaintext")
	}
	if encryptedSecret == secret {
		t.Error("Secret stored in plaintext")
	}

	// Verify we can still decrypt
	retrievedKey, retrievedSecret, err := store.Retrieve(userID)
	if err != nil {
		t.Fatalf("Failed to retrieve credentials: %v", err)
	}

	if retrievedKey != apiKey || retrievedSecret != secret {
		t.Error("Decryption failed")
	}
}

func TestCredentialStore_EmptyValues(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	// Test storing empty values
	err = store.Store("user1", "", "")
	if err != nil {
		t.Fatalf("Failed to store empty credentials: %v", err)
	}

	key, secret, err := store.Retrieve("user1")
	if err != nil {
		t.Fatalf("Failed to retrieve empty credentials: %v", err)
	}

	if key != "" || secret != "" {
		t.Error("Empty values not preserved")
	}
}

func TestCredentialStore_NonexistentUser(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	store, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Logf("Failed to close store: %v", closeErr)
		}
	}()

	_, _, err = store.Retrieve("nonexistent_user")
	if err == nil {
		t.Error("Expected error for nonexistent user")
	}
}

func TestCredentialStore_MasterKeyConsistency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_credentials.db")

	// Set consistent master key
	if err := os.Setenv("RTM_MASTER_KEY", "test_master_key_123"); err != nil {
		t.Fatalf("Failed to set env var: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("RTM_MASTER_KEY"); err != nil {
			t.Logf("Failed to unset env var: %v", err)
		}
	}()

	// Create store, add data, close
	store1, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store1: %v", err)
	}

	if err := store1.Store("user1", "key1", "secret1"); err != nil {
		t.Fatalf("Failed to store in store1: %v", err)
	}
	if err := store1.Close(); err != nil {
		t.Fatalf("Failed to close store1: %v", err)
	}

	// Reopen store, verify data readable
	store2, err := NewCredentialStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store2: %v", err)
	}
	defer func() {
		if closeErr := store2.Close(); closeErr != nil {
			t.Logf("Failed to close store2: %v", closeErr)
		}
	}()

	key, secret, err := store2.Retrieve("user1")
	if err != nil {
		t.Fatalf("Failed to retrieve from store2: %v", err)
	}

	if key != "key1" || secret != "secret1" {
		t.Error("Data inconsistent across store instances")
	}
}
