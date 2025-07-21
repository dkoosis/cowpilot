package mcp

import (
	"encoding/json"
	"fmt"
)

// Server implements the MCP server
type Server struct {
	tools map[string]Tool
}

// Tool represents an MCP tool
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines tool parameters
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property defines a parameter property
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Request represents a JSON-RPC request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Response represents a JSON-RPC response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewServer creates a new MCP server
func NewServer() *Server {
	s := &Server{
		tools: make(map[string]Tool),
	}

	// Register hello tool
	s.RegisterTool(Tool{
		Name:        "hello",
		Description: "Says hello to the world",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: make(map[string]Property),
		},
	})

	return s
}

// RegisterTool registers a new tool
func (s *Server) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
}

// HandleRequest processes a JSON-RPC request
func (s *Server) HandleRequest(req Request) Response {
	switch req.Method {
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32601,
				Message: "Method not found",
			},
			ID: req.ID,
		}
	}
}

func (s *Server) handleToolsList(req Request) Response {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return Response{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"tools": tools,
		},
		ID: req.ID,
	}
}

func (s *Server) handleToolCall(req Request) Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32602,
				Message: "Invalid params",
			},
			ID: req.ID,
		}
	}

	if params.Name != "hello" {
		return Response{
			JSONRPC: "2.0",
			Error: &Error{
				Code:    -32602,
				Message: fmt.Sprintf("Unknown tool: %s", params.Name),
			},
			ID: req.ID,
		}
	}

	// Execute hello tool
	return Response{
		JSONRPC: "2.0",
		Result: map[string]interface{}{
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Hello, World!",
				},
			},
		},
		ID: req.ID,
	}
}
