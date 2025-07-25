package debug

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ConversationRecord represents a single MCP message in the conversation log
type ConversationRecord struct {
	ID            int64     `json:"id"`
	SessionID     string    `json:"session_id"`
	Timestamp     time.Time `json:"timestamp"`
	Direction     string    `json:"direction"` // "inbound" or "outbound"
	Method        string    `json:"method"`
	Params        string    `json:"params"`        // JSON string
	Result        string    `json:"result"`        // JSON string  
	Error         string    `json:"error"`         // JSON string
	PerformanceMS int64     `json:"performance_ms"`
}

// ConversationStorage handles SQLite database operations for conversation logging
type ConversationStorage struct {
	db       *sql.DB
	dbPath   string
	enabled  bool
}

// NewConversationStorage creates a new conversation storage instance
func NewConversationStorage(dbPath string) (*ConversationStorage, error) {
	if dbPath == "" {
		dbPath = "./debug_conversations.db"
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &ConversationStorage{
		db:      db,
		dbPath:  dbPath,
		enabled: true,
	}

	if err := storage.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return storage, nil
}

// createTables creates the necessary database tables
func (cs *ConversationStorage) createTables() error {
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
		UNIQUE(session_id, timestamp, direction) ON CONFLICT IGNORE
	);

	CREATE INDEX IF NOT EXISTS idx_conversations_session ON conversations(session_id);
	CREATE INDEX IF NOT EXISTS idx_conversations_timestamp ON conversations(timestamp);
	CREATE INDEX IF NOT EXISTS idx_conversations_method ON conversations(method);
	CREATE INDEX IF NOT EXISTS idx_conversations_direction ON conversations(direction);
	
	-- Table for session metadata
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT UNIQUE NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		client_info TEXT,
		total_messages INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON sessions(session_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time);
	`

	_, err := cs.db.Exec(query)
	return err
}

// LogMessage stores a conversation message in the database
func (cs *ConversationStorage) LogMessage(sessionID, direction, method string, params, result, errorMsg interface{}, performanceMS int64) error {
	if !cs.enabled {
		return nil
	}

	// Convert interfaces to JSON strings
	paramsJSON := ""
	if params != nil {
		if data, err := json.Marshal(params); err == nil {
			paramsJSON = string(data)
		}
	}

	resultJSON := ""
	if result != nil {
		if data, err := json.Marshal(result); err == nil {
			resultJSON = string(data)
		}
	}

	errorJSON := ""
	if errorMsg != nil {
		if data, err := json.Marshal(errorMsg); err == nil {
			errorJSON = string(data)
		}
	}

	query := `
	INSERT INTO conversations (session_id, timestamp, direction, method, params, result, error, performance_ms)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := cs.db.Exec(query, sessionID, time.Now(), direction, method, paramsJSON, resultJSON, errorJSON, performanceMS)
	if err != nil {
		log.Printf("Failed to log message: %v", err)
		return err
	}

	// Update session message count
	cs.updateSessionMessageCount(sessionID)

	return nil
}

// updateSessionMessageCount increments the message count for a session
func (cs *ConversationStorage) updateSessionMessageCount(sessionID string) {
	// Insert or update session record
	query := `
	INSERT INTO sessions (session_id, start_time, total_messages)
	VALUES (?, ?, 1)
	ON CONFLICT(session_id) DO UPDATE SET
		total_messages = total_messages + 1,
		end_time = CURRENT_TIMESTAMP`

	_, err := cs.db.Exec(query, sessionID, time.Now())
	if err != nil {
		log.Printf("Failed to update session count: %v", err)
	}
}

// GetConversation retrieves all messages for a specific session
func (cs *ConversationStorage) GetConversation(sessionID string) ([]ConversationRecord, error) {
	query := `
	SELECT id, session_id, timestamp, direction, method, params, result, error, performance_ms
	FROM conversations
	WHERE session_id = ?
	ORDER BY timestamp ASC`

	rows, err := cs.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ConversationRecord
	for rows.Next() {
		var record ConversationRecord
		err := rows.Scan(
			&record.ID,
			&record.SessionID,
			&record.Timestamp,
			&record.Direction,
			&record.Method,
			&record.Params,
			&record.Result,
			&record.Error,
			&record.PerformanceMS,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// GetRecentSessions retrieves the most recent sessions
func (cs *ConversationStorage) GetRecentSessions(limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
	SELECT DISTINCT session_id
	FROM conversations
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := cs.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// GetMessagesByMethod retrieves messages filtered by method
func (cs *ConversationStorage) GetMessagesByMethod(method string, limit int) ([]ConversationRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
	SELECT id, session_id, timestamp, direction, method, params, result, error, performance_ms
	FROM conversations
	WHERE method = ?
	ORDER BY timestamp DESC
	LIMIT ?`

	rows, err := cs.db.Query(query, method, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ConversationRecord
	for rows.Next() {
		var record ConversationRecord
		err := rows.Scan(
			&record.ID,
			&record.SessionID,
			&record.Timestamp,
			&record.Direction,
			&record.Method,
			&record.Params,
			&record.Result,
			&record.Error,
			&record.PerformanceMS,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// GetStats returns basic statistics about the conversation log
func (cs *ConversationStorage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total messages
	var totalMessages int64
	err := cs.db.QueryRow("SELECT COUNT(*) FROM conversations").Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	stats["total_messages"] = totalMessages

	// Total sessions
	var totalSessions int64
	err = cs.db.QueryRow("SELECT COUNT(DISTINCT session_id) FROM conversations").Scan(&totalSessions)
	if err != nil {
		return nil, err
	}
	stats["total_sessions"] = totalSessions

	// Messages by method
	methodRows, err := cs.db.Query(`
		SELECT method, COUNT(*) 
		FROM conversations 
		WHERE method IS NOT NULL AND method != ''
		GROUP BY method 
		ORDER BY COUNT(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer methodRows.Close()

	methodCounts := make(map[string]int64)
	for methodRows.Next() {
		var method string
		var count int64
		if err := methodRows.Scan(&method, &count); err != nil {
			return nil, err
		}
		methodCounts[method] = count
	}
	stats["methods"] = methodCounts

	// Average performance
	var avgPerformance float64
	err = cs.db.QueryRow("SELECT AVG(performance_ms) FROM conversations WHERE performance_ms > 0").Scan(&avgPerformance)
	if err != nil {
		avgPerformance = 0
	}
	stats["avg_performance_ms"] = avgPerformance

	return stats, nil
}

// CleanupOldRecords removes records older than the specified duration
func (cs *ConversationStorage) CleanupOldRecords(maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)

	query := `DELETE FROM conversations WHERE timestamp < ?`
	result, err := cs.db.Exec(query, cutoff)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d old conversation records", rowsAffected)
	}

	// Also cleanup sessions with no messages
	sessionQuery := `DELETE FROM sessions WHERE session_id NOT IN (SELECT DISTINCT session_id FROM conversations)`
	sessionResult, err := cs.db.Exec(sessionQuery)
	if err != nil {
		return err
	}

	sessionRowsAffected, _ := sessionResult.RowsAffected()
	if sessionRowsAffected > 0 {
		log.Printf("Cleaned up %d orphaned session records", sessionRowsAffected)
	}

	return nil
}

// Close closes the database connection
func (cs *ConversationStorage) Close() error {
	if cs.db != nil {
		return cs.db.Close()
	}
	return nil
}

// Enable/Disable logging
func (cs *ConversationStorage) SetEnabled(enabled bool) {
	cs.enabled = enabled
}

func (cs *ConversationStorage) IsEnabled() bool {
	return cs.enabled
}
