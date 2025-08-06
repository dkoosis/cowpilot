#!/bin/bash
set -e

cd /Users/vcto/Projects/cowpilot

echo "Cleaning module cache..."
go clean -modcache

echo "Downloading dependencies..."
go mod download

echo "Tidying modules..."
go mod tidy

echo "Running tests..."
make test

echo "✓ Dependencies fixed"
