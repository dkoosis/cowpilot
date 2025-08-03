# Script Cleanup - What to Use Instead

## ❌ Deleted Scripts → ✅ Use These Commands

### Building
```bash
# Instead of: build_rtm.sh, test-build.sh, quick_build.sh, etc.
make build          # Build core server
make build-rtm      # Build RTM server  
make build-all      # Build all servers
make build-debug    # Build debug proxy
```

### Testing
```bash
# Instead of: test-*.sh scripts
make test           # Run local tests
make test-all       # Run comprehensive tests
make test-verbose   # Verbose output
make unit-test      # Unit tests only
make integration-test  # Integration tests
```

### Code Quality
```bash
# Instead of: check-fmt.sh, run-gofmt.sh
make fmt            # Format code
make lint           # Run linter
make vet            # Run go vet
```

### Deployment
```bash
# Instead of: ad-hoc deploy scripts
make deploy-rtm     # Deploy RTM server
make deploy-spektrix # Deploy Spektrix server
```

### Documentation
```bash
# Instead of: docs_*.sh scripts
make docs-tree      # Generate project structure
scripts/cleanup-docs.sh  # Clean up docs (the ONE cleanup script)
```

### Development
```bash
# Instead of: various dev scripts
make dev            # Run with hot reload (requires air)
make run            # Run locally
make run-debug-proxy # Run with debug proxy
```

## 📁 What's Left

### Scripts Used by Makefile
- `scripts/test/project-health-check.sh`
- `scripts/test/test-mcp-integration.sh`
- `scripts/test/run-tests.sh`
- `scripts/utils/track-performance.sh`

### Deploy Scripts (in scripts/deploy/)
Keep these for complex deployment scenarios

### Test Scenarios (in tests/scenarios/)
Keep these for manual testing

### The ONE Cleanup Script
- `scripts/cleanup-docs.sh` - For docs cleanup only

## 🎯 Key Point
**Use the Makefile!** It's the source of truth for all build and test operations.
