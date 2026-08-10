package api_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/MontFerret/specs/pkg/api"
)

func TestParseDocumentation(t *testing.T) {
	for _, test := range []struct {
		name string
		text string
		want api.Documentation
	}{
		{
			name: "prose only",
			text: "Decode decodes content.",
			want: api.Documentation{Description: "Decode decodes content."},
		},
		{
			name: "structured metadata",
			text: "Decode decodes content.\n\nThe result uses Ferret-native values.\n\n" +
				"@param data {String|Binary} Source content.\n" +
				"@param options {Object?} Decode options.\n" +
				"@return {Object} Normalized document.\n" +
				"@throws {ParseError} Input is malformed.\n" +
				"@throws {LimitError} Input is too large.\n" +
				"@deprecated Use Parse instead.",
			want: api.Documentation{
				Description: "Decode decodes content.\n\nThe result uses Ferret-native values.",
				Parameters: []api.Parameter{
					{Name: "data", Type: "String|Binary", Description: "Source content."},
					{Name: "options", Type: "Object?", Description: "Decode options."},
				},
				Return: &api.Return{Type: "Object", Description: "Normalized document."},
				Throws: []api.Throw{
					{Error: "ParseError", Description: "Input is malformed."},
					{Error: "LimitError", Description: "Input is too large."},
				},
				Deprecated: "Use Parse instead.",
			},
		},
		{
			name: "collection type",
			text: "@param names {Array<String>} Field names.",
			want: api.Documentation{Parameters: []api.Parameter{{Name: "names", Type: "Array<String>", Description: "Field names."}}},
		},
		{
			name: "variadic type",
			text: "@param values {Any...} Values to concatenate.",
			want: api.Documentation{Parameters: []api.Parameter{{Name: "values", Type: "Any...", Description: "Values to concatenate."}}},
		},
		{
			name: "opaque type spacing",
			text: "@param data { String | Binary } Source content.",
			want: api.Documentation{Parameters: []api.Parameter{{Name: "data", Type: " String | Binary ", Description: "Source content."}}},
		},
		{
			name: "opaque nested braces",
			text: "@throws {Error<{code: String}>} Remote response is invalid.",
			want: api.Documentation{Throws: []api.Throw{{Error: "Error<{code: String}>", Description: "Remote response is invalid."}}},
		},
		{
			name: "unknown annotation remains prose",
			text: "Decode decodes content.\n@example Decode value.",
			want: api.Documentation{Description: "Decode decodes content.\n@example Decode value."},
		},
		{
			name: "indented supported annotation remains prose",
			text: "Decode decodes content.\n  @param data {String} Source content.",
			want: api.Documentation{Description: "Decode decodes content.\n  @param data {String} Source content."},
		},
		{
			name: "structured deprecation removes standard paragraph",
			text: "Decode decodes content.\n\nDeprecated: use Parse instead.\n\nMore details remain.\n@deprecated Use Parse instead.",
			want: api.Documentation{Description: "Decode decodes content.\n\nMore details remain.", Deprecated: "Use Parse instead."},
		},
		{
			name: "standard deprecation remains without annotation",
			text: "Decode decodes content.\n\nDeprecated: use Parse instead.",
			want: api.Documentation{Description: "Decode decodes content.\n\nDeprecated: use Parse instead."},
		},
		{
			name: "CRLF input",
			text: "Decode decodes content.\r\n\r\n@param data {Binary} Source content.",
			want: api.Documentation{
				Description: "Decode decodes content.",
				Parameters:  []api.Parameter{{Name: "data", Type: "Binary", Description: "Source content."}},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := api.ParseDocumentation(test.text)
			if err != nil {
				t.Fatalf("parse documentation: %v", err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("documentation = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseDocumentationRejectsMalformedAnnotations(t *testing.T) {
	for _, test := range []struct {
		name       string
		annotation string
		want       string
	}{
		{name: "empty parameter", annotation: "@param", want: "expected"},
		{name: "parameter name only", annotation: "@param data", want: "expected"},
		{name: "parameter without braces", annotation: "@param data String description", want: "opening brace"},
		{name: "type first parameter", annotation: "@param {String} data", want: "expected"},
		{name: "missing parameter description", annotation: "@param data {String}", want: "description"},
		{name: "JSDoc separator", annotation: "@param data {String} - Source content.", want: "JSDoc"},
		{name: "empty return", annotation: "@return", want: "expected"},
		{name: "return without braces", annotation: "@return Object value", want: "expected"},
		{name: "missing return description", annotation: "@return {Object}", want: "description"},
		{name: "empty throws", annotation: "@throws", want: "expected"},
		{name: "missing throws description", annotation: "@throws {ParseError}", want: "description"},
		{name: "empty deprecated", annotation: "@deprecated", want: "expected"},
		{name: "blank type", annotation: "@param data { } Source content.", want: "must not be blank"},
		{name: "missing closing brace", annotation: "@param data {String Source content.", want: "closing brace"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := api.ParseDocumentation("Prose.\n" + test.annotation)
			var documentationErr *api.DocumentationError
			if !errors.As(err, &documentationErr) {
				t.Fatalf("error = %T %v, want DocumentationError", err, err)
			}

			if documentationErr.Kind != api.DocumentationErrorMalformedAnnotation || documentationErr.Line != 2 || documentationErr.Annotation != test.annotation || !strings.Contains(documentationErr.Detail, test.want) {
				t.Fatalf("unexpected documentation error: %#v", documentationErr)
			}

			if !strings.Contains(documentationErr.Error(), test.annotation) {
				t.Fatalf("error %q omits annotation", documentationErr)
			}
		})
	}
}

func TestParseDocumentationRejectsDuplicateMetadata(t *testing.T) {
	for _, test := range []struct {
		name string
		text string
		kind api.DocumentationErrorKind
	}{
		{
			name: "parameter",
			text: "@param data {String} Source content.\n@param data {Binary} Binary content.",
			kind: api.DocumentationErrorDuplicateParameter,
		},
		{
			name: "return",
			text: "@return {String} Text.\n@return {Binary} Binary.",
			kind: api.DocumentationErrorMultipleReturns,
		},
		{
			name: "deprecated",
			text: "@deprecated Use Parse.\n@deprecated Use Read.",
			kind: api.DocumentationErrorMultipleDeprecations,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := api.ParseDocumentation(test.text)
			var documentationErr *api.DocumentationError
			if !errors.As(err, &documentationErr) || documentationErr.Kind != test.kind || documentationErr.Line != 2 {
				t.Fatalf("error = %#v, want kind %q on line 2", documentationErr, test.kind)
			}
		})
	}
}
