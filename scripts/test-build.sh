#!/bin/bash
set -e

echo "=== Testing mcp adapters ==="
go mod download
make test
make build
echo "=== Ready to deploy! ==="
