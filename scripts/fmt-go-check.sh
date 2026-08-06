#!/bin/sh

set -eu

GOFMT=${GOFMT:-gofmt}

files=$(find . -type f -name '*.go' -not -path './vendor/*' -exec "$GOFMT" -l {} +)
if [ -n "$files" ]; then
	printf '%s\n' "Go files need formatting:" "$files"
	exit 1
fi
