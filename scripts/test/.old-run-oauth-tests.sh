#!/bin/bash
# Run OAuth-specific tests

set -e

echo "=== Running OAuth Tests ==="

# Unit tests
echo -e "\n--- OAuth Adapter Unit Tests ---"
go test -v ./internal/auth -run TestOAuthAdapter

echo -e "\n--- CSRF Token Tests ---"
go test -v ./internal/auth -run TestCSRFTokens

echo -e "\n--- Token Store Tests ---"
go test -v ./internal/auth -run TestTokenStore

echo -e "\n--- OAuth Middleware Tests ---"
go test -v ./internal/auth -run TestOAuthMiddleware

echo -e "\n--- OAuth Scenario Tests ---"
go test -v ./tests -run TestOAuthFlow

echo -e "\n=== OAuth Test Summary ==="
go test ./internal/auth ./tests -run "OAuth|CSRF|Token" -count=1 | grep -E "PASS|FAIL|ok" | sort | uniq -c

echo -e "\n✓ OAuth tests complete"
