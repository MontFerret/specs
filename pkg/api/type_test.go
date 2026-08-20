package api_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/MontFerret/specs/pkg/api"
	"github.com/MontFerret/specs/pkg/validation"
)

func TestParseType(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  api.Type
	}{
		{name: "named", input: "String", want: namedTypeValue("String")},
		{name: "trimmed named", input: "  Page  ", want: namedTypeValue("Page")},
		{name: "compact union", input: "String|Array", want: unionTypeValue(namedTypeValue("String"), namedTypeValue("Array"))},
		{name: "spaced union", input: " String | Array | Object ", want: unionTypeValue(namedTypeValue("String"), namedTypeValue("Array"), namedTypeValue("Object"))},
		{name: "list", input: "[Int]", want: listTypeValue(namedTypeValue("Int"))},
		{name: "list union", input: "[Int | Float]", want: listTypeValue(unionTypeValue(namedTypeValue("Int"), namedTypeValue("Float")))},
		{name: "nested list", input: "[[String]]", want: listTypeValue(listTypeValue(namedTypeValue("String")))},
		{name: "mixed list union", input: "[String | [Int]]", want: listTypeValue(unionTypeValue(namedTypeValue("String"), listTypeValue(namedTypeValue("Int"))))},
		{name: "top level list union", input: "[String] | None", want: unionTypeValue(listTypeValue(namedTypeValue("String")), namedTypeValue("None"))},
		{name: "legacy optional atom", input: "Object?", want: namedTypeValue("Object?")},
		{name: "legacy variadic atom", input: "Any...", want: namedTypeValue("Any...")},
		{name: "legacy generic atom", input: "Iterator<T>", want: namedTypeValue("Iterator<T>")},
		{name: "legacy generic union atom", input: "Object<{value: String | None}>", want: namedTypeValue("Object<{value: String | None}>")},
		{name: "legacy postfix list atom", input: "T[]", want: namedTypeValue("T[]")},
		{name: "deduplicated union", input: "A | B | A", want: unionTypeValue(namedTypeValue("A"), namedTypeValue("B"))},
		{name: "collapsed union", input: "A | A", want: namedTypeValue("A")},
		{name: "nested union is not flattened", input: "A | [A | B]", want: unionTypeValue(namedTypeValue("A"), listTypeValue(unionTypeValue(namedTypeValue("A"), namedTypeValue("B"))))},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := api.ParseType(test.input)
			if err != nil {
				t.Fatalf("ParseType(%q): %v", test.input, err)
			}

			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParseType(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}

func TestParseTypeRejectsMalformedExpressions(t *testing.T) {
	for _, input := range []string{
		"",
		" ",
		"[]",
		"A |",
		"| B",
		"A || B",
		"[A",
		"A]",
		"[A]B",
		"(A",
		"A>",
		"[A|]",
		"A\nB",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := api.ParseType(input); err == nil {
				t.Fatalf("ParseType(%q) unexpectedly succeeded", input)
			}
		})
	}
}

func TestReferenceParseNormalizesUnionTypes(t *testing.T) {
	reference := validReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[1]
	signature.Parameters[0].Type = &api.Type{
		Kind: api.TypeKindList,
		Element: &api.Type{
			Kind: api.TypeKindUnion,
			Types: []api.Type{
				namedTypeValue("String"),
				namedTypeValue("Binary"),
				namedTypeValue("String"),
			},
		},
	}
	signature.Return = &api.Return{
		Type: &api.Type{
			Kind:  api.TypeKindUnion,
			Types: []api.Type{namedTypeValue("Any"), namedTypeValue("Any")},
		},
		Description: "Result value.",
	}

	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := api.Parse(data)
	if err != nil {
		t.Fatalf("parse reference: %v", err)
	}

	want := listTypeValue(unionTypeValue(namedTypeValue("String"), namedTypeValue("Binary")))
	got := *parsed.Namespaces[1].Functions[0].Signatures[1].Parameters[0].Type
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized type = %#v, want %#v", got, want)
	}

	if got := parsed.Namespaces[1].Functions[0].Signatures[1].Return.Type; !reflect.DeepEqual(got, namedType("Any")) {
		t.Fatalf("collapsed return type = %#v, want named Any", got)
	}
}

