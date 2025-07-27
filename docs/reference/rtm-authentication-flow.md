# RTM Authentication Flow - CRITICAL REFERENCE

## RTM is NOT OAuth!

RTM uses a custom frob-based authentication flow that we need to adapt to work with Claude's OAuth connector expectations.

## RTM's Actual Flow

1. **Get Frob**
   ```
   GET https://api.rememberthemilk.com/services/rest/?method=rtm.auth.getFrob&api_key=YOUR_API_KEY&api_sig=SIGNATURE
   Response: {"rsp":{"stat":"ok","frob":"FROB_VALUE"}}
   ```

2. **Direct User to Auth URL**
   ```
   https://www.rememberthemilk.com/services/auth/?api_key=YOUR_API_KEY&perms=delete&frob=FROB_VALUE&api_sig=SIGNATURE
   ```
   
3. **User Authorizes on RTM**
   - User clicks "OK, I'll allow it" button
   - RTM redirects to... WHERE? (This was our problem!)

4. **Exchange Frob for Token**
   ```
   GET https://api.rememberthemilk.com/services/rest/?method=rtm.auth.getToken&api_key=YOUR_API_KEY&frob=FROB_VALUE&api_sig=SIGNATURE
   Response: {"rsp":{"stat":"ok","auth":{"token":"TOKEN","perms":"delete","user":{...}}}}
   ```

## The Critical Problem

**RTM doesn't redirect anywhere by default!** After auth, it just shows a success page on rememberthemilk.com.

### Solution Approaches We Considered:

1. **Custom Callback URL Parameter** (DOESN'T WORK)
   - RTM API doesn't support OAuth-style redirect_uri parameter
   - No way to specify where to send user after auth

2. **Polling Approach** (WHAT WE NEED)
   - Store frob in our server
   - Open RTM auth URL in browser
   - Poll rtm.auth.getToken with the frob
   - When it succeeds, we have the token

3. **Manual Token Entry** (FALLBACK)
   - User authorizes on RTM
   - We show instructions to close the window
   - User returns to Claude and we exchange frob for token

## Adapting to Claude's OAuth Flow

Claude expects:
1. User clicks "Connect" → goes to `/oauth/authorize`
2. Server redirects to external auth
3. External service redirects back with code
4. Server exchanges code for token

We need to fake this with RTM:

### Our OAuth Adapter Strategy

1. **OAuth Authorize Endpoint** (`/oauth/authorize`)
   - Generate and store frob
   - Create "fake" OAuth code that maps to frob
   - Redirect to RTM auth URL
   - Start background polling for token

2. **Fake Callback** 
   - Since RTM won't redirect, we need to:
   - Either: Show custom page with "Click here after authorizing"
   - Or: Use JavaScript to detect window close and callback

3. **OAuth Token Endpoint** (`/oauth/token`)
   - Receive our fake code
   - Look up associated frob
   - Call rtm.auth.getToken
   - Return token in OAuth format

## Implementation Plan

### Phase 1: Basic Flow
```go
// In OAuth authorize handler:
1. Call RTM to get frob
2. Store mapping: fake_code → frob
3. Redirect to RTM auth URL with instructions

// In OAuth token handler:
1. Look up frob from fake_code
2. Try rtm.auth.getToken
3. If success, return token
4. If fail, return appropriate error
```

### Phase 2: Improved UX
- Add intermediate page with:
  - "Authorizing with Remember The Milk..."
  - "Click here after you've authorized"
  - Auto-detect completion via polling

### Key Data Structures

```go
type RTMAuthSession struct {
    Code      string    // Our fake OAuth code
    Frob      string    // RTM frob
    CreatedAt time.Time
    Token     string    // Set after successful exchange
}
```

## Testing Strategy

1. **Manual Test Flow**:
   ```bash
   # Set credentials
   export RTM_API_KEY=xxx
   export RTM_API_SECRET=xxx
   
   # Start server
   ./bin/cowpilot
   
   # In browser, go to:
   http://localhost:8080/oauth/authorize?client_id=claude_desktop
   
   # Complete RTM auth
   # Server should poll and get token
   ```

2. **Debug Endpoints**:
   - `/debug/rtm/frobs` - List active frobs
   - `/debug/rtm/test-exchange` - Test frob→token exchange

## Common Issues

1. **Signature Calculation**
   - Must be MD5 of: secret + sorted params
   - All params alphabetically sorted
   - Format: secretkey1value1key2value2...

2. **Frob Expiry**
   - Frobs expire after ~1 hour if unused
   - Must exchange quickly after user auth

3. **Token Persistence**
   - RTM tokens don't expire
   - Store securely for reuse

## References
- RTM Auth Docs: https://www.rememberthemilk.com/services/api/authentication.rtm
- API Explorer: https://www.rememberthemilk.com/services/api/
