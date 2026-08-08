package artifact

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// Rule identifies a stable artifact validation rule.
	Rule string

	// Violation describes one artifact validation failure.
	// Path is an RFC 6901 JSON Pointer; an empty path identifies the document root.
	Violation struct {
		Path    string `json:"path"`
		Rule    Rule   `json:"rule"`
		Message string `json:"message"`
	}

	// ValidationErrors contains every violation found in one validation phase.
	ValidationErrors struct {
		Violations []Violation `json:"violations"`
	}

	// UnsupportedVersionError reports a positive artifact schema version other than v1.
	UnsupportedVersionError struct {
		Version int
	}
)

const (
	RuleDecode      Rule = "decode"
	RuleSchema      Rule = "schema"
	RuleDuplicate   Rule = "duplicate"
	RuleIdentity    Rule = "identity"
	RuleReference   Rule = "reference"
	RulePackagePath Rule = "package-path"
	RuleSource      Rule = "source"
	RuleTimestamp   Rule = "timestamp"
)

func (e *ValidationErrors) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "registry artifact validation failed"
	}

	if len(e.Violations) == 1 {
		violation := e.Violations[0]
		location := violation.Path

		if location == "" {
			location = "document root"
		}

		return fmt.Sprintf("registry artifact validation failed at %s (%s): %s", location, violation.Rule, violation.Message)
	}

	return fmt.Sprintf("registry artifact validation failed with %d violations", len(e.Violations))
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("unsupported registry artifact schema version %d", e.Version)
}

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
	for index, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		escaped[index] = strings.ReplaceAll(part, "/", "~1")
	}

	return "/" + strings.Join(escaped, "/")
}