func TestReferenceValidateDoesNotNormalizeDuplicateUnions(t *testing.T) {
	reference := validReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[0]
	signature.Parameters[0].Type = &api.Type{
		Kind:  api.TypeKindUnion,
		Types: []api.Type{namedTypeValue("Any"), namedTypeValue("Any")},
	}
	signature.Parameters[0].Description = "Input value."
	want := *signature.Parameters[0].Type

	if err := api.Validate(reference); err != nil {
		t.Fatalf("validate duplicate union: %v", err)
	}

	if !reflect.DeepEqual(*signature.Parameters[0].Type, want) {
		t.Fatalf("Validate mutated type to %#v, want %#v", *signature.Parameters[0].Type, want)
	}
}

func TestReferenceRejectsMalformedTypeVariants(t *testing.T) {
	for _, test := range []struct {
		name  string
		value *api.Type
	}{
		{name: "unknown kind", value: &api.Type{Kind: "unknown", Name: "Any"}},
		{name: "blank named type", value: &api.Type{Kind: api.TypeKindNamed, Name: " \t"}},
		{name: "named cross field", value: &api.Type{Kind: api.TypeKindNamed, Name: "Any", Types: []api.Type{namedTypeValue("String"), namedTypeValue("Binary")}}},
		{name: "short union", value: &api.Type{Kind: api.TypeKindUnion, Types: []api.Type{namedTypeValue("Any")}}},
		{name: "malformed union child", value: &api.Type{Kind: api.TypeKindUnion, Types: []api.Type{namedTypeValue("Any"), {Kind: api.TypeKindNamed}}}},
		{name: "union cross field", value: &api.Type{Kind: api.TypeKindUnion, Name: "Any", Types: []api.Type{namedTypeValue("String"), namedTypeValue("Binary")}}},
		{name: "missing list element", value: &api.Type{Kind: api.TypeKindList}},
		{name: "list cross field", value: &api.Type{Kind: api.TypeKindList, Name: "Any", Element: namedType("String")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := validReference()
			signature := &reference.Namespaces[1].Functions[0].Signatures[1]
			signature.Parameters[0].Type = test.value
			requireValidationErrors(t, api.Validate(reference))
		})
	}
}

func TestReferenceRejectsLegacyStringType(t *testing.T) {
	data := []byte(`{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[{"name":"ARCHIVE","functions":[{"name":"READ","signatures":[{"parameters":[{"name":"value","type":"Any","description":"Input."}]}]}]}]}`)
	_, err := api.Parse(data)
	requireRule(t, requireValidationErrors(t, err), validation.Rule("type"))
}

func TestTypesDoNotCreateSignatureOverloads(t *testing.T) {
	reference := validReference()
	function := &reference.Namespaces[1].Functions[0]
	function.Signatures[0].Parameters[0].Type = namedType("String")
	function.Signatures[0].Parameters[0].Description = "Input."
	function.Signatures = append(function.Signatures, api.Signature{
		Parameters: []api.Parameter{{Name: "value", Type: namedType("Object"), Description: "Input."}},
	})

	requireRule(t, requireValidationErrors(t, api.Validate(reference)), validation.RuleDuplicate)
}

func namedTypeValue(name string) api.Type {
	return api.Type{Kind: api.TypeKindNamed, Name: name}
}

func unionTypeValue(members ...api.Type) api.Type {
	return api.Type{Kind: api.TypeKindUnion, Types: members}
}

func listTypeValue(element api.Type) api.Type {
	return api.Type{Kind: api.TypeKindList, Element: &element}
}
