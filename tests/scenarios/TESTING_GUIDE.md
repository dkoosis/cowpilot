# MCP Testing Guide

## Testing Philosophy

Every build must verify that MCP conversations work correctly. We test at two levels:

1. **MCP Protocol Scenarios** - Primary verification that conversations work
2. **Go Unit Tests** - Safety net for our implementation

We trust the underlying libraries (mark3labs/mcp-go) for transport details.

## Quick Build Verification

```bash
# 1. Run MCP conversation tests
make e2e-test-prod    # Against production
# OR
make e2e-test-local   # Against local server

# 2. Run unit tests
make unit-test

# Both together
make test && make e2e-test-prod
```

## MCP Conversation Scenarios

These test real user interactions with the MCP server:

### Current Scenarios
- **Tool Discovery**: Can clients list available tools?
- **Tool Execution**: Can clients call the hello tool?
- **Error Handling**: Does server properly reject invalid requests?
- **Resource Discovery**: Does server respond to resource queries?

### Adding New Scenarios
When you add capabilities, add corresponding MCP scenarios:
```bash
# Example: After adding a "calculate" tool
# Add test in mcp_scenarios.sh:
# - List tools shows "calculate" 
# - Call calculate with valid args
# - Call calculate with invalid args
```

## Test Commands Reference

| Purpose | Command | When to Use |
|---------|---------|-------------|
| MCP Scenarios (prod) | `make e2e-test-prod` | Before deployment |
| MCP Scenarios (local) | `make e2e-test-local` | During development |
| Unit Tests | `make unit-test` | Every build |
| All Tests | `make test && make e2e-test-prod` | Pre-commit |

## Development Workflow

1. Make changes
2. Run `make unit-test` - Verify functions work
3. Run `make e2e-test-local` - Verify MCP conversation works
4. Commit
5. CI runs `make test && make e2e-test-prod`

## Debugging (When Needed)

If MCP tests fail, use Inspector directly:
```bash
# Interactive testing
npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list
```

For protocol-level debugging, see [DEBUG_GUIDE.md](./DEBUG_GUIDE.md).
