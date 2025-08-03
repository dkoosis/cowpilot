# ADR-014: OAuth Adapter Pattern for RTM Frob Authentication

**Date:** 2025-07-30
**Status:** Accepted
**Context:** Adapting RTM's frob-based auth to Claude's OAuth expectations

## Decision

We implemented an OAuth adapter that translates between Claude's OAuth flow and RTM's custom frob-based authentication, using a polling mechanism to handle RTM's lack of redirect support.

## Context

RTM uses a custom frob-based authentication flow that is NOT OAuth:
1. Get a frob from RTM API
2. User authorizes on RTM website  
3. RTM doesn't redirect anywhere (shows success page)
4. Exchange frob for permanent token

Claude Desktop expects standard OAuth 2.0 flow with redirects.

## Decision Details

### OAuth Adapter Architecture
```
Claude → [OAuth] → Our Adapter → [Frob] → RTM API
```

### Key Implementation Points

1. **Fake OAuth Codes**: Map OAuth codes to RTM frobs in memory
2. **Polling Mechanism**: Background polling of `rtm.auth.getToken` 
3. **Manual Fallback**: User clicks "continue" after RTM auth
4. **CSRF Protection**: SameSite=None cookies for cross-origin flow

### Critical Discovery
**RTM doesn't support redirect_uri** - After auth, it stays on rememberthemilk.com. This required us to implement a polling pattern with manual user continuation.

## Consequences

### Positive
- Claude gets expected OAuth flow
- Users authenticate once, tokens persist
- Works with existing RTM API

### Negative  
- UX requires manual "continue" click
- Polling creates potential race conditions
- Fake OAuth codes add complexity

### Mitigation
- Clear user instructions during auth
- 5-minute polling timeout
- Visual indicators for auth status

## Alternatives Considered

1. **Direct Token Entry**: User manually enters RTM token
   - Rejected: Poor UX, error-prone

2. **Browser Extension**: Detect RTM auth completion
   - Rejected: Too complex, platform-specific

3. **Custom RTM App**: Request official callback URL
   - Future option: Could improve UX significantly

## References
- RTM Auth Docs: https://www.rememberthemilk.com/services/api/authentication.rtm
- OAuth 2.0: RFC 6749
- Implementation: `internal/rtm/oauth_adapter.go`
