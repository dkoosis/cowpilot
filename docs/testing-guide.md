# Cowpilot Testing Guide

## 🧪 Testing Philosophy

Every build must verify that MCP conversations work correctly. We test at multiple levels:

1. **Unit Tests** - Function and package level correctness
2. **Integration Tests** - Component interactions
3. **Scenario Tests** - Real MCP protocol conversations
4. **Shell Scripts** - Client compatibility and transport behavior

## 🚀 Quick Testing Commands

```bash
# Fastest validation
make test              # All Go tests + scenarios

# Development workflow  
make test-verbose      # Human-readable output
make build            # Tests first, then builds

# Manual inspection
npx @modelcontextprotocol/inspector ./bin/cowpilot
```

## 📊 Test Types

### Unit Tests (`./internal/...`)
- Test individual functions
- Mock dependencies
- Fast execution (<100ms per test)
- Run on every commit

### Integration Tests (`./tests/integration/`)
- Test component interactions
- Real dependencies
- Medium speed (<1s per test)
- Run on every commit

### Scenario Tests (`./tests/scenarios/`)
- Test full MCP conversations
- Real network calls
- Protocol compliance
- Run before deployment

### Shell Script Tests (`./scripts/test/`)
- Test client compatibility
- Transport behavior (HTTP vs SSE)
- Debug system integration
- Real-world scenarios

## 🛠️ Development Workflow

```bash
# 1. Make changes
vim cmd/cowpilot/main.go

# 2. Quick validation
make unit-test         # Fast feedback

# 3. Full validation
make test             # All tests including scenarios

# 4. Manual testing
npx @modelcontextprotocol/inspector ./bin/cowpilot

# 5. Commit
git commit -m "Add weather tool"
```

## 📝 Writing Tests

### Test Naming Convention

Go tests should clearly state what they test:

```go
// Good - descriptive BDD style
func TestHealthEndpoint_ReturnsStatusOK_When_ServerIsRunning(t *testing.T) {}
func TestEchoTool_ReturnsError_When_MessageIsMissing(t *testing.T) {}

// Use t.Run for sub-tests
t.Run("JSON-RPC request with wrong version returns invalid request error", func(t *testing.T) {
    // test implementation
})
```

### Adding Tool Tests

When adding a new tool, add tests at all levels:

```go
// 1. Unit test (in main_test.go or tool_test.go)
func TestWeatherHandler_ReturnsWeather_When_ValidLocation(t *testing.T) {
    // Test the handler function
}

// 2. Scenario test (in scenario_test.go)
t.Run("Weather tool returns forecast when given valid city", func(t *testing.T) {
    // Test via MCP protocol
})
```

```bash
# 3. Shell script test (add to mcp-protocol-smoke-test.sh)
if ! run_test "Call weather tool with valid city" \
    '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"weather","arguments":{"location":"Seattle"}}}'; then
    ((FAILED++))
fi
```

## 🔍 Test Output Standards

All tests use consistent formatting:

### Go Tests (via gotestsum)
```
=== RUN   TestHealthEndpoint
    --- Testing health endpoint returns OK
        ✓ Status code is 200
        ✓ Body contains OK
--- PASS  TestHealthEndpoint (0.01s)
```

### Shell Scripts
```
=== RUN   MCP Protocol Smoke Test
    --- Testing basic MCP protocol operations via curl
    --- Initialize protocol
        ✓ Success
    --- List available tools
        ✓ Success
--- PASS  MCP Protocol Smoke Test
```

## 🐛 Debugging Failed Tests

### Check Logs
```bash
# Enable debug mode
MCP_DEBUG=true make test

# Check specific test
go test -v -run TestEchoTool ./...
```

### Manual Protocol Testing
```bash
# Test with curl
curl -X POST http://localhost:8080/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | jq

# Test with MCP Inspector
npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list
```

### Shell Script Tests
```bash
cd scripts/test
./run-tests.sh           # Menu of all tests
./run-tests.sh 1         # Run specific test
./mcp-protocol-smoke-test.sh  # Run directly
```

## 📈 Test Coverage

```bash
# Generate coverage report
make coverage

# View in browser
open coverage.html

# Coverage targets
# - Unit tests: >80%
# - Integration: >70% 
# - Overall: >75%
```

## 🔧 Test Configuration

### Environment Variables
```bash
# Server URL for scenario tests
MCP_SERVER_URL=http://localhost:8080/

# Debug mode
MCP_DEBUG=true
MCP_DEBUG_LEVEL=DEBUG

# Test timeouts
TEST_TIMEOUT=30s
```

### Makefile Targets

| Target | Description | When to Use |
|--------|-------------|-------------|
| `make test` | All tests with scenarios | Before commit |
| `make unit-test` | Just unit tests | Quick feedback |
| `make integration-test` | Component tests | After refactoring |
| `make scenario-test-local` | E2E with local server | Before deploy |
| `make test-verbose` | Detailed output | Debugging |

## ✅ Test Checklist

Before committing:
- [ ] Unit tests pass
- [ ] Integration tests pass  
- [ ] Scenario tests pass
- [ ] New features have tests
- [ ] Test names are descriptive
- [ ] Coverage hasn't decreased

Before deploying:
- [ ] All local tests pass
- [ ] Shell script tests pass
- [ ] Manual inspection works
- [ ] Production scenario tests pass

## 📚 Additional Resources

- [Go Testing Guide](https://go.dev/doc/tutorial/add-a-test)
- [MCP Protocol Spec](docs/reference/schema.ts)
- [Debug Guide](docs/debug/mcp-conformance-plan.md)
- [Shell Test Scripts](scripts/test/README.md)
