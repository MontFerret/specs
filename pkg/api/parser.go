package api

import (
	"fmt"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

// Parse strictly decodes and validates one Ferret API Reference document.
func Parse(data []byte) (*Reference, error) {
	document, err := jsondocument.Decode(data)
	if err != nil {
		return nil, validation.NewErrors(validation.ScopeRegistryArtifact, []validation.Violation{{Rule: validation.RuleDecode, Message: err.Error()}})
	}

	if err := validateDocument(document); err != nil {
		return nil, err
	}

	reference, err := jsondocument.Into[Reference](document)
	if err != nil {
		return nil, fmt.Errorf("decode schema-validated Ferret API Reference: %w", err)
	}

	if err := validateSemantics(reference); err != nil {
		return nil, err
	}

	return reference, nil
}
