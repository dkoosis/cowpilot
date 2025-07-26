# OAuth Adapter - June 2025 Spec Compliance

## Required Endpoints

### 1. Protected Resource Metadata (NEW)
`GET /.well-known/oauth-protected-resource`
```json
{
  "resource": "https://cowpilot.fly.dev",
  "authorization_servers": ["https://cowpilot.fly.dev"]
}
```

### 2. Authorization Server Metadata  
`GET /.well-known/oauth-authorization-server`
```json
{
  "issuer": "https://cowpilot.fly.dev",
  "authorization_endpoint": "https://cowpilot.fly.dev/oauth/authorize",
  "token_endpoint": "https://cowpilot.fly.dev/oauth/token",
  "registration_endpoint": "https://cowpilot.fly.dev/oauth/register",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code"],
  "code_challenge_methods_supported": ["S256"]
}
```

### 3. OAuth Flow
- Accept `resource` parameter in `/oauth/authorize` and `/oauth/token`
- Validate resource matches our server URL
- Check MCP-Protocol-Version header

## Simplified Implementation
Since we're adapting RTM API keys, we can:
- Ignore the `resource` parameter (we only have one resource)
- Always return same token regardless of resource
- Focus on minimal compliance for Claude.ai
