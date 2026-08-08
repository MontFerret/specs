package registry

import (
	"fmt"
	"io"
	"os"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
)

// ParseModuleManifest parses and validates one JSON Registry Module Manifest.
func ParseModuleManifest(data []byte) (*ModuleManifest, error) {
	document, err := decodeDocument(data)
	if err != nil {
		return nil, err
	}

	if err := validateModuleDocument(document); err != nil {
		return nil, err
	}

	manifest, err := decodeValidated[ModuleManifest](document)
	if err != nil {
		return nil, fmt.Errorf("decode validated registry module manifest: %w", err)
	}

	if err := validateModuleSemantics(manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// LoadModuleManifest reads, parses, and validates one JSON Registry Module Manifest.
// It does not close reader.
func LoadModuleManifest(reader io.Reader) (*ModuleManifest, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read registry module manifest: %w", err)
	}

	return ParseModuleManifest(data)
}

// LoadModuleManifestFile opens, parses, and validates one JSON Registry Module Manifest.
func LoadModuleManifestFile(filePath string) (*ModuleManifest, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open registry module manifest %q: %w", filePath, err)
	}

	defer file.Close()

	manifest, err := LoadModuleManifest(file)
	if err != nil {
		return nil, fmt.Errorf("load registry module manifest %q: %w", filePath, err)
	}

	return manifest, nil
}

// ParseVersionRecord parses and validates one JSON Registry Version Record.
func ParseVersionRecord(data []byte) (*VersionRecord, error) {
	document, err := decodeDocument(data)
	if err != nil {
		return nil, err
	}

	if err := validateVersionDocument(document); err != nil {
		return nil, err
	}

	record, err := decodeValidated[VersionRecord](document)
	if err != nil {
		return nil, fmt.Errorf("decode validated registry version record: %w", err)
	}

	if err := validateVersionSemantics(record); err != nil {
		return nil, err
	}

	return record, nil
}

// LoadVersionRecord reads, parses, and validates one JSON Registry Version Record.
// It does not close reader.
func LoadVersionRecord(reader io.Reader) (*VersionRecord, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read registry version record: %w", err)
	}

	return ParseVersionRecord(data)
}

// LoadVersionRecordFile opens, parses, and validates one JSON Registry Version Record.
func LoadVersionRecordFile(filePath string) (*VersionRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open registry version record %q: %w", filePath, err)
	}

	defer file.Close()

	record, err := LoadVersionRecord(file)
	if err != nil {
		return nil, fmt.Errorf("load registry version record %q: %w", filePath, err)
	}

	return record, nil
}

func decodeDocument(data []byte) (any, error) {
	value, err := jsondocument.Decode(data)
	if err != nil {
		return nil, validation.NewErrors(validation.ScopeRegistry, []validation.Violation{{Rule: validation.RuleDecode, Message: err.Error()}})
	}

	return value, nil
}

func decodeValidated[T any](document any) (*T, error) {
	return jsondocument.Into[T](document)
}
