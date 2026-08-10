package api_test

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/MontFerret/specs/pkg/api"
	"github.com/MontFerret/specs/pkg/validation"
)

func TestReferenceParsingIsStrict(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate key":     `{"schemaVersion":1,"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[]}`,
		"trailing document": `{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[]} {}`,
		"unknown field":     `{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[],"extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := api.Parse([]byte(data))
			requireValidationErrors(t, err)
		})
	}
}

func TestReferenceSchemaRejectsMalformedMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*api.Reference)
	}{
		{name: "version", mutate: func(reference *api.Reference) { reference.Version = "latest" }},
		{name: "namespace", mutate: func(reference *api.Reference) { reference.Namespaces[1].Name = "ARCHIVE::" }},
		{name: "function", mutate: func(reference *api.Reference) { reference.Namespaces[1].Functions[0].Name = "READ-FILE" }},
		{name: "parameter", mutate: func(reference *api.Reference) {
			reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0].Name = "file path"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := validReference()
			test.mutate(reference)

			data, err := json.Marshal(reference)
			if err != nil {
				t.Fatal(err)
			}

			_, err = api.Parse(data)
			requireValidationErrors(t, err)
		})
	}
}

func TestReferenceRejectsDuplicateMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*api.Reference)
	}{
		{
			name: "namespace",
			mutate: func(reference *api.Reference) {
				reference.Namespaces = append(reference.Namespaces, reference.Namespaces[1])
			},
		},
		{
			name: "function",
			mutate: func(reference *api.Reference) {
				function := reference.Namespaces[1].Functions[0]
				reference.Namespaces[1].Functions = append(reference.Namespaces[1].Functions, function)
			},
		},
		{
			name: "fixed signature",
			mutate: func(reference *api.Reference) {
				reference.Namespaces[1].Functions[0].Signatures = append(
					reference.Namespaces[1].Functions[0].Signatures,
					api.Signature{Parameters: []api.Parameter{{Name: "other"}}},
				)
			},
		},
		{
			name: "variadic signature",
			mutate: func(reference *api.Reference) {
				reference.Namespaces[1].Functions[0].Signatures = append(
					reference.Namespaces[1].Functions[0].Signatures,
					api.Signature{Parameters: []api.Parameter{{Name: "values"}}, Variadic: true},
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := validReference()
			test.mutate(reference)
			requireRule(t, requireValidationErrors(t, api.Validate(reference)), validation.RuleDuplicate)
		})
	}
}

func TestReferenceVariadicSignatureAllowsLogicalParameters(t *testing.T) {
	reference := validReference()
	reference.Namespaces[1].Functions[0].Signatures[1].Parameters = []api.Parameter{{Name: "first"}, {Name: "rest"}}

	if err := api.Validate(reference); err != nil {
		t.Fatalf("validate variadic logical parameters: %v", err)
	}
}

func TestReferenceRejectsSignatureParameterConstraints(t *testing.T) {
	for _, test := range []struct {
		rule   validation.Rule
		mutate func(*api.Signature)
	}{
		{
			rule: "maxItems",
			mutate: func(signature *api.Signature) {
				signature.Parameters = []api.Parameter{{Name: "one"}, {Name: "two"}, {Name: "three"}, {Name: "four"}, {Name: "five"}}
			},
		},
		{
			rule: "minItems",
			mutate: func(signature *api.Signature) {
				signature.Parameters = []api.Parameter{}
				signature.Variadic = true
			},
		},
	} {
		reference := validReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[0]
		test.mutate(signature)
		requireRule(t, requireValidationErrors(t, api.Validate(reference)), test.rule)
	}
}

func TestReferenceAllowsNoNamespaces(t *testing.T) {
	reference := validReference()
	reference.Namespaces = []api.Namespace{}

	if err := api.Validate(reference); err != nil {
		t.Fatalf("validate empty API Reference: %v", err)
	}

	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := api.Parse(data); err != nil {
		t.Fatalf("parse empty API Reference: %v", err)
	}
}

