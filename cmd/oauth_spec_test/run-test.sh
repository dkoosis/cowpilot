#!/bin/bash

# OAuth Spec Compliance Test Script
# Tests if Claude.ai supports the new MCP OAuth spec (June 18, 2025)

set -e

echo "🧪 MCP OAuth Spec Compliance Test"
echo "=================================="
echo ""

# Build the test server
echo "📦 Building test server..."
cd "$(dirname "$0")"
go build -o oauth-spec-test main.go

# Start the test server
echo "🚀 Starting OAuth test servers..."
echo "   Resource Server: http://localhost:8090/mcp"
echo "   Auth Server: http://localhost:8091"
echo ""

# Run in background and capture PID
./oauth-spec-test &
TEST_PID=$!

# Give servers time to start
sleep 2

echo "✅ Test servers running (PID: $TEST_PID)"
echo ""

# Test the metadata endpoints
echo "📋 Testing OAuth 2.0 Protected Resource Metadata (RFC 9728):"
echo "curl http://localhost:8090/.well-known/oauth-protected-resource"
curl -s http://localhost:8090/.well-known/oauth-protected-resource | jq . || echo "Failed to get resource metadata"
echo ""

echo "🔐 Testing Authorization Server Metadata:"
echo "curl http://localhost:8091/.well-known/oauth-authorization-server"  
curl -s http://localhost:8091/.well-known/oauth-authorization-server | jq . || echo "Failed to get auth server metadata"
echo ""

echo "🎫 Testing Token Validation (should fail without token):"
echo "curl http://localhost:8090/mcp"
curl -s -o /dev/null -w "HTTP %{http_code}\n" http://localhost:8090/mcp
echo ""

echo "✅ Testing with valid token:"
echo "curl -H 'Authorization: Bearer test-token-123' http://localhost:8090/mcp"
curl -s -H 'Authorization: Bearer test-token-123' http://localhost:8090/mcp || echo "MCP request failed"
echo ""

echo ""
echo "🎯 Manual Claude.ai Test Instructions:"
echo "======================================"
echo "1. Keep this test server running"
echo "2. In Claude.ai, try to add MCP server:"
echo "   URL: http://localhost:8090/mcp"
echo "3. Claude should:"
echo "   ✅ Discover auth server via /.well-known/oauth-protected-resource"
echo "   ✅ Use resource indicators (RFC 8707)" 
echo "   ✅ Handle separate auth server pattern"
echo "4. If successful, Claude supports new spec!"
echo "5. If failed, Claude still uses old pattern"
echo ""
echo "🔍 Watch the server logs above for Claude's requests"
echo ""
echo "Press Ctrl+C to stop test server..."

# Wait for interrupt
trap "echo ''; echo '🛑 Stopping test server...'; kill $TEST_PID 2>/dev/null; exit 0" INT
wait $TEST_PID
