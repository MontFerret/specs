package artifact

import (
	"fmt"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

// ParseRootIndex parses and validates a Registry root index artifact.
func ParseRootIndex(data []byte) (*RootIndex, error) {
	return parseArtifact(data, RootSchemaV1, validateRootIndexSemantics)
}

// ParseModuleIndex parses and validates a Registry module index artifact.
func ParseModuleIndex(data []byte) (*ModuleIndex, error) {
	return parseArtifact(data, ModuleIndexSchemaV1, validateModuleIndexSemantics)
}

// ParseModuleDocument parses and validates a Registry module artifact.
func ParseModuleDocument(data []byte) (*ModuleDocument, error) {
	return parseArtifact(data, ModuleSchemaV1, validateModuleDocumentSemantics)
}

// ParseVersionDocument parses and validates a Registry version artifact.
func ParseVersionDocument(data []byte) (*VersionDocument, error) {
	return parseArtifact(data, VersionSchemaV1, validateVersionDocumentSemantics)
}

// ParseAPIReference parses and validates an API Reference artifact.
func ParseAPIReference(data []byte) (*APIReference, error) {
	return parseArtifact(data, APISchemaV1, validateAPIReferenceSemantics)
}

// ParseCategoryIndex parses and validates a Registry category index artifact.
func ParseCategoryIndex(data []byte) (*CategoryIndex, error) {
	return parseArtifact(data, CategoryIndexSchemaV1, validateCategoryIndexSemantics)
}

// ParseCategoryDocument parses and validates a Registry category artifact.
func ParseCategoryDocument(data []byte) (*CategoryDocument, error) {
	return parseArtifact(data, CategorySchemaV1, validateCategoryDocumentSemantics)
}

// ParsePluginIndex parses and validates a Registry plugin index artifact.
func ParsePluginIndex(data []byte) (*PluginIndex, error) {
	return parseArtifact(data, PluginIndexSchemaV1, noSemanticValidation[PluginIndex])
}

func parseArtifact[T any](data []byte, schemaID string, semantic func(*T) error) (*T, error) {
	document, err := jsondocument.Decode(data)
	if err != nil {
		return nil, validation.NewErrors(validation.ScopeRegistryArtifact, []validation.Violation{{Rule: validation.RuleDecode, Message: err.Error()}})
	}

	if err := validateDocument(schemaID, document); err != nil {
		return nil, err
	}

	value, err := jsondocument.Into[T](document)
	if err != nil {
		return nil, fmt.Errorf("decode schema-validated registry artifact: %w", err)
	}

	if err := semantic(value); err != nil {
		return nil, err
	}

	return value, nil
}
