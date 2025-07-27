package auth

import (
	"database/sql"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteTokenStore implements persistent token storage
type SQLiteTokenStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewSQLiteTokenStore creates a new SQLite-backed token store
func NewSQLiteTokenStore(dbPath string) (*SQLiteTokenStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
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

	_, _ = s.db.Exec(
		"INSERT OR REPLACE INTO oauth_tokens (token, api_key, created_at, last_used) VALUES (?, ?, ?, ?)",
		token, apiKey, time.Now(), time.Now(),
	)
}

// Get retrieves apiKey for token and updates last_used
func (s *SQLiteTokenStore) Get(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var apiKey string
	err := s.db.QueryRow("SELECT api_key FROM oauth_tokens WHERE token = ?", token).Scan(&apiKey)
	if err != nil {
		return "", false
	}

	// Update last used
	_, _ = s.db.Exec("UPDATE oauth_tokens SET last_used = ? WHERE token = ?", time.Now(), token)

	return apiKey, true
}

// Delete removes a token
func (s *SQLiteTokenStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, _ = s.db.Exec("DELETE FROM oauth_tokens WHERE token = ?", token)
}

// CleanupExpired removes tokens not used in the specified duration
func (s *SQLiteTokenStore) CleanupExpired(maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	_, err := s.db.Exec("DELETE FROM oauth_tokens WHERE last_used < ?", cutoff)
	return err
}

// Close closes the database connection
func (s *SQLiteTokenStore) Close() error {
	return s.db.Close()
}
