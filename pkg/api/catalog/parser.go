package catalog

import (
	"fmt"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

// Parse strictly decodes and validates one API Catalog document.
func Parse(data []byte) (*Catalog, error) {
	document, err := jsondocument.Decode(data)
	if err != nil {
		return nil, validation.NewErrors(validation.ScopeAPICatalog, []validation.Violation{{Rule: validation.RuleDecode, Message: err.Error()}})
	}

	if err := validateDocument(document); err != nil {
		return nil, err
	}

	catalog, err := jsondocument.Into[Catalog](document)
	if err != nil {
		return nil, fmt.Errorf("decode schema-validated API Catalog: %w", err)
	}

	if err := validateSemantics(catalog); err != nil {
		return nil, err
	}

	return catalog, nil
}
