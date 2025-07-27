#!/bin/bash

# Update imports in all Go files
find . -name "*.go" -type f -not -path "./.git/*" -exec sed -i '' \
    -e 's|github.com/vcto/cowpilot/internal/validator|github.com/vcto/cowpilot/internal/protocol/validation|g' \
    -e 's|github.com/vcto/cowpilot/internal/contracts|github.com/vcto/cowpilot/internal/protocol/specs|g' \
    -e 's|github.com/vcto/cowpilot/internal/testutil|github.com/vcto/cowpilot/internal/testing|g' \
    {} \;

echo "Imports updated"
