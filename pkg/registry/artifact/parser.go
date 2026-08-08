package artifact

import (
	"fmt"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

func ParseRootIndex(data []byte) (*RootIndex, error) {
	return parseArtifact(data, RootSchemaV1, validateRootIndexSemantics)
}

func ParseModuleIndex(data []byte) (*ModuleIndex, error) {
	return parseArtifact(data, ModuleIndexSchemaV1, validateModuleIndexSemantics)
}

func ParseModuleDocument(data []byte) (*ModuleDocument, error) {
	return parseArtifact(data, ModuleSchemaV1, validateModuleDocumentSemantics)
}

func ParseVersionDocument(data []byte) (*VersionDocument, error) {
	return parseArtifact(data, VersionSchemaV1, validateVersionDocumentSemantics)
}

func ParseCategoryIndex(data []byte) (*CategoryIndex, error) {
	return parseArtifact(data, CategoryIndexSchemaV1, validateCategoryIndexSemantics)
}

func ParseCategoryDocument(data []byte) (*CategoryDocument, error) {
	return parseArtifact(data, CategorySchemaV1, validateCategoryDocumentSemantics)
}

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
