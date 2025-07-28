# mcp adapters Test Strategy

## Test Layers

### 1. Unit Tests (`make unit-test`)
- **Purpose**: Test individual functions and packages
- **Speed**: Fast (< 10 seconds)
- **When**: Every commit, PR checks
- **Location**: `*_test.go` files throughout codebase

### 2. Integration Tests (`make integration-test`)
- **Purpose**: Test component interactions
- **Speed**: Medium (< 30 seconds)
- **When**: Every commit, PR checks
- **Location**: `tests/integration/`

### 3. Scenario Tests (`make scenario-test`)
- **Purpose**: Test MCP protocol compliance
- **Speed**: Medium (< 1 minute)
- **When**: Pre-deploy, staging validation
- **Location**: `tests/scenarios/`

### 4. E2E Protocol Tests (`make e2e-test`)
- **Purpose**: Real client compatibility & transport behavior
- **Speed**: Slow (1-2 minutes)
- **When**: Pre-deploy, release validation
- **Location**: `scripts/test/`

## When to Run Each

### Local Development
```bash
make test          # Quick unit + integration tests
make e2e-test      # Quick protocol smoke tests
```

### Before Committing
```bash
make all           # Full validation including lint
```

### CI/CD Pipeline
```bash
make test-ci       # Unit + integration with JUnit output
make scenario-test # Protocol compliance
```

### Pre-Deployment
```bash
make test          # All Go tests
make e2e-test-all  # All shell script tests
make scenario-test # Protocol scenarios
```

## Shell Scripts Purpose

The shell scripts in `scripts/test/` are NOT replacements for Go tests. They serve unique purposes:

1. **Protocol Testing**: Verify actual wire protocol behavior
2. **Client Compatibility**: Test with real clients (curl, MCP Inspector)
3. **Transport Behavior**: SSE vs HTTP auto-detection
4. **Debug Integration**: Runtime configuration testing
5. **Production Validation**: Health checks against deployed services

These tests use real network calls and external tools, making them slower but more realistic than mocked Go tests.

## Test Selection Guide

- **Bug in business logic?** → Write a unit test
- **Component integration issue?** → Write an integration test  
- **Protocol compliance concern?** → Write a scenario test
- **Client compatibility issue?** → Write a shell script test
- **Transport behavior question?** → Write a shell script test

## Performance Targets

- Unit tests: < 100ms per test
- Integration tests: < 1s per test
- Scenario tests: < 5s per test
- E2E tests: < 30s per test

Keep tests focused and fast. Long-running tests belong in separate suites.
