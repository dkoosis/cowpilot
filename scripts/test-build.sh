#!/bin/bash
set -e

echo "=== Testing Cowpilot ==="
go mod download
make test
make build
echo "=== Ready to deploy! ==="
