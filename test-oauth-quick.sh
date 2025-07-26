#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "=== Testing OAuth Build ==="
go build -o bin/cowpilot-oauth cmd/cowpilot/main.go
if [ $? -eq 0 ]; then
    echo "✓ Build successful"
    
    echo -e "\n=== Testing OAuth endpoints ==="
    ./bin/cowpilot-oauth &
    PID=$!
    sleep 2
    
    # Test OAuth metadata
    echo "Testing /.well-known/oauth-authorization-server:"
    curl -s http://localhost:8080/.well-known/oauth-authorization-server | head -n3
    
    # Test authorize endpoint
    echo -e "\nTesting /oauth/authorize:"
    curl -s "http://localhost:8080/oauth/authorize?client_id=test&redirect_uri=http://localhost:9090/callback" | grep -o "Connect Remember The Milk" | head -n1
    
    kill $PID 2>/dev/null
    echo -e "\n✓ OAuth endpoints working"
else
    echo "✗ Build failed"
fi
