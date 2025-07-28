#!/bin/bash
cd /Users/vcto/Projects/cowpilot

echo "🧹 Cleaning up migration artifacts..."
echo ""

# Remove backup files
echo "Removing backup files..."
find . -name "*.cowpilot-backup" -type f -delete
echo "✅ Backup files removed"

# Remove migration scripts
echo ""
echo "Removing migration scripts..."
rm -f migrate-project.sh
rm -f run-migration.sh
rm -f cleanup-backups.sh
echo "✅ Migration scripts removed"

# Remove this cleanup script itself (last)
echo ""
echo "Removing final cleanup script..."
rm -f final-cleanup.sh
echo "✅ All migration artifacts cleaned up"

echo ""
echo "🎉 Project migration complete and cleaned up!"
echo ""
echo "Your project is now: mcp-adapters"
echo "Next steps:"
echo "  1. Test the build: make test"
echo "  2. Commit changes: git add . && git commit -m 'Rename project from cowpilot to mcp-adapters'"
