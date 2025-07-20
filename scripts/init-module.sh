#!/bin/bash
set -e

echo "=== Initializing Go module ==="
cd /Users/vcto/Projects/cowpilot

# Clean and reinitialize
rm -f go.sum
go mod tidy

echo "=== Dependencies downloaded ==="
