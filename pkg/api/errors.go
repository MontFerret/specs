package api

import (
	"fmt"
	"strings"
)

// DocumentationErrorKind identifies a stable structured-documentation failure category.
type DocumentationErrorKind string

const (
	// DocumentationErrorMalformedAnnotation identifies invalid supported-tag grammar.
	DocumentationErrorMalformedAnnotation DocumentationErrorKind = "malformed-annotation"
	// DocumentationErrorDuplicateParameter identifies a repeated parameter name.
	DocumentationErrorDuplicateParameter DocumentationErrorKind = "duplicate-parameter"
	// DocumentationErrorMultipleReturns identifies more than one @return annotation.
	DocumentationErrorMultipleReturns DocumentationErrorKind = "multiple-returns"
	// DocumentationErrorMultipleDeprecations identifies more than one @deprecated annotation.
	DocumentationErrorMultipleDeprecations DocumentationErrorKind = "multiple-deprecations"
)

// DocumentationError reports one invalid structured annotation.
// Line is one-based within the normalized documentation body.
type DocumentationError struct {
	// Kind identifies the stable documentation failure category.
	Kind DocumentationErrorKind
	// Line is one-based within the normalized documentation body.
	Line int
	// Annotation is the original malformed or conflicting annotation line.
	Annotation string
	// Detail describes the expected grammar or semantic conflict.
	Detail string
}

func (e *DocumentationError) Error() string {
	if e == nil {
		return "invalid Ferret API documentation"
	}

	tag, _, _ := strings.Cut(e.Annotation, " ")
	if tag == "" {
		tag = "annotation"
	}

	return fmt.Sprintf("malformed %s %q: %s", tag, e.Annotation, e.Detail)
}

// UnsupportedVersionError reports a positive API Reference schema version other than v1.
type UnsupportedVersionError struct {
	// Version is the unsupported positive schema version.
	Version int
}

func (e *UnsupportedVersionError) Error() string {
	return fmt.Sprintf("unsupported Ferret API Reference schema version %d", e.Version)
}
