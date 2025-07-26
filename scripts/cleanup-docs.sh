#!/bin/bash
# Documentation cleanup script

echo "🧹 Cleaning up documentation sprawl..."

# Create archive directory for historical files
mkdir -p docs/archive

# 1. Archive temporary/historical files
echo "📦 Archiving temporary files..."
mv PROJECT_CLEANUP_SUMMARY.md docs/archive/ 2>/dev/null || echo "  - PROJECT_CLEANUP_SUMMARY.md already moved"
mv TEST_REORGANIZATION_SUMMARY.md docs/archive/ 2>/dev/null || echo "  - TEST_REORGANIZATION_SUMMARY.md already moved"
mv docs/TRANSPORT_STATUS.md docs/archive/ 2>/dev/null || echo "  - TRANSPORT_STATUS.md already moved"
mv docs/MCP_INSPECTOR_SESSION_ISSUE.md docs/archive/ 2>/dev/null || echo "  - MCP_INSPECTOR_SESSION_ISSUE.md already moved"
mv docs/sessions/2025-01-20-handoff.md docs/archive/ 2>/dev/null || echo "  - Old handoff doc moved"

# 2. Remove AI-generated review files
echo "🗑️  Removing temporary review files..."
rm -f docs/reviews/*.md
rmdir docs/reviews 2>/dev/null || true

# 3. Remove test scenario artifacts
echo "🗑️  Removing test scenario artifacts..."
rm -f tests/scenarios/ENHANCED_SUMMARY.md
rm -f tests/scenarios/FILE_INVENTORY.md
rm -f tests/scenarios/IMPLEMENTATION_REVIEW.md
rm -f tests/scenarios/IMPLEMENTATION_SUMMARY.md
rm -f tests/scenarios/RTFM_CORRECTION.md

# 4. Remove other temporary files
echo "🗑️  Removing other temporary files..."
rm -f debug-tools-test.go  # This appears to be a stray test file
rm -f cowpilot  # Binary in wrong location
rm -f cleanup_dead_code.sh  # One-time script
rm -f docs/test-formatting.md  # Will merge content
rm -f docs/README.md  # Will merge with root README
rm -f PROJECT_OVERVIEW_FOR_CLAUDE.md  # Will merge with README

# 5. Clean up misc files
rm -f docs/.DS_Store
rm -f docs/mcp-go-main-tags
rm -f docs/tags
rm -f docs/tree.txt
rm -f .gitignore.tmp
rm -f Makefile.test

echo "✅ Cleanup complete!"
echo ""
echo "📋 Next steps:"
echo "  1. Merge PROJECT_OVERVIEW_FOR_CLAUDE.md content into README.md"
echo "  2. Merge docs/README.md content into root README.md"
echo "  3. Merge docs/test-formatting.md into docs/testing-guide.md"
echo "  4. Merge docs/sessions/quick-start-next.md into README.md or contributing.md"
echo "  5. Merge tests/scenarios/DEBUG_GUIDE.md into docs/"
echo "  6. Merge tests/scenarios/TESTING_GUIDE.md into docs/testing-guide.md"
echo "  7. Update .gitignore to exclude temporary files"
