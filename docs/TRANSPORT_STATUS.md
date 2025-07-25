# Transport Status - CRITICAL CLARIFICATION

## Current Implementation (As of July 25, 2025)

### What We Actually Implement:
1. **Local/CLI mode**: `server.ServeStdio()` - Standard I/O transport
2. **HTTP mode**: `server.NewSSEServer()` - SSE transport (DEPRECATED in MCP spec since 2024-11-05)

### What Our Tests Expect:
1. **raw_sse_test.sh**: Tests SSE transport directly - WORKS
2. **scenario_test.go**: Uses MCP inspector tool which expects Streamable HTTP - FAILS

## The Problem:
- **SSE is deprecated** but that's what mcp-go provides via `NewSSEServer()`
- **Streamable HTTP** is the modern replacement but we don't know if mcp-go supports it
- **MCP inspector tool** (our main test tool) expects Streamable HTTP, not SSE
- Error message "Arguments cannot be passed to a URL-based MCP server" indicates protocol mismatch

## Transport Evolution Timeline:
1. **SSE Transport** (deprecated 2024-11-05)
   - Separate endpoints for SSE stream and POST messages
   - What we currently use
2. **Streamable HTTP** (current standard)
   - Single endpoint that can return JSON or SSE
   - What modern tools expect

## Resolution (July 25, 2025):
**Decision**: Use StreamableHTTP transport (Option 1)
- mcp-go v0.34+ supports `server.NewStreamableHTTPServer()`
- Changed line 473 in main.go from SSE to StreamableHTTP
- Tests should now pass with MCP inspector tool
- See [ADR-013](../adr/013-mcp-transport-selection.md) for full details

## Evidence:
- Line 473 in main.go: `sseServer := server.NewSSEServer(mcpServer)`
- ADR-009 claims "all transports" but predates the SSE deprecation awareness
- No documentation about StreamableHTTP in our codebase
