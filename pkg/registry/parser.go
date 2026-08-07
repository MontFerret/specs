package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := decodeValue(decoder)
	if err != nil {
		return nil, newValidationErrors([]Violation{{Rule: RuleDecode, Message: err.Error()}})
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("document must contain exactly one JSON value")
		}
		return nil, newValidationErrors([]Violation{{Rule: RuleDecode, Message: err.Error()}})
	}
	return value, nil
}

func decodeValue(decoder *json.Decoder) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delimiter {
	case '{':
		value := make(map[string]any)
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := keyToken.(string)
			if !ok {
				return nil, fmt.Errorf("object key must be a string")
			}
			if _, exists := value[key]; exists {
				return nil, fmt.Errorf("duplicate object key %q", key)
			}
			item, err := decodeValue(decoder)
			if err != nil {
				return nil, err
			}
			value[key] = item
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return nil, fmt.Errorf("unterminated object")
		}
		return value, nil
	case '[':
		value := make([]any, 0)
		for decoder.More() {
			item, err := decodeValue(decoder)
			if err != nil {
				return nil, err
			}
			value = append(value, item)
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return nil, fmt.Errorf("unterminated array")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unexpected delimiter %q", delimiter)
	}
}

func decodeValidated[T any](document any) (*T, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return &value, nil
}
