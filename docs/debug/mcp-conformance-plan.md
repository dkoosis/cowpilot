# MCP Protocol Conformance & Debug System Plan

**Date:** July 25, 2025  
**Status:** Planning Phase  
**Context:** Addressing persistent lack of rigorous MCP protocol conformance validation and debugging capabilities

## Executive Summary

This plan addresses the critical gap in MCP protocol debugging and conformance validation by implementing a comprehensive system that provides:
- Real-time conversation logging and analysis
- Automated protocol conformance validation
- Interactive debugging dashboard
- Security monitoring foundation
- Automated testing infrastructure

## Problem Statement

**Current Issues:**
- No rigorous way to enforce protocol conformance
- Limited debugging capabilities for MCP conversations
- Missing security audit trail for MCP interactions
- Lack of automated conformance testing

**Impact:**
- Protocol violations go undetected
- Debugging MCP issues is manual and time-intensive
- Security vulnerabilities may be missed
- Quality assurance gaps in MCP implementation

## Current State Analysis

**Existing Infrastructure:**
- Basic SSE/HTTP transport testing (`raw_sse_test.sh`)
- Low-level protocol debugging guide (`tests/scenarios/DEBUG_GUIDE.md`)
- Manual test scripts and inspector verification
- Protocol standards documentation (`docs/protocol-standards.md`)
- Basic integration tests in Go

**Technology Stack:**
- mark3labs/mcp-go SDK (per ADR-009)
- JSON-RPC 2.0 over SSE transport
- SQLite for data storage
- Go for server implementation

## Implementation Plan

### Phase 1: Real-time Conversation Logging
**Duration:** 1-2 weeks  
**Goal:** Capture and store all MCP conversations for analysis

**Components:**

1. **Message Interceptor Middleware**
   ```go
   type MessageInterceptor struct {
       logger ConversationLogger
       next   mcp.Handler
   }
   ```
   - Hook into mark3labs/mcp-go transport layer
   - Log all JSON-RPC messages (request/response/notification)
   - Include timestamps, connection IDs, message flow direction
   - Non-invasive proxy/middleware approach

2. **Structured Log Storage**
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
   - SQLite database for conversation history
   - Indexed for fast querying by session, method, timeframe

3. **Log Rotation & Management**
   - Configurable retention policies
   - Performance impact monitoring
   - Optional enable/disable via environment flag

**Deliverables:**
- `internal/debug/interceptor.go` - Message interception middleware
- `internal/debug/storage.go` - Conversation logging storage
- `cmd/mcp-debug-proxy/` - Standalone debug proxy server

### Phase 2: Enhanced Protocol Conformance Validator
**Duration:** 2-3 weeks  
**Goal:** Automatically validate every message against MCP specification

**2A. Core Protocol Validation**

1. **JSON-RPC 2.0 Validator**
   - Validate message structure, required fields
   - Check method names against MCP spec
   - Verify parameter schemas

2. **MCP Semantic Validator**
   **Tools Validation:**
   - Tool call argument validation
   - Schema compliance checking
   - Execution flow validation

   **Resources Validation:**
   - Resource URI format checking
   - Content type validation
   - Access pattern validation

   **Prompts Validation (EXPLICIT):**
   - Prompt template structure validation
   - Argument schema enforcement (string types only)
   - Message role/content validation (user/assistant)
   - Template variable resolution testing
   - PromptMessage format compliance

**2B. Security Pattern Detection**

1. **Parameter Injection Detection**
   - SQL injection patterns in tool parameters
   - Command injection attempts
   - Path traversal attempts in resource URIs

2. **Unusual Access Pattern Flagging**
   - Rapid-fire tool calls
   - Resource access outside normal patterns
   - Cross-session correlation analysis

3. **Error Message Sanitization Validation**
   - Check for information leakage in error responses
   - Validate error codes match MCP specification

**Deliverables:**
- `internal/validator/` package for all validation logic
- `internal/security/` package for security pattern detection
- `docs/security/mcp-audit-guide.md` - Security analysis documentation

### Phase 2.5: Security Audit Infrastructure
**Duration:** 1-2 weeks  
**Goal:** Establish foundation for security monitoring and audit

**Components:**

1. **Security Event Correlator**
   - Pattern matching engine for suspicious activities
   - Baseline behavior learning
   - Anomaly detection algorithms

2. **Security Dashboard Integration**
   - Real-time security alerts
   - Tool usage heat maps
   - Resource access patterns
   - Conversation replay for incident analysis

3. **Audit Report Generator**
   - Compliance reports for security reviews
   - Tool usage accountability reports
   - Data access audit trails
   - Security incident documentation

**Deliverables:**
- `internal/security/correlator.go` - Security event correlation
- `docs/security/incident-response.md` - Incident response procedures
- Security dashboard components in web interface

### Phase 3: Interactive Debug Dashboard
**Duration:** 3-4 weeks  
**Goal:** Web-based interface for real-time MCP debugging

**Components:**

1. **Live Conversation Monitor**
   - WebSocket connection for real-time updates
   - Message flow visualization
   - Session tree view (init → discovery → operations)
   - Real-time conformance scoring

