# Debug Guide - Low-Level Protocol Testing

This guide covers low-level debugging when MCP conversation tests fail.

## When to Use This Guide

Only when:
- MCP conversation tests are failing
- You need to understand what's happening at the protocol level
- Inspector output isn't clear enough

**Note**: We trust mark3labs/mcp-go for SSE/JSON-RPC implementation. This is for debugging, not regular testing.

## Raw SSE Testing

### View Raw SSE Stream
```bash
# See exactly what the server sends
curl -s -N -X POST https://cowpilot.fly.dev/ \
  -H 'Content-Type: application/json' \
  -H 'Accept: text/event-stream' \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'
```

### Parse SSE to JSON
```bash
# Extract JSON from SSE format
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | \
  curl -s -N -X POST https://cowpilot.fly.dev/ \
    -H 'Content-Type: application/json' \
    -H 'Accept: text/event-stream' \
    -d @- | grep '^data: ' | sed 's/^data: //' | jq .
```

## SSE Format Reference

Valid SSE response:
```
data: {"jsonrpc":"2.0","id":1,"result":{...}}

data: [DONE]

```

Requirements:
- `data: ` prefix (with space)
- Blank line after each message
- Optional `data: [DONE]` terminator

## Common Issues

### No Response
```bash
# Add verbose flag to see connection details
curl -v -s -N -X POST http://localhost:8080/ ...
```

### Malformed JSON
```bash
# Save raw output to inspect
curl ... > raw_output.txt
cat raw_output.txt
```

### SSE Not Working
```bash
# Check headers
curl -I http://localhost:8080/
# Should include: Content-Type: text/event-stream
```

## Manual Test Scripts

For repeated debugging:
```bash
# Run raw SSE test suite
make e2e-test-raw
```

This runs `raw_sse_test.sh` which tests:
- Basic connectivity
- SSE format compliance
- JSON-RPC structure
- Error responses
