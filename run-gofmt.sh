#!/bin/bash
# Run gofmt and show results

cd /Users/vcto/Projects/cowpilot

echo "Running gofmt -w . to format all Go files..."
gofmt -w .

echo ""
echo "Checking formatting results..."

# Check if any files still need formatting
unformatted=$(gofmt -l .)

if [ -z "$unformatted" ]; then
    echo "✓ All Go files are now properly formatted"
    exit 0
else
    echo "❌ These files still have formatting issues:"
    echo "$unformatted"
    exit 1
fi
