---
title: "ADR-010: MCP Debug System Architecture"
status: "Accepted"
date: "2025-07-25"
tags: ["debug", "mcp", "architecture", "proxy", "logging"]
supersedes: ""
superseded-by: ""
---

# ADR-010: MCP Debug System Architecture

## Status
Accepted

## Context
The mcp adapters MCP server currently lacks comprehensive debugging and protocol conformance validation capabilities. This creates several critical gaps:

1. **Protocol Conformance**: No automated validation that our MCP implementation follows the specification correctly
2. **Debugging Difficulties**: Limited visibility into MCP conversations makes debugging client-server interactions challenging
3. **Security Audit Trail**: No systematic logging of MCP interactions for security monitoring
4. **Quality Assurance**: Missing automated testing infrastructure for protocol compliance

The current debugging approach relies on manual inspection using the MCP inspector tool and ad-hoc logging, which is insufficient for production-grade reliability and security requirements.

## Decision
We will implement a comprehensive **MCP Debug System** using a **non-invasive proxy/middleware architecture** that provides:

1. **Real-time Conversation Logging**: Capture all MCP JSON-RPC messages with metadata
2. **Protocol Conformance Validation**: Automated validation against MCP specification
3. **Security Monitoring**: Pattern detection for suspicious activities
4. **Interactive Debug Dashboard**: Web-based interface for real-time debugging
5. **Automated Testing Infrastructure**: Continuous conformance validation

**Architecture Overview:**
```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Client    │◄──►│  Debug Proxy     │◄──►│   MCP Server    │
│   (Claude)      │    │  - Intercept     │    │   (mcp adapters)    │
└─────────────────┘    │  - Log           │    └─────────────────┘
                       │  - Validate      │
                       │  - Monitor       │
                       └──────────────────┘
                              │
                       ┌──────────────────┐
                       │   Data Layer     │
                       │  - SQLite Store  │
                       │  - Log Archive   │
                       │  - Metrics DB    │
                       └──────────────────┘
```

## Alternatives Considered

1. **Inline Instrumentation** - Modify cowpilot server code directly
   - Rejected: Invasive, increases complexity, harder to disable, couples debug logic with business logic

2. **External Log Analysis** - Parse logs post-hoc using external tools
   - Rejected: No real-time capabilities, limited protocol validation, poor UX for debugging

3. **MCP SDK Extension** - Extend mark3labs/mcp-go with debug capabilities
   - Rejected: Would require upstream changes, less control over implementation, SDK dependency risk

4. **Separate Debug Server** - Standalone debugging tool
   - Rejected: More complex setup, no real-time interception, requires manual correlation

## Rationale

- **Non-Invasive Design**: Proxy/middleware approach requires no changes to cowpilot core, allowing optional enable/disable
- **Real-Time Capability**: Message interception provides immediate visibility during development and debugging
- **Protocol Compliance**: Automated validation catches conformance issues before they reach production
- **Security Foundation**: Systematic logging enables security monitoring and audit trails
- **Developer Experience**: Interactive dashboard dramatically improves debugging workflow
- **Production Ready**: Optional activation means zero performance impact when disabled

## Consequences

### Positive
- **Improved Quality**: Automated protocol conformance validation catches issues early
- **Faster Debugging**: Real-time message inspection reduces debugging time significantly
- **Security Monitoring**: Systematic audit trail enables security pattern detection
- **Better Testing**: Automated conformance testing improves CI/CD pipeline
- **Documentation**: Live examples and conversation logs serve as implementation documentation

### Negative
- **Complexity**: Additional components increase system complexity
- **Performance Overhead**: Message interception adds latency (~5ms target)
- **Storage Requirements**: Conversation logs require disk space management
- **Development Time**: Significant implementation effort across multiple phases

### Mitigation
- **Optional Activation**: Environment flag enables/disables entire system
- **Performance Monitoring**: Built-in latency tracking with alerts
- **Log Rotation**: Configurable retention policies prevent storage bloat
- **Phased Implementation**: Incremental delivery spreads development effort
- **Fallback Mechanism**: Direct connection if proxy fails

## Implementation Plan

**Phase 1 (1-2 weeks)**: Message Interceptor & Storage
- `internal/debug/interceptor.go` - Message interception middleware
- `internal/debug/storage.go` - SQLite conversation logging
- `cmd/mcp-debug-proxy/` - Standalone debug proxy server

**Phase 2 (2-3 weeks)**: Protocol Validation & Security
- `internal/validator/` - JSON-RPC and MCP semantic validation
- `internal/security/` - Security pattern detection
- Enhanced prompts validation (addressing spec coverage gap)

**Phase 3 (3-4 weeks)**: Interactive Dashboard
- `web/dashboard/` - React-based debug interface
- Real-time WebSocket updates
- Interactive testing panel

**Phase 4 (2-3 weeks)**: Automated Testing
- `tests/conformance/` - Automated conformance test suite
- CI/CD integration
- Compliance reporting

## Technical Specifications

**Message Interception:**
- Hook into transport layer (HTTP/SSE and stdio)
- JSON-RPC message parsing and metadata extraction
- Bidirectional message flow tracking
- Performance impact: <5ms latency target

**Data Storage:**
```sql
CREATE TABLE conversations (
    id INTEGER PRIMARY KEY,
    session_id TEXT,
    timestamp DATETIME,
    direction TEXT, -- 'inbound'|'outbound'
    method TEXT,
    params JSON,
    result JSON,
    error JSON,
    performance_ms INTEGER
);
```

**Configuration:**
```bash
# Enable debug system
export MCP_DEBUG_ENABLED=true
export MCP_DEBUG_LEVEL=INFO
export MCP_SECURITY_MONITORING=true

# Run with debug proxy
./bin/mcp-debug-proxy --target=./bin/cowpilot --port=8080
```

## References
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io)
- [mark3labs/mcp-go SDK](https://github.com/mark3labs/mcp-go)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [Debug System Plan](../debug/mcp-conformance-plan.md)
- [MCP Security Best Practices](https://modelcontextprotocol.io/docs/tools/debugging)

## Notes
- This ADR establishes the architectural foundation for the comprehensive debug system outlined in `docs/debug/mcp-conformance-plan.md`
- Implementation will be tracked through Phase 1 deliverables and subsequent ADRs for complex components
- Success metrics: 100% message coverage, <5ms latency overhead, 99.9% protocol conformance score
- Future consideration: Integration with observability platforms (Prometheus, Grafana) for production monitoring
