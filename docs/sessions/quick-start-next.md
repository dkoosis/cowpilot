# Quick Start for Next Session

## Essential Commands
```bash
# Test production server instantly
curl -s https://cowpilot.fly.dev/health

# See what tools are available
npx @modelcontextprotocol/inspector --cli https://cowpilot.fly.dev/ --method tools/list

# Call the hello tool
npx @modelcontextprotocol/inspector --cli https://cowpilot.fly.dev/ --method tools/call --tool-name hello
```

## Adding a New Tool
1. Open `/cmd/cowpilot/main.go`
2. Copy the `helloHandler` pattern
3. Add new tool with `mcp.NewTool()` and `s.AddTool()`
4. Test locally: `go run cmd/cowpilot/main.go`
5. Run tests: `make test && make e2e-test-local`
6. Deploy: `fly deploy`

## Current State Summary
- **Live Server**: https://cowpilot.fly.dev/ ✅
- **Tools**: hello ✅
- **Tests**: Comprehensive E2E suite with dual approach ✅
- **Next**: Add more tools, auth, resources, monitoring

## Key Files
- Server code: `/cmd/cowpilot/main.go`
- Tests: `/tests/e2e/mcp_scenarios.sh` and `raw_sse_test.sh`
- State: `/docs/STATE.yaml` (main context file)
