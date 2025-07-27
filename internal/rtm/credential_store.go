package rtm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// CredentialStore manages encrypted RTM credentials
type CredentialStore interface {
	Store(userID, apiKey, secret string) error
	Retrieve(userID string) (apiKey, secret string, err error)
	Delete(userID string) error
	Close() error
}

// SQLiteCredentialStore implements encrypted credential storage
type SQLiteCredentialStore struct {
	db        *sql.DB
	masterKey []byte
}

// RTMCredential represents stored credentials
type RTMCredential struct {
	UserID          string
	EncryptedKey    string
	EncryptedSecret string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewCredentialStore creates credential store with encryption
func NewCredentialStore(dbPath string) (CredentialStore, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Generate or retrieve master key
	masterKey, err := getMasterKey()
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// Ignore close error, return original error
			_ = closeErr
		}
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	store := &SQLiteCredentialStore{
		db:        db,
		masterKey: masterKey,
	}

	if err := store.createTables(); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			// Ignore close error, return original error
			_ = closeErr
		}
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return store, nil
}

func (s *SQLiteCredentialStore) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS rtm_credentials (
		user_id TEXT PRIMARY KEY,
		encrypted_api_key TEXT NOT NULL,
		encrypted_secret TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TRIGGER IF NOT EXISTS update_rtm_credentials_timestamp 
	AFTER UPDATE ON rtm_credentials
	BEGIN
		UPDATE rtm_credentials SET updated_at = CURRENT_TIMESTAMP WHERE user_id = NEW.user_id;
	END;`

	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteCredentialStore) Store(userID, apiKey, secret string) error {
	encryptedKey, err := s.encrypt(apiKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	encryptedSecret, err := s.encrypt(secret)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	query := `
	INSERT OR REPLACE INTO rtm_credentials (user_id, encrypted_api_key, encrypted_secret, updated_at)
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)`

	_, err = s.db.Exec(query, userID, encryptedKey, encryptedSecret)
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	return nil
}

func (s *SQLiteCredentialStore) Retrieve(userID string) (string, string, error) {
	var cred RTMCredential
	query := `SELECT encrypted_api_key, encrypted_secret FROM rtm_credentials WHERE user_id = ?`

	err := s.db.QueryRow(query, userID).Scan(&cred.EncryptedKey, &cred.EncryptedSecret)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("credentials not found for user %s", userID)
		}
		return "", "", fmt.Errorf("failed to retrieve credentials: %w", err)
	}

	apiKey, err := s.decrypt(cred.EncryptedKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	secret, err := s.decrypt(cred.EncryptedSecret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return apiKey, secret, nil
}

func (s *SQLiteCredentialStore) Delete(userID string) error {
	query := `DELETE FROM rtm_credentials WHERE user_id = ?`
	_, err := s.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}
	return nil
}

func (s *SQLiteCredentialStore) Close() error {
	return s.db.Close()
}

// Encryption helpers
func (s *SQLiteCredentialStore) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *SQLiteCredentialStore) decrypt(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.masterKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// getMasterKey generates or retrieves encryption key
func getMasterKey() ([]byte, error) {
	// In production, this should be from a secure key management system
	// For now, derive from environment or generate
	keySource := os.Getenv("RTM_MASTER_KEY")
	if keySource == "" {
		keySource = "cowpilot-rtm-encryption-key-2025" // Default for development
	}

	hash := sha256.Sum256([]byte(keySource))
	return hash[:], nil
}
