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

find "$@" -type f -name '*.json' -print | sort | while IFS= read -r file; do
	tmp="${file}.tmp.$$"
	trap 'rm -f "$tmp"' 0 1 2 3 15

	if ! "$JQ" --indent "$JSON_INDENT" . "$file" > "$tmp"; then
		printf 'Failed to format JSON: %s\n' "$file" >&2
		exit 1
	fi

	if cmp -s "$file" "$tmp"; then
		rm -f "$tmp"
	else
		mv "$tmp" "$file"
	fi

	trap - 0 1 2 3 15
done
