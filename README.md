# mcp adapters

MCP (Model Context Protocol) server implementation in Go - comprehensive everything server with tools, resources, and prompts.

[![Production Status](https://img.shields.io/badge/status-operational-green)](https://mcp-adapters.fly.dev/health)
[![MCP Version](https://img.shields.io/badge/MCP-v2025--03--26-blue)](docs/protocol-standards.md)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8)](go.mod)

## 🚀 Quick Start

```bash
# Clone and test production server
git clone https://github.com/vcto/mcp-adapters.git
cd mcp-adapters
curl https://mcp-adapters.fly.dev/health  # Should return: OK

# Build and test locally
make build  # Runs all tests then builds
./bin/mcp-adapters  # Run in stdio mode

# Run with HTTP/SSE server
FLY_APP_NAME=local-test ./bin/mcp-adapters
```

## 📋 What's Implemented

### Tools (11 implemented)
- `hello` - Simple greeting
- `echo` - Echo with prefix
- `add` - Add two numbers
- `get_time` - Current time (unix/iso/human)
- `base64_encode`/`base64_decode` - Base64 operations
- `string_operation` - Text transformations (upper/lower/reverse/length)
- `format_json` - JSON formatting/minification
- `long_running_operation` - Progress simulation
- `get_test_image` - Returns test image
- `get_resource_content` - Embeds resources

### Resources (4 types)
- Static text resources
- Binary resources (images)
- Dynamic templates
- Embedded content support

### Prompts (2 templates)
- Simple greeting prompt
- Code review prompt with arguments

## 🧪 Testing

```bash
# Quick validation
make test          # Unit + integration + scenarios

# Development workflow
make test-verbose  # Human-readable output
make coverage      # Generate coverage report

# Manual testing
npx @modelcontextprotocol/inspector ./bin/mcp-adapters
```

See [Testing Guide](docs/testing-guide.md) for comprehensive testing documentation.

## 🏗️ Development

### Adding a New Tool

```go
// In cmd/everything/main.go
tool := mcp.NewTool("weather",
    mcp.WithDescription("Get weather for a location"),
    mcp.WithString("location", mcp.Required(), mcp.Description("City name")),
)
s.AddTool(tool, weatherHandler)

func weatherHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args := request.Params.Arguments.(map[string]any)
    location := args["location"].(string)
    // Implementation
    return mcp.NewToolResultText(fmt.Sprintf("Weather in %s: Sunny", location)), nil
}
```

### Development Commands

```bash
make all      # Clean, format, lint, test, build
make build    # Test and build
make run      # Build and run
make deploy   # Test and deploy to production
```

## 🚢 Deployment

Production deployment on Fly.io:

```bash
fly deploy  # Automated via GitHub Actions
```

See [fly.toml](fly.toml) for configuration.

## 📖 Documentation

### Essential Docs
- [STATE.yaml](docs/STATE.yaml) - Machine-readable project context
- [Testing Guide](docs/testing-guide.md) - Comprehensive testing documentation
- [Contributing](docs/contributing.md) - Development guidelines
- [Protocol Standards](docs/protocol-standards.md) - MCP implementation details

### Architecture Decisions
- [ADR-009](docs/adr/009-mcp-sdk-selection.md) - Why mark3labs/mcp-go
- [ADR-010](docs/adr/010-mcp-debug-system-architecture.md) - Debug system design
- [ADR-012](docs/adr/012-runtime-debug-configuration.md) - Runtime debug configuration
- [ADR-013](docs/adr/013-mcp-transport-selection.md) - StreamableHTTP transport

### Debug & Troubleshooting
- [Debug System](docs/debug/mcp-conformance-plan.md) - Protocol conformance testing
- [Known Issues](docs/KNOWN-ISSUES.md) - Current limitations

## 🛠️ Technical Details

- **SDK**: [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- **Protocol**: MCP v2025-03-26 over JSON-RPC 2.0
- **Transport**: StreamableHTTP (HTTP POST + SSE)
- **Runtime**: Go 1.22+
- **Deployment**: Fly.io with auto-scaling

## 🔮 Next Steps

1. **Authentication** - API keys or basic auth
2. **More Tools** - Weather, search, calculations
3. **Persistence** - Resource storage
4. **Monitoring** - Metrics and observability
5. **Performance** - Load testing and optimization

See [ROADMAP.md](docs/ROADMAP.md) for detailed plans.

## 🤝 Contributing

We welcome contributions! Please read our [Contributing Guide](docs/contributing.md) first.

## 📄 License

[MIT License](LICENSE)

---

**Questions?** Check [STATE.yaml](docs/STATE.yaml) for current context or open an issue.
