# RTM Quick Reference

## Is RTM Working in Production?
```bash
make rtm-status
```

## Need to Debug?

### Production Issues
```bash
make rtm-status        # Quick check
make diagnose-prod     # Full diagnostics  
make rtm-logs         # View logs
make monitor-oauth    # Watch OAuth flow
```

### Local Development
```bash
# First set env vars (for local only!)
export RTM_API_KEY=your_key
export RTM_API_SECRET=your_secret

# Then test
make rtm-test         # Run tests
make diagnose-local   # Check setup
make run             # Start server
```

## Deploy Updates
```bash
make deploy-rtm
```

## Key Points

1. **Fly.io secrets** (production) and **local env vars** are completely separate
2. Your production app uses **Fly secrets** (already set via `fly secrets set`)
3. Local testing needs **local env vars** (set with `export`)
4. Use `make rtm-status` to quickly check if production is working

## If Claude Desktop Can't Connect

1. Run `make rtm-status` - is it online?
2. Check the URL: `https://rtm-mcp.fly.dev/mcp`
3. Run `make rtm-logs` to see what's happening
4. The user must click "OK, I'll allow it" on RTM's page

That's all you need to know! 🎉
