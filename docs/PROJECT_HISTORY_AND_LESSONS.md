# Cowpilot Project History & Lessons Learned

## The Journey: TypeScript → Go

### Phase 1: TypeScript + Cloudflare Workers (FAILED)
**Timeline**: Early January 2025

**What We Built**:
- TypeScript MCP server on Cloudflare Workers
- OAuth authentication flow
- agents-mcp SDK integration

**Why It Failed**:
- **Critical Issue**: Cloudflare Workers don't support stdio transport
- MCP requires stdio for local development
- CF Workers are request/response only
- Wasted significant time before discovering incompatibility

**Lessons Learned**:
1. Always verify transport compatibility FIRST
2. Web platforms (CF Workers, Vercel, etc.) can't do stdio
3. Read MCP transport requirements before choosing platform

**Dead Code to Avoid Recreating**:
- OAuth implementation for CF Workers
- TypeScript MCP server structure
- Wrangler.toml configurations

### Phase 2: Go + Fly.io (SUCCESS)
**Timeline**: Mid-January 2025

**Key Decisions**:
1. **Language**: Go chosen for:
   - Better server deployment story
   - Strong HTTP/SSE support
   - Simple deployment binaries
   - mark3labs/mcp-go had what we needed

2. **SDK**: mark3labs/mcp-go over official because:
   - Built-in SSE support
   - Simpler API
   - Actually worked out of the box
   - Official SDK was more complex

3. **Platform**: Fly.io chosen for:
   - Native SSE support
   - Simple Go deployment
   - Good health check integration
   - Supports both HTTP and stdio modes

4. **Transport**: SSE (Server-Sent Events) because:
   - Works over HTTP
   - Browser compatible
   - Simple to implement
   - Good for real-time streaming

## Critical Technical Discoveries

### SSE Format Specifics
```
data: {"jsonrpc":"2.0","id":1,"result":{...}}\n
\n
```
- MUST have "data: " prefix (with space!)
- MUST have blank line after JSON
- Each message is a complete JSON-RPC object

### Environment Detection Pattern
```go
if os.Getenv("FLY_APP_NAME") != "" {
    // Run HTTP/SSE server
} else {
    // Run stdio server
}
```
This allows same binary for local dev and production.

### MCP Inspector Syntax (Learned the Hard Way)
❌ WRONG:
```bash
mcp-inspector-cli '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

✅ CORRECT:
```bash
npx @modelcontextprotocol/inspector --cli https://server/ --method tools/list
```

## Valuable Patterns from Cowgnition Project

They built a complete MCP implementation before official SDK existed:

1. **FSM for Connection State**
   - Uses looplab/fsm
   - Clean state transitions
   - Good for complex connection management

2. **Error Handling**
   - cockroachdb/errors for stack traces
   - Structured error types
   - Context propagation

3. **Middleware Pattern**
   - Validation middleware
   - Logging middleware
   - Clean separation of concerns

4. **Transport Abstraction**
   - Clean interface between transport and logic
   - Easy to add new transports
   - Good test boundaries

## Dead Ends to Never Revisit

1. **Cloudflare Workers for MCP** - Fundamentally incompatible
2. **Vercel Functions for MCP** - Same stdio problem
3. **Official Go SDK for SSE** - Doesn't support it well
4. **Raw WebSocket transport** - Overcomplicated vs SSE
5. **Authentication before MVP** - Get protocol working first

## Testing Evolution

### Testing Mistake #1: Not Reading Docs
- Implemented wrong mcp-inspector syntax
- Wasted time debugging non-existent issues
- LESSON: Always RTFM first

### Testing Success: Dual Approach
1. **High-Level**: MCP Inspector for protocol compliance
2. **Low-Level**: curl+jq for debugging
- Both approaches caught different issues
- Raw testing essential for SSE format debugging

## Architecture Decision Log

### Decision 1: Minimal First Implementation
**Choice**: Just implement "hello" tool
**Rationale**: Get end-to-end working before complexity
**Result**: ✅ Correct - found transport issues early

### Decision 2: Skip Authentication
**Choice**: No auth in v1
**Rationale**: Focus on protocol compliance first
**Result**: ✅ Correct - avoided complexity

### Decision 3: SSE over WebSockets
**Choice**: SSE transport
**Rationale**: Simpler, unidirectional, HTTP-compatible
**Result**: ✅ Correct - much easier to debug

### Decision 4: mark3labs over official SDK
**Choice**: Third-party SDK
**Rationale**: Had features we needed
**Result**: ✅ Correct - saved significant time

## What We Know Works

1. **Server Structure**:
   - main.go pattern for tools
   - SSE transport implementation
   - Fly.io deployment
   - Health check endpoint

2. **Testing**:
   - Dual approach (Inspector + raw)
   - E2E test suite
   - Make targets
   - CI/CD template

3. **Deployment**:
   - fly deploy
   - Automatic health checks
   - SSE endpoint at root
   - Environment-based transport switching

## Next Phase Considerations

### From Cowgnition, Consider Adopting:
1. FSM for connection management (if adding complex state)
2. Middleware pattern (when adding auth)
3. Structured errors (as complexity grows)
4. Config management (for multi-environment)

### Avoid Until Needed:
1. Complex authentication (basic auth first)
2. Database integration (memory first)
3. Microservices (monolith works fine)
4. Kubernetes (Fly.io is enough)

## The Most Important Lesson

**RTFM** - Read The F***ing Manual
- Would have avoided CF Workers dead end
- Would have avoided Inspector syntax errors
- Would have saved days of work

When in doubt, check:
1. Official MCP spec
2. Tool documentation
3. Platform capabilities
4. Transport requirements

## Recovery Patterns

When hitting a dead end:
1. Document why it failed
2. Save any reusable patterns
3. Switch approach completely
4. Don't try to salvage incompatible tech

This project succeeded because we:
- Abandoned TypeScript/CF completely
- Started fresh with proper research
- Chose boring, proven technology
- Built minimal implementation first
- Added comprehensive tests