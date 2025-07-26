# OAuth Protocol Requirements

## Critical Implementation Details

### Authorization Endpoint (GET /oauth/authorize)
**Query Parameters Required:**
- `client_id` - The client identifier
- `redirect_uri` - Where to redirect after authorization  
- `state` - Client state parameter for CSRF protection

**Response:** HTML form containing:
- Hidden `csrf_state` field with generated token
- Form fields for user to fill

### Authorization Endpoint (POST /oauth/authorize)
**Form Body Required (NOT query params):**
- `client_id` - Must match GET request
- `csrf_state` - Token from GET response
- `client_state` - Original state parameter
- `api_key` - User's RTM API key

**Common Mistake:** Passing client_id in query string instead of form body

### Token Endpoint (POST /oauth/token)
**Headers Required:**
- `Content-Type: application/x-www-form-urlencoded`

**Form Body Required:**
- `grant_type=authorization_code`
- `code` - Authorization code from redirect

### Bearer Token Usage
**Format:** `Authorization: Bearer <token>`
- Must include "Bearer " prefix with space
- Token cannot be empty

## Testing Checklist
- [ ] GET authorize includes all query params
- [ ] POST authorize uses form body, not query params
- [ ] CSRF token extracted from GET and sent in POST
- [ ] Cookies maintained between GET and POST
- [ ] Token endpoint uses correct Content-Type
- [ ] Bearer tokens include proper prefix
