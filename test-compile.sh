#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "Testing OAuth compilation..."
go build -o /tmp/cowpilot-oauth-test cmd/cowpilot/main.go 2>&1
