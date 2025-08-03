#!/bin/bash
# Check and fix Go formatting

echo "Running gofmt..."
cd /Users/vcto/Projects/cowpilot

# Find files that need formatting
unformatted=$(gofmt -l .)

if [ -z "$unformatted" ]; then
    echo "✓ All Go files are properly formatted"
else
    echo "Files that need formatting:"
    echo "$unformatted"
    echo ""
    echo "Running gofmt -w . to fix formatting..."
    gofmt -w .
    echo "✓ Formatting fixed"
fi
