---
title: "ADR-009: Use mark3labs/mcp-go for MCP Implementation"
status: "Accepted"
date: "2025-01-18"
tags: ["go", "mcp", "sdk", "dependency", "architecture"]
---

# ADR-009: Use mark3labs/mcp-go for MCP Implementation

## Status
Accepted

## Context
We need to implement an MCP server in Go for the Cowpilot project. Two primary SDK options exist:
- `github.com/modelcontextprotocol/go-sdk` - Official SDK maintained by Anthropic/Google
- `github.com/mark3labs/mcp-go` - Community SDK by Mark III Labs

## Decision
Use `github.com/mark3labs/mcp-go` for our initial MCP implementation.

## Alternatives Considered
1. **Official SDK** - Unreleased, marked unstable, warns "don't use in real projects"
2. **Direct implementation** - Too complex, reinventing the wheel
3. **Other community SDKs** - Less mature than mark3labs

## Rationale
- **Production readiness**: mark3labs has 4.5k stars, active releases, production usage
- **Completeness**: Supports all transports (stdio, SSE, HTTP streaming)
- **Documentation**: Working examples and clear API
- **Timeline**: Official SDK targets August 2025 for stability
- **Protocol compliance**: Both implement the same MCP protocol

## Consequences
### Positive
- Immediate development with stable foundation
- Proven production usage patterns
- Active community support

### Negative
- May need migration when official SDK stabilizes
- Not the "blessed" implementation
- Potential API differences from official SDK

### Mitigation
- Isolate MCP logic in internal packages for easier migration
- Monitor official SDK progress
- Document any workarounds or custom implementations

## References
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [Official Go SDK Discussion](https://github.com/orgs/modelcontextprotocol/discussions/364)
