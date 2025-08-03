#!/bin/bash

# Final cleanup script

cd /Users/vcto/Projects/cowpilot/docs

echo "Starting cleanup..."

# Delete archive and old folders
rm -rf archive/
rm -rf design/
rm -rf reviews/ 
rm -rf sessions/

# Delete OAuth implementation docs (keep only rtm-authentication-flow as reference)
if [ -d oauth ]; then
    # Keep rtm-authentication-flow.md as reference
    [ -f oauth/rtm-authentication-flow.md ] && mv oauth/rtm-authentication-flow.md reference/
    rm -rf oauth/
fi

# Delete MCP folder (content moved to ADRs)
rm -rf mcp/

# Delete contributing (not needed for our sessions)
rm -rf contributing/

# Move active files from backlog to root
if [ -d backlog ]; then
    [ -f backlog/todo.md ] && mv backlog/todo.md TODO.md
    [ -f backlog/rtm-enhancements-backlog.yaml ] && mv backlog/rtm-enhancements-backlog.yaml RTM_BACKLOG.yaml
    rm -f backlog/history.md  # Delete history - it's in git
    rmdir backlog/
fi

# Delete .DS_Store files
find . -name ".DS_Store" -delete

echo "✅ Cleanup complete!"
echo ""
echo "Remaining structure:"
ls -la
