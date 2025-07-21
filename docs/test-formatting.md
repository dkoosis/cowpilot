# Test Formatting Setup

## Install gotestsum
```bash
go install gotest.tools/gotestsum@latest
```

## Features Added to Makefile

1. **Auto-detection** - Uses gotestsum if installed, falls back to go test
2. **Format options**:
   - `testname` - Clean output for unit/integration tests
   - `dots-v2` - Compact dots for E2E tests
   - `--junitfile` - JUnit XML for CI

## Usage

```bash
# Install gotestsum first
go install gotest.tools/gotestsum@latest

# Run tests with better formatting
make test              # Unit + integration with clean output
make e2e-test-prod    # E2E tests with progress dots
make test-ci          # All tests with JUnit XML output

# Watch mode (great for development)
gotestsum --watch ./...
```

## Output Formats

- **testname**: Shows test names with pass/fail
- **dots-v2**: One dot per test (good for many tests)
- **pkgname**: Groups by package
- **standard-verbose**: Like go test -v but cleaner

## CI Integration

```yaml
# GitHub Actions example
- run: make test-ci
- uses: actions/upload-artifact@v3
  if: always()
  with:
    name: test-results
    path: test-results.xml
```

The Makefile now automatically uses gotestsum when available!