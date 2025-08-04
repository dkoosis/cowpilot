# Project History & Resolved Issues

## Completed Milestones

### 2025-08-03: Long-Running Tasks Implementation
- **What**: Full implementation of MCP protocol-compliant progress tracking
- **Solution**: Custom task manager with progress notifications and cancellation
- **Impact**: 1213 lines of production code + 393 lines of tests
- **Files**: `internal/longrunning/` package
- **Key Features**:
  - Thread-safe task state management
  - Progress rate limiting
  - Graceful degradation when progressToken not provided
  - Nil task handling for synchronous operations

### 2025-08-03: Concurrency & Thread Safety Fixes
- **Issue**: Race conditions in OAuth adapter and task manager
- **Root Cause**: Multiple mutex acquisitions in different orders
- **Solution**: 
  - Fixed lock ordering (authSessions before tokenStore)
  - Added comprehensive mutex protection
  - Implemented safe concurrent operations
- **Tests Added**:
  - TestConcurrentSessions
  - TestAuthorizationPendingRetry
  - TestFullOAuthFlow with race detection

### 2025-08-02: RTM OAuth Flow Fix
- **Issue**: Users not clicking "Allow" button on RTM authorization page
- **Root Cause**: Unclear instructions and missing visual guidance
- **Solution**:
  - Clearer step-by-step instructions
  - Visual indicator of allow button requirement
  - Warning messages
  - Better status messages during polling
- **Result**: OAuth flow working in production with Claude.ai

### 2025-07-30: MCP OAuth 2.1 Compliance
- **Implementation**: Full OAuth 2.1 spec compliance
- **Features Added**:
  - PKCE (S256)
  - Dynamic Client Registration (/oauth/register)
  - Resource parameter validation
  - Well-known discovery endpoints
  - 401 with WWW-Authenticate headers

### 2025-07-29: Infrastructure Consolidation
- **Problem**: Duplication across servers, missing endpoints
- **Solution**: Extracted shared infrastructure to `internal/core/infrastructure.go`
- **Pattern**: All servers call `core.SetupInfrastructure()`
- **Benefits**:
  - Single source of truth
  - Claude.ai compatibility guaranteed
  - Fix once, apply everywhere

### 2025-07-29: Build System Fixes
- **Issues Fixed**:
  - Makefile path updates
  - Package naming conflicts
  - Import path corrections
  - Linter violations
  - Integration test paths
- **Result**: Comprehensive build/test/deploy system working

## Lessons Learned

### Architecture Decisions

1. **OAuth Pattern Evolution**
   - March 2025: MCP spec had server as both auth and resource server
   - June 2025: Spec changed to separate concerns
   - Decision: Maintain current pattern until Claude requirements stabilize
   - Lesson: Build abstraction layers for evolving specs

2. **Progress Tracking Without Library Support**
   - mcp-go v0.34.1 lacks notification transport
   - Built custom task manager as workaround
   - Lesson: Don't wait for library updates, build what you need

3. **Infrastructure Sharing**
   - Started with copy-paste between servers
   - Led to inconsistencies and bugs
   - Solution: Extract common patterns early
   - Lesson: DRY principle applies to server setup code

### Testing Insights

1. **Race Condition Detection**
   - Always run tests with `-race` flag
   - Test concurrent operations explicitly
   - Use sync.WaitGroup for coordinating test goroutines

2. **OAuth Flow Testing**
   - Mock the entire flow, not just parts
   - Test state transitions explicitly
   - Verify CSRF protection and timeouts

3. **Integration Testing**
   - Test against deployed services when possible
   - Use real Claude.ai for final validation
   - Keep scenario tests separate from unit tests

### Production Insights

1. **User Experience**
   - Clear instructions > clever code
   - Visual feedback for async operations
   - Progressive status updates for long operations

2. **Error Handling**
   - Users need actionable error messages
   - Log full context server-side
   - Graceful degradation > perfect features

3. **Performance**
   - Pagination from day one
   - Cache expensive operations
   - Rate limit client notifications

## Archived Code Patterns

### Old OAuth Implementation (Pre-2025-08-02)
- Used popup windows (blocked by browsers)
- Lost CSRF tokens on redirect
- No visual feedback during authorization

### Old Error Handling (Pre-2025-08-03)
- Basic error returns without context
- No structured error types
- Missing stack traces

### Old Concurrency Pattern (Pre-2025-08-03)
- Inconsistent lock ordering
- Missing mutex protection on shared state
- No cancellation propagation

## Migration Notes

### For Future OAuth Spec Changes
When migrating to separate auth server:
1. Keep current adapter as legacy support
2. Build new adapter alongside
3. Use feature flags to switch
4. Test with Claude.ai extensively

### For Error Handling Upgrade
When adding cockroachdb/errors:
1. Start with new code only
2. Gradually migrate old code
3. Add slog in parallel
4. Keep error codes consistent

### For Spektrix Implementation
Use existing patterns from RTM:
1. Copy oauth_adapter pattern
2. Reuse batch_handlers structure
3. Implement client following RTM client pattern
4. Add to shared infrastructure setup