# OAuth Connection Issue Diagnostic Guide

## Problem
"Successfully connected but still shows Connect button" in Claude

## Diagnostic Tools Created

### 1. Quick Diagnosis
```bash
# Check production OAuth endpoints and logs
make diagnose-prod

# Full local server diagnostics
make diagnose-local
```

### 2. Real-Time Monitoring
```bash
# Start server with colored OAuth event monitoring
make monitor-oauth

# Then in Claude, try to connect and watch the logs
```

### 3. Manual Testing
```bash
# Test OAuth endpoints directly
go run scripts/diagnostics/oauth_trace.go

# Check specific endpoints
curl https://rtm.fly.dev/.well-known/oauth-protected-resource
curl https://rtm.fly.dev/.well-known/oauth-authorization-server
```

## What to Look For

### ✓ Success Indicators
- `[OAuth] Authorize request: ...` - OAuth flow started
- `[OAuth] Generated auth code: ...` - Code created
- `[OAuth] Token request: ...` - Token exchange initiated
- `[OAuth] Generated bearer token: ...` - Token created
- `[OAuth Callback] SUCCESS: ...` - Callback received
- Window closes automatically after success

### ✗ Failure Indicators
- Missing WWW-Authenticate header on 401
- `[OAuth] ERROR: Token not found in store`
- `[OAuth] ERROR: Invalid or expired code`
- `[OAuth Callback] ERROR: Missing parameters`
- Window stays open after "success"
- No logs appearing at all

## Browser-Side Debugging

1. Open Chrome/Firefox Developer Tools (F12)
2. Go to Network tab
3. Try to connect in Claude
4. Look for:
   - `/oauth/authorize` request
   - Redirect to callback URL
   - Any failed requests (in red)

5. In Console tab:
```javascript
// Check for errors
console.log(window.location.href);
console.log(document.cookie);

// Check if postMessage is being sent
window.addEventListener('message', (e) => console.log('Message:', e));
```

## Enhanced Logging Added

All OAuth operations now log with `[OAuth]` prefix:
- Authorization requests
- Token generation
- Token validation
- Callback processing
- Success/error states

## Common Issues & Solutions

### Issue 1: Window doesn't close
**Symptom**: Success page shows but window stays open
**Cause**: JavaScript auto-close blocked by browser
**Solution**: Added multiple notification methods (window.close, postMessage)

### Issue 2: Token not found
**Symptom**: `[OAuth] ERROR: Token not found in store`
**Cause**: Token storage/retrieval mismatch
**Solution**: Check token store implementation

### Issue 3: Missing WWW-Authenticate
**Symptom**: 401 without WWW-Authenticate header
**Cause**: Auth middleware not properly configured
**Solution**: Already fixed in recent updates

### Issue 4: Callback not received
**Symptom**: No `[OAuth Callback]` logs
**Cause**: Callback server not started or wrong port
**Solution**: Check callback server initialization

## Next Steps

1. **Deploy with logging**:
```bash
make deploy-rtm
```

2. **Monitor production**:
```bash
make diagnose-prod
# or
flyctl logs --app rtm --tail
```

3. **Test connection in Claude** while monitoring logs

4. **Report findings** - Look for:
   - Where in the flow it stops
   - Any error messages
   - Whether callback is received
   - If window closes properly

## Files Modified

- `internal/auth/oauth_adapter.go` - Added comprehensive logging
- `internal/auth/oauth_callback_server.go` - Added auto-close and postMessage
- `scripts/diagnostics/*` - New diagnostic tools
- `Makefile` - Added diagnostic commands

## Confidence Level

**40% confidence** in the window.close/postMessage fix because:
- Haven't verified Claude's specific OAuth requirements
- Multiple possible root causes
- Need diagnostic data to confirm actual issue

Run diagnostics first to identify the real problem before deploying the fix.
