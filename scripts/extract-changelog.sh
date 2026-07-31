#!/bin/bash
# Extract the changelog section for a specific version from CHANGELOG.md
# Usage: extract-changelog.sh <version>

VERSION=$1

if [ -z "$VERSION" ]; then
	echo "Usage: $0 <version>"
	exit 1
fi

# Remove 'v' prefix if present
VERSION=${VERSION#v}

# Extract the section for this version
awk -v version="$VERSION" '
/^## \[/ {
    if (found) exit
    if ($0 ~ "\\[" version "\\]") {
        found=1
        next
    }
}
found && /^## \[/ { exit }
found { print }
' CHANGELOG.md
