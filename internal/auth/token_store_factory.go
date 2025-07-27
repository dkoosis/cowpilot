package auth

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TokenStoreInterface defines token storage operations
type TokenStoreInterface interface {
	Store(token, apiKey string)
	Get(token string) (string, bool)
	Delete(token string)
}

// CreateTokenStore creates appropriate token store based on environment
func CreateTokenStore() TokenStoreInterface {
	// Check if we should use SQLite
	dbPath := os.Getenv("TOKEN_DB_PATH")
	if dbPath != "" {
		store, err := NewSQLiteTokenStore(dbPath)
		if err != nil {
			log.Printf("Failed to create SQLite token store: %v, falling back to in-memory", err)
			return NewTokenStore() // Fall back to in-memory
		}
		log.Printf("Using SQLite token store at %s", dbPath)
		
		// Start cleanup routine
		go func() {
			ticker := time.NewTicker(1 * time.Hour)
			defer ticker.Stop()
			for range ticker.C {
				if err := store.CleanupExpired(24 * time.Hour); err != nil {
					log.Printf("Token cleanup error: %v", err)
				}
			}
		}()
		
		return store
	}
	
	log.Println("Using in-memory token store (set TOKEN_DB_PATH for persistence)")
	return NewTokenStore()
}

// SQLiteTokenStore implements persistent token storage
type SQLiteTokenStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteTokenStore creates a new SQLite-backed token store
func NewSQLiteTokenStore(dbPath string) (*SQLiteTokenStore, error) {
	// Ensure directory exists
	if dir := os.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS oauth_tokens (
			token TEXT PRIMARY KEY,
			api_key TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_api_key ON oauth_tokens(api_key);
		CREATE INDEX IF NOT EXISTS idx_last_used ON oauth_tokens(last_used);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteTokenStore{db: db}, nil
}

// Store saves a token-apiKey mapping
func (s *SQLiteTokenStore) Store(token, apiKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO oauth_tokens (token, api_key, created_at, last_used) VALUES (?, ?, ?, ?)",
		token, apiKey, time.Now(), time.Now(),
	)
	if err != nil {
		log.Printf("Failed to store token: %v", err)
	}
}

// Get retrieves apiKey for token and updates last_used
func (s *SQLiteTokenStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var apiKey string
	err := s.db.QueryRow("SELECT api_key FROM oauth_tokens WHERE token = ?", token).Scan(&apiKey)
	if err != nil {
		return "", false
	}

	// Update last used (async to avoid blocking)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		_, _ = s.db.Exec("UPDATE oauth_tokens SET last_used = ? WHERE token = ?", time.Now(), token)
	}()

	return apiKey, true
}

// Delete removes a token
func (s *SQLiteTokenStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec("DELETE FROM oauth_tokens WHERE token = ?", token)
	if err != nil {
		log.Printf("Failed to delete token: %v", err)
	}
}

// CleanupExpired removes tokens not used in the specified duration
func (s *SQLiteTokenStore) CleanupExpired(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	result, err := s.db.Exec("DELETE FROM oauth_tokens WHERE last_used < ?", cutoff)
	if err != nil {
		return err
	}
	
	if rows, err := result.RowsAffected(); err == nil && rows > 0 {
		log.Printf("Cleaned up %d expired tokens", rows)
	}
	
	return nil
}

// Close closes the database connection
func (s *SQLiteTokenStore) Close() error {
	return s.db.Close()
}
