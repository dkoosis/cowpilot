# Cowpilot

MCP (Model Context Protocol) server implementation in Go - comprehensive everything server with tools, resources, and prompts.

**Production Server**: https://cowpilot.fly.dev/

## Quick Start

```bash
# Build and run locally (stdio mode)
make build && ./bin/cowpilot

# Run with HTTP/SSE server mode
FLY_APP_NAME=local-test ./bin/cowpilot

# Quick validation
./validate_status.sh
```

## Testing Guide 🧪

### Essential Commands
```bash
# Quick validation (recommended first step)
./validate_status.sh                    # Build + basic validation

# Unit + integration tests
make test-verbose                       # Human-readable test output

# End-to-end testing
make scenario-test-local               # Test against local server
make scenario-test-raw                 # CLI protocol testing (no browser)
```

### All Test Options

| Command | What It Does | When To Use |
|---------|-------------|-------------|
| `./validate_status.sh` | Quick build + health check | First validation, CI checks |
| `make test-verbose` | Unit + integration tests | Development, code changes |
| `make scenario-test-local` | E2E tests against local server | Full validation before deploy |
| `make scenario-test-raw` | CLI MCP protocol testing | Protocol compliance, debugging |
| `make scenario-test-prod` | E2E tests against production | Post-deployment validation |
| `npx @modelcontextprotocol/inspector ./bin/cowpilot` | Interactive MCP debugging | Manual testing (opens browser) |

### Legacy Scripts (⚠️ Avoid)
These exist but are superseded by Makefile targets:
- `test.sh`, `quick_test.sh`, `run-tests.sh` - Use `make test-verbose` instead
- `test_*.sh` scripts - Use appropriate `make` targets instead

## Features

**Tools (11)**: hello, echo, add, get_time, base64_encode/decode, string_operation, format_json, long_running_operation, get_test_image, get_resource_content

**Resources (4)**: text/hello, text/readme, image/logo, dynamic/{id} template  

**Prompts (2)**: simple_greeting, code_review (with arguments)

**Transport**: Dual mode - stdio (local) + HTTP/SSE (production)

## Development

```bash
# Full development cycle
make all                               # Clean, format, lint, test, build

# Individual steps  
make clean fmt vet lint                # Code quality
make build                            # Build binary
make run                              # Build and run
make coverage                         # Generate coverage report
```

## Deployment

```bash
# Deploy to production
make deploy                           # Test + build + deploy to Fly.io

# Manual deployment
fly deploy
```

## Architecture

- **Library**: `github.com/mark3labs/mcp-go` (native support)
- **Protocol**: MCP v2025-03-26 over JSON-RPC 2.0
- **Transport**: Server-Sent Events (SSE) for HTTP mode
- **Deployment**: Fly.io with auto-scaling

## Documentation

- [State Documentation](docs/STATE.yaml) - Machine-optimized context
- [Testing Guide](docs/testing-guide.md) - Detailed testing information  
- [Protocol Standards](docs/protocol-standards.md) - MCP implementation details
- [Development Roadmap](docs/ROADMAP.md) - Future enhancements

## Project Structure

```
cmd/cowpilot/main.go           # Everything server implementation
internal/mcp/                  # MCP protocol handling
internal/transport/            # HTTP/SSE transport
tests/scenarios/               # E2E test scenarios
docs/STATE.yaml               # Machine context (for Claude)
Makefile                      # Build automation
```