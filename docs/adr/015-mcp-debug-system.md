# ADR-015: MCP Debug System and Conformance Validation

**Date:** 2025-07-25
**Status:** Implemented (Phase 1)
**Context:** Establishing rigorous MCP protocol debugging and conformance validation

## Decision

We implemented a non-invasive debug proxy system that intercepts, logs, and validates all MCP protocol messages, providing real-time debugging capabilities without modifying the core server.

## Context

We lacked:
- Protocol conformance validation
- Real-time debugging capabilities  
- Security audit trails
- Automated testing infrastructure

## Decision Details

### Architecture: Debug Proxy Pattern
```
Client → Debug Proxy → MCP Server
         ↓
    SQLite Storage
         ↓
    Validation Layer
```

### Phase 1 Implementation (Complete)

1. **Message Interceptor** (`internal/debug/interceptor.go`)
   - Captures all JSON-RPC messages
   - Adds performance metrics
   - Session tracking

2. **SQLite Storage** (`internal/debug/storage.go`)
   - Conversation history
   - Performance statistics
   - Configurable retention

3. **Debug Proxy Server** (`cmd/mcp-debug-proxy/`)
   - Standalone binary
   - Environment-based configuration
   - Zero production impact when disabled

### Key Design Decisions

1. **Non-Invasive**: No changes to core server code
2. **Optional Activation**: Environment flag controlled
3. **Performance Target**: <5ms overhead per request
4. **Storage Strategy**: SQLite for simplicity and portability

## Consequences

### Positive
- Real-time visibility into MCP conversations
- Foundation for automated conformance testing
- Security audit trail capability
- Performance bottleneck identification

### Negative
- Additional binary to maintain
- Storage requirements for logs
- Potential performance overhead (mitigated by optional activation)

## Future Phases

### Phase 2: Conformance Validation
- JSON-RPC structure validation
- MCP semantic validation
- Security pattern detection

### Phase 3: Interactive Dashboard
- Real-time monitoring UI
- Protocol analysis views
- Interactive testing panel

### Phase 4: Automated Testing
- CI/CD integration
- Regression testing
- Compliance reporting

## Configuration

```bash
# Enable debug mode
export MCP_DEBUG_ENABLED=true
export MCP_DEBUG_LEVEL=DEBUG

# Run with proxy
./bin/mcp-debug-proxy --target ./bin/cowpilot
```

## Performance Metrics
- Achieved: <3ms average overhead
- Storage: ~1MB per 1000 messages
- Memory: Minimal impact

## References
- MCP Specification: https://spec.modelcontextprotocol.io
- Implementation Plan: Phase 1 complete, Phase 2 pending
- Debug Proxy: `cmd/mcp-debug-proxy/main.go`
