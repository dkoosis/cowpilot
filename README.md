# Cowpilot

MCP (Model Context Protocol) server implementation in Go, deployed on Fly.io.

## Project Structure
```
/cmd/cowpilot/        # Main application entry point
/internal/mcp/        # MCP protocol implementation
/internal/transport/  # HTTP streaming transport
/tests/              # Test suites
/docs/               # Documentation
```

## Quick Start
```bash
# Install dependencies
go mod download

# Run locally
go run cmd/cowpilot/main.go

# Run tests
go test ./...

# Deploy to Fly.io
fly deploy
```

## Documentation
- [Testing Guide](docs/HOW-TO-TEST.md)
- [MCP Protocol Standards](docs/MCP-PROTOCOL-STANDARDS.md)
- [Development Roadmap](docs/ROADMAP.md)

## Development
Following MCP protocol standards for AI model tool interaction via JSON-RPC 2.0.
