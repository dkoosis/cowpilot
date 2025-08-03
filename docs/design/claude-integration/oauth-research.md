# MCP OAuth Research & Implementation Plan

## Current State (July 2025)

### Spec Versions
- **2025-03-26**: OAuth 2.1 with authorization framework
- **2025-06-18**: Added Protected Resource Metadata (RFC 8707)

### Key Requirements (Latest Spec)
1. OAuth 2.1 (PKCE mandatory)
2. Dynamic Client Registration (SHOULD support)
3. Authorization Server Metadata (RFC 8414)
4. Resource Indicators (RFC 8707) - prevent token misuse

### Critical Issues
1. **MCP server = Resource Server + Auth Server** (problematic)
   - Aaron Parecki proposes separation (RFC 9728)
   - Enterprise friction - doesn't integrate with existing IdPs
2. **Claude.ai specific**: 
   - Callback: `https://claude.ai/api/mcp/auth_callback`
   - Requires DCR or manual client registration

## Implementation Options

### Option 1: Full Spec Compliance (Complex)
- Implement both resource + auth server
- Support DCR
- Handle token issuance/refresh
- **Risk**: High complexity, enterprise unfriendly

### Option 2: Cloudflare MCP (Recommended)
- Uses `workers-oauth-provider` library
- Handles OAuth complexity
- Supports third-party IdPs
- **Benefit**: Battle-tested, less code

### Option 3: Minimal Implementation
- Basic OAuth 2.1 flow
- No DCR (manual client registration)
- External auth server
- **Risk**: May not work with claude.ai

## Systematic Plan

### Step 1: Research & Decision (TODAY)
- [ ] Review SDK examples (TypeScript/Python)
- [ ] Test Cloudflare MCP approach
- [ ] Document decision rationale

### Step 2: Prototype (Day 1)
- [ ] Basic OAuth endpoints
- [ ] Test with Postman/curl
- [ ] Document each endpoint

### Step 3: Integration (Day 2)
- [ ] Connect to mcp-go server
- [ ] Test with MCP Inspector
- [ ] Document issues

### Step 4: Claude.ai Test (Day 3)
- [ ] Deploy to fly.io
- [ ] Test connector setup
- [ ] Document any failures

## Tracking Template
```
Date: 
Attempt: 
Approach: 
Result: 
Issues: 
Next Step: 
```
