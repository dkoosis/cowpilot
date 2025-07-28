#!/bin/bash
cd /Users/vcto/Projects/cowpilot
echo "🧹 Removing backup files..."
find . -name "*.cowpilot-backup" -type f -delete
echo "✅ Backup files removed"
echo ""
echo "📊 Cleanup summary:"
echo "Removed all files ending with .cowpilot-backup"
