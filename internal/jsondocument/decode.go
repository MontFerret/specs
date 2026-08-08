// Package jsondocument strictly decodes one JSON document for specs packages.
package jsondocument

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Decode decodes exactly one JSON value and rejects duplicate object keys.
func Decode(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	value, err := decodeValue(decoder)
	if err != nil {
		return nil, err
	}

	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("document must contain exactly one JSON value")
		}

		return nil, err
	}

	return value, nil
}

// Into converts a decoded JSON document into a typed value.
func Into[T any](document any) (*T, error) {
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
