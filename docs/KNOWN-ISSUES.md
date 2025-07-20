# Known Issues

## Go/Fly.io Specific

### Deployment
- First deploy requires `fly launch` to create app
- Secrets must be set via `fly secrets set`
- Auto-scaling needs careful configuration

### MCP Protocol
- HTTP streaming requires proper buffering
- Connection lifecycle management is critical
- JSON-RPC error handling must be precise

## Development Environment
- Go 1.22+ required
- Fly CLI must be authenticated
- Local testing needs mock transport

## Common Pitfalls
- Don't forget to handle context cancellation
- Always validate JSON-RPC messages
- Tool errors != protocol errors
