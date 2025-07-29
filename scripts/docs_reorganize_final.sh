#!/bin/bash
# Complete documentation reorganization with standardized naming

set -e

echo "📚 Starting documentation reorganization..."

# 1. Create archive directory
mkdir -p docs/archive

# 2. Archive old/temporary files
echo "📦 Archiving temporary files..."
mv PROJECT_CLEANUP_SUMMARY.md docs/archive/ 2>/dev/null || true
mv TEST_REORGANIZATION_SUMMARY.md docs/archive/ 2>/dev/null || true
mv docs/TRANSPORT_STATUS.md docs/archive/transport-status.md 2>/dev/null || true
mv docs/MCP_INSPECTOR_SESSION_ISSUE.md docs/archive/mcp-inspector-session-issue.md 2>/dev/null || true
mv docs/sessions/2025-01-20-handoff.md docs/archive/ 2>/dev/null || true

# 3. Remove duplicate/temporary files
echo "🗑️  Removing temporary files..."
rm -rf docs/reviews/
rm -rf docs/sessions/
rm -f tests/scenarios/ENHANCED_SUMMARY.md
rm -f tests/scenarios/FILE_INVENTORY.md
rm -f tests/scenarios/IMPLEMENTATION_REVIEW.md
rm -f tests/scenarios/IMPLEMENTATION_SUMMARY.md
rm -f tests/scenarios/RTFM_CORRECTION.md
rm -f tests/scenarios/TESTING_GUIDE.md
rm -f tests/scenarios/DEBUG_GUIDE.md
rm -f debug-tools-test.go
rm -f cowpilot
rm -f cleanup_dead_code.sh
rm -f docs/.DS_Store
rm -f docs/mcp-go-main-tags
rm -f docs/tags
rm -f docs/tree.txt
rm -f .gitignore.tmp
rm -f Makefile.test
rm -f PROJECT_OVERVIEW_FOR_CLAUDE.md
rm -f docs/README.md
rm -f docs/test-formatting.md
rm -f docs/sessions/quick-start-next.md 2>/dev/null || true

# 4. Standardize file names (UPPERCASE.md -> lowercase.md)
echo "📝 Standardizing file names..."
mv docs/KNOWN-ISSUES.md docs/known-issues.md 2>/dev/null || true
mv docs/ROADMAP.md docs/roadmap.md 2>/dev/null || true
mv docs/TODO.md docs/todo.md 2>/dev/null || true
mv docs/TESTING_STRATEGY.md docs/testing-strategy.md 2>/dev/null || true
# Note: STATE.yaml stays uppercase as it's machine-readable

# 5. Replace files with updated versions
echo "📝 Installing updated documentation..."
mv README.new.md README.md
mv docs/testing-guide.new.md docs/testing-guide.md

# 6. Update .gitignore
echo "🔧 Updating .gitignore..."
if ! grep -q "# Temporary files" .gitignore; then
    echo "" >> .gitignore
    echo "# Temporary files" >> .gitignore
    echo "*.tmp" >> .gitignore
    echo ".DS_Store" >> .gitignore
    echo "cowpilot" >> .gitignore
    echo "debug-tools-test.go" >> .gitignore
    echo "docs/archive/" >> .gitignore
fi

# 7. Final cleanup
echo "🧹 Final cleanup..."
find . -name ".DS_Store" -delete 2>/dev/null || true
rm -f scripts/cleanup-docs.sh
rm -f scripts/cleanup-old-tests.sh
rm -f scripts/reorganize-docs.sh

echo ""
echo "✅ Documentation reorganization complete!"
echo ""
echo "Summary of changes:"
echo "  • Consolidated README.md with all project info"
echo "  • Updated testing-guide.md with complete test documentation" 
echo "  • Standardized file naming (lowercase with hyphens)"
echo "  • Archived old/temporary files to docs/archive/"
echo "  • Removed duplicate and AI-generated files"
echo "  • Updated .gitignore"
echo ""
echo "📋 Documentation structure is now clean and organized!"
