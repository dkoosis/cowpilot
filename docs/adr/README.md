# Architectural Decision Records (ADRs) for Cowpilot

## ADR-001: Migration from TypeScript to Go
**Date**: 2025-01-18  
**Status**: Implemented

### Context
Initial implementation was TypeScript on Cloudflare Workers with OAuth and agents-mcp framework.

### Decision
Migrate to Go with simpler architecture.

### Consequences
- ✅ Better streaming support
- ✅ Single binary deployment
- ✅ Native SSE handling
- ❌ Lost TypeScript ecosystem
- ❌ Team needs Go expertise

### Dead End Avoided
Cloudflare Workers has fundamental limitations with streaming responses needed for MCP.

---

## ADR-002: Use mark3labs/mcp-go Instead of Official SDK
**Date**: 2025-01-18  
**Status**: Implemented

### Context
Official modelcontextprotocol/go-sdk exists but evaluation showed limitations.

### Decision
Use community mark3labs/mcp-go SDK.

### Consequences
- ✅ Built-in SSE transport
- ✅ Simpler API
- ✅ Active maintenance
- ❌ Not official
- ❌ Potential future migration

### Dead End Avoided
Official SDK lacked SSE support required for Fly.io deployment.

---

## ADR-003: SSE Transport Over stdio
**Date**: 2025-01-18  
**Status**: Implemented

### Context
MCP supports multiple transports. Fly.io requires HTTP-based transport.

### Decision
Implement SSE (Server-Sent Events) as primary transport.

### Consequences
- ✅ Fly.io compatible
- ✅ Browser testable
- ✅ Standard HTTP
- ❌ More complex than stdio
- ❌ Requires SSE parsing

### Dead End Avoided
stdio transport doesn't work with HTTP-based platforms.

---

## ADR-004: Defer Authentication
**Date**: 2025-01-18  
**Status**: Decided

### Context
Original design included complex OAuth flow.

### Decision
Launch without auth, add later when requirements clear.

### Consequences
- ✅ Faster initial deployment
- ✅ Simpler testing
- ✅ Focus on core MCP
- ❌ Public endpoint
- ❌ Need auth eventually

### Dead End Avoided
Over-engineering auth before core functionality proven.

---

## ADR-005: Start with Single "hello" Tool
**Date**: 2025-01-18  
**Status**: Implemented

### Context
Could implement multiple tools immediately.

### Decision
Ship with minimal "hello" tool, prove infrastructure first.

### Consequences
- ✅ Fast deployment
- ✅ Easy testing
- ✅ Clear success metric
- ❌ Limited functionality
- ❌ Appears basic

### Dead End Avoided
Complex implementation before basic connectivity proven.

---

## ADR-006: Dual Testing Strategy
**Date**: 2025-01-20  
**Status**: Implemented

### Context
Need comprehensive E2E testing for MCP protocol compliance.

### Decision
Implement both high-level (Inspector) and low-level (curl/jq) tests.

### Consequences
- ✅ Complete coverage
- ✅ Protocol debugging
- ✅ Multiple validation
- ❌ More maintenance
- ❌ Duplicate tests

### Lesson Learned
RTFM! Initial Inspector implementation used wrong syntax.

---

## Testing Evolution

### Attempt 1: Raw JSON-RPC to Inspector ❌
Tried sending `{"jsonrpc":"2.0"...}` directly to mcp-inspector-cli.

**Why Failed**: Inspector has its own CLI syntax, not raw JSON-RPC.

### Attempt 2: Correct Inspector Usage ✅
Used proper flags: `--cli`, `--method`, `--tool-name`.

**Lesson**: Always read tool documentation thoroughly.

### Enhancement: Raw SSE Testing ✅
Added curl+jq tests for protocol-level validation.

**Benefit**: Deep debugging capability, no tool abstraction.

---

## Platform Journey

### Stage 1: TypeScript + Cloudflare Workers ❌
- **Problem**: Streaming limitations
- **Dead End**: CF Workers can't maintain long connections

### Stage 2: TypeScript + Alternative Hosting ❌
- **Problem**: Complexity with TS streaming
- **Dead End**: Too much glue code needed

### Stage 3: Go + Fly.io ✅
- **Success**: Native streaming, simple deployment
- **Current**: Production operational

---

## Valuable Patterns from Cowgnition

### Worth Considering Later
1. **FSM for Connection State**: Clean state management
2. **Structured Errors**: Better debugging with context
3. **Middleware Chain**: Extensible validation/logging
4. **NDJSON Transport**: Reference implementation

### Not Needed Yet
1. **Service Registry**: Overkill for single service
2. **Complex Routing**: Single endpoint sufficient
3. **Prometheus Metrics**: Add when scaling
4. **YAML Config**: Env vars work fine

---

## Key Lessons for Future Development

1. **Start Minimal**: "hello" tool proved infrastructure
2. **Test at Protocol Level**: Tools can hide issues  
3. **Read Documentation**: Inspector syntax incident
4. **Avoid Premature Optimization**: No auth/config yet
5. **Platform Constraints Matter**: CF → Fly.io migration
6. **Community Solutions Valid**: mark3labs > official SDK

---

## What NOT to Do Again

1. ❌ Don't assume CLI tools accept raw protocol
2. ❌ Don't start with complex auth flows
3. ❌ Don't pick platform without streaming research
4. ❌ Don't implement multiple features before one works
5. ❌ Don't skip writing tests for "simple" things

---

*This document preserves institutional knowledge to prevent repeating mistakes.*