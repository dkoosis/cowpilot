#!/bin/bash
cd /Users/vcto/Projects/cowpilot
go vet ./...
echo "Exit code: $?"
