package debug

import (
	"encoding/json"
	"log"
	"time"

	"github.com/vcto/cowpilot/internal/protocol/validation"
)

// ValidatedMessageInterceptor extends MessageInterceptor with validation
type ValidatedMessageInterceptor struct {
	*MessageInterceptor
	validationEngine  *validation.ValidationEngine
	validationStorage ValidationStorage
}

// ValidationStorage interface for storing validation reports
type ValidationStorage interface {
	StoreValidationReport(report *validation.ValidationReport) error
	GetValidationReports(sessionID string, limit int) ([]*validation.ValidationReport, error)
	GetValidationStats() (map[string]interface{}, error)
}

// NewValidatedMessageInterceptor creates an interceptor with validation
func NewValidatedMessageInterceptor(storage Storage, config *DebugConfig) *ValidatedMessageInterceptor {
	baseInterceptor := NewMessageInterceptor(storage, config)

	// Create validation engine
	validationConfig := &validation.ValidatorConfig{
		Enabled:    config.Enabled,
		StrictMode: config.Level == "DEBUG",
	}

	engine := validation.NewValidationEngine(validationConfig)

	// Register validators
	engine.RegisterValidator(validation.NewJSONRPCValidator())
	engine.RegisterValidator(validation.NewMCPValidator())
	engine.RegisterValidator(validation.NewSecurityValidator())

	// Use the same storage for validation reports if it implements ValidationStorage
	var validationStorage ValidationStorage
	if vs, ok := storage.(ValidationStorage); ok {
		validationStorage = vs
	} else {
		validationStorage = &NoOpValidationStorage{}
	}

	return &ValidatedMessageInterceptor{
		MessageInterceptor: baseInterceptor,
		validationEngine:   engine,
		validationStorage:  validationStorage,
	}
}

// LogRequestWithValidation logs and validates an incoming request
func (vmi *ValidatedMessageInterceptor) LogRequestWithValidation(method string, params interface{}) *validation.ValidationReport {
	// First log normally
	vmi.LogRequest(method, params)

	if !vmi.validationEngine.GetStats()["enabled"].(bool) {
		return nil
	}

	// Create message data for validation
	messageData := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      generateMessageID(),
	}

	return vmi.validateMessage(messageData, "request")
}

// LogResponseWithValidation logs and validates a response
func (vmi *ValidatedMessageInterceptor) LogResponseWithValidation(method string, result interface{}, errorMsg interface{}, performanceMS int64) *validation.ValidationReport {
	// First log normally
	vmi.LogResponse(method, result, errorMsg, performanceMS)

	if !vmi.validationEngine.GetStats()["enabled"].(bool) {
		return nil
	}

	// Create message data for validation
	messageData := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      generateMessageID(),
	}

	if errorMsg != nil {
		messageData["error"] = errorMsg
	} else {
		messageData["result"] = result
	}

	return vmi.validateMessage(messageData, "response")
}

func (vmi *ValidatedMessageInterceptor) validateMessage(messageData map[string]interface{}, msgType string) *validation.ValidationReport {
	// Convert to JSON for validation
	jsonData, err := json.Marshal(messageData)
	if err != nil {
		log.Printf("Validation error: failed to marshal message: %v", err)
		return nil
	}

	// Run validation
	messageID := generateMessageID()
	context := map[string]string{
		"session_id": vmi.GetSessionID(),
		"direction":  msgType,
	}

	report, err := vmi.validationEngine.ValidateMessage(
		vmi.GetSessionID(),
		messageID,
		jsonData,
		context,
	)

	if err != nil {
		log.Printf("Validation error: %v", err)
		return nil
	}

	// Store validation report
	if err := vmi.validationStorage.StoreValidationReport(report); err != nil {
		log.Printf("Failed to store validation report: %v", err)
	}

	// Log validation results if debug level
	if vmi.config.Level == "DEBUG" && len(report.Results) > 0 {
		log.Printf("[VALIDATION] %s: %d issues, score: %.1f, valid: %v",
			msgType, len(report.Results), report.Score, report.IsValid)

		for _, result := range report.Results {
			log.Printf("[VALIDATION] %s: %s - %s",
				result.Level.String(), result.ID, result.Message)
		}
	}

	return report
}

func generateMessageID() string {
	return time.Now().Format("20060102150405.000000")
}

// NoOpValidationStorage provides no-op validation storage
type NoOpValidationStorage struct{}

func (n *NoOpValidationStorage) StoreValidationReport(report *validation.ValidationReport) error {
	return nil
}

func (n *NoOpValidationStorage) GetValidationReports(sessionID string, limit int) ([]*validation.ValidationReport, error) {
	return nil, nil
}

func (n *NoOpValidationStorage) GetValidationStats() (map[string]interface{}, error) {
	return map[string]interface{}{
		"validation_enabled": false,
	}, nil
}

// Enhanced FileStorage with validation support
func (fs *FileStorage) StoreValidationReport(report *validation.ValidationReport) error {
	if !fs.enabled {
		return nil
	}

	// Serialize the report
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}

	query := `
	INSERT INTO validation_reports (
		session_id, message_id, message_type, method, score, is_valid, 
		results_json, processing_ms, timestamp
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = fs.db.Exec(query,
		report.SessionID,
		report.MessageID,
		report.MessageType,
		report.Method,
		report.Score,
		report.IsValid,
		string(reportJSON),
		report.ProcessingMS,
		report.Timestamp,
	)

	return err
}

func (fs *FileStorage) GetValidationReports(sessionID string, limit int) ([]*validation.ValidationReport, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
	SELECT results_json FROM validation_reports 
	WHERE session_id = ? 
	ORDER BY timestamp DESC 
	LIMIT ?`

	rows, err := fs.db.Query(query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var reports []*validation.ValidationReport
	for rows.Next() {
		var reportJSON string
		if err := rows.Scan(&reportJSON); err != nil {
			continue
		}

		var report validation.ValidationReport
		if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
			continue
		}

		reports = append(reports, &report)
	}

	return reports, nil
}

func (fs *FileStorage) GetValidationStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"validation_enabled": true,
	}

	// Total validation reports
	var totalReports int64
	if err := fs.db.QueryRow("SELECT COUNT(*) FROM validation_reports").Scan(&totalReports); err != nil {
		log.Printf("Failed to get total reports: %v", err)
	}
	stats["total_reports"] = totalReports

	// Average score
	var avgScore float64
	if err := fs.db.QueryRow("SELECT AVG(score) FROM validation_reports").Scan(&avgScore); err != nil {
		log.Printf("Failed to get average score: %v", err)
	}
	stats["average_score"] = avgScore

	// Validity rate
	var validCount int64
	if err := fs.db.QueryRow("SELECT COUNT(*) FROM validation_reports WHERE is_valid = 1").Scan(&validCount); err != nil {
		log.Printf("Failed to get valid count: %v", err)
	}
	stats["validity_rate"] = float64(validCount) / float64(totalReports) * 100.0

	return stats, nil
}
