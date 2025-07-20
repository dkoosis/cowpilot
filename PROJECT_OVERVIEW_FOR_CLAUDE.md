# Cowpilot Project Overview - Optimized for Claude

## 🚀 Project Status: OPERATIONAL

### Quick Test
```bash
curl https://cowpilot.fly.dev/health
# Should return: OK
```

### What Is This?
- **MCP Server** implemented in Go
- **Live at**: https://cowpilot.fly.dev/
- **Current Tools**: "hello" (returns "Hello, World!")
- **Transport**: SSE (Server-Sent Events)
- **Protocol**: MCP v2025-03-26

### Key Files for Next Session
```
/cmd/cowpilot/main.go          # Add new tools here
/docs/STATE.yaml               # Session context (START HERE)
/tests/e2e/mcp_scenarios.sh    # High-level tests
/tests/e2e/raw_sse_test.sh     # Low-level tests
```

### How to Add a New Tool
```go
// In main.go, copy this pattern:
tool := mcp.NewTool("toolname",
    mcp.WithDescription("What it does"),
    mcp.WithInputSchema(mcp.InputSchema{
        Type: "object",
        Properties: map[string]interface{}{
            "param": map[string]string{"type": "string"},
        },
    }),
)
s.AddTool(tool, toolnameHandler)

func toolnameHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Implementation
    return mcp.NewToolResultText("Response"), nil
}
```

### Testing Workflow
```bash
# Local development
go run cmd/cowpilot/main.go

# Test everything
make test
make e2e-test-local

# Deploy
fly deploy

# Test production
make e2e-test-prod
```

### Next Priorities
1. **More Tools**: Weather, search, calculations, etc.
2. **Authentication**: Basic auth or API keys
3. **Resources**: File serving capability
4. **Monitoring**: Metrics and logging
5. **Performance**: Load testing and optimization

### Important Notes
- Uses `mark3labs/mcp-go` SDK (not official)
- SSE transport required for Fly.io
- Comprehensive E2E tests with dual approach
- All patterns established and documented
- Ready for rapid feature development

### Session Entry Point
**ALWAYS START WITH**: `/docs/STATE.yaml`

This file contains optimized context for Claude sessions including:
- Current state and capabilities
- Quick commands
- Established patterns
- Next steps
- Gotchas and learnings

---
*Project is stable, tested, and ready for expansion. E2E testing was the last infrastructure piece needed.*