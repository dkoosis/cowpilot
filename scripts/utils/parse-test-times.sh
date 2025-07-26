#!/bin/bash
# Parse Go test JSON output for timing information

if [ ! -f test-results.json ]; then
    echo "No test-results.json found"
    exit 0
fi

echo "Individual test times:"
echo "====================="

# Parse JSON and extract test times
jq -r 'select(.Action == "pass" and .Test != null) | "\(.Test): \(.Elapsed)s"' test-results.json 2>/dev/null | sort -k2 -nr

echo ""
echo "Failed tests:"
jq -r 'select(.Action == "fail" and .Test != null) | "\(.Test): FAILED (\(.Elapsed)s)"' test-results.json 2>/dev/null

# Also update .gitignore to exclude test artifacts
if ! grep -q "test-results.json" ../.gitignore 2>/dev/null; then
    echo "test-results.json" >> ../../.gitignore
    echo ".test-times.log" >> ../../.gitignore
fi
