// Package contracts defines protocol requirements as executable specifications
package specs

import (
	"net/http"
	"net/url"
	"strings"
)

// OAuthContract defines the OAuth2 protocol requirements
type OAuthContract struct {
	violations []string
}

// ValidateAuthorizeRequest checks OAuth authorize endpoint requirements
func (c *OAuthContract) ValidateAuthorizeRequest(r *http.Request) []string {
	c.violations = nil

	switch r.Method {
	case "GET":
		c.validateAuthorizeGET(r)
	case "POST":
		c.validateAuthorizePOST(r)
	}

	return c.violations
}

func (c *OAuthContract) validateAuthorizeGET(r *http.Request) {
	params := r.URL.Query()

	// Required parameters for GET
	if params.Get("client_id") == "" {
		c.violations = append(c.violations, "GET /authorize: missing required 'client_id' parameter")
	}
	if params.Get("redirect_uri") == "" {
		c.violations = append(c.violations, "GET /authorize: missing required 'redirect_uri' parameter")
	}
	if params.Get("state") == "" {
		c.violations = append(c.violations, "GET /authorize: missing required 'state' parameter")
	}
}

func (c *OAuthContract) validateAuthorizePOST(r *http.Request) {
	// POST must have form data
	if err := r.ParseForm(); err != nil {
		c.violations = append(c.violations, "POST /authorize: invalid form data")
		return
	}

	// Required form fields
	required := map[string]string{
		"client_id":    "client identifier",
		"csrf_state":   "CSRF protection token",
		"client_state": "client state parameter",
		"api_key":      "RTM API key",
	}

	for field, desc := range required {
		if r.PostForm.Get(field) == "" {
			c.violations = append(c.violations, "POST /authorize: missing required form field '"+field+"' ("+desc+")")
		}
	}

	// Ensure form fields, not query params
	if r.URL.Query().Get("client_id") != "" && r.PostForm.Get("client_id") == "" {
		c.violations = append(c.violations, "POST /authorize: 'client_id' must be in form body, not query string")
	}
}

// ValidateTokenRequest checks token endpoint requirements
func (c *OAuthContract) ValidateTokenRequest(r *http.Request) []string {
	c.violations = nil

	if r.Method != "POST" {
		c.violations = append(c.violations, "Token endpoint must use POST method")
		return c.violations
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		c.violations = append(c.violations, "Token endpoint requires Content-Type: application/x-www-form-urlencoded")
	}

	if err := r.ParseForm(); err != nil {
		c.violations = append(c.violations, "Token endpoint: invalid form data")
		return c.violations
	}

	grantType := r.PostForm.Get("grant_type")
	if grantType == "" {
		c.violations = append(c.violations, "Token endpoint: missing 'grant_type'")
	} else if grantType != "authorization_code" {
		c.violations = append(c.violations, "Token endpoint: unsupported grant_type '"+grantType+"'")
	}

	if r.PostForm.Get("code") == "" {
		c.violations = append(c.violations, "Token endpoint: missing 'code' parameter")
	}

	return c.violations
}

// ValidateAuthorizationHeader checks Bearer token format
func (c *OAuthContract) ValidateAuthorizationHeader(header string) []string {
	c.violations = nil

	if header == "" {
		c.violations = append(c.violations, "Missing Authorization header")
		return c.violations
	}

	if !strings.HasPrefix(header, "Bearer ") {
		c.violations = append(c.violations, "Authorization header must use Bearer scheme")
		return c.violations
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		c.violations = append(c.violations, "Bearer token cannot be empty")
	}

	return c.violations
}

// ValidateRedirectURI checks redirect URI requirements
func (c *OAuthContract) ValidateRedirectURI(uri string) []string {
	c.violations = nil

	if uri == "" {
		c.violations = append(c.violations, "Redirect URI cannot be empty")
		return c.violations
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		c.violations = append(c.violations, "Invalid redirect URI format: "+err.Error())
		return c.violations
	}

	if parsed.Scheme == "" {
		c.violations = append(c.violations, "Redirect URI must include scheme (http/https)")
	}

	if parsed.Host == "" {
		c.violations = append(c.violations, "Redirect URI must include host")
	}

	if parsed.Fragment != "" {
		c.violations = append(c.violations, "Redirect URI must not contain fragment")
	}

	return c.violations
}
