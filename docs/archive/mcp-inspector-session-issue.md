# MCP Inspector & StreamableHTTP Session Management Issue

## Problem Summary
MCP Inspector fails with "SSE error: TypeError: terminated: Body Timeout Error" when testing our StreamableHTTP server, while raw curl/JSON-RPC tests work fine.

## Root Causes

### 1. Transport Auto-Detection
MCP Inspector selects transport based on URL pattern:
- URL ending with `/mcp` → HTTP transport (stateful POST/response)
- URL ending with `/sse` → SSE transport (Server-Sent Events)
- Any other URL → **SSE transport by default**

Our server at `http://localhost:8080/` triggers SSE mode, not HTTP POST mode.

### 2. Session Management Requirements
StreamableHTTPServer default behavior:
- Uses `InsecureStatefulSessionIdManager` by default
- `initialize` request returns `Mcp-Session-Id` header
- All subsequent requests MUST include this header
- Missing/invalid session ID → "Invalid session ID" error

### 3. The Mismatch
- MCP Inspector in SSE mode doesn't handle session IDs the same way
- Raw tests work because they're stateless single requests
- Inspector expects either stateless operation or different session flow

## Solutions

### Option 1: Change Server Endpoint (Recommended)
```go
streamableServer := server.NewStreamableHTTPServer(
    mcpServer,
    server.WithEndpointPath("/mcp"), // Makes URL end with /mcp
)
```
Then test with: `http://localhost:8080/mcp`

### Option 2: Enable Stateless Mode
```go
streamableServer := server.NewStreamableHTTPServer(
    mcpServer,
    server.WithStateLess(true), // Disables session validation
)
```
Trade-off: Loses per-session features (tools, logging levels)

### Option 3: Force HTTP Transport in Tests
```bash
npx @modelcontextprotocol/inspector --cli http://localhost:8080/ \
  --transport http --method tools/list
```
Note: Doesn't solve session ID handling, just forces HTTP mode

### Option 4: Implement Session Handling in Tests
```bash
# Initialize and capture session ID
RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}' \
  http://localhost:8080/)
SESSION_ID=$(echo "$RESPONSE" | grep -i "mcp-session-id:" | cut -d' ' -f2)

# Use in subsequent requests
curl -H "Mcp-Session-Id: $SESSION_ID" ...
```

## Implementation Examples

### For Production (Stateful)
```go
// Keep sessions for multi-client isolation
streamableServer := server.NewStreamableHTTPServer(
    mcpServer,
    server.WithEndpointPath("/mcp"), // Inspector-compatible endpoint
    server.WithHeartbeatInterval(30 * time.Second),
)
```

### For Testing/Development (Stateless)
```go
// Simple, no session overhead
streamableServer := server.NewStreamableHTTPServer(
    mcpServer,
    server.WithStateLess(true),
)
```

### Custom Session Manager
```go
type MySessionManager struct{}

func (m *MySessionManager) Generate() string {
    return "" // JWT, OAuth token, etc.
}

func (m *MySessionManager) Validate(sessionID string) (isTerminated bool, err error) {
    // Custom validation logic
    return false, nil
}

func (m *MySessionManager) Terminate(sessionID string) (isNotAllowed bool, err error) {
    // Custom termination logic
    return false, nil
}

streamableServer := server.NewStreamableHTTPServer(
    mcpServer,
    server.WithSessionIdManager(&MySessionManager{}),
)
```

## Quick Reference

| Issue | Solution |
|-------|----------|
| Inspector uses SSE by default | Use `/mcp` endpoint or `--transport http` |
| Session ID validation fails | Enable stateless mode or handle session flow |
| Need multi-client isolation | Keep stateful mode, fix endpoint |
| Testing is complex | Use stateless mode for tests |

## Sources
- StreamableHTTP implementation: `/Users/vcto/Projects/mcp-go-main/server/streamable_http.go`
- Inspector transport logic: `/Users/vcto/Projects/inspector-main/cli/src/transport.ts`
- MCP Spec: https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http

## Key Insight
The `--cli` flag in MCP Inspector only affects output format (CLI vs web UI), NOT transport selection. Transport is determined by URL pattern or explicit `--transport` flag.
