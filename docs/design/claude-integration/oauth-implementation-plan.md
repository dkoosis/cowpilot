# OAuth Implementation Decision

## Decision: OAuth Proxy Approach

After reviewing TypeScript SDK examples and community implementations, we'll use an **OAuth Proxy** approach rather than implementing a full auth server.

### Rationale
1. TypeScript SDK provides `ProxyOAuthServerProvider`
2. Community boilerplate (chrisleekr) successfully uses this pattern
3. Avoids the MCP spec issue of server being both resource + auth server
4. Simpler to implement and maintain

## Implementation Steps

### Step 1: Study Reference Implementation
```bash
# Clone TypeScript SDK
git clone https://github.com/modelcontextprotocol/typescript-sdk
cd typescript-sdk/src/examples

# Key files to study:
- server/simpleStreamableHttp.ts (auth endpoints)
- server/simple-auth/* (OAuth server example)
- client/simpleOAuthClient.ts (client flow)
```

### Step 2: Go Implementation Plan
Since mcp-go doesn't have OAuth examples, we'll port the TypeScript approach:

1. **OAuth Endpoints** (internal/auth/oauth.go):
   - /.well-known/oauth-authorization-server (metadata)
   - /oauth/authorize (redirect to auth provider)
   - /oauth/token (token exchange)
   - /oauth/register (DCR support)

2. **Session Management** (internal/auth/session.go):
   - Token storage
   - Session validation
   - PKCE support

3. **Middleware** (internal/auth/middleware.go):
   - Bearer token validation
   - 401 with WWW-Authenticate header

### Step 3: Test Locally
1. Mock auth flow with hardcoded tokens
2. Test with MCP Inspector
3. Document each endpoint's behavior

### Step 4: Real Auth Provider
Options:
1. Auth0 (like chrisleekr boilerplate)
2. Google OAuth
3. Simple built-in provider

## Tracking Log

### Attempt 1: Reference Study
Date: 2025-07-26
Goal: Understand TypeScript OAuth implementation
Files reviewed:
- [x] TypeScript SDK OAuth approach (ProxyOAuthServerProvider)
- [x] June 2025 spec changes (RFC 8707, RFC 9728)
- [x] Community implementations (chrisleekr boilerplate)
Result: Decided on OAuth adapter approach for API key services
Next: Implement OAuth endpoints

### Attempt 2: Go OAuth Adapter Implementation
Date: 2025-07-26  
Goal: Implement OAuth facade for RTM API keys
Implemented:
- [x] oauth_adapter.go - All OAuth endpoints
- [x] token_store.go - Bearer token to API key mapping
- [x] middleware.go - Auth validation
- [x] main.go integration
- [x] Test script (test-oauth-flow.sh)
Result: Complete OAuth adapter with June 2025 compliance
Issues: None
Next: Deploy and test with Claude.ai
