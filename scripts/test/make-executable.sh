#!/bin/bash
# Make all test scripts executable

cd /Users/vcto/Projects/cowpilot/scripts/test

echo "Making test scripts executable..."
chmod +x *.sh
echo "Done!"

ls -la *.sh