2. **Protocol Analysis View**
   - Conformance score per session
   - Violation details with suggested fixes
   - Message timing analysis
   - Error pattern detection
   - Security alert integration

3. **Interactive Testing Panel**
   - Send test messages directly to server
   - Validate responses in real-time
   - Save/replay message sequences
   - Load testing scenarios
   - Prompt template testing interface

**Technology Stack:**
- React-based frontend
- WebSocket for real-time updates
- Go backend API server
- Chart.js for visualizations

**Deliverables:**
- `web/dashboard/` - React-based debug interface
- `internal/api/` - Dashboard API server
- `cmd/mcp-dashboard/` - Standalone dashboard server

### Phase 4: Automated Conformance Testing
**Duration:** 2-3 weeks  
**Goal:** Continuous validation of protocol compliance

**Components:**

1. **Conformance Test Suite**
   - Happy path scenarios (init → list → call)
   - Error condition testing
   - Edge case validation
   - Performance benchmarks
   - Prompt template testing scenarios

2. **CI/CD Integration**
   - Automated conformance checks on PRs
   - Regression testing against MCP spec changes
   - Performance degradation detection
   - Security regression testing

3. **Compliance Reporting**
   - Generate conformance reports
   - Track compliance trends over time
   - Compare against MCP reference implementations
   - Automated security audit reports

**Deliverables:**
- `tests/conformance/` - Automated test suite
- `.github/workflows/conformance.yml` - CI/CD integration
- `cmd/conformance-runner/` - Standalone test runner

## Technical Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   MCP Client    │◄──►│  Debug Proxy     │◄──►│   MCP Server    │
│   (Claude)      │    │  - Intercept     │    │   (Cowpilot)    │
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
                              │
                       ┌──────────────────┐
                       │   Web Dashboard  │
                       │  - Live Monitor  │
                       │  - Analysis UI   │
                       │  - Test Runner   │
                       │  - Security View │
                       └──────────────────┘
```

## Integration Strategy

**Non-invasive Approach:**
- Proxy/middleware pattern - no cowpilot core changes needed
- Optional activation via environment flags
- Minimal performance overhead when disabled
- Works with existing mark3labs/mcp-go implementation

**Configuration:**
```bash
# Enable debug mode
export MCP_DEBUG_ENABLED=true
export MCP_DEBUG_LEVEL=INFO
export MCP_SECURITY_MONITORING=true

# Run with debug proxy
./bin/mcp-debug-proxy --target=./bin/cowpilot --port=8080
```

## Implementation Priorities

**Phase 1 (High Priority):** 
- Critical foundation for all other phases
- Immediate value for debugging current issues

**Phase 2A (High Priority):**
- Enhanced prompts validation (addressing spec coverage gap)
- Core protocol conformance validation

**Phase 2B + 2.5 (Medium Priority):**
- Security monitoring foundation
- Audit documentation deliverable

**Phase 3 (Medium Priority):**
- User experience improvement
- Visual debugging capabilities

**Phase 4 (Lower Priority):**
- Long-term quality assurance
- CI/CD automation

## Success Metrics

**Technical Metrics:**
- 100% MCP message coverage in logs
- <5ms latency overhead for debug proxy
- 99.9% protocol conformance score
- Zero false positives in security detection

**Operational Metrics:**
- Reduced debugging time for MCP issues
- Faster identification of protocol violations
- Improved security incident response time
- Higher confidence in MCP implementation quality

## Risk Assessment

**Technical Risks:**
- Performance impact of logging/validation
- Proxy introducing connection instability
- Storage requirements for conversation logs

**Mitigation Strategies:**
- Comprehensive performance testing
- Optional enable/disable mechanisms
- Configurable log retention policies
- Fallback to direct connection if proxy fails

## Next Steps

1. **Immediate (Next Session):**
   - Create ADR for debug system architecture
   - Implement Phase 1 message interceptor
   - Set up basic SQLite storage schema

2. **Short Term (1-2 weeks):**
   - Complete Phase 1 implementation
   - Begin Phase 2A protocol validation
   - Create security documentation deliverable

3. **Medium Term (1-2 months):**
   - Complete conformance validation system
   - Implement basic dashboard interface
   - Add security monitoring capabilities

## Files Created/Modified

**New Files:**
- `docs/debug/mcp-conformance-plan.md` (this document)
- `internal/debug/` package
- `internal/validator/` package
- `internal/security/` package
- `web/dashboard/` frontend
- `cmd/mcp-debug-proxy/` binary
- `tests/conformance/` test suite
- `docs/security/mcp-audit-guide.md`

**Modified Files:**
- Update `docs/STATE.yaml` with debug capabilities
- Enhance `Makefile` with debug targets
- Update CI/CD workflows for conformance testing

## References

- [MCP Protocol Specification](https://spec.modelcontextprotocol.io)
- [mark3labs/mcp-go SDK](https://github.com/mark3labs/mcp-go)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [MCP Security Best Practices](https://modelcontextprotocol.io/docs/tools/debugging)
- ADR-009: MCP SDK Selection (existing)

---

**Document Status:** Ready for Implementation  
**Next Review:** After Phase 1 completion  
**Owner:** Development Team