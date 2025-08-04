#!/bin/bash
cd /Users/vcto/Projects/cowpilot
go test -v ./internal/rtm -run TestFullOAuthFlow 2>&1
