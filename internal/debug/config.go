package debug

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DebugConfig holds runtime configuration for the debug system
type DebugConfig struct {
	Enabled        bool   // Enable/disable debug system
	StorageType    string // "disabled", "memory", "file"
	StoragePath    string // File path for file storage
	MaxMemoryMB    int    // Memory storage limit in MB
	MaxFileMB      int    // File storage limit in MB
	RetentionH     int    // Auto-cleanup hours
	Level          string // Debug level: DEBUG, INFO, WARN, ERROR
	ValidateProto  bool   // Enable protocol validation
	ValidateMode   string // "monitor" or "enforce"
}

// LoadDebugConfig loads debug configuration from environment variables
func LoadDebugConfig() *DebugConfig {
	enabled := getEnvBool("MCP_DEBUG", false)
	if !enabled {
		return &DebugConfig{Enabled: false} // Zero overhead when disabled
	}

	return &DebugConfig{
		Enabled:        true,
		StorageType:    getEnvDefault("MCP_DEBUG_STORAGE", "memory"),
		StoragePath:    getEnvDefault("MCP_DEBUG_PATH", "./debug.db"),
		MaxMemoryMB:    getEnvInt("MCP_DEBUG_MAX_MB", 100),
		MaxFileMB:      getEnvInt("MCP_DEBUG_FILE_MAX_MB", 500),
		RetentionH:     getEnvInt("MCP_DEBUG_RETENTION_H", 24),
		Level:          getEnvDefault("MCP_DEBUG_LEVEL", "INFO"),
		ValidateProto:  getEnvBool("MCP_VALIDATE_PROTOCOL", true),
		ValidateMode:   getEnvDefault("MCP_VALIDATE_MODE", "monitor"),
	}
}

// Helper functions for environment variables
func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// ConversationRecord represents a single MCP message in the conversation log
type ConversationRecord struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	Timestamp     time.Time `json:"timestamp"`
	Direction     string    `json:"direction"` // "inbound" or "outbound"
	Method        string    `json:"method"`
	Params        string    `json:"params"` // JSON string
	Result        string    `json:"result"` // JSON string
	Error         string    `json:"error"`  // JSON string
	PerformanceMS int64     `json:"performance_ms"`
}

// Storage interface for different storage backends
type Storage interface {
	LogMessage(sessionID, direction, method string, params, result, errorMsg interface{}, performanceMS int64) error
	LogValidation(sessionID, method string, violations []string, severity string) error
	GetConversation(sessionID string) ([]ConversationRecord, error)
	GetRecentSessions(limit int) ([]string, error)
	GetMessagesByMethod(method string, limit int) ([]ConversationRecord, error)
	GetStats() (map[string]interface{}, error)
	GetValidationStats() (map[string]interface{}, error)
	CleanupOldRecords(maxAge time.Duration) error
	Close() error
	IsEnabled() bool
}

// NoOpStorage provides a no-op implementation when debug is disabled
type NoOpStorage struct{}

func (n *NoOpStorage) LogMessage(sessionID, direction, method string, params, result, errorMsg interface{}, performanceMS int64) error {
	return nil
}

func (n *NoOpStorage) GetConversation(sessionID string) ([]ConversationRecord, error) {
	return nil, nil
}

func (n *NoOpStorage) GetRecentSessions(limit int) ([]string, error) {
	return nil, nil
}

func (n *NoOpStorage) GetMessagesByMethod(method string, limit int) ([]ConversationRecord, error) {
	return nil, nil
}

func (n *NoOpStorage) GetStats() (map[string]interface{}, error) {
	return map[string]interface{}{
		"debug_enabled": false,
		"storage_type":  "disabled",
	}, nil
}

func (n *NoOpStorage) CleanupOldRecords(maxAge time.Duration) error {
	return nil
}

func (n *NoOpStorage) Close() error {
	return nil
}

func (n *NoOpStorage) IsEnabled() bool {
	return false
}

