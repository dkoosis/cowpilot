# Cowpilot Testing Guide

## Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/mcp/...

# Verbose output
go test -v ./...
```

## Test Structure

### Unit Tests
- Located alongside code files as `*_test.go`
- Focus on individual functions/methods
- Mock external dependencies

### Integration Tests
- Located in `/tests/integration/`
- Test component interactions
- Use real transport layers

### E2E Tests
- Located in `/tests/e2e/`
- Test full MCP protocol flows
- Include Fly.io deployment tests

## Writing Tests

### Naming Convention
```go
// Follow Go conventions
func TestComponentName_MethodName(t *testing.T) {}
func TestComponentName_MethodName_WhenCondition(t *testing.T) {}
```

### Table-Driven Tests
```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 2, 3, 5},
        {"negative", -1, -2, -3},
        {"zero", 0, 0, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

## MCP Protocol Testing

### Key Areas
1. JSON-RPC message validation
2. Transport layer (HTTP streaming)
3. Tool execution and error handling
4. State management

### Error Codes
Ensure proper JSON-RPC error codes:
- Parse Error: -32700
- Invalid Request: -32600
- Method Not Found: -32601
- Invalid Params: -32602
- Internal Error: -32603

## Local Development

```bash
# Run local server
go run cmd/cowpilot/main.go

# Test with curl
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{},"id":1}'
```

## Fly.io Testing

```bash
# Deploy to staging
fly deploy --config fly.staging.toml

# Run remote tests
fly ssh console -C "go test ./..."

# Check logs
fly logs
```
