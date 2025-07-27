package testutil

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// NewCallToolRequest creates a CallToolRequest for testing tool handlers
func NewCallToolRequest(name string, params map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: params,
		},
	}
}
