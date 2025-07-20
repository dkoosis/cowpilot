# Comprehensive MCP Testing Guide

This guide shows all the different ways to test the Cowpilot MCP server, from high-level to low-level approaches.

## Testing Approaches Overview

### 1. High-Level: MCP Inspector CLI
Best for: Quick protocol compliance checks, standard testing

```bash
# List tools
npx @modelcontextprotocol/inspector --cli https://cowpilot.fly.dev/ --method tools/list

# Call a tool
npx @modelcontextprotocol/inspector --cli https://cowpilot.fly.dev/ --method tools/call --tool-name hello
```

### 2. Low-Level: Raw curl with JSON-RPC
Best for: Protocol debugging, understanding SSE format, custom testing

```bash
# Send raw JSON-RPC over SSE
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .
```

### 3. Automated: Test Suites
Best for: CI/CD, regression testing, comprehensive validation

```bash
# Inspector-based tests
make e2e-test-prod

# Raw SSE tests
make e2e-test-raw

# All tests
make e2e-test-prod && make e2e-test-raw
```

## Complete Testing Matrix

| Test Type | Command | What It Tests |
|-----------|---------|---------------|
| Unit Tests | `make unit-test` | Individual functions |
| Integration Tests | `make integration-test` | Component interactions |
| E2E Inspector Tests | `make e2e-test-prod` | Protocol compliance via Inspector |
| E2E Raw Tests | `make e2e-test-raw` | Direct SSE/JSON-RPC protocol |
| Manual Inspector | `npx @modelcontextprotocol/inspector --cli ...` | Interactive testing |
| Manual Raw | `curl + jq` | Protocol debugging |

## Understanding SSE Format

MCP over SSE uses Server-Sent Events format:

```
data: {"jsonrpc":"2.0","id":1,"result":{...}}

data: [DONE]

```

Each JSON-RPC response is prefixed with `data: ` and terminated with a blank line.

## Common Test Scenarios

### 1. Basic Connectivity Test
```bash
# Health check
curl -s https://cowpilot.fly.dev/health

# SSE endpoint test
curl -s -N https://cowpilot.fly.dev/ -H 'Accept: text/event-stream' --max-time 2
```

### 2. Full Protocol Flow
```bash
# Initialize
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .

# List tools
echo '{"jsonrpc":"2.0","method":"tools/list","id":2}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .

# Call tool
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"hello","arguments":{}},"id":3}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .
```

### 3. Error Testing
```bash
# Non-existent method
echo '{"jsonrpc":"2.0","method":"invalid/method","id":4}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .
```

## Debugging Tips

1. **View Raw SSE Stream**: Remove the `grep` and `jq` to see raw output
2. **Test Timeouts**: Use `--max-time` with curl to handle hanging connections
3. **Verbose Output**: Add `-v` to curl for connection details
4. **Save Responses**: Use `tee` to save while viewing: `... | tee response.txt | jq .`

## References

- [MCP Inspector Docs](https://modelcontextprotocol.io/docs/tools/inspector)
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [SSE Testing Blog Post](https://blog.fka.dev/blog/2025-03-25-inspecting-mcp-servers-using-cli/)
