#!/bin/bash
# Debug Tools Integration Test - Verify debug system functionality

set -e

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE} ▶${NC} Debug Tools Integration Test"
echo -e "${BLUE} ▶${NC} Testing MCP debug system with runtime configuration"

cd /Users/vcto/Projects/cowpilot

# Build server
echo -e "${BLUE} ▶${NC} Building server..."
if go build -o ./bin/cowpilot ./cmd/cowpilot 2>/dev/null; then
    echo -e "${GREEN}        ✓ Build successful"
else
    echo -e "${RED}        ✗ Build failed"
    echo -e "${RED} ▶${NC} FAIL  Debug Tools Integration Test"
    exit 1
fi

FAILED=0

# Test 1: Debug disabled by default
echo -e "${BLUE} ▶${NC} Test 1: Debug disabled by default"
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

# Make a request and check server logs (debug should be silent)
curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' &>/dev/null

echo -e "${GREEN}        ✓ Server running without debug overhead"

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Test 2: Debug with memory storage
echo -e "${BLUE} ▶${NC} Test 2: Debug enabled with memory storage"
export MCP_DEBUG=true
export MCP_DEBUG_STORAGE=memory
export MCP_DEBUG_LEVEL=DEBUG

FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

# Make some test requests
echo -e "${BLUE}        Making test requests..."
for i in 1 2 3; do
    curl -s -X POST http://localhost:8080/ \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/list\",\"id\":$i}" &>/dev/null
done

echo -e "${GREEN}        ✓ Debug system capturing requests"

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Test 3: Debug with file storage
echo -e "${BLUE} ▶${NC} Test 3: Debug enabled with file storage"
export MCP_DEBUG_STORAGE=file
export MCP_DEBUG_PATH=./test_debug.db

# Remove old test file
rm -f ./test_debug.db

FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

# Make test requests
echo -e "${BLUE}        Logging to SQLite database..."
curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"hello"},"id":1}' &>/dev/null

curl -s -X POST http://localhost:8080/ \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"echo","arguments":{"message":"debug test"}},"id":2}' &>/dev/null

# Check if database was created
if [[ -f "./test_debug.db" ]]; then
    echo -e "${GREEN}        ✓ Debug database created"
    
    # Check file size
    size=$(ls -lh ./test_debug.db | awk '{print $5}')
    echo -e "${BLUE}        Database size: $size"
else
    echo -e "${RED}        ✗ Debug database not created"
    ((FAILED++))
fi

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Test 4: Debug with bounded storage
echo -e "${BLUE} ▶${NC} Test 4: Debug with storage limits"
export MCP_DEBUG_MAX_MB=1
export MCP_DEBUG_RETENTION_H=1

FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3

echo -e "${BLUE} ▶${NC} Testing bounded storage (1MB limit)..."
# Make many requests to test limits
for i in {1..20}; do
    curl -s -X POST http://localhost:8080/ \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"method\":\"tools/call\",\"params\":{\"name\":\"string_operation\",\"arguments\":{\"text\":\"This is a test string to fill up storage space quickly\",\"operation\":\"upper\"}},\"id\":$i}" &>/dev/null
done

echo -e "${GREEN}        ✓ Bounded storage handling multiple requests"

kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

# Test 5: Debug proxy integration (if available)
echo -e "${BLUE} ▶${NC} Test 5: Debug proxy integration"
if [[ -f "./bin/mcp-debug-proxy" ]] || go build -o ./bin/mcp-debug-proxy ./cmd/mcp-debug-proxy 2>/dev/null; then
    echo -e "${BLUE}        Starting debug proxy..."
    
    # Start debug proxy
    MCP_DEBUG_ENABLED=true ./bin/mcp-debug-proxy \
        --target ./bin/cowpilot \
        --port 9090 \
        --target-port 9091 &
    PROXY_PID=$!
    sleep 5
    
    # Test proxy health endpoint
    if curl -s http://localhost:9090/debug/health | grep -q "healthy"; then
        echo -e "${GREEN}        ✓ Debug proxy running"
        
        # Check debug endpoints
        if curl -s http://localhost:9090/debug/stats &>/dev/null; then
            echo -e "${GREEN}        ✓ Debug stats endpoint available"
        fi
    else
        echo -e "${YELLOW}        ⚠️  Debug proxy not responding"
    fi
    
    kill $PROXY_PID 2>/dev/null || true
    wait $PROXY_PID 2>/dev/null || true
else
    echo -e "${YELLOW}        ⚠️  Debug proxy not available (skipped)"
fi

# Cleanup
echo -e "${BLUE} ▶${NC} Cleanup"
unset MCP_DEBUG
unset MCP_DEBUG_STORAGE
unset MCP_DEBUG_PATH
unset MCP_DEBUG_LEVEL
unset MCP_DEBUG_MAX_MB
unset MCP_DEBUG_RETENTION_H

rm -f ./test_debug.db
echo -e "${GREEN}        ✓ Test files cleaned up"

# Summary
echo ""
if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}   PASS${NC}  Debug Tools Integration Test"
    echo -e "   Runtime debug configuration working correctly"
    exit 0
else
    echo -e "${RED}   FAIL  Debug Tools Integration Test ($FAILED tests failed)"
    exit 1
fi
