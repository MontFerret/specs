package catalog

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/MontFerret/specs/pkg/validation"
)

const validDocument = `{
  "schemaVersion": 1,
  "id": "montferret/core",
  "version": "2.0.0-alpha.47",
  "categories": [
    {
      "id": "arrays",
      "title": "Arrays",
      "description": "Global functions for working with arrays.",
      "functions": ["append", "first", "flatten"]
    },
    {
      "id": "math",
      "title": "Math",
      "description": "Mathematical and numeric global functions.",
      "functions": ["abs", "acos"]
    }
  ],
  "namespaceRoots": ["io", "t"]
}`

func TestParseValidCatalog(t *testing.T) {
	catalog, err := Parse([]byte(validDocument))
	if err != nil {
		t.Fatal(err)
	}

	if catalog.ID != "montferret/core" || catalog.Version != "2.0.0-alpha.47" {
		t.Fatalf("catalog identity = %s@%s", catalog.ID, catalog.Version)
	}

	if got, want := catalog.Categories[0].Functions, []string{"append", "first", "flatten"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("array functions = %v, want %v", got, want)
	}
}

func TestParseNamespacedOnlyCatalog(t *testing.T) {
	document := `{
  "schemaVersion": 1,
  "id": "montferret/http",
  "version": "1.0.0",
  "categories": [],
  "namespaceRoots": ["http"]
}`

	catalog, err := Parse([]byte(document))
	if err != nil {
		t.Fatal(err)
	}

	if len(catalog.Categories) != 0 || !reflect.DeepEqual(catalog.NamespaceRoots, []string{"http"}) {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestParseRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "malformed", data: `{`, want: "decode"},
		{name: "trailing", data: validDocument + `{}`, want: "exactly one JSON value"},
		{name: "duplicate key", data: strings.Replace(validDocument, `"id": "montferret/core",`, `"id": "montferret/core", "id": "montferret/core",`, 1), want: "duplicate object key"},
		{name: "unknown root field", data: strings.Replace(validDocument, `"namespaceRoots":`, `"unknown": true, "namespaceRoots":`, 1), want: "additionalProperties"},
		{name: "unknown category field", data: strings.Replace(validDocument, `"functions": ["append"`, `"unknown": true, "functions": ["append"`, 1), want: "additionalProperties"},
		{name: "invalid category id", data: strings.Replace(validDocument, `"arrays"`, `"Arrays"`, 1), want: "pattern"},
		{name: "blank title", data: strings.Replace(validDocument, `"title": "Arrays"`, `"title": "  "`, 1), want: "pattern"},
		{name: "invalid function", data: strings.Replace(validDocument, `"append"`, `"append-value"`, 1), want: "pattern"},
		{name: "duplicate function in category", data: strings.Replace(validDocument, `"append", "first"`, `"append", "append", "first"`, 1), want: "uniqueItems"},
		{name: "unsorted functions", data: strings.Replace(validDocument, `"append", "first", "flatten"`, `"first", "append", "flatten"`, 1), want: "sorted"},
		{name: "duplicate function across categories", data: strings.Replace(validDocument, `"abs", "acos"`, `"abs", "append"`, 1), want: "already assigned"},
		{name: "duplicate category id", data: strings.Replace(validDocument, `"id": "math"`, `"id": "arrays"`, 1), want: "duplicated"},
		{name: "unsorted roots", data: strings.Replace(validDocument, `["io", "t"]`, `["t", "io"]`, 1), want: "namespace roots must be sorted"},
		{name: "duplicate roots", data: strings.Replace(validDocument, `["io", "t"]`, `["io", "io"]`, 1), want: "uniqueItems"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.data))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Parse error = %v, want error containing %q", err, test.want)
			}

			var validationErr *validation.Errors
			if !errors.As(err, &validationErr) {
				t.Fatalf("Parse error type = %T, want *validation.Errors", err)
			}
		})
	}
}

func TestParseRejectsUnsupportedVersion(t *testing.T) {
	_, err := Parse([]byte(strings.Replace(validDocument, `"schemaVersion": 1`, `"schemaVersion": 2`, 1)))
	var unsupported *UnsupportedVersionError
	if !errors.As(err, &unsupported) || unsupported.Version != 2 {
		t.Fatalf("Parse error = %v, want unsupported version 2", err)
	}
}

func TestValidateAndRoundTrip(t *testing.T) {
	catalog, err := Parse([]byte(validDocument))
	if err != nil {
		t.Fatal(err)
	}

	if err := Validate(catalog); err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(parsed, catalog) {
		t.Fatalf("round trip = %#v, want %#v", parsed, catalog)
	}
}

func TestValidateNil(t *testing.T) {
	err := Validate(nil)
	if err == nil || !strings.Contains(err.Error(), "API catalog validation failed") {
		t.Fatalf("Validate(nil) error = %v", err)
	}
}
