package module

import (
	"sort"
	"strings"
)

func newValidationErrors(violations []Violation) error {
	if len(violations) == 0 {
		return nil
	}

	sort.SliceStable(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}

		if violations[i].Rule != violations[j].Rule {
			return violations[i].Rule < violations[j].Rule
		}

		return violations[i].Message < violations[j].Message
	})

	return &ValidationErrors{Violations: violations}
}

func jsonPointer(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	escaped := make([]string, len(parts))

	for i, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		escaped[i] = strings.ReplaceAll(part, "/", "~1")
	}

	return "/" + strings.Join(escaped, "/")
}
