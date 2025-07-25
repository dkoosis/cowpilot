#!/bin/bash
# Clean up dead code directories

echo "Removing dead code directories..."

# Remove unused transport implementation
if [ -d "internal/transport" ]; then
    echo "Removing internal/transport..."
    rm -rf internal/transport
fi

# Remove unused mcp implementation
if [ -d "internal/mcp" ]; then
    echo "Removing internal/mcp..."
    rm -rf internal/mcp
fi

echo "Dead code cleanup complete!"
