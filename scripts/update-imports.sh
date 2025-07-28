#!/bin/bash

# Update imports in all Go files
find . -name "*.go" -type f -not -path "./.git/*" -exec sed -i '' \
    -e 's|github.com/vcto/mcp-adapters/internal/validator|github.com/vcto/mcp-adapters/internal/protocol/validation|g' \
    -e 's|github.com/vcto/mcp-adapters/internal/contracts|github.com/vcto/mcp-adapters/internal/protocol/specs|g' \
    -e 's|github.com/vcto/mcp-adapters/internal/testutil|github.com/vcto/mcp-adapters/internal/testing|g' \
    {} \;

echo "Imports updated"
