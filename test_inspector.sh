#!/bin/bash
# Test Inspector CLI flags

echo "Testing Inspector CLI mode..."

# Try with explicit non-interactive environment
export CI=true
export NONINTERACTIVE=true

# Test different Inspector invocations
echo "1. Testing with --help flag..."
npx @modelcontextprotocol/inspector --help

echo -e "\n2. Testing SSE connection directly..."
npx @modelcontextprotocol/inspector test sse https://cowpilot.fly.dev/

echo -e "\n3. Testing with explicit transport..."
npx @modelcontextprotocol/inspector https://cowpilot.fly.dev/ --transport sse
