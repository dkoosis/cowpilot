# OAuth Adapter Implementation Plan

## Architecture: OAuth Facade for API Key Service

```
Claude.ai → [OAuth 2.1] → Our MCP Server → [API Key] → Remember The Milk
```

## Implementation

### 1. OAuth Endpoints (Facade)
- `/.well-known/oauth-authorization-server` - Metadata
- `/oauth/authorize` - Show RTM API key input form
- `/oauth/token` - Exchange auth code for our token
- `/oauth/register` - DCR (store client info)

### 2. Token Mapping
```go
// internal/auth/token_store.go
type TokenStore struct {
    // OAuth token → RTM API key mapping
    tokens map[string]string  // bearer_token -> rtm_api_key
}
```

### 3. Flow
1. Claude.ai initiates OAuth
2. We redirect to our authorize page
3. User enters RTM API key
4. We generate bearer token, store mapping
5. Claude.ai uses bearer token
6. We translate to RTM API key for backend calls

### 4. Simplified Implementation
- No real OAuth provider needed
- No user database
- Just token↔API key mapping
- Session storage for token persistence

## Benefits
- Simpler than full OAuth
- Works with API key services
- Claude.ai gets expected OAuth flow
- User experience: enter API key once
