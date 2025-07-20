# Quick Reference: What NOT to Do

## ❌ DEAD ENDS - Don't Waste Time Here

### 1. **Cloudflare Workers/Vercel/Edge Functions**
- **Why Not**: No stdio transport support
- **Time Wasted**: ~1 week
- **Key Learning**: MCP needs stdio or SSE, not request/response

### 2. **Official modelcontextprotocol/go-sdk** 
- **Why Not**: No built-in SSE support
- **Better Choice**: mark3labs/mcp-go
- **Key Learning**: Third-party can be better

### 3. **MCP Inspector Wrong Syntax**
```bash
# ❌ WRONG - This doesn't work
mcp-inspector-cli '{"jsonrpc":"2.0","method":"tools/list"}'

# ✅ RIGHT - This is correct
npx @modelcontextprotocol/inspector --cli URL --method tools/list
```

### 4. **WebSocket Transport**
- **Why Not**: Overcomplicated vs SSE
- **Better Choice**: SSE (unidirectional is fine)
- **Key Learning**: Simple > Complex

### 5. **Authentication Before Basic Functionality**
- **Why Not**: Premature optimization
- **Better Approach**: Get protocol working first
- **Key Learning**: MVP first, security second

## ✅ What DOES Work

### Current Working Stack
- **Language**: Go 1.23
- **SDK**: mark3labs/mcp-go
- **Transport**: SSE (Server-Sent Events)
- **Platform**: Fly.io
- **Testing**: Inspector + raw curl/jq

### SSE Format (Exact)
```
data: {"jsonrpc":"2.0","id":1,"result":{...}}

```
Note: Space after "data:", blank line after JSON

### Environment Detection
```go
if os.Getenv("FLY_APP_NAME") != "" {
    runHTTPServer(s)  // Production
} else {
    server.ServeStdio(s)  // Local dev
}
```

## 🎯 If You're Stuck

1. **Check transport compatibility first**
2. **Read the actual documentation**
3. **Start with minimal implementation**
4. **Use established patterns from main.go**
5. **Test with both Inspector and curl**

## 📚 Required Reading

1. `/docs/STATE.yaml` - Current context
2. `/docs/PROJECT_HISTORY_AND_LESSONS.md` - Full journey
3. `/docs/reference/schema.ts` - MCP protocol spec
4. `/tests/e2e/TESTING_GUIDE.md` - How to test