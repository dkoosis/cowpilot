package specs_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vcto/mcp-adapters/internal/protocol/specs"
)

func TestOAuthContract_ValidatesRequest_When_AuthorizeEndpointIsCalled(t *testing.T) {
	contract := &specs.OAuthContract{}

	t.Run("ValidateAuthorizeRequest_Fails_When_GETRequestIsMissingParams", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/oauth/authorize", nil)
		violations := contract.ValidateAuthorizeRequest(req)

		if len(violations) != 3 {
			t.Errorf("Expected 3 violations, got %d", len(violations))
		}

		expectedViolations := []string{"client_id", "redirect_uri", "state"}
		for _, expected := range expectedViolations {
			found := false
			for _, v := range violations {
				if strings.Contains(v, expected) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Missing expected violation for %s", expected)
			}
		}
	})

	t.Run("ValidateAuthorizeRequest_Fails_When_POSTRequestMisusesQueryParams", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/oauth/authorize?client_id=test", nil)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		violations := contract.ValidateAuthorizeRequest(req)

		found := false
		for _, v := range violations {
			if strings.Contains(v, "form body, not query string") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Failed to detect client_id in query string violation")
		}
	})
}

func TestOAuthContract_ValidatesHeader_When_BearerTokenIsPresent(t *testing.T) {
	contract := &specs.OAuthContract{}
	tests := []struct {
		header      string
		shouldFail  bool
		description string
	}{
		{"", true, "empty header"},
		{"token123", true, "missing Bearer prefix"},
		{"Bearer", true, "missing token"},
		{"Bearer ", true, "empty token"},
		{"Bearer token123", false, "valid format"},
	}

	for _, tt := range tests {
		t.Run("ValidateAuthorizationHeader_Succeeds_When_FormatIsValid", func(t *testing.T) {
			violations := contract.ValidateAuthorizationHeader(tt.header)
			hasFailed := len(violations) > 0

			if hasFailed != tt.shouldFail {
				t.Errorf("%s: expected failure=%v, got %v",
					tt.description, tt.shouldFail, hasFailed)
			}
		})
	}
}
