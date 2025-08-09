# TODO - Active Development Tasks

## 🔴 P0 - Critical (Blocking Production)

### Fix RTM RRULE Parsing
- [ ] Change RRule field to `json.RawMessage` in `internal/rtm/client.go`
- [ ] Add test for recurring task parsing
- [ ] Test with various RRULE patterns from RTM
- **Blocks**: Recurring task functionality
- **Estimate**: 2 hours

## 🟠 P1 - High Priority

### Implement Simple Metrics
- [ ] Add counter to each MCP tool handler
- [ ] Track: tool calls, errors, response times
- [ ] Weekly metrics.json dump
- [ ] Start tracking wrong tool selections
- [ ] Create weekly review habit
- **Reference**: `prompts/G-Metrics-Implementation.yaml`
- **Impact**: Actually know if we're improving
- **Estimate**: 4 hours initial, 15 min/week ongoing

### Secure Credential Storage
- [ ] Research keyring libraries (99designs/keyring recommended)
- [ ] Implement secure storage interface
- [ ] Migrate from environment variables
- [ ] Add rotation mechanism
- **Security**: Prevents credential exposure
- **Estimate**: 4 hours

### RTM Task Volume Limiting
- [ ] Implement server-side pagination
- [ ] Add configurable result limits
- [ ] Cache paginated results
- [ ] Add "load more" mechanism
- **Impact**: Prevents UI freezing with large task lists
- **Estimate**: 6 hours

### Deploy RTM with Batch Tools
- [ ] Run final integration tests
- [ ] Update fly.toml if needed
- [ ] Deploy with `make deploy-rtm`
- [ ] Verify batch operations in production
- [ ] Monitor for errors
- **Estimate**: 1 hour

## 🟡 P2 - Medium Priority

### Implement Task Results Cache
- [ ] Design cache structure
- [ ] Add TTL configuration
- [ ] Implement invalidation logic
- [ ] Add cache metrics
- **Benefit**: Reduces API calls, enables position-based operations
- **Estimate**: 8 hours

### Extract Shared Infrastructure Package
- [ ] Create `internal/mcpserver/` package
- [ ] Move common setup from `internal/core/infrastructure.go`
- [ ] Update all servers to use shared package
- [ ] Test all servers still work
- **Benefit**: Single source of truth for server setup
- **Estimate**: 4 hours

### Start Spektrix Server Implementation
- [ ] Copy RTM server structure as template
- [ ] Implement HMAC authentication
- [ ] Add customer creation endpoint
- [ ] Add address management
- [ ] Add tag operations
- **Reference**: `docs/AUTH_FLOWS.yaml#SPEKTRIX_HMAC_FLOW`
- **Estimate**: 16 hours

## 🔵 P3 - Nice to Have

### Enhanced Error Handling
- [ ] Add cockroachdb/errors dependency
- [ ] Create error categories and codes
- [ ] Implement slog for structured logging
- [ ] Migrate existing error handling
- **Benefit**: Better debugging and monitoring
- **Estimate**: 12 hours

### Implement Smart Search Features
- [ ] Add search preset saving
- [ ] Implement numbered result lists
- [ ] Add position-based task retrieval
- [ ] Create favorites resource
- **Reference**: `docs/RTM_ENHANCEMENTS.yaml#SEARCH_ENHANCEMENTS`
- **Estimate**: 8 hours

### Add Integration Tests for OAuth
- [ ] Complete `TestRTMOAuthRaceCondition`
- [ ] Complete `TestRTMOAuthSuccessfulFlow`
- [ ] Add timeout tests
- [ ] Add CSRF validation tests
- **Files**: `tests/integration/rtm_oauth_test.go`
- **Estimate**: 4 hours

## 📋 Quick Wins (< 1 hour each)

- [ ] Remove `.removed` and `.disabled` files properly
- [ ] Update README with current deployment status
- [ ] Add health check enhancements
- [ ] Document batch tool usage examples
- [ ] Add performance metrics collection

## 🔄 Ongoing Maintenance

- [ ] Monitor error logs for new issues
- [ ] Review and respond to Claude.ai connection issues
- [ ] Keep dependencies updated
- [ ] Maintain test coverage above 80%

## 📅 Sprint Planning

### This Week
1. Fix RTM RRULE parsing (P0)
2. Deploy RTM with batch tools
3. Implement task cache

### Next Week
1. Secure credential storage
2. RTM task volume limiting
3. Start Spektrix implementation

### Future Sprints
- Enhanced error handling
- Smart search features
- OAuth spec migration planning

## 📝 Notes

- Always check STATE.yaml before starting work
- Run tests with `-race` flag
- Update STATE.yaml after completing tasks
- Document any new patterns in HISTORY.md

## 🎯 Success Metrics

- [ ] Zero P0 issues in production
- [ ] < 100ms response time for cached operations
- [ ] 100% OAuth success rate
- [ ] No memory leaks or goroutine leaks
- [ ] Test coverage > 80%