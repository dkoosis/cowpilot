#!/bin/bash

# Project Rename Migration Script
# Renames cowpilot to a new project name with minimal disruption

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
OLD_NAME="cowpilot"
OLD_MODULE="github.com/vcto/cowpilot"
DEFAULT_NEW_NAME="mcp-adapters"
BACKUP_SUFFIX=".cowpilot-backup"

# Parse command line arguments
NEW_NAME="${1:-$DEFAULT_NEW_NAME}"
DRY_RUN="${2:-false}"

if [[ "$NEW_NAME" == "--help" ]] || [[ "$NEW_NAME" == "-h" ]]; then
    echo "Usage: $0 [new-name] [dry-run]"
    echo ""
    echo "Examples:"
    echo "  $0 mcp-adapters          # Rename to mcp-adapters"
    echo "  $0 mcp-platform true     # Dry run for mcp-platform"
    echo "  $0 prism                 # Rename to prism"
    echo ""
    echo "Default new name: $DEFAULT_NEW_NAME"
    exit 0
fi

NEW_MODULE="github.com/vcto/$NEW_NAME"

echo -e "${BLUE}🚀 Project Rename Migration${NC}"
echo -e "From: ${RED}$OLD_NAME${NC} → To: ${GREEN}$NEW_NAME${NC}"
echo -e "Module: ${RED}$OLD_MODULE${NC} → ${GREEN}$NEW_MODULE${NC}"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}⚠️  DRY RUN MODE - No changes will be made${NC}"
    echo ""
fi

# Function to log actions
log_action() {
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}[DRY RUN]${NC} $1"
    else
        echo -e "${GREEN}[DONE]${NC} $1"
    fi
}

# Function to safely update file
update_file() {
    local file="$1"
    local description="$2"
    
    if [[ ! -f "$file" ]]; then
        return 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}Would update:${NC} $file - $description"
        return 0
    fi
    
    # Create backup
    cp "$file" "${file}${BACKUP_SUFFIX}"
    
    # Update the file
    sed -i.tmp \
        -e "s|$OLD_MODULE|$NEW_MODULE|g" \
        -e "s|github\.com/vcto/$OLD_NAME|github.com/vcto/$NEW_NAME|g" \
        -e "s|cowpilot\.fly\.dev|$NEW_NAME.fly.dev|g" \
        -e "s|FLY_APP_NAME=cowpilot|FLY_APP_NAME=$NEW_NAME|g" \
        -e "s|app.*=.*cowpilot|app = \"$NEW_NAME\"|g" \
        -e "s|BINARY_NAME=cowpilot|BINARY_NAME=$NEW_NAME|g" \
        -e "s|cowpilot-spektrix|$NEW_NAME-spektrix|g" \
        -e "s|Cowpilot|$(echo $NEW_NAME | sed 's/-/ /g' | sed 's/\b\w/\U&/g')|g" \
        "$file"
    
    # Remove temp file
    rm -f "${file}.tmp"
    
    log_action "Updated $file - $description"
}

# Function to rename files/directories
rename_item() {
    local old_path="$1"
    local new_path="$2"
    local description="$3"
    
    if [[ ! -e "$old_path" ]]; then
        return 0
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "${YELLOW}Would rename:${NC} $old_path → $new_path ($description)"
        return 0
    fi
    
    mv "$old_path" "$new_path"
    log_action "Renamed $old_path → $new_path ($description)"
}

echo -e "${BLUE}📋 Phase 1: Go Module and Imports${NC}"

# Update go.mod
update_file "go.mod" "module name"

# Find and update all Go files
if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}Would update all Go files with import statements${NC}"
else
    find . -name "*.go" -not -path "./.git/*" -not -path "./bin/*" | while read -r file; do
        update_file "$file" "Go imports and references"
    done
fi

echo ""
echo -e "${BLUE}📋 Phase 2: Configuration Files${NC}"

# Update key configuration files
update_file "Dockerfile" "Docker configuration"
update_file "fly.toml" "Fly.io deployment config"
update_file "Makefile" "build configuration"
update_file "Makefile.test" "test configuration"

echo ""
echo -e "${BLUE}📋 Phase 3: Documentation${NC}"

# Update documentation
update_file "README.md" "main documentation"
update_file "PROJECT_OVERVIEW_FOR_CLAUDE.md" "Claude context"

# Update docs directory
find docs -name "*.md" -o -name "*.yaml" -o -name "*.yml" | while read -r file; do
    update_file "$file" "documentation file"
done

echo ""
echo -e "${BLUE}📋 Phase 4: Scripts and Tests${NC}"

# Update scripts
find scripts -name "*.sh" | while read -r file; do
    update_file "$file" "script file"
done

# Update test files
find tests -name "*.go" -o -name "*.sh" -o -name "*.md" | while read -r file; do
    update_file "$file" "test file"
done

echo ""
echo -e "${BLUE}📋 Phase 5: Binary and Build Artifacts${NC}"

# Rename the main binary if it exists
rename_item "cowpilot" "$NEW_NAME" "main binary"
rename_item "bin/cowpilot" "bin/$NEW_NAME" "binary in bin directory"

echo ""
echo -e "${BLUE}📋 Phase 6: Cleanup${NC}"

if [[ "$DRY_RUN" != "true" ]]; then
    # Run go mod tidy to clean up
    go mod tidy
    log_action "Ran go mod tidy"
    
    # Update go.sum
    if [[ -f "go.sum" ]]; then
        log_action "go.sum will be updated on next build"
    fi
fi

echo ""
echo -e "${GREEN}✅ Migration Summary${NC}"
echo ""
echo "Old project name: $OLD_NAME"
echo "New project name: $NEW_NAME"
echo "Old module: $OLD_MODULE"
echo "New module: $NEW_MODULE"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}This was a dry run. To execute the changes, run:${NC}"
    echo -e "${BLUE}$0 $NEW_NAME false${NC}"
    echo ""
    echo -e "${YELLOW}Or to proceed with the default name (mcp-adapters):${NC}"
    echo -e "${BLUE}$0${NC}"
else
    echo -e "${GREEN}🎉 Migration completed successfully!${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. Test the build: make test"
    echo "2. Update GitHub repository name (if desired)"
    echo "3. Update Fly.io app name: fly apps rename cowpilot $NEW_NAME"
    echo "4. Update any external references or documentation"
    echo ""
    echo -e "${YELLOW}💾 Backup files created with suffix: $BACKUP_SUFFIX${NC}"
    echo "   Remove them when you're confident everything works:"
    echo "   find . -name '*$BACKUP_SUFFIX' -delete"
    echo ""
    echo -e "${BLUE}🔍 Review these files manually:${NC}"
    echo "   - Any hardcoded URLs or references you might have missed"
    echo "   - Third-party integrations or configs"
    echo "   - API documentation or external references"
fi

echo ""
echo -e "${BLUE}🏗️  Project Structure After Migration:${NC}"
echo "   cmd/"
echo "   ├── demo-server/     (consider renaming from 'everything')"
echo "   ├── rtm-server/      (keep - fits the cow theme!)"
echo "   ├── spektrix-server/ (keep as-is)"
echo "   └── mcp-debug-proxy/ (keep as-is)"
echo ""
echo -e "${YELLOW}💡 Optional next step: Rename cmd/everything → cmd/demo-server${NC}"
echo "   This can be done as a separate, smaller change."