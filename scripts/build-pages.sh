#!/bin/sh

set -eu

if [ "$#" -gt 1 ]; then
	printf 'Usage: %s [destination]\n' "$0" >&2
	exit 2
fi

source_dir=schemas
destination=${1:-_site}

case "$destination" in
	'' | / | .)
		printf 'Unsafe Pages destination: %s\n' "$destination" >&2
		exit 2
		;;
esac

if [ ! -d "$source_dir" ]; then
	printf 'Schema directory not found: %s\n' "$source_dir" >&2
	exit 1
fi

if [ -e "$destination" ]; then
	printf 'Pages destination already exists: %s\n' "$destination" >&2
	exit 1
fi

schema_list=$(mktemp)
cleanup() {
	rm -f "$schema_list"
}
trap cleanup 0
trap 'cleanup; exit 1' 1 2 3 15

find "$source_dir" -type f -name '*.json' -print | sort > "$schema_list"
if [ ! -s "$schema_list" ]; then
	printf 'No JSON schemas found under: %s\n' "$source_dir" >&2
	exit 1
fi

mkdir -p "$destination"
index="$destination/index.html"

{
	printf '%s\n' '<!doctype html>'
	printf '%s\n' '<html lang="en">'
	printf '%s\n' '<head>'
	printf '%s\n' '  <meta charset="utf-8">'
	printf '%s\n' '  <meta name="viewport" content="width=device-width, initial-scale=1">'
	printf '%s\n' '  <title>Ferret Schemas</title>'
	printf '%s\n' '</head>'
	printf '%s\n' '<body>'
	printf '%s\n' '  <h1>Ferret Schemas</h1>'
	printf '%s\n' '  <p>Canonical JSON Schemas for the Ferret ecosystem.</p>'
	printf '%s\n' '  <ul>'

	while IFS= read -r file; do
		prefix="$source_dir/"
		relative=${file#"$prefix"}
		case "$relative" in
			*[!A-Za-z0-9._/-]*)
				printf 'Unsupported schema path: %s\n' "$file" >&2
				exit 1
				;;
		esac

		target="$destination/$relative"
		mkdir -p "$(dirname "$target")"
		cp "$file" "$target"
		printf '    <li><a href="/%s">%s</a></li>\n' "$relative" "$relative"
	done < "$schema_list"

	printf '%s\n' '  </ul>'
	printf '%s\n' '</body>'
	printf '%s\n' '</html>'
} > "$index"

rm -f "$schema_list"
trap - 0 1 2 3 15
