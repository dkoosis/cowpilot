#!/bin/bash

# Clean up duplicate/unnecessary files in project root

echo "Cleaning up temporary and duplicate files..."

# Remove duplicate MD files (RTM docs are now in README)
rm -f RTM_OAUTH_STATUS.md RTM_LOCAL_VS_PRODUCTION.md RTM_QUICK_GUIDE.md RTM_SOLUTION.md
echo "  ✓ Removed duplicate RTM documentation"

# Remove old/backup files
rm -f Makefile.rtm.backup Old.Makefile
echo "  ✓ Removed backup Makefiles"

# Remove duplicate scripts in root (moved to scripts/)
rm -f fix-rtm-now.sh fix_deps.sh fix_docker.sh
rm -f run_prod_diagnostic.sh run_test.sh
rm -f test-build-verbose.sh test-build.sh test-new-structure.sh
rm -f test_auth.sh test_build.sh test_runner.sh
rm -f set-rtm-credentials.sh update-deps.sh
rm -f run_test.go
echo "  ✓ Removed duplicate scripts from root"

# Remove other temp files
rm -f rtm_deploy_check.mk makefile_help.mk SCRIPT_REFERENCE.md
rm -f tree-tst.txt tags
rm -f cleanup_temp.sh  # Remove this script itself
echo "  ✓ Removed temporary files"

echo ""
echo "Project cleaned up! Key files remaining:"
echo "  - Makefile (main build system)"
echo "  - README.md (includes RTM connection info)"
echo "  - fly-rtm.toml (RTM deployment config)"
echo "  - fly-core-tmp.toml (test deployment config)"
echo "  - scripts/ (all diagnostic scripts)"
echo "  - tests/ (all test files including new production tests)"
