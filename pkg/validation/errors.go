// Package validation contains shared structured validation errors.
package validation

import (
	"fmt"
	"sort"
)

type (
	// Scope identifies the subject of a validation failure.
	Scope string

	// Rule identifies a stable validation rule.
	Rule string

	// Violation describes one validation failure.
	// Path is an RFC 6901 JSON Pointer; an empty path identifies the document root.
	Violation struct {
		Path    string `json:"path"`
		Rule    Rule   `json:"rule"`
		Message string `json:"message"`
	}

	// Errors contains every violation found in one validation phase.
	Errors struct {
		Violations []Violation `json:"violations"`
		scope      Scope
	}
)

const (
	// ScopeManifest identifies module manifest validation.
	ScopeManifest Scope = "manifest"
	// ScopeRegistry identifies Registry source document validation.
	ScopeRegistry Scope = "registry"
	// ScopeRegistryArtifact identifies Registry distribution artifact validation.
	ScopeRegistryArtifact Scope = "registry artifact"
)

const (
	RuleDecode              Rule = "decode"
	RuleSchema              Rule = "schema"
	RuleSemVer              Rule = "semver"
	RuleVersionRange        Rule = "version-range"
	RuleSPDX                Rule = "spdx"
	RuleDuplicate           Rule = "duplicate"
	RuleNamespaceScope      Rule = "namespace-scope"
	RuleSelfDependency      Rule = "self-dependency"
	RuleRepositoryDirectory Rule = "repository-directory"
	RuleRepositoryURL       Rule = "repository-url"
	RuleSourcePath          Rule = "source-path"
	RuleTag                 Rule = "git-tag"
	RuleIdentity            Rule = "identity"
	RuleReference           Rule = "reference"
	RulePackagePath         Rule = "package-path"
	RuleSource              Rule = "source"
	RuleTimestamp           Rule = "timestamp"
)

// Error summarizes the contained validation violations.
func (e *Errors) Error() string {
	prefix := "validation failed"
	if e != nil && e.scope != "" {
		prefix = string(e.scope) + " validation failed"
	}

	if e == nil || len(e.Violations) == 0 {
		return prefix
	}

	if len(e.Violations) == 1 {
		violation := e.Violations[0]
		location := violation.Path
		if location == "" {
			location = "document root"
		}

		return fmt.Sprintf("%s at %s (%s): %s", prefix, location, violation.Rule, violation.Message)
	}

	return fmt.Sprintf("%s with %d violations", prefix, len(e.Violations))
}

// NewErrors returns a sorted aggregate error for violations.
// It returns nil when violations is empty.
func NewErrors(scope Scope, violations []Violation) error {
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

	return &Errors{scope: scope, Violations: violations}
}