func TestReferenceRejectsReservedParameterName(t *testing.T) {
	reference := validReference()
	reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0].Name = "_"

	violations := requireValidationErrors(t, api.Validate(reference))
	requireRule(t, violations, validation.RuleSchema)

	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	_, err = api.Parse(data)
	requireRule(t, requireValidationErrors(t, err), validation.RuleSchema)
}

func TestReferenceRejectsLegacySignatureShape(t *testing.T) {
	for name, signature := range map[string]string{
		"string parameter": `{"parameters":["value"]}`,
		"documentation":    `{"parameters":[],"documentation":"Legacy prose."}`,
	} {
		t.Run(name, func(t *testing.T) {
			data := []byte(`{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[{"name":"ARCHIVE","functions":[{"name":"READ","signatures":[` + signature + `]}]}]}`)
			_, err := api.Parse(data)
			requireValidationErrors(t, err)
		})
	}
}

func TestReferenceRejectsIncompleteParameterDocumentation(t *testing.T) {
	for _, mutate := range []func(*api.Parameter){
		func(parameter *api.Parameter) { parameter.Description = "" },
		func(parameter *api.Parameter) { parameter.Type = "" },
	} {
		reference := validReference()
		parameter := &reference.Namespaces[1].Functions[0].Signatures[1].Parameters[0]
		mutate(parameter)
		requireValidationErrors(t, api.Validate(reference))
	}
}

func TestReferenceRejectsDuplicateParameterNames(t *testing.T) {
	reference := validReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[1]
	signature.Parameters = append(signature.Parameters, signature.Parameters[0])
	requireRule(t, requireValidationErrors(t, api.Validate(reference)), validation.RuleDuplicate)
}

func TestReferenceRejectsBlankStructuredMetadata(t *testing.T) {
	for _, mutate := range []func(*api.Signature){
		func(signature *api.Signature) { signature.Description = " \t" },
		func(signature *api.Signature) { signature.Parameters[0].Type = " \t" },
		func(signature *api.Signature) { signature.Parameters[0].Description = " \t" },
		func(signature *api.Signature) { signature.Return.Type = " \t" },
		func(signature *api.Signature) { signature.Return.Description = " \t" },
		func(signature *api.Signature) { signature.Throws[0].Error = " \t" },
		func(signature *api.Signature) { signature.Throws[0].Description = " \t" },
		func(signature *api.Signature) { signature.Deprecated = " \t" },
	} {
		reference := validReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[1]
		signature.Description = "Reads archive paths."
		signature.Return = &api.Return{Type: "Any", Description: "Archive content."}
		mutate(signature)
		requireRule(t, requireValidationErrors(t, api.Validate(reference)), validation.RuleSchema)
	}
}

func TestReferenceRejectsMultilineTypeExpressions(t *testing.T) {
	for _, mutate := range []func(*api.Signature){
		func(signature *api.Signature) { signature.Parameters[0].Type = "String\nBinary" },
		func(signature *api.Signature) { signature.Return.Type = "String\rBinary" },
		func(signature *api.Signature) { signature.Throws[0].Error = "Parse\nError" },
	} {
		reference := validReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[1]
		signature.Return = &api.Return{Type: "Any", Description: "Archive content."}
		mutate(signature)
		requireRule(t, requireValidationErrors(t, api.Validate(reference)), validation.RulePattern)
	}
}

func TestReferencePreservesRepeatedThrows(t *testing.T) {
	reference := validReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[1]
	signature.Throws = append(signature.Throws, api.Throw{Error: "ReadError", Description: "Another read failure."})

	if err := api.Validate(reference); err != nil {
		t.Fatalf("validate repeated throws: %v", err)
	}

	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := api.Parse(data)
	if err != nil {
		t.Fatalf("parse repeated throws: %v", err)
	}

	got := parsed.Namespaces[1].Functions[0].Signatures[1].Throws
	if len(got) != 2 || got[0].Description != "An archive path cannot be read." || got[1].Description != "Another read failure." {
		t.Fatalf("throws order differs: %#v", got)
	}
}

