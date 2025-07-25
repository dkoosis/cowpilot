---
title: "ADR-013: MCP Transport Selection - StreamableHTTP over SSE"
status: "Accepted"
date: "2025-07-25"
tags: ["mcp", "transport", "protocol", "http", "sse"]
---

# ADR-013: MCP Transport Selection - StreamableHTTP over SSE

## Status
Accepted

## Context
We discovered significant confusion about MCP transport protocols when our tests failed with "Arguments cannot be passed to a URL-based MCP server" errors. Investigation revealed:

1. **SSE Transport is Deprecated**: The MCP specification deprecated SSE transport on 2024-11-05
2. **StreamableHTTP is Current**: The modern replacement supports both JSON responses and SSE streaming
3. **Inspector Tool Incompatibility**: The MCP inspector tool (our primary test tool) expects StreamableHTTP, not SSE
4. **Library Support**: mark3labs/mcp-go v0.34+ supports StreamableHTTP via `NewStreamableHTTPServer()`

### Transport Evolution
- **SSE Transport** (deprecated): Separate endpoints for SSE stream and POST messages
- **StreamableHTTP Transport** (current): Single endpoint that returns either JSON or SSE based on request

## Decision
Use StreamableHTTP transport for HTTP mode, replacing the deprecated SSE transport.

## Consequences
### Positive
- **Protocol Compliance**: Aligns with current MCP specification
- **Tool Compatibility**: Works with modern MCP tools including inspector
- **Future Proof**: Uses the actively supported transport
- **Unified Endpoint**: Single endpoint simplifies client implementation

### Negative
- **Breaking Change**: Clients using SSE-specific endpoints must update
- **Documentation Gap**: Limited documentation on transport differences

### Implementation
```go
// OLD (deprecated):
sseServer := server.NewSSEServer(mcpServer)

// NEW (current):
streamableServer := server.NewStreamableHTTPServer(mcpServer)
```

## Lessons Learned
1. **Track Protocol Evolution**: MCP spec changes must be actively monitored
2. **Test Tool Alignment**: Ensure test tools match server implementation
3. **Document Transport**: Be explicit about which transport layer we support

## References
- [MCP Transports Specification](https://modelcontextprotocol.io/docs/concepts/transports)
- [mcp-go StreamableHTTPServer](https://pkg.go.dev/github.com/mark3labs/mcp-go/server#StreamableHTTPServer)
- [Transport Status Documentation](/docs/TRANSPORT_STATUS.md)
