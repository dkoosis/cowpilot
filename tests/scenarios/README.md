### Manual Testing Examples

```bash
# Using MCP Inspector
npx @modelcontextprotocol/inspector --cli https://cowpilot.fly.dev/ --method tools/list

# Using raw curl/jq
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .
```# Scenario Tests for Cowpilot

This directory contains scenario tests that validate MCP protocol compliance for the cowpilot server.

## Prerequisites

1. Install `@modelcontextprotocol/inspector`:
   ```bash
   npm install -g @modelcontextprotocol/inspector
   ```

2. Ensure the cowpilot server is running (either locally or deployed)

## Test Approaches

### 1. MCP Inspector Tests (`mcp_scenarios.sh`)
Uses the official `@modelcontextprotocol/inspector` CLI tool for high-level protocol testing.

### 2. Raw SSE/JSON-RPC Tests (`raw_sse_test.sh`)
Direct protocol testing using `curl` and `jq` based on the approach from [this blog post](https://blog.fka.dev/blog/2025-03-25-inspecting-mcp-servers-using-cli/). This provides:
- Lower-level protocol inspection
- Direct JSON-RPC message sending
- SSE stream format validation
- Useful for debugging protocol issues

## Running Tests

### Against Production (Fly.io)

```bash
MCP_SERVER_URL=https://cowpilot.fly.dev/ go test -v ./tests/scenarios/
```

### Against Local Server

```bash
# Terminal 1: Start the server
go run cmd/cowpilot/main.go

# Terminal 2: Run tests
go test -v ./tests/scenarios/
```

### Direct Shell Script Execution

```bash
# Make script executable
chmod +x tests/scenarios/mcp_scenarios.sh

# Run against production
./tests/scenarios/mcp_scenarios.sh https://cowpilot.fly.dev/

# Raw SSE tests
./tests/scenarios/raw_sse_test.sh https://cowpilot.fly.dev/

# Run against local
./tests/scenarios/mcp_scenarios.sh http://localhost:8080/
```

## Test Scenarios

1. **Initialization Flow**: Validates the MCP handshake with protocol version 2025-03-26
2. **Tool Discovery**: Ensures the "hello" tool is properly advertised
3. **Tool Execution**: Verifies the hello tool returns "Hello, World!"
4. **Error Handling**: Tests proper error responses for non-existent tools
5. **Protocol Compliance**: Validates JSON-RPC 2.0 compliance
6. **SSE Stream Format**: Validates Server-Sent Events format (raw tests)

## CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Run Scenario Tests
  env:
    MCP_SERVER_URL: https://cowpilot.fly.dev/
  run: |
    npm install -g @modelcontextprotocol/inspector
    go test -v ./tests/scenarios/
```

## Troubleshooting

- If tests fail with "@modelcontextprotocol/inspector not found", ensure it's installed globally
- The inspector uses `npx` to run, so Node.js must be installed
- For SSE transport issues, verify the server supports Server-Sent Events
- Check server logs if tests fail unexpectedly

## Expected Output

Successful test run shows:
- ✓ PASS markers for each test
- Total/Passed/Failed summary
- Exit code 0

Failed tests show:
- ✗ FAIL markers with details
- Expected vs Actual output
- Exit code 1
