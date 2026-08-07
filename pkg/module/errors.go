package module

import (
	"fmt"
)

// Rule identifies a stable validation rule.
type Rule string

const (
	// RuleDecode identifies malformed JSON or YAML input.
	RuleDecode Rule = "decode"
	// RuleSchema identifies a schema failure without a more specific keyword.
	RuleSchema Rule = "schema"
	// RuleSemVer identifies an invalid strict semantic version.
	RuleSemVer Rule = "semver"
	// RuleVersionRange identifies an invalid npm-compatible version range.
	RuleVersionRange Rule = "version-range"
	// RuleSPDX identifies an invalid SPDX license expression.
	RuleSPDX Rule = "spdx"
	// RuleDuplicate identifies a duplicate dependency or export.
	RuleDuplicate Rule = "duplicate"
	// RuleNamespaceScope identifies an export outside the module namespace.
	RuleNamespaceScope Rule = "namespace-scope"
	// RuleSelfDependency identifies a direct dependency on the declaring module.
	RuleSelfDependency Rule = "self-dependency"
	// RuleRepositoryDirectory identifies a non-normalized repository directory.
	RuleRepositoryDirectory Rule = "repository-directory"
)

type (
	// Violation describes one validation failure.
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

// Error summarizes the contained validation violations.
func (e *ValidationErrors) Error() string {
	if e == nil || len(e.Violations) == 0 {
		return "manifest validation failed"
	}

	if len(e.Violations) == 1 {
		violation := e.Violations[0]
		location := violation.Path
		if location == "" {
			location = "document root"
		}

		return fmt.Sprintf("manifest validation failed at %s (%s): %s", location, violation.Rule, violation.Message)
	}

	return fmt.Sprintf("manifest validation failed with %d violations", len(e.Violations))
}
