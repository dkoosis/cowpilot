package validation

import (
	"fmt"
	"reflect"
	"strings"
)

// MCPValidator validates MCP protocol-specific semantics
type MCPValidator struct{}

// NewMCPValidator creates a new MCP semantic validator
func NewMCPValidator() *MCPValidator {
	return &MCPValidator{}
}

func (v *MCPValidator) GetName() string {
	return "mcp_semantic_validator"
}

func (v *MCPValidator) GetDescription() string {
	return "Validates MCP protocol semantics for tools, resources, and prompts"
}

func (v *MCPValidator) Validate(message *MCPMessage, context map[string]string) *ValidationReport {
	report := &ValidationReport{Results: make([]ValidationResult, 0)}

	// Route to specific validators based on method
	switch message.Method {
	case "tools/list":
		v.validateToolsList(message, report)
	case "tools/call":
		v.validateToolsCall(message, report)
	case "resources/list":
		v.validateResourcesList(message, report)
	case "resources/read":
		v.validateResourcesRead(message, report)
	case "prompts/list":
		v.validatePromptsList(message, report)
	case "prompts/get":
		v.validatePromptsGet(message, report)
	case "initialize":
		v.validateInitialize(message, report)
	case "ping":
		v.validatePing(message, report)
	default:
		if strings.HasPrefix(message.Method, "tools/") ||
			strings.HasPrefix(message.Method, "resources/") ||
			strings.HasPrefix(message.Method, "prompts/") {
			report.AddWarning("mcp_unknown_method",
				fmt.Sprintf("Unknown MCP method: %s", message.Method),
				map[string]string{"method": message.Method})
		}
	}

	return report
}

// Tools validation
func (v *MCPValidator) validateToolsList(message *MCPMessage, report *ValidationReport) {
	// tools/list should have no required parameters
	if len(message.Params) > 0 {
		report.AddWarning("tools_list_unexpected_params",
			"tools/list typically doesn't require parameters",
			map[string]string{"param_count": fmt.Sprintf("%d", len(message.Params))})
	}
}

func (v *MCPValidator) validateToolsCall(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		report.AddCritical("tools_call_missing_params", "tools/call requires parameters", nil)
		return
	}

	// Check required fields
	name, hasName := message.Params["name"]
	arguments, hasArguments := message.Params["arguments"]

	if !hasName {
		report.AddCritical("tools_call_missing_name", "tools/call requires 'name' parameter", nil)
	} else if nameStr, ok := name.(string); !ok || nameStr == "" {
		report.AddError("tools_call_invalid_name", "Tool name must be a non-empty string", nil)
	}

	if !hasArguments {
		report.AddError("tools_call_missing_arguments", "tools/call requires 'arguments' parameter", nil)
	} else if reflect.TypeOf(arguments).Kind() != reflect.Map {
		report.AddError("tools_call_invalid_arguments", "Tool arguments must be an object", nil)
	}
}

// Resources validation
func (v *MCPValidator) validateResourcesList(message *MCPMessage, report *ValidationReport) {
	// resources/list may have cursor for pagination
	if message.Params != nil {
		if cursor, exists := message.Params["cursor"]; exists {
			if _, ok := cursor.(string); !ok {
				report.AddError("resources_list_invalid_cursor", "Cursor must be a string", nil)
			}
		}
	}
}

func (v *MCPValidator) validateResourcesRead(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		report.AddCritical("resources_read_missing_params", "resources/read requires parameters", nil)
		return
	}

	uri, hasURI := message.Params["uri"]
	if !hasURI {
		report.AddCritical("resources_read_missing_uri", "resources/read requires 'uri' parameter", nil)
	} else if uriStr, ok := uri.(string); !ok || uriStr == "" {
		report.AddError("resources_read_invalid_uri", "URI must be a non-empty string", nil)
	} else {
		v.validateResourceURI(uriStr, report)
	}
}

func (v *MCPValidator) validateResourceURI(uri string, report *ValidationReport) {
	if !strings.Contains(uri, "://") {
		report.AddWarning("resource_uri_no_scheme",
			"Resource URI should include a scheme (e.g., file://, http://, custom://)",
			map[string]string{"uri": uri})
	}

	// Check for path traversal patterns
	if strings.Contains(uri, "../") || strings.Contains(uri, "..\\") {
		report.AddError("resource_uri_path_traversal",
			"Resource URI contains potential path traversal patterns",
			map[string]string{"uri": uri})
	}
}

// *** CRITICAL: Prompts validation (explicit) ***
func (v *MCPValidator) validatePromptsList(message *MCPMessage, report *ValidationReport) {
	// prompts/list may have cursor for pagination
	if message.Params != nil {
		if cursor, exists := message.Params["cursor"]; exists {
			if _, ok := cursor.(string); !ok {
				report.AddError("prompts_list_invalid_cursor", "Cursor must be a string", nil)
			}
		}
	}
}

