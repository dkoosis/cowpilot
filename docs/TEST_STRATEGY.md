# MCP Integration Test Strategy

## Overview
Integration tests run against the DEPLOYED instance at https://mcp-adapters.fly.dev/mcp by default.
This ensures we're testing the actual production environment.

## Test Types

### 1. Unit Tests (`make unit-test`)
- Run locally without any server
- Test individual components in isolation
- Fast, no external dependencies

### 2. Integration Tests (`make integration-test`)
- Run against DEPLOYED instance (https://mcp-adapters.fly.dev/mcp)
- Test actual MCP protocol compliance
- Verify deployed service is working
- Default for CI/CD and `make test`

### 3. Local Integration Tests (`make integration-test-local`)
- Run against local server (http://localhost:8080/mcp)
- For development/debugging only
- NOT part of standard build

## Standard Build Process

```bash
make build  # Runs: unit tests → integration tests (deployed) → build binary
make test   # Runs: unit tests → integration tests (deployed)
```

## Environment Variables

- `MCP_SERVER_URL`: Override test target (defaults to https://mcp-adapters.fly.dev/mcp)
- `LOCAL_TEST=true`: Force local testing (for development)

## Why Test Against Deployed?

1. **Real Environment**: Tests the actual production setup
2. **Auth Disabled**: Deployed with `--disable-auth` for testing
3. **Always Available**: No need to manage local servers
4. **CI/CD Ready**: Works in GitHub Actions without setup
5. **Catches Deployment Issues**: Verifies Fly.io configuration

## Local Testing (Development Only)

For local development/debugging:
```bash
# Option 1: Use make target
make integration-test-local

# Option 2: Set environment variable
MCP_SERVER_URL=http://localhost:8080/mcp make integration-test
```

## CI/CD Configuration

GitHub Actions should use default behavior:
```yaml
- name: Run tests
  run: make test  # Tests against deployed instance
```
