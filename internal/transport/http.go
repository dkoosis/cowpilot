package transport

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/vcto/cowpilot/internal/mcp"
)

// HTTPTransport handles HTTP streaming for MCP
type HTTPTransport struct {
	server *mcp.Server
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(server *mcp.Server) *HTTPTransport {
	return &HTTPTransport{
		server: server,
	}
}

// HandleMCP handles MCP requests over HTTP streaming
func (t *HTTPTransport) HandleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	
	// Enable flushing for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, -32700, "Parse error")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request
	var req mcp.Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, -32700, "Parse error")
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		writeError(w, -32600, "Invalid request")
		return
	}

	// Handle request
	resp := t.server.HandleRequest(req)

	// Write response
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Can't write error at this point, just log it
		return
	}
	
	flusher.Flush()
}

func writeError(w http.ResponseWriter, code int, message string) {
	resp := mcp.Response{
		JSONRPC: "2.0",
		Error: &mcp.Error{
			Code:    code,
			Message: message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

// StreamReader handles streaming JSON-RPC messages
type StreamReader struct {
	reader  *bufio.Reader
	decoder *json.Decoder
}

// NewStreamReader creates a new stream reader
func NewStreamReader(r io.Reader) *StreamReader {
	reader := bufio.NewReader(r)
	return &StreamReader{
		reader:  reader,
		decoder: json.NewDecoder(reader),
	}
}

// ReadMessage reads a single JSON-RPC message from the stream
func (sr *StreamReader) ReadMessage() (*mcp.Request, error) {
	var req mcp.Request
	if err := sr.decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}
	return &req, nil
}
