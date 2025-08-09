# RTM: Local vs Production - Quick Reference

## The Key Point
**Fly.io secrets ≠ Local environment variables**

They are completely separate. Setting one doesn't affect the other.

## Quick Status Check

### Check Production (Your Deployed App)
```bash
# Is it working?
make -f Makefile.rtm quick-check

# Full diagnostic
make -f Makefile.rtm diagnose-production

# View logs
make -f Makefile.rtm logs-rtm
```

### Check Local (Your Mac)
```bash
# Check env vars
make -f Makefile.rtm check-env

# Full diagnostic  
make -f Makefile.rtm diagnose-local
```

## Setting Secrets

### For Production (One Time)
```bash
fly secrets set RTM_API_KEY=your_real_key -a rtm-mcp
fly secrets set RTM_API_SECRET=your_real_secret -a rtm-mcp
fly secrets set SERVER_URL=https://rtm-mcp.fly.dev -a rtm-mcp
```

### For Local Testing
```bash
export RTM_API_KEY=your_key
export RTM_API_SECRET=your_secret
export SERVER_URL=http://localhost:8080
```

## Common Scenarios

### "I want to test if production is working"
```bash
make -f Makefile.rtm quick-check
```

### "I want to run tests locally"
```bash
# First set local env vars
export RTM_API_KEY=test_key
export RTM_API_SECRET=test_secret

# Then run tests
make -f Makefile.rtm test-all
```

### "I want to connect Claude Desktop"
Use the production URL: `https://rtm-mcp.fly.dev/mcp`

### "I want to debug locally"
```bash
# Set local env vars
export RTM_API_KEY=your_key
export RTM_API_SECRET=your_secret

# Run local server
make -f Makefile.rtm run-debug

# Test with local URL
curl http://localhost:8080/rtm/authorize?client_id=test
```

## Why This Confusion Exists

1. **Fly.io runs on their servers**, not your Mac
2. **Environment variables are machine-specific**
3. **Fly secrets are encrypted** and only available to the deployed app
4. **Your Mac can't see Fly secrets** - they're on different computers!

## Bottom Line

- **Testing locally?** Need local env vars
- **Running on Fly.io?** Uses Fly secrets  
- **Want to know if production works?** Run `make -f Makefile.rtm quick-check`

That's it! 🎉
