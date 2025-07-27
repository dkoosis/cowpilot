package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// SecurityValidator detects security issues in MCP messages
type SecurityValidator struct {
	sqlInjectionPatterns     []*regexp.Regexp
	commandInjectionPatterns []*regexp.Regexp
	pathTraversalPatterns    []*regexp.Regexp
	xssPatterns              []*regexp.Regexp
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{
		sqlInjectionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(union\s+select|drop\s+table|delete\s+from|insert\s+into|update\s+set)`),
			regexp.MustCompile(`(?i)('\s*or\s*'\s*=\s*'|'\s*or\s*1\s*=\s*1|--\s*|/\*|\*/)`),
			regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|sp_executesql)`),
		},
		commandInjectionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(;|\||&&|\$\(|` + "`" + `)`),
			regexp.MustCompile(`(?i)(rm\s+-rf|sudo|curl\s+|wget\s+|nc\s+|netcat)`),
			regexp.MustCompile(`(>\s*/dev/null|2>&1|</dev/null)`),
		},
		pathTraversalPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(\.\./|\.\\\\|\.\.%2f|\.\.%5c)`),
			regexp.MustCompile(`(%2e%2e%2f|%2e%2e%5c|%252e%252e%252f)`),
		},
		xssPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(<script|javascript:|on\w+\s*=|<iframe|<object|<embed)`),
			regexp.MustCompile(`(?i)(alert\s*\(|confirm\s*\(|prompt\s*\(|eval\s*\()`),
		},
	}
}

func (v *SecurityValidator) GetName() string {
	return "security_validator"
}

func (v *SecurityValidator) GetDescription() string {
	return "Detects security vulnerabilities in MCP messages"
}

func (v *SecurityValidator) Validate(message *MCPMessage, context map[string]string) *ValidationReport {
	report := &ValidationReport{Results: make([]ValidationResult, 0)}

	// Check all string values in the message
	v.checkMessageForThreats(message, "", report)

	// Method-specific security checks
	switch message.Method {
	case "tools/call":
		v.validateToolCallSecurity(message, report)
	case "resources/read":
		v.validateResourceReadSecurity(message, report)
	case "prompts/get":
		v.validatePromptGetSecurity(message, report)
	}

	// Check for suspicious patterns in context
	v.checkContext(context, report)

	return report
}

func (v *SecurityValidator) checkMessageForThreats(obj interface{}, path string, report *ValidationReport) {
	switch val := obj.(type) {
	case string:
		v.scanString(val, path, report)
	case map[string]interface{}:
		for key, value := range val {
			newPath := path
			if newPath != "" {
				newPath += "."
			}
			newPath += key
			v.checkMessageForThreats(value, newPath, report)
		}
	case []interface{}:
		for i, value := range val {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			v.checkMessageForThreats(value, newPath, report)
		}
	}
}

func (v *SecurityValidator) scanString(value, path string, report *ValidationReport) {
	// SQL Injection Detection
	for _, pattern := range v.sqlInjectionPatterns {
		if pattern.MatchString(value) {
			report.AddCritical("security_sql_injection_detected",
				fmt.Sprintf("Potential SQL injection detected in %s", path),
				map[string]string{
					"field":   path,
					"pattern": pattern.String(),
					"value":   v.truncateValue(value),
				})
		}
	}

	// Command Injection Detection
	for _, pattern := range v.commandInjectionPatterns {
		if pattern.MatchString(value) {
			report.AddCritical("security_command_injection_detected",
				fmt.Sprintf("Potential command injection detected in %s", path),
				map[string]string{
					"field":   path,
					"pattern": pattern.String(),
					"value":   v.truncateValue(value),
				})
		}
	}

	// Path Traversal Detection
	for _, pattern := range v.pathTraversalPatterns {
		if pattern.MatchString(value) {
			report.AddError("security_path_traversal_detected",
				fmt.Sprintf("Potential path traversal detected in %s", path),
				map[string]string{
					"field":   path,
					"pattern": pattern.String(),
					"value":   v.truncateValue(value),
				})
		}
	}

	// XSS Detection
	for _, pattern := range v.xssPatterns {
		if pattern.MatchString(value) {
			report.AddError("security_xss_detected",
				fmt.Sprintf("Potential XSS payload detected in %s", path),
				map[string]string{
					"field":   path,
					"pattern": pattern.String(),
					"value":   v.truncateValue(value),
				})
		}
	}

	// Additional checks
	v.checkForSuspiciousPatterns(value, path, report)
}

