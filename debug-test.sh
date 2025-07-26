#!/bin/bash
cd /Users/vcto/Projects/cowpilot
go build -o ./bin/cowpilot ./cmd/cowpilot
FLY_APP_NAME=local-test ./bin/cowpilot &
SERVER_PID=$!
sleep 3
MCP_SERVER_URL="http://localhost:8080/" go run debug-tools-test.go
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null
