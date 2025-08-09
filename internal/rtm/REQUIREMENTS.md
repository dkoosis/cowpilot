# RTM OAuth Integration Requirements

## Critical Requirements for RTM Registration to Work

### 1. Environment Variables (REQUIRED)
```bash
RTM_API_KEY=your_api_key_here        # From RTM developer console
RTM_API_SECRET=your_secret_here      # From RTM developer console  
SERVER_URL=https://your-app.fly.dev  # MUST be HTTPS in production
PORT=8080                             # Default Fly.io port
```

### 2. OAuth Flow Sequence (MUST FOLLOW)
```
1. Claude → GET /rtm/authorize?client_id=X&state=Y&redirect_uri=Z
2. Server → Generate frob from RTM API
3. Server → Store session with fake OAuth code
4. Server → Show auth form with CSRF token
5. User → Clicks "Connect" button
6. Server → Redirects to RTM auth page
7. User → Clicks "OK, I'll allow it" on RTM
8. User → Returns to our intermediate page
9. Page → Polls /rtm/check-auth every 2 seconds
10. Server → Exchanges frob for token via rtm.auth.getToken
11. Server → Stores token in session
12. Page → Redirects to /rtm/callback
13. Server → Returns to Claude's redirect_uri with code
14. Claude → POST /rtm/token with code
15. Server → Returns access_token (RTM token)
```

### 3. Cookie Requirements
- **Production (HTTPS)**: `SameSite=None; Secure=true`
- **Development (HTTP)**: `SameSite=Lax; Secure=false`
- **HttpOnly**: Always true
- **MaxAge**: 1800 seconds (30 minutes)

### 4. RTM API Signature Requirements
```go
// Signature = MD5(secret + alphabetically_sorted_params)
// Example: MD5("SECRETapi_keyVALUEfrobVALUEmethodrtm.auth.getToken")
```

### 5. Error Codes to Handle
- **98**: Invalid signature (check parameter sorting)
- **101**: Invalid frob (user hasn't authorized yet)
- **102**: Login required (user not logged in to RTM)

### 6. Timing Constraints
- **Frob expiry**: 60 minutes (start fresh after 55 minutes)
- **Polling interval**: 2 seconds
- **Polling timeout**: 5 minutes
- **Session cleanup**: After 10 minutes

### 7. CORS Requirements (for Fly.io)
```go
// Must allow Claude Desktop's origin
AllowedOrigins: []string{
    "http://localhost:*",
    "https://claude.ai",
    "https://*.anthropic.com",
}
```

### 8. MCP Integration Points
- **OAuth endpoints**: `/rtm/authorize`, `/rtm/callback`, `/rtm/token`
- **Resource path**: `/mcp`
- **SSE endpoint**: `/mcp/sse`

## Testing Checklist

### Pre-Flight Checks
- [ ] RTM_API_KEY is set and valid
- [ ] RTM_API_SECRET is set and valid  
- [ ] SERVER_URL is HTTPS in production
- [ ] Fly.io app is deployed and running
- [ ] DNS resolves correctly

### OAuth Flow Tests
- [ ] Can generate frob from RTM
- [ ] CSRF token is set and validated
- [ ] Auth URL has correct signature
- [ ] Polling mechanism detects authorization
- [ ] Token exchange succeeds
- [ ] Session stores token correctly
- [ ] Callback redirects with code
- [ ] Token endpoint returns valid access_token

### Error Handling Tests
- [ ] Handles expired frob gracefully
- [ ] Handles user denial properly
- [ ] Handles network timeouts
- [ ] Cleans up expired sessions
- [ ] Returns proper OAuth error codes

### Integration Tests
- [ ] Claude Desktop can complete full flow
- [ ] Token persists across requests
- [ ] RTM API calls work with token
- [ ] Resources load correctly

## Common Failure Points

### 1. "Authorization Pending" Loop
**Symptom**: Claude keeps showing "authorization pending"
**Causes**:
- User didn't click "OK, I'll allow it" on RTM
- Frob expired (> 60 minutes)
- Token exchange failed silently

**Fix**: Check `/rtm/check-auth` endpoint logs

### 2. CSRF Token Mismatch
**Symptom**: "Invalid CSRF token" error
**Causes**:
- Cookie blocked by browser
- SameSite setting incorrect
- Page refreshed between steps

**Fix**: Check cookie settings match environment (HTTP vs HTTPS)

### 3. Invalid Signature (Error 98)
**Symptom**: RTM returns error 98
**Causes**:
- Parameters not sorted alphabetically
- Secret prepended incorrectly
- Special characters not escaped

**Fix**: Log signature calculation and verify sorting

### 4. No Redirect After Auth
**Symptom**: Stuck on RTM success page
**Causes**:
- RTM doesn't support redirect_uri
- User doesn't know to return to our tab

**Fix**: This is expected - user must manually return

## Monitoring Points

### Log These Events
```go
log.Printf("[RTM AUTH] Frob generated: %s", frob)
log.Printf("[RTM AUTH] Session created: code=%s", code)
log.Printf("[RTM AUTH] Check auth attempt: code=%s, has_token=%v", code, session.Token != "")
log.Printf("[RTM AUTH] Token exchange: success=%v", err == nil)
log.Printf("[RTM AUTH] Callback redirect: uri=%s", redirectURL)
```

### Metrics to Track
- Frob generation success rate
- Authorization completion rate
- Average time to complete auth
- Session timeout rate
- Token validation success rate

## Recovery Procedures

### If Registration Fails
1. Check environment variables
2. Verify RTM API credentials are valid
3. Check server logs for specific error
4. Test frob generation independently
5. Verify signature calculation
6. Check cookie settings
7. Test with curl commands (see Testing section)

### Manual Testing Commands
```bash
# Test frob generation
curl "https://api.rememberthemilk.com/services/rest/?method=rtm.auth.getFrob&api_key=KEY&api_sig=SIG&format=json"

# Test OAuth flow start
curl -c cookies.txt "https://your-app.fly.dev/rtm/authorize?client_id=test&state=xyz&redirect_uri=http://localhost:3000/callback"

# Test auth check (after manual RTM auth)
curl -b cookies.txt "https://your-app.fly.dev/rtm/check-auth?code=YOUR_CODE"

# Test token exchange
curl -X POST "https://your-app.fly.dev/rtm/token" \
  -d "code=YOUR_CODE&grant_type=authorization_code"
```

## Maintenance Tasks

### Daily
- Check error logs for auth failures
- Monitor auth success rate

### Weekly  
- Review session cleanup performance
- Check for expired frobs in logs

### Monthly
- Rotate RTM API credentials if needed
- Review and update timeout values
- Test full OAuth flow manually

## Version Compatibility

### Tested With
- Go 1.21+
- mark3labs/mcp-go v0.1.0
- Fly.io platform v2
- Claude Desktop 1.0+

### Breaking Changes to Watch
- RTM API version changes
- OAuth 2.1 spec updates
- MCP protocol changes
- Fly.io platform updates
