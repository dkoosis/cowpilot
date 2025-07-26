#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "=== OAuth Implementation Compilation Test ==="
echo "Building cowpilot with OAuth callback server..."

# Run go build and capture output
OUTPUT=$(go build -o /tmp/cowpilot-oauth-test cmd/cowpilot/main.go 2>&1)
BUILD_EXIT=$?

if [ $BUILD_EXIT -eq 0 ]; then
    echo "✓ Build successful!"
    rm -f /tmp/cowpilot-oauth-test
else
    echo "✗ Build failed with errors:"
    echo "$OUTPUT"
    echo ""
    echo "=== Checking specific issues ==="
    
    # Check for common issues
    if echo "$OUTPUT" | grep -q "auth.Middleware"; then
        echo "Missing: auth.Middleware function"
    fi
    
    if echo "$OUTPUT" | grep -q "cannot find package"; then
        echo "Missing package imports detected"
    fi
fi

echo ""
echo "=== Next Steps ==="
echo "1. Fix any compilation errors"
echo "2. Run integration tests"
echo "3. Test with Claude.ai"