func (v *MCPValidator) validatePromptsGet(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		report.AddCritical("prompts_get_missing_params", "prompts/get requires parameters", nil)
		return
	}

	// Validate required 'name' field
	name, hasName := message.Params["name"]
	if !hasName {
		report.AddCritical("prompts_get_missing_name", "prompts/get requires 'name' parameter", nil)
	} else if nameStr, ok := name.(string); !ok || nameStr == "" {
		report.AddError("prompts_get_invalid_name", "Prompt name must be a non-empty string", nil)
	}

	// Validate 'arguments' field if present
	if arguments, hasArguments := message.Params["arguments"]; hasArguments {
		v.validatePromptArguments(arguments, report)
	}
}

// *** CRITICAL: Explicit prompt arguments validation ***
func (v *MCPValidator) validatePromptArguments(arguments interface{}, report *ValidationReport) {
	// Arguments must be an object (map)
	argsMap, ok := arguments.(map[string]interface{})
	if !ok {
		report.AddCritical("prompts_invalid_arguments_type",
			"Prompt arguments must be an object/map",
			map[string]string{"actual_type": fmt.Sprintf("%T", arguments)})
		return
	}

	// Validate each argument value
	for key, value := range argsMap {
		// Check key format
		if key == "" {
			report.AddError("prompts_empty_argument_key", "Argument keys cannot be empty", nil)
			continue
		}

		// Validate argument value types (MCP spec: arguments should be simple types)
		v.validatePromptArgumentValue(key, value, report)
	}
}

func (v *MCPValidator) validatePromptArgumentValue(key string, value interface{}, report *ValidationReport) {
	switch v := value.(type) {
	case string:
		// Strings are valid - check for potential injection patterns
		if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
			report.AddWarning("prompts_nested_template_syntax",
				fmt.Sprintf("Argument '%s' contains template-like syntax", key),
				map[string]string{"argument": key, "pattern": "{{...}}"})
		}
	case bool, int, int64, float64:
		// Basic types are valid
	case nil:
		// Null is valid
	case map[string]interface{}, []interface{}:
		// Complex types should be warned about
		report.AddWarning("prompts_complex_argument_type",
			fmt.Sprintf("Argument '%s' uses complex type %T - simple types preferred", key, value),
			map[string]string{"argument": key, "type": fmt.Sprintf("%T", value)})
	default:
		report.AddError("prompts_unsupported_argument_type",
			fmt.Sprintf("Argument '%s' has unsupported type %T", key, value),
			map[string]string{"argument": key, "type": fmt.Sprintf("%T", value)})
	}
}

// Core MCP methods validation
func (v *MCPValidator) validateInitialize(message *MCPMessage, report *ValidationReport) {
	if message.Params == nil {
		report.AddCritical("initialize_missing_params", "initialize requires parameters", nil)
		return
	}

	// Check protocol version
	if protocolVersion, exists := message.Params["protocolVersion"]; exists {
		if version, ok := protocolVersion.(string); ok {
			v.validateProtocolVersion(version, report)
		} else {
			report.AddError("initialize_invalid_protocol_version", "protocolVersion must be a string", nil)
		}
	} else {
		report.AddError("initialize_missing_protocol_version", "initialize requires protocolVersion parameter", nil)
	}

	// Check capabilities
	if capabilities, exists := message.Params["capabilities"]; exists {
		if capsMap, ok := capabilities.(map[string]interface{}); ok {
			v.validateCapabilities(capsMap, report)
		} else {
			report.AddError("initialize_invalid_capabilities", "capabilities must be an object", nil)
		}
	}

	// Check client info
	if clientInfo, exists := message.Params["clientInfo"]; exists {
		if infoMap, ok := clientInfo.(map[string]interface{}); ok {
			v.validateClientInfo(infoMap, report)
		} else {
			report.AddError("initialize_invalid_client_info", "clientInfo must be an object", nil)
		}
	}
}

func (v *MCPValidator) validateProtocolVersion(version string, report *ValidationReport) {
	// Check for semantic versioning format
	if !strings.Contains(version, ".") {
		report.AddWarning("protocol_version_format",
			"Protocol version should follow semantic versioning (e.g., '2024-11-05')",
			map[string]string{"version": version})
	}
}

func (v *MCPValidator) validateCapabilities(capabilities map[string]interface{}, report *ValidationReport) {
	knownCapabilities := []string{"roots", "sampling", "tools", "resources", "prompts", "logging"}

	for capability := range capabilities {
		found := false
		for _, known := range knownCapabilities {
			if capability == known {
				found = true
				break
			}
		}
		if !found {
			report.AddWarning("unknown_capability",
				fmt.Sprintf("Unknown capability: %s", capability),
				map[string]string{"capability": capability})
		}
	}
}

func (v *MCPValidator) validateClientInfo(clientInfo map[string]interface{}, report *ValidationReport) {
	if name, exists := clientInfo["name"]; exists {
		if _, ok := name.(string); !ok {
			report.AddError("client_info_invalid_name", "clientInfo.name must be a string", nil)
		}
	}

	if version, exists := clientInfo["version"]; exists {
		if _, ok := version.(string); !ok {
			report.AddError("client_info_invalid_version", "clientInfo.version must be a string", nil)
		}
	}
}

func (v *MCPValidator) validatePing(message *MCPMessage, report *ValidationReport) {
	// ping should have no parameters or empty parameters
	if len(message.Params) > 0 {
		report.AddWarning("ping_unexpected_params", "ping method typically has no parameters", nil)
	}
}
