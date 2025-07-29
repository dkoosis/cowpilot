# Testing Strategy

## Philosophy
**"Better to wait 2 minutes 100 times than spend half that time debugging something dumb."**

This project prioritizes **comprehensive validation by default** over speed optimization. Every build runs the full test suite to catch issues early.

## Default Behavior

### `make` or `make test`
Runs **comprehensive tests**:
1. ✅ Unit tests (fast, isolated)  
2. ✅ Local integration tests (everything server, all tools)
3. ✅ Deployed integration tests (RTM server, real environment)

**Auto-deploy option:** If the deployed server isn't responding, you'll be prompted to deploy automatically.

### `make all`  
Full validation pipeline:
1. ✅ Clean artifacts
2. ✅ Format code  
3. ✅ Run linter
4. ✅ Comprehensive tests
5. ✅ Local scenarios

## Why Comprehensive by Default?

### Catches Issues Early
- **Local tests** verify basic functionality and debug features
- **Deployed tests** catch real environment issues (HTTPS, CORS, fly.io routing)
- **Both together** provide confidence in production deployment

### Real Environment Validation
- HTTPS/TLS certificate handling
- CORS policy enforcement  
- Fly.io proxy and routing behavior
- RTM server vs everything server differences

### Time Investment Philosophy
- Spending 2 minutes on comprehensive testing prevents hours of debugging
- Network timeouts and build times are acceptable for quality assurance
- Comprehensive testing scales better than reactive debugging

## Test Structure

```
tests/
├── integration/
│   ├── debug_tools_test.go     # Local only (everything server)
│   ├── rtm_server_test.go      # Deployed only (RTM server)  
│   └── server_test.go          # Both environments
└── scenarios/
    └── *.go                    # Local and deployed variants
```

## When Tests Run

### Development
```bash
make              # Comprehensive validation
make test         # Same as above
```

### Build Pipeline
```bash
make build-rtm    # Comprehensive tests → RTM build
make deploy-rtm   # Build → Deploy
```

### Manual Options
```bash
make unit-test                # Unit tests only
make integration-test-local   # Local integration only  
make integration-test         # Deployed integration only
```

## Auto-Deploy Feature

When `integration-test` detects the server is down:
```
⚠️  Server not responding at https://mcp-adapters.fly.dev

Options:
  1. Auto-deploy: make deploy-rtm (recommended)
  2. Check status: fly status && fly logs  
  3. Skip deployed tests: make unit-test integration-test-local

Deploy RTM server now? [y/N]
```

Choose 'y' for automatic deployment and continued testing.

## Design Rationale

**Speed is not the priority** - comprehensive validation is. This approach:
- ✅ Prevents production issues
- ✅ Builds confidence in deployments  
- ✅ Scales better than reactive debugging
- ✅ Provides clear expectations about deployment requirements

The 2-minute comprehensive test run is an investment in reliability, not a cost to be optimized away.
