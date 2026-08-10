package main

import (
	"fmt"
	"io"
)

func writeRootUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `ferret-spec validates Ferret specification documents.

Usage:
  ferret-spec validate module [--format text|json] FILE...`)
}

func writeValidateUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Usage:
  ferret-spec validate module [--format text|json] FILE...`)
}

func writeModuleUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, `Usage:
  ferret-spec validate module [--format text|json] FILE...

Arguments:
  FILE    JSON or YAML module manifest; use - to read stdin

Options:
  --format text|json    Output format (default: text)`)
}
