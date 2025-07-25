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

**Transport**: Dual mode - stdio (local) + StreamableHTTP (production) - [See ADR-013](docs/adr/013-mcp-transport-selection.md)

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
- **Transport**: StreamableHTTP (supports both JSON and SSE responses) - [See ADR-013](docs/adr/013-mcp-transport-selection.md)
- **Deployment**: Fly.io with auto-scaling

## Documentation Directory 📚

### 🏠 Project Overview
- [README](README.md) - Main project documentation (this file)
- [Project Overview for Claude](PROJECT_OVERVIEW_FOR_CLAUDE.md) - AI-optimized project summary
- [Project Cleanup Summary](PROJECT_CLEANUP_SUMMARY.md) - Recent improvements and cleanups
- [State Documentation](docs/STATE.yaml) - Machine-optimized context for Claude

### 🔧 Development & Testing
- [Testing Guide](docs/testing-guide.md) - Comprehensive testing documentation
- [Test Formatting](docs/test-formatting.md) - Test output formatting standards
- [User Guide](docs/user-guide.md) - End-user documentation
- [Contributing Guide](docs/contributing.md) - Development contribution guidelines

### 🏗️ Architecture & Design
- [Protocol Standards](docs/protocol-standards.md) - MCP implementation details
- [Known Issues](docs/KNOWN-ISSUES.md) - Current limitations and workarounds
- [Roadmap](docs/ROADMAP.md) - Future development plans
- [TODO](docs/TODO.md) - Immediate development tasks
- [LLM Documentation](docs/llm.md) - AI/LLM related documentation
- [Project History](docs/project-history.md) - Development timeline and decisions

### 🐛 Debug & Tools
- **[MCP Conformance Plan](docs/debug/mcp-conformance-plan.md)** - Comprehensive protocol debugging system
- **[Phase 1 Implementation Complete](docs/debug/phase1-implementation-complete.md)** - Debug system status (NEW)
- [Debug Guide](tests/scenarios/DEBUG_GUIDE.md) - Low-level protocol debugging
- [Testing Guide (Scenarios)](tests/scenarios/TESTING_GUIDE.md) - Scenario-based testing

### 📋 Architecture Decision Records (ADRs)
- [ADR Directory](docs/adr/README.md) - Overview of all architecture decisions
- [ADR-009: MCP SDK Selection](docs/adr/009-mcp-sdk-selection.md) - Why we chose mark3labs/mcp-go
- **[ADR-010: MCP Debug System Architecture](docs/adr/010-mcp-debug-system-architecture.md)** - Debug system architecture (NEW)
- **[ADR-011: Conditional Compilation for Lightweight Debug System](docs/adr/011-conditional-compilation-lightweight-debug.md)** - Lightweight debug strategy (SUPERSEDED)
- **[ADR-012: Runtime Debug Configuration](docs/adr/012-runtime-debug-configuration.md)** - Runtime debug approach (NEW)
- **[ADR-013: MCP Transport Selection](docs/adr/013-mcp-transport-selection.md)** - StreamableHTTP over SSE decision (NEW)
- [ADR Template](docs/adr/adr-template.md) - Template for new ADRs

### 📝 Development Sessions & History
- [2025-01-20 Handoff](docs/sessions/2025-01-20-handoff.md) - Project handoff documentation
- [Quick Start Next](docs/sessions/quick-start-next.md) - Next steps documentation

### 🎯 Prompts & Templates
- [Prompts README](prompts/README.md) - Overview of available prompts
- [Review Guide](prompts/REVIEW-GUIDE.md) - Code review prompt guidelines
- [001P: Review Semantic Naming](prompts/001P-review-semantic-naming.md) - Naming review prompt
- [002P: Analyze Code Smells](prompts/002P-analyze-code-smells.md) - Code quality analysis prompt

### 📊 Test Documentation
- [Scenarios README](tests/scenarios/README.md) - Test scenarios overview
- [Enhanced Summary](tests/scenarios/ENHANCED_SUMMARY.md) - Detailed test analysis
- [Implementation Review](tests/scenarios/IMPLEMENTATION_REVIEW.md) - Implementation testing review
- [Implementation Summary](tests/scenarios/IMPLEMENTATION_SUMMARY.md) - Implementation test results
- [File Inventory](tests/scenarios/FILE_INVENTORY.md) - Test file organization
- [RTFM Correction](tests/scenarios/RTFM_CORRECTION.md) - Documentation corrections

### 📁 Documentation Organization
- [docs/README.md](docs/README.md) - Docs directory index
- All documentation follows the project's architectural decision for comprehensive documentation

---

### 🔍 Quick Navigation
**New to the project?** Start with [Project Overview for Claude](PROJECT_OVERVIEW_FOR_CLAUDE.md)

**Want to contribute?** Read [Contributing Guide](docs/contributing.md) and [Testing Guide](docs/testing-guide.md)

**Debugging issues?** Check [MCP Conformance Plan](docs/debug/mcp-conformance-plan.md) and [Debug Guide](tests/scenarios/DEBUG_GUIDE.md)

**Understanding decisions?** Browse [ADR Directory](docs/adr/README.md)

## Legacy Documentation

## Project Structure

```
cmd/cowpilot/main.go           # Everything server implementation
internal/mcp/                  # MCP protocol handling
internal/transport/            # HTTP/SSE transport
tests/scenarios/               # E2E test scenarios
docs/STATE.yaml               # Machine context (for Claude)
Makefile                      # Build automation
```