# Claude.ai Integration

Connect mcp adapters MCP server to Claude.ai for enhanced AI capabilities.

## Status

- ✅ Phase 1: CORS + Remote Deployment (COMPLETE)
- 🚧 Phase 2: OAuth Implementation (TODO)
- ⏳ Phase 3: Integration Testing (TODO)

## Current Features

- 11 working tools (encoding, math, text operations, etc.)
- 4 resources (text, markdown, image, dynamic)
- 2 prompts (greeting, code review)
- CORS enabled for claude.ai
- SSE transport support

## Quick Setup

1. Deploy: `fly deploy`
2. Test: `npx @modelcontextprotocol/inspector https://mcp-adapters.fly.dev/mcp --transport sse`
3. Add to Claude.ai: Settings → Connectors → Add More

## Documentation

- [Deployment Guide](./deployment.md)
- [Troubleshooting](./troubleshooting.md)
- OAuth Guide (coming soon)

## Known Issues

- No authentication (Phase 2)
- Stateless mode only
- Browser security blocks local servers