// FileStorage implements SQLite-based storage
type FileStorage struct {
	db       *sql.DB
	dbPath   string
	enabled  bool
	maxBytes int64
}

// NewFileStorage creates a new file-based storage
func NewFileStorage(config *DebugConfig) (*FileStorage, error) {
	var dbPath string
	switch config.StorageType {
	case "memory":
		dbPath = ":memory:"
	case "file":
		dbPath = config.StoragePath
		// Create directory if needed
		if dbPath != ":memory:" {
			if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &FileStorage{
		db:       db,
		dbPath:   dbPath,
		enabled:  true,
		maxBytes: int64(config.MaxFileMB) * 1024 * 1024,
	}

	if err := storage.createTablesWithValidation(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// createTablesWithValidation creates database tables and validates they exist correctly.
func (fs *FileStorage) createTablesWithValidation() error {
	if err := fs.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Validate tables exist
	var count int
	err := fs.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('conversations', 'sessions')").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to validate tables: %w", err)
	}
	if count != 2 {
		return fmt.Errorf("expected 2 tables but found %d", count)
	}

	return nil
}

func (fs *FileStorage) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
		method TEXT,
		params TEXT,
		result TEXT,
		error TEXT,
		performance_ms INTEGER DEFAULT 0,
		size_bytes INTEGER DEFAULT 0
	);

	CREATE INDEX IF NOT EXISTS idx_conversations_session ON conversations(session_id);
	CREATE INDEX IF NOT EXISTS idx_conversations_timestamp ON conversations(timestamp);
	CREATE INDEX IF NOT EXISTS idx_conversations_method ON conversations(method);
	
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT UNIQUE NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		total_messages INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := fs.db.Exec(query)
	return err
}

