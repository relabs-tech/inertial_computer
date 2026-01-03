#!/bin/bash
# Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
# SPDX-License-Identifier: MIT

# Add license headers to all project files

GO_HEADER="// Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
// SPDX-License-Identifier: MIT
// See LICENSE file for full license text

"

HTML_HEADER="<!--
  Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
  SPDX-License-Identifier: MIT
-->

"

SCRIPT_HEADER="# Copyright (c) 2026 Daniel Alarcon Rubio / Relabs Tech
# SPDX-License-Identifier: MIT

"

# Add to Go files (skip if already has copyright)
echo "Processing Go files..."
find . -name "*.go" -type f ! -path "./vendor/*" ! -path "./.git/*" | while read -r file; do
    if ! grep -q "Copyright (c)" "$file"; then
        echo "  Adding header to $file"
        echo "$GO_HEADER" | cat - "$file" > temp && mv temp "$file"
    else
        echo "  Skipping $file (already has copyright)"
    fi
done

# Add to HTML files
echo "Processing HTML files..."
find . -name "*.html" -type f ! -path "./.git/*" | while read -r file; do
    if ! grep -q "Copyright (c)" "$file"; then
        echo "  Adding header to $file"
        # Insert after <!DOCTYPE> if present, otherwise at beginning
        if head -1 "$file" | grep -q "<!DOCTYPE"; then
            sed -i "1a\\$HTML_HEADER" "$file"
        else
            echo "$HTML_HEADER" | cat - "$file" > temp && mv temp "$file"
        fi
    else
        echo "  Skipping $file (already has copyright)"
    fi
done

# Add to shell scripts
echo "Processing shell scripts..."
find . -name "*.sh" -type f ! -path "./.git/*" ! -name "add_license_headers.sh" | while read -r file; do
    if ! grep -q "Copyright (c)" "$file"; then
        echo "  Adding header to $file"
        # Preserve shebang if present
        if head -1 "$file" | grep -q "^#!"; then
            shebang=$(head -1 "$file")
            tail -n +2 "$file" > temp
            echo "$shebang" > "$file"
            echo "$SCRIPT_HEADER" >> "$file"
            cat temp >> "$file"
            rm temp
        else
            echo "$SCRIPT_HEADER" | cat - "$file" > temp && mv temp "$file"
        fi
    else
        echo "  Skipping $file (already has copyright)"
    fi
done

echo ""
echo "License headers added successfully!"
echo "Files modified:"
echo "  - Go source files (*.go)"
echo "  - HTML files (*.html)"
echo "  - Shell scripts (*.sh)"
