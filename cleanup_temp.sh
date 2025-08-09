#!/bin/bash

# Clean up unnecessary files created during troubleshooting

echo "Cleaning up temporary files..."

# Remove duplicate MD files (keeping only the essential ones)
files_to_remove=(
    "RTM_OAUTH_STATUS.md"
    "RTM_LOCAL_VS_PRODUCTION.md"
    "RTM_QUICK_GUIDE.md"
    "RTM_SOLUTION.md"
)

for file in "${files_to_remove[@]}"; do
    if [ -f "$file" ]; then
        rm "$file"
        echo "  Removed: $file"
    fi
done

echo "Cleanup complete"
