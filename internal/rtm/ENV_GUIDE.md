# RTM Environment Configuration Guide

## Understanding Local vs Production Environments

### 🌍 Production (Fly.io)
When your app is **deployed on Fly.io**, it uses **Fly secrets** - NOT local environment variables.

**Check production secrets:**
```bash
# See what secrets are set (won't show values)
fly secrets list -a rtm-mcp

# Check production diagnostics
make -f Makefile.rtm diagnose-production
```

**Set production secrets (one time):**
```bash
fly secrets set RTM_API_KEY=your_actual_key -a rtm-mcp
fly secrets set RTM_API_SECRET=your_actual_secret -a rtm-mcp
fly secrets set SERVER_URL=https://rtm-mcp.fly.dev -a rtm-mcp
```

### 💻 Local Development
When running the server **on your Mac**, you need local environment variables.

**Set local environment variables:**
```bash
# Option 1: Export in terminal (temporary)
export RTM_API_KEY=your_key
export RTM_API_SECRET=your_secret
export SERVER_URL=http://localhost:8080

# Option 2: Create .env file (permanent)
cat > .env << EOF
RTM_API_KEY=your_key
RTM_API_SECRET=your_secret
SERVER_URL=http://localhost:8080
EOF

# Then source it
source .env
```

**Check local environment:**
```bash
make -f Makefile.rtm check-env
make -f Makefile.rtm diagnose-local
```

## Quick Reference

| Task | Command | Where Secrets Are |
|------|---------|-------------------|
| Test deployed app | `make diagnose-production` | Fly.io secrets |
| Test local server | `make diagnose-local` | Local env vars |
| Run server locally | `make run-local` | Local env vars |
| Deploy to Fly | `make deploy-rtm` | Fly.io secrets |
| View production logs | `make logs-rtm` | Fly.io secrets |

## Common Confusion Points

### ❌ Wrong: "I set Fly secrets, why doesn't local testing work?"
Fly secrets are ONLY available to your deployed app on Fly.io servers. Your local machine can't see them.

### ❌ Wrong: "I exported env vars, why doesn't production work?"
Your local environment variables don't transfer to Fly.io. You must use `fly secrets set`.

### ✅ Right: Two separate configurations
- **Local testing**: Needs local env vars
- **Production**: Needs Fly secrets

## Testing Claude Desktop Connection

### Against Production (Recommended)
```bash
# 1. Ensure secrets are set on Fly.io
fly secrets list -a rtm-mcp

# 2. Deploy latest code
make -f Makefile.rtm deploy-rtm

# 3. Test the deployed app
make -f Makefile.rtm diagnose-production

# 4. Connect Claude Desktop to:
# https://rtm-mcp.fly.dev/mcp
```

### Against Local Server
```bash
# 1. Set local env vars
export RTM_API_KEY=your_key
export RTM_API_SECRET=your_secret

# 2. Start local server
make -f Makefile.rtm run-local

# 3. Test locally
make -f Makefile.rtm diagnose-local

# 4. Connect Claude Desktop to:
# http://localhost:8080/mcp
```

## Quick Diagnostic

Run this to see what's configured where:
```bash
echo "=== LOCAL ==="
env | grep RTM_ || echo "No local RTM vars set"
echo ""
echo "=== PRODUCTION ==="
fly secrets list -a rtm-mcp 2>/dev/null || echo "No Fly secrets or not logged in"
```

## TL;DR

- **Deployed app (Fly.io)** = Uses Fly secrets (set with `fly secrets set`)
- **Local testing** = Uses local env vars (set with `export` or `.env` file)
- **They are completely separate** - setting one doesn't affect the other

The diagnostic error you saw was because you ran a LOCAL diagnostic command which checks LOCAL env vars. Your Fly.io production app already has the secrets it needs!
