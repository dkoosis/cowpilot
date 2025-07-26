# Cowpilot Test Scripts

This directory contains organized test scripts for the Cowpilot MCP server. All scripts use formatted output similar to `gotestsum` for consistency and readability.

## Quick Start

```bash
# Make all scripts executable (first time only)
chmod +x *.sh

# Run the test runner to see all available tests
./run-tests.sh

# Run a specific test by number
./run-tests.sh 1

# Run all tests
./run-tests.sh all

# Run quick smoke tests
./run-tests.sh quick
```

## Available Tests

### 1. **project-health-check.sh**
Comprehensive project validation including:
- Project structure verification
- Build test and binary size check
- Dependency verification
- Feature implementation counts
- Debug system configuration
- Documentation presence
- Git repository status

### 2. **mcp-protocol-smoke-test.sh**
Basic MCP protocol verification via direct HTTP/JSON-RPC:
- Protocol initialization
- Tools listing and calling
- Resources listing
- Prompts listing
- Error handling

### 3. **mcp-inspector-integration-test.sh**
Tests compatibility with the official MCP Inspector tool:
- Inspector availability check
- HTTP transport testing
- Tool calling via inspector
- Transport fallback behavior
- Endpoint routing

### 4. **mcp-transport-diagnostics.sh**
Advanced transport testing and diagnostics:
- HTTP/SSE auto-detection
- Client type detection
- Content negotiation
- Protocol diagnostics endpoint
- Multiple transport methods

### 5. **sse-transport-test.sh**
Server-Sent Events protocol verification:
- SSE connection establishment
- Event stream format validation
- Multiple concurrent requests
- Browser client simulation
- HTTP override testing

### 6. **debug-tools-integration-test.sh**
Debug system functionality testing:
- Runtime configuration via environment variables
- Memory storage mode
- File storage mode (SQLite)
- Bounded storage limits
- Debug proxy integration

## Test Output Format

All tests use consistent formatting:
- `🔵 Blue` - Test sections and information
- `🟢 Green` - Successful tests
- `🔴 Red` - Failed tests
- `🟡 Yellow` - Warnings or skipped tests
- `🔷 Cyan` - Additional information

## Environment Variables

Some tests respect these environment variables:
- `MCP_SERVER_URL` - Override server URL for remote testing
- `MCP_DEBUG` - Enable debug mode
- `MCP_DEBUG_STORAGE` - Set storage type (memory/file)
- `MCP_DEBUG_PATH` - Path for debug database

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed
- Other codes may indicate specific errors

## Writing New Tests

When adding new test scripts:
1. Use the same formatting style (see existing scripts)
2. Add clear test descriptions in the output
3. Return proper exit codes
4. Add the test to `run-tests.sh` descriptions
5. Make the script executable

## Makefile Integration

These scripts complement the Makefile targets:
```bash
make test           # Run Go unit tests
make scenario-test  # Run scenario tests
make test-verbose   # Run tests with detailed output
```

The shell scripts provide additional validation and integration testing beyond the Go test suite.
