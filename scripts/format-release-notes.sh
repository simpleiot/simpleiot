#!/bin/bash
# Format release notes for GitHub: join wrapped lines into paragraphs and point
# documentation links at the docs site.
#
# GitHub renders every newline in a release body as a line break, so the 80
# column wrapping used in CHANGELOG.md arrives as ragged text. Links in the
# changelog are relative to the repository root, which resolves on the
# repository pages but not on the release page.
#
# Usage: format-release-notes.sh <file>   (edits the file in place)
#        format-release-notes.sh          (reads stdin, writes stdout)

set -euo pipefail

DOCS_URL="https://docs.simpleiot.org"

format() {
	# Join wrapped lines. Headings, list items, table rows, block quotes, and
	# horizontal rules each start a new line; anything that follows one is a
	# continuation of it. Fenced code blocks are left alone.
	awk '
    function flush() {
        if (buf != "") print buf
        buf = ""
    }
    { sub(/\r$/, "") }
    /^[ \t]*(```|~~~)/ {
        flush()
        fence = !fence
        print
        next
    }
    fence { print; next }
    /^[ \t]*$/ { flush(); print ""; next }
    # a row of a table stands on its own line
    /^[ \t]*\|/ { flush(); print; next }
    /^(#{1,6} |[ \t]*([-*+]|[0-9]+\.) |> |(---|___|\*\*\*)[ \t]*$)/ {
        flush()
        buf = $0
        next
    }
    {
        sub(/^[ \t]+/, "")
        buf = (buf == "" ? $0 : buf " " $0)
    }
    END { flush() }
    ' |
		# Documentation links are relative to the repository root, and the book
		# is built with its source root there, so docs/user/sync.md becomes
		# <docs site>/docs/user/sync.html with any anchor carried over.
		sed -E "s%\]\((docs/[^)#]*)\.md(#[^)]*)?\)%](${DOCS_URL}/\1.html\2)%g" |
		# collapse the blank lines that joining can leave behind, and drop
		# the ones at the start and the end
		cat -s |
		awk '
    NF == 0 { if (started) held++; next }
    {
        while (held-- > 0) print ""
        held = 0
        started = 1
        print
    }
    '
}

if [ $# -eq 0 ]; then
	format
	exit 0
fi

FILE=$1

if [ ! -f "$FILE" ]; then
	echo "$0: no such file: $FILE" >&2
	exit 1
fi

TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
format <"$FILE" >"$TMP"
cat "$TMP" >"$FILE"
