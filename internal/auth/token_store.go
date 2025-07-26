package auth

import (
	"sync"
	"time"
)

// TokenStore manages OAuth tokens in memory
type TokenStore struct {
	mu     sync.RWMutex
	tokens map[string]*Token
}

// Token represents an OAuth token with metadata
type Token struct {
	Value     string
	RTMAPIKey string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	store := &TokenStore{
		tokens: make(map[string]*Token),
	}
	// Start cleanup goroutine
	go store.cleanupExpired()
	return store
}

// Store saves a token mapping
func (s *TokenStore) Store(token, apiKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token] = &Token{
		Value:     token,
		RTMAPIKey: apiKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
}

// Get retrieves API key for token
func (s *TokenStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.tokens[token]
	if !exists || time.Now().After(t.ExpiresAt) {
		return "", false
	}
	return t.RTMAPIKey, true
}

// Delete removes a token
func (s *TokenStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

// cleanupExpired removes expired tokens periodically
func (s *TokenStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, t := range s.tokens {
			if now.After(t.ExpiresAt) {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}