func (v *SecurityValidator) checkForSuspiciousPatterns(value, path string, report *ValidationReport) {
	// Excessively long strings (potential DoS)
	if len(value) > 10000 {
		report.AddWarning("security_excessive_length",
			fmt.Sprintf("Unusually long string in %s (%d chars)", path, len(value)),
			map[string]string{"field": path, "length": fmt.Sprintf("%d", len(value))})
	}

	// Binary data in string fields
	if v.containsBinaryData(value) {
		report.AddWarning("security_binary_data",
			fmt.Sprintf("Potential binary data in string field %s", path),
			map[string]string{"field": path})
	}

	// Multiple encoding attempts
	if strings.Count(value, "%") > 5 {
		report.AddWarning("security_multiple_encoding",
			fmt.Sprintf("Multiple URL encoding patterns in %s", path),
			map[string]string{"field": path, "percent_count": fmt.Sprintf("%d", strings.Count(value, "%"))})
	}
}

func (v *SecurityValidator) validateToolCallSecurity(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		return
	}

	// Check tool name for suspicious patterns
	if name, exists := message.Params["name"]; exists {
		if nameStr, ok := name.(string); ok {
			if strings.Contains(nameStr, "..") || strings.Contains(nameStr, "/") || strings.Contains(nameStr, "\\") {
				report.AddWarning("security_suspicious_tool_name",
					"Tool name contains path-like characters",
					map[string]string{"tool_name": nameStr})
			}
		}
	}

	// Check arguments for injection patterns
	if args, exists := message.Params["arguments"]; exists {
		if argsMap, ok := args.(map[string]interface{}); ok {
			v.validateToolArguments(argsMap, report)
		}
	}
}

func (v *SecurityValidator) validateToolArguments(args map[string]interface{}, report *ValidationReport) {
	suspiciousArgs := []string{"command", "cmd", "exec", "eval", "system", "shell", "script"}

	for argName := range args {
		for _, suspicious := range suspiciousArgs {
			if strings.ToLower(argName) == suspicious {
				report.AddWarning("security_suspicious_argument_name",
					fmt.Sprintf("Tool argument name '%s' could indicate command execution", argName),
					map[string]string{"argument": argName})
			}
		}
	}
}

func (v *SecurityValidator) validateResourceReadSecurity(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		return
	}

	if uri, exists := message.Params["uri"]; exists {
		if uriStr, ok := uri.(string); ok {
			// Check for suspicious schemes
			suspiciousSchemes := []string{"file://", "ftp://", "smb://", "ldap://"}
			for _, scheme := range suspiciousSchemes {
				if strings.HasPrefix(strings.ToLower(uriStr), scheme) {
					report.AddWarning("security_suspicious_uri_scheme",
						fmt.Sprintf("Resource URI uses potentially risky scheme: %s", scheme),
						map[string]string{"uri": uriStr, "scheme": scheme})
				}
			}

			// Check for internal/private addresses
			internalPatterns := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "192.168.", "10.", "172.1", "172.2", "172.3"}
			for _, pattern := range internalPatterns {
				if strings.Contains(strings.ToLower(uriStr), pattern) {
					report.AddError("security_internal_address",
						"Resource URI targets internal/private address",
						map[string]string{"uri": uriStr, "pattern": pattern})
				}
			}
		}
	}
}

func (v *SecurityValidator) validatePromptGetSecurity(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		return
	}

	// Check prompt arguments for template injection
	if args, exists := message.Params["arguments"]; exists {
		if argsMap, ok := args.(map[string]interface{}); ok {
			for key, value := range argsMap {
				if valueStr, ok := value.(string); ok {
					// Check for template injection patterns
					templatePatterns := []string{"${", "#{", "{{", "<%", "[["}
					for _, pattern := range templatePatterns {
						if strings.Contains(valueStr, pattern) {
							report.AddWarning("security_template_injection",
								fmt.Sprintf("Prompt argument '%s' contains template-like syntax", key),
								map[string]string{"argument": key, "pattern": pattern})
						}
					}
				}
			}
		}
	}
}

func (v *SecurityValidator) checkContext(context map[string]string, report *ValidationReport) {
	if context == nil {
		return
	}

	// Check for suspicious context values
	for key, value := range context {
		if strings.Contains(strings.ToLower(key), "password") ||
			strings.Contains(strings.ToLower(key), "token") ||
			strings.Contains(strings.ToLower(key), "secret") {
			report.AddCritical("security_sensitive_context",
				fmt.Sprintf("Context contains potentially sensitive key: %s", key),
				map[string]string{"context_key": key})
		}

		// Scan context values for threats
		v.scanString(value, fmt.Sprintf("context.%s", key), report)
	}
}

func (v *SecurityValidator) containsBinaryData(s string) bool {
	// Simple check for non-printable characters
	for _, r := range s {
		if r < 32 && r != 9 && r != 10 && r != 13 { // Allow tab, LF, CR
			return true
		}
	}
	return false
}

func (v *SecurityValidator) truncateValue(value string) string {
	if len(value) > 100 {
		return value[:100] + "..."
	}
	return value
}
