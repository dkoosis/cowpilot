# 🧪 MCP OAuth Spec Compliance Test - Quick Start

## Fixed compilation errors and ready to test!

### ✅ **Step 1: Build and Test**
```bash
# From project root
bash scripts/test-oauth-build.sh
```

### ✅ **Step 2: Run the Test Server**
```bash
# If build succeeds
cd cmd/oauth_spec_test 
./run-test.sh
```

### ✅ **Step 3: Test with Claude.ai**
1. **Keep the test server running** (from Step 2)
2. **In Claude.ai web interface**, try to add MCP server:
   - URL: `http://localhost:8090/mcp`
3. **Watch the server logs** for Claude's requests

### 📊 **Expected Results**

| Claude.ai Behavior | Spec Support | Decision |
|-------------------|--------------|----------|
| Fetches `/.well-known/oauth-protected-resource` | ✅ **New spec** | Migrate to resource server pattern |
| Redirects to auth server at `:8091` | ✅ **New spec** | Migrate architecture |
| Uses resource indicators | ✅ **New spec** | Future-proof approach |
| Expects `/oauth/authorize` on MCP server | ❌ **Old spec** | Fix current auth UX first |

### 🎯 **What This Determines**

- **✅ New spec supported** → Refactor RTM server to resource-only pattern  
- **❌ Old spec only** → Fix current auth UX issues first, migrate later

### 🔍 **Debug Info**

The test server logs will show exactly what Claude.ai requests:
- `📋 Served resource metadata` = Claude supports new spec discovery
- `🔐 Authorization request` = Claude uses separate auth server  
- `🎫 Issued token` = Claude uses resource indicators
- No requests = Claude expects old pattern

### 📋 **Files Created**
```
cmd/oauth_spec_test/
├── main.go           # Fixed compilation issues
├── run-test.sh       # Test runner 
└── README.md         # Full documentation

scripts/
└── test-oauth-build.sh  # Build and setup script
```

**This test will definitively answer whether we should migrate to the new MCP OAuth spec or fix the current auth UX first.**