func TestReferenceUnsupportedVersion(t *testing.T) {
	data := []byte(`{"schemaVersion":2,"id":"acme/archive","version":"1.0.0","namespaces":[]}`)
	_, err := api.Parse(data)

	var versionErr *api.UnsupportedVersionError
	if !errors.As(err, &versionErr) || versionErr.Version != 2 {
		t.Fatalf("error = %T %v, want UnsupportedVersionError for version 2", err, err)
	}

	reference := validReference()
	reference.SchemaVersion = 2
	err = api.Validate(reference)
	if !errors.As(err, &versionErr) || versionErr.Version != 2 {
		t.Fatalf("validation error = %T %v, want UnsupportedVersionError for version 2", err, err)
	}
}

func TestReferenceValidationRejectsNil(t *testing.T) {
	requireValidationErrors(t, api.Validate(nil))
}

func TestReferenceViolationsAreDeterministic(t *testing.T) {
	reference := validReference()
	reference.Namespaces = append(reference.Namespaces, reference.Namespaces[1])
	reference.Namespaces[1].Functions = append(reference.Namespaces[1].Functions, reference.Namespaces[1].Functions[0])

	first := requireValidationErrors(t, api.Validate(reference))
	second := requireValidationErrors(t, api.Validate(reference))
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("violations differ:\nfirst:  %#v\nsecond: %#v", first, second)
	}

	for index := 1; index < len(first); index++ {
		if first[index-1].Path > first[index].Path {
			t.Fatalf("violations are not path ordered: %#v", first)
		}
	}
}

func TestBarnGeneratedReferenceFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/barn-v1.json")
	if err != nil {
		t.Fatal(err)
	}

	reference, err := api.Parse(data)
	if err != nil {
		t.Fatalf("parse Barn-generated fixture: %v", err)
	}

	if err := api.Validate(reference); err != nil {
		t.Fatalf("validate Barn-generated fixture: %v", err)
	}

	encoded, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	roundTripped, err := api.Parse(encoded)
	if err != nil {
		t.Fatalf("parse round-tripped fixture: %v", err)
	}

	if !reflect.DeepEqual(roundTripped, reference) {
		t.Fatalf("round-tripped fixture = %#v, want %#v", roundTripped, reference)
	}
}

func validReference() *api.Reference {
	return &api.Reference{
		SchemaVersion: api.SchemaVersion,
		ID:            "acme/archive",
		Version:       "1.0.0",
		Namespaces: []api.Namespace{
			{
				Name: "",
				Functions: []api.Function{{
					Name: "VERSION",
					Signatures: []api.Signature{{
						Parameters:  []api.Parameter{},
						Description: "Version returns the archive module version.",
						Return: &api.Return{
							Type:        "String",
							Description: "Current module version.",
						},
					}},
				}},
			},
			{
				Name: "ARCHIVE",
				Functions: []api.Function{{
					Name: "READ",
					Signatures: []api.Signature{
						{Parameters: []api.Parameter{{Name: "path"}}},
						{
							Parameters: []api.Parameter{{
								Name:        "paths",
								Type:        "String...",
								Description: "Archive paths.",
							}},
							Variadic:   true,
							Throws:     []api.Throw{{Error: "ReadError", Description: "An archive path cannot be read."}},
							Deprecated: "Use READ_FILE instead.",
						},
					},
				}},
			},
		},
	}
}

func requireValidationErrors(t *testing.T, err error) []validation.Violation {
	t.Helper()

	var validationErr *validation.Errors
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected validation errors, got %T: %v", err, err)
	}

	return validationErr.Violations
}

func requireRule(t *testing.T, violations []validation.Violation, rule validation.Rule) {
	t.Helper()

	for _, violation := range violations {
		if violation.Rule == rule {
			return
		}
	}

	t.Fatalf("violations %#v do not contain rule %q", violations, rule)
}
