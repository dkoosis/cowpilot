#!/bin/bash
set -e

echo "Fixing Docker build issue..."
cd /Users/vcto/Projects/cowpilot

echo "Running go mod tidy..."
go mod tidy

echo "Attempting deployment..."
make deploy-rtm

echo "✓ Deployment should work now"
