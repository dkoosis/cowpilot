#!/bin/bash
set -e
# Test basic functionality of the everything server

echo "Testing cowpilot everything server..."

# Build the server
echo "Building server..."
cd /Users/vcto/Projects/cowpilot
go build -o bin/cowpilot-test cmd/cowpilot/main.go

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"

# Start the server in background
echo "Starting server..."
./bin/cowpilot-test &
SERVER_PID=$!

# Give it a moment to start
sleep 2

# Test with npx inspector
echo "Testing with MCP inspector..."
npx @modelcontextprotocol/inspector ./bin/cowpilot-test &
INSPECTOR_PID=$!

echo "Server PID: $SERVER_PID"
echo "Inspector PID: $INSPECTOR_PID"
echo ""
echo "Test the following capabilities:"
echo "1. Tools tab - try various tools like echo, add, string_operation"
echo "2. Try list_resources and read_resource tools"
echo "3. Try list_prompts and get_prompt tools"
echo "4. Try get_test_image for image content"
echo "5. Try get_resource_content for embedded resources"
echo ""
echo "Press Ctrl+C to stop both processes"

# Wait for interrupt
trap "kill $SERVER_PID $INSPECTOR_PID 2>/dev/null; exit" INT
wait
