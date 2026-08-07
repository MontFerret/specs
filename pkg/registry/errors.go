package registry

import (
	"fmt"
	"sort"
	"strings"
)

type (
	// Rule identifies a stable validation rule.
	Rule string

	// Violation describes one registry validation failure.
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
)

const (
	RuleDecode        Rule = "decode"
	RuleSchema        Rule = "schema"
	RuleSemVer        Rule = "semver"
	RuleRepositoryURL Rule = "repository-url"
	RuleSourcePath    Rule = "source-path"
	RuleTag           Rule = "git-tag"
)

func (e *ValidationErrors) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "registry validation failed"
	}

	if len(e.Violations) == 1 {
		violation := e.Violations[0]
		location := violation.Path
		if location == "" {
			location = "document root"
		}

		return fmt.Sprintf("registry validation failed at %s (%s): %s", location, violation.Rule, violation.Message)
	}

	return fmt.Sprintf("registry validation failed with %d violations", len(e.Violations))
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

	for i, part := range parts {
		part = strings.ReplaceAll(part, "~", "~0")
		escaped[i] = strings.ReplaceAll(part, "/", "~1")
	}

	return "/" + strings.Join(escaped, "/")
}