func (fs *FileStorage) LogMessage(sessionID, direction, method string, params, result, errorMsg interface{}, performanceMS int64) error {
	if !fs.enabled {
		return nil
	}

	// Convert to JSON
	paramsJSON, _ := json.Marshal(params)
	resultJSON, _ := json.Marshal(result)
	errorJSON, _ := json.Marshal(errorMsg)

	// Calculate size
	sizeBytes := len(paramsJSON) + len(resultJSON) + len(errorJSON) + len(method) + len(sessionID)

	// Check size limits and cleanup if needed
	if err := fs.enforceStorageLimits(); err != nil {
		log.Printf("Storage limit enforcement failed: %v", err)
	}

	query := `
	INSERT INTO conversations (session_id, timestamp, direction, method, params, result, error, performance_ms, size_bytes)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := fs.db.Exec(query, sessionID, time.Now(), direction, method, string(paramsJSON), string(resultJSON), string(errorJSON), performanceMS, sizeBytes)
	if err != nil {
		return err
	}

	fs.updateSessionCount(sessionID)
	return nil
}

func (fs *FileStorage) enforceStorageLimits() error {
	if fs.maxBytes <= 0 {
		return nil
	}

	// Get current database size
	var totalSize int64
	err := fs.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM conversations").Scan(&totalSize)
	if err != nil {
		return err
	}

	// If over limit, delete oldest records
	if totalSize > fs.maxBytes {
		_, err = fs.db.Exec(`
			DELETE FROM conversations 
			WHERE id IN (
				SELECT id FROM conversations 
				ORDER BY timestamp ASC 
				LIMIT (SELECT COUNT(*) * 0.2 FROM conversations)
			)`)
		if err != nil {
			return err
		}
		log.Printf("Cleaned up old records due to size limit")
	}

	return nil
}

func (fs *FileStorage) updateSessionCount(sessionID string) {
	query := `
	INSERT INTO sessions (session_id, start_time, total_messages)
	VALUES (?, ?, 1)
	ON CONFLICT(session_id) DO UPDATE SET
		total_messages = total_messages + 1,
		end_time = CURRENT_TIMESTAMP`

	if _, err := fs.db.Exec(query, sessionID, time.Now()); err != nil {
		log.Printf("Failed to update session activity: %v", err)
	}
}

func (fs *FileStorage) GetConversation(sessionID string) ([]ConversationRecord, error) {
	query := `
	SELECT id, session_id, timestamp, direction, method, params, result, error, performance_ms
	FROM conversations WHERE session_id = ? ORDER BY timestamp ASC`

	rows, err := fs.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var records []ConversationRecord
	for rows.Next() {
		var record ConversationRecord
		err := rows.Scan(&record.ID, &record.SessionID, &record.Timestamp, &record.Direction,
			&record.Method, &record.Params, &record.Result, &record.Error, &record.PerformanceMS)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (fs *FileStorage) GetRecentSessions(limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	rows, err := fs.db.Query("SELECT DISTINCT session_id FROM conversations ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var sessions []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, err
		}
		sessions = append(sessions, sessionID)
	}
	return sessions, nil
}

func (fs *FileStorage) GetMessagesByMethod(method string, limit int) ([]ConversationRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
	SELECT id, session_id, timestamp, direction, method, params, result, error, performance_ms
	FROM conversations WHERE method = ? ORDER BY timestamp DESC LIMIT ?`

	rows, err := fs.db.Query(query, method, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var records []ConversationRecord
	for rows.Next() {
		var record ConversationRecord
		err := rows.Scan(&record.ID, &record.SessionID, &record.Timestamp, &record.Direction,
			&record.Method, &record.Params, &record.Result, &record.Error, &record.PerformanceMS)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (fs *FileStorage) GetStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"debug_enabled": true,
		"storage_type":  "file",
		"database_path": fs.dbPath,
	}

	var totalMessages, totalSessions int64
	if err := fs.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&totalMessages); err != nil {
		log.Printf("Failed to get total messages: %v", err)
	}
	if err := fs.db.QueryRow("SELECT COUNT(DISTINCT session_id) FROM conversations").Scan(&totalSessions); err != nil {
		log.Printf("Failed to get total sessions: %v", err)
	}

	stats["total_messages"] = totalMessages
	stats["total_sessions"] = totalSessions

	// Storage size
	var totalSize int64
	if err := fs.db.QueryRow("SELECT COALESCE(SUM(size_bytes), 0) FROM conversations").Scan(&totalSize); err != nil {
		log.Printf("Failed to get total size: %v", err)
	}
	stats["storage_bytes"] = totalSize
	stats["storage_mb"] = float64(totalSize) / (1024 * 1024)

	return stats, nil
}

func (fs *FileStorage) CleanupOldRecords(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	result, err := fs.db.Exec("DELETE FROM conversations WHERE timestamp < ?", cutoff)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old conversation records", rowsAffected)
	}
	return nil
}

func (fs *FileStorage) Close() error {
	if fs.db != nil {
		return fs.db.Close()
	}
	return nil
}

func (fs *FileStorage) IsEnabled() bool {
	return fs.enabled
}

// NewStorage creates appropriate storage based on config
func NewStorage(config *DebugConfig) (Storage, error) {
	if !config.Enabled || config.StorageType == "disabled" {
		return &NoOpStorage{}, nil
	}

	return NewFileStorage(config)
}

// StartDebugSystem initializes debug system based on runtime configuration
func StartDebugSystem() (Storage, *DebugConfig, error) {
	config := LoadDebugConfig()

	if !config.Enabled {
		log.Println("Debug system disabled")
		return &NoOpStorage{}, config, nil
	}

	log.Printf("Starting MCP Debug System (storage: %s, level: %s)", config.StorageType, config.Level)

	storage, err := NewStorage(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Set up cleanup routine
	if config.RetentionH > 0 {
		go func() {
			ticker := time.NewTicker(time.Hour)
			defer ticker.Stop()

			for range ticker.C {
				maxAge := time.Duration(config.RetentionH) * time.Hour
				if err := storage.CleanupOldRecords(maxAge); err != nil {
					log.Printf("Cleanup error: %v", err)
				}
			}
		}()
	}

	log.Println("Debug system started successfully")
	return storage, config, nil
}
