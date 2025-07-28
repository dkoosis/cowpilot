#!/bin/bash
# Manual testing examples using MCP Inspector CLI

echo "MCP Inspector CLI - Manual Testing Examples"
echo "=========================================="
echo ""
echo "These commands can be run directly to test the cowpilot server:"
echo ""

SERVER="https://mcp-adapters.fly.dev/"

echo "1. List available tools:"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --method tools/list"
echo ""

echo "2. Call the hello tool:"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --method tools/call --tool-name hello"
echo ""

echo "3. List resources (if implemented):"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --method resources/list"
echo ""

echo "4. List prompts (if implemented):"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --method prompts/list"
echo ""

echo "5. Test with HTTP streaming transport (if supported):"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --transport http --method tools/list"
echo ""

echo "6. Call tool with arguments (example):"
echo "   npx @modelcontextprotocol/inspector --cli $SERVER --method tools/call --tool-name mytool --tool-arg key=value --tool-arg another=value2"
echo ""

echo "7. Test against local server:"
echo "   npx @modelcontextprotocol/inspector --cli http://localhost:8080/ --method tools/list"
echo ""

echo "Note: SSE transport is used by default for HTTP(S) URLs"
echo "Note: For stdio transport, use: npx @modelcontextprotocol/inspector --cli node path/to/server.js"
