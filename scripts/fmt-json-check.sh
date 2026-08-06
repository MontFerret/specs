#!/bin/sh

set -eu

JQ=${JQ:-jq}
JSON_INDENT=${JSON_INDENT:-2}

if [ "$#" -eq 0 ]; then
	set -- schemas testdata
fi

command -v "$JQ" >/dev/null 2>&1 || {
	printf '%s\n' "JSON formatter not found: $JQ" >&2
	exit 127
}

find "$@" -type f -name '*.json' -print | sort | (
	status=0
	tmp=

	cleanup() {
		if [ -n "$tmp" ]; then
			rm -f "$tmp"
		fi
	}

	trap cleanup 0
	trap 'cleanup; exit 1' 1 2 3 15

	while IFS= read -r file; do
		tmp=$(mktemp)
		if ! "$JQ" --indent "$JSON_INDENT" . "$file" > "$tmp"; then
			printf 'Invalid JSON: %s\n' "$file" >&2
			status=1
		elif ! diff -u "$file" "$tmp"; then
			status=1
		fi
		rm -f "$tmp"
		tmp=
	done

	exit "$status"
)
