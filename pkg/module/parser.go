package module

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
)

// LoadFile reads, parses, and validates a JSON or YAML module manifest.
func LoadFile(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open module manifest %q: %w", path, err)
	}

	defer file.Close()

	manifest, err := Load(file)
	if err != nil {
		return nil, fmt.Errorf("load module manifest %q: %w", path, err)
	}

	return manifest, nil
}

// Load reads, parses, and validates a JSON or YAML module manifest.
// Load does not close the supplied reader.
func Load(reader io.Reader) (*Manifest, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read module manifest: %w", err)
	}

	return Parse(data)
}

// Parse parses and validates a JSON or YAML module manifest.
func Parse(data []byte) (*Manifest, error) {
	document, err := decodeDocument(data)
	if err != nil {
		return nil, err
	}

	if err := validateSchema(document); err != nil {
		return nil, err
	}

	normalized, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode normalized module manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(normalized, &manifest); err != nil {
		return nil, fmt.Errorf("decode validated module manifest: %w", err)
	}

	if err := validateSemantics(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func decodeDocument(data []byte) (any, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))

	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, newValidationErrors([]Violation{{
			Rule:    RuleDecode,
			Message: err.Error(),
		}})
	}

	var extra any
	err := decoder.Decode(&extra)
	if err == nil {
		return nil, newValidationErrors([]Violation{{
			Rule:    RuleDecode,
			Message: "manifest must contain exactly one document",
		}})
	}

	if err != io.EOF {
		return nil, newValidationErrors([]Violation{{
			Rule:    RuleDecode,
			Message: err.Error(),
		}})
	}

	normalized, err := json.Marshal(document)
	if err != nil {
		return nil, newValidationErrors([]Violation{{
			Rule:    RuleDecode,
			Message: "manifest mapping keys must be strings",
		}})
	}

	jsonDecoder := json.NewDecoder(bytes.NewReader(normalized))
	jsonDecoder.UseNumber()

	var value any
	if err := jsonDecoder.Decode(&value); err != nil {
		return nil, newValidationErrors([]Violation{{
			Rule:    RuleDecode,
			Message: err.Error(),
		}})
	}

	return value, nil
}
