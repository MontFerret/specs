package api

import (
	"fmt"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

// ParseIndex strictly decodes and validates one Ferret API Reference Index document.
func ParseIndex(data []byte) (*Index, error) {
	document, err := jsondocument.Decode(data)
	if err != nil {
		return nil, validation.NewErrors(validation.ScopeRegistryArtifact, []validation.Violation{{Rule: validation.RuleDecode, Message: err.Error()}})
	}

	if err := validateIndexDocument(document); err != nil {
		return nil, err
	}

	index, err := jsondocument.Into[Index](document)
	if err != nil {
		return nil, fmt.Errorf("decode schema-validated Ferret API Reference Index: %w", err)
	}

	if err := validateIndexSemantics(index); err != nil {
		return nil, err
	}

	return index, nil
}
