#!/bin/bash

# Test script for Phase 1 Debug System Implementation
set -e

echo "=== Testing Phase 1 Debug System Implementation ==="
echo ""

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || [[ ! -d "cmd/cowpilot" ]]; then
    echo "Error: Please run this script from the project root directory"
    exit 1
fi

echo "1. Updating Go dependencies..."
go mod tidy

echo ""
echo "2. Building main application..."
make build

echo ""
echo "3. Building debug proxy..."
make build-debug

echo ""
echo "4. Verifying binaries exist..."
if [[ -f "./bin/cowpilot" ]]; then
    echo "✓ Main application binary built successfully"
else
    echo "✗ Main application binary not found"
    exit 1
fi

if [[ -f "./bin/mcp-debug-proxy" ]]; then
    echo "✓ Debug proxy binary built successfully"
else
    echo "✗ Debug proxy binary not found"
    exit 1
fi

echo ""
echo "5. Testing debug proxy help..."
./bin/mcp-debug-proxy --help

echo ""
echo "6. Running basic syntax check on Go files..."
go vet ./internal/debug/...

echo ""
echo "7. Testing storage functionality..."
cat > test_debug.go << 'EOF'
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/vcto/mcp-adapters/internal/debug"
)

func main() {
    // Test storage creation
    storage, err := debug.NewConversationStorage("./test_debug.db")
    if err != nil {
        log.Fatalf("Failed to create storage: %v", err)
    }
    defer storage.Close()
    defer os.Remove("./test_debug.db")

    // Test logging a message
    err = storage.LogMessage("test-session", "inbound", "initialize", 
        map[string]interface{}{"test": "data"}, nil, nil, 10)
    if err != nil {
        log.Fatalf("Failed to log message: %v", err)
    }

    // Test getting stats
    stats, err := storage.GetStats()
    if err != nil {
        log.Fatalf("Failed to get stats: %v", err)
    }

    fmt.Printf("✓ Storage test passed. Stats: %+v\n", stats)
}
EOF

go run test_debug.go
rm test_debug.go

echo ""
echo "=== Phase 1 Implementation Test Complete ==="
echo ""
echo "✅ All components built and tested successfully!"
echo ""
echo "Next steps:"
echo "1. Run: make run-debug-proxy"
echo "2. Test with: npx @modelcontextprotocol/inspector http://localhost:8080"
echo "3. Check debug endpoints:"
echo "   - http://localhost:8080/debug/health"
echo "   - http://localhost:8080/debug/stats"
echo "   - http://localhost:8080/debug/sessions"
echo ""
