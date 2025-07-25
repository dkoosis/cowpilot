package validator

import (
	"fmt"
	"reflect"
)

// JSONRPCValidator validates JSON-RPC 2.0 compliance
type JSONRPCValidator struct{}

// NewJSONRPCValidator creates a new JSON-RPC validator
func NewJSONRPCValidator() *JSONRPCValidator {
	return &JSONRPCValidator{}
}

// GetName returns the validator name
func (v *JSONRPCValidator) GetName() string {
	return "json_rpc_validator"
}

// GetDescription returns the validator description
func (v *JSONRPCValidator) GetDescription() string {
	return "Validates JSON-RPC 2.0 protocol compliance"
}

// Validate validates a message against JSON-RPC 2.0 specification
func (v *JSONRPCValidator) Validate(message *MCPMessage, context map[string]string) *ValidationReport {
	report := &ValidationReport{
		Results: make([]ValidationResult, 0),
	}

	// 1. Check JSON-RPC version
	if !IsValidJSONRPCVersion(message.JSONRPCVersion) {
		report.AddCritical("jsonrpc_invalid_version", 
			fmt.Sprintf("Invalid JSON-RPC version: %s (must be '2.0')", message.JSONRPCVersion),
			map[string]string{
				"expected": "2.0",
				"actual":   message.JSONRPCVersion,
			})
	}

	// 2. Determine message type and validate structure
	if message.Method != "" {
		// Request or Notification
		v.validateRequest(message, report)
	} else if message.Result != nil || message.Error != nil {
		// Response
		v.validateResponse(message, report)
	} else {
		report.AddCritical("jsonrpc_invalid_structure",
			"Invalid message structure: missing method (request/notification) or result/error (response)",
			nil)
	}

	return report
}

// validateRequest validates request and notification messages
func (v *JSONRPCValidator) validateRequest(message *MCPMessage, report *ValidationReport) {
	// Method is required and must be a string
	if message.Method == "" {
		report.AddError("jsonrpc_missing_method", "Method field is required for requests", nil)
	} else if !IsValidMCPMethod(message.Method) {
		report.AddError("jsonrpc_invalid_method", 
			fmt.Sprintf("Invalid method name: %s", message.Method),
			map[string]string{"method": message.Method})
	}

	// ID validation for requests vs notifications
	if message.ID != nil {
		// Request - ID should be valid
		if !IsValidID(message.ID) {
			report.AddError("jsonrpc_invalid_id",
				fmt.Sprintf("Invalid ID type: %T (must be string, number, or null)", message.ID),
				map[string]string{"id_type": fmt.Sprintf("%T", message.ID)})
		}
	}
	// Note: Notifications have no ID field requirement

	// Params should be structured data if present
	if message.Params != nil {
		v.validateParams(message.Params, report)
	}
}

// validateResponse validates response messages
func (v *JSONRPCValidator) validateResponse(message *MCPMessage, report *ValidationReport) {
	// Response must have ID
	if message.ID == nil {
		report.AddError("jsonrpc_missing_response_id", "Response messages must include an ID", nil)
	} else if !IsValidID(message.ID) {
		report.AddError("jsonrpc_invalid_response_id",
			fmt.Sprintf("Invalid response ID type: %T", message.ID),
			map[string]string{"id_type": fmt.Sprintf("%T", message.ID)})
	}

	// Response must have either result or error, but not both
	hasResult := message.Result != nil
	hasError := message.Error != nil

	if !hasResult && !hasError {
		report.AddCritical("jsonrpc_missing_result_error",
			"Response must contain either 'result' or 'error' field", nil)
	} else if hasResult && hasError {
		report.AddError("jsonrpc_both_result_error",
			"Response cannot contain both 'result' and 'error' fields", nil)
	}

	// Validate error structure if present
	if hasError {
		v.validateError(message.Error, report)
	}
}

// validateParams validates parameter structure
func (v *JSONRPCValidator) validateParams(params map[string]interface{}, report *ValidationReport) {
	// Params should be an object (map) - arrays are discouraged in MCP
	if params == nil {
		return // null params are valid
	}

	// Check for array-style params (discouraged)
	if reflect.TypeOf(params).Kind() == reflect.Slice {
		report.AddWarning("jsonrpc_array_params",
			"Array-style parameters are discouraged in MCP, use object-style instead",
			map[string]string{"recommendation": "Use object with named parameters"})
	}
}

// validateError validates error object structure
func (v *JSONRPCValidator) validateError(error *MCPError, report *ValidationReport) {
	if error == nil {
		return
	}

	// Error code is required
	if error.Code == 0 {
		report.AddError("jsonrpc_missing_error_code", "Error object must include a non-zero code", nil)
	}

	// Error message is required
	if error.Message == "" {
		report.AddError("jsonrpc_missing_error_message", "Error object must include a message", nil)
	}

	// Validate standard error codes
	v.validateErrorCode(error.Code, report)
}

// validateErrorCode validates JSON-RPC error codes
func (v *JSONRPCValidator) validateErrorCode(code int, report *ValidationReport) {
	validCodes := map[int]string{
		-32700: "Parse error",
		-32600: "Invalid Request",
		-32601: "Method not found",
		-32602: "Invalid params",
		-32603: "Internal error",
	}

	// Standard JSON-RPC errors (-32768 to -32000)
	if code >= -32768 && code <= -32000 {
		if description, exists := validCodes[code]; exists {
			// Known standard error
			report.AddResult(ValidationResult{
				ID:      "jsonrpc_standard_error_code",
				Level:   LevelInfo,
				Message: fmt.Sprintf("Using standard JSON-RPC error code: %s (%d)", description, code),
				Context: map[string]string{"error_code": fmt.Sprintf("%d", code)},
			})
		} else {
			report.AddWarning("jsonrpc_reserved_error_code",
				fmt.Sprintf("Error code %d is in reserved range (-32768 to -32000) but not a standard code", code),
				map[string]string{"error_code": fmt.Sprintf("%d", code)})
		}
	}

	// Application-specific errors should be outside reserved range
	if code > -32000 || code < -32768 {
		report.AddResult(ValidationResult{
			ID:      "jsonrpc_application_error_code",
			Level:   LevelInfo,
			Message: fmt.Sprintf("Using application-specific error code: %d", code),
			Context: map[string]string{"error_code": fmt.Sprintf("%d", code)},
		})
	}
}
