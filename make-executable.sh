#!/bin/bash
# Make all scripts executable

cd /Users/vcto/Projects/cowpilot

chmod +x build-check.sh
chmod +x check-fmt.sh
chmod +x run-gofmt.sh
chmod +x test-build.sh
chmod +x test-longrunning.sh
chmod +x update-deps.sh

echo "✓ All scripts are now executable"
