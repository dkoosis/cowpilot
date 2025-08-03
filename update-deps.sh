#!/bin/bash
# Update dependencies

echo "Running go mod tidy..."
cd /Users/vcto/Projects/cowpilot
go mod tidy

echo "Dependencies updated"
