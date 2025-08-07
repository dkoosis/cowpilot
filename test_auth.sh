#!/bin/bash

echo "Testing OAuth fixes..."
echo "====================="
echo ""

# Set test environment
export GO_TEST=1

echo "Running auth package tests only..."
go test -v -race -timeout 10s ./internal/auth/...

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ All auth tests passed!"
    echo ""
    echo "You can now run the full test suite:"
    echo "  make test"
else
    echo ""
    echo "❌ Tests failed. Check the output above."
fi
