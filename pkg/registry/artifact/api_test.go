package artifact_test

import (
	"encoding/json"
	"testing"

	"github.com/MontFerret/specs/pkg/registry/artifact"
	"github.com/MontFerret/specs/pkg/validation"
)

func TestAPIReferenceParsingIsStrict(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate key":     `{"schemaVersion":1,"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[]}`,
		"trailing document": `{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[]} {}`,
		"unknown field":     `{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[],"extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := artifact.ParseAPIReference([]byte(data))
			requireValidationErrors(t, err)
		})
	}
}

func TestAPIReferenceSchemaRejectsMalformedMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifact.APIReference)
	}{
		{name: "version", mutate: func(reference *artifact.APIReference) { reference.Version = "latest" }},
		{name: "namespace", mutate: func(reference *artifact.APIReference) { reference.Namespaces[1].Name = "ARCHIVE::" }},
		{name: "function", mutate: func(reference *artifact.APIReference) { reference.Namespaces[1].Functions[0].Name = "READ-FILE" }},
		{name: "parameter", mutate: func(reference *artifact.APIReference) {
			reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0].Name = "file path"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := validAPIReference()
			test.mutate(reference)
			data, err := json.Marshal(reference)
			if err != nil {
				t.Fatal(err)
			}
			_, err = artifact.ParseAPIReference(data)
			requireValidationErrors(t, err)
		})
	}
}

func TestAPIReferenceRejectsDuplicateMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*artifact.APIReference)
	}{
		{
			name: "namespace",
			mutate: func(reference *artifact.APIReference) {
				reference.Namespaces = append(reference.Namespaces, reference.Namespaces[1])
			},
		},
		{
			name: "function",
			mutate: func(reference *artifact.APIReference) {
				function := reference.Namespaces[1].Functions[0]
				reference.Namespaces[1].Functions = append(reference.Namespaces[1].Functions, function)
			},
		},
		{
			name: "fixed signature",
			mutate: func(reference *artifact.APIReference) {
				reference.Namespaces[1].Functions[0].Signatures = append(
					reference.Namespaces[1].Functions[0].Signatures,
					artifact.APIFunctionSignature{Parameters: []artifact.APIParameter{{Name: "other"}}},
				)
			},
		},
		{
			name: "variadic signature",
			mutate: func(reference *artifact.APIReference) {
				reference.Namespaces[1].Functions[0].Signatures = append(
					reference.Namespaces[1].Functions[0].Signatures,
					artifact.APIFunctionSignature{Parameters: []artifact.APIParameter{{Name: "values"}}, Variadic: true},
				)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := validAPIReference()
			test.mutate(reference)
			requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.RuleDuplicate)
		})
	}
}

func TestAPIReferenceVariadicSignatureAllowsLogicalParameters(t *testing.T) {
	reference := validAPIReference()
	reference.Namespaces[1].Functions[0].Signatures[1].Parameters = []artifact.APIParameter{{Name: "first"}, {Name: "rest"}}
	if err := artifact.ValidateAPIReference(reference); err != nil {
		t.Fatalf("validate variadic logical parameters: %v", err)
	}
}

func TestAPIReferenceRejectsSignatureParameterConstraints(t *testing.T) {
	for _, test := range []struct {
		rule   string
		mutate func(*artifact.APIFunctionSignature)
	}{
		{
			rule: "maxItems",
			mutate: func(signature *artifact.APIFunctionSignature) {
				signature.Parameters = []artifact.APIParameter{{Name: "one"}, {Name: "two"}, {Name: "three"}, {Name: "four"}, {Name: "five"}}
			},
		},
		{
			rule: "minItems",
			mutate: func(signature *artifact.APIFunctionSignature) {
				signature.Parameters = []artifact.APIParameter{}
				signature.Variadic = true
			},
		},
	} {
		reference := validAPIReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[0]
		test.mutate(signature)
		requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.Rule(test.rule))
	}
}

func TestAPIReferenceAllowsNoNamespaces(t *testing.T) {
	reference := validAPIReference()
	reference.Namespaces = []artifact.APINamespace{}
	if err := artifact.ValidateAPIReference(reference); err != nil {
		t.Fatalf("validate empty API Reference: %v", err)
	}
	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := artifact.ParseAPIReference(data); err != nil {
		t.Fatalf("parse empty API Reference: %v", err)
	}
}

func TestAPIReferenceRejectsReservedParameterName(t *testing.T) {
	reference := validAPIReference()
	reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0].Name = "_"
	violations := requireValidationErrors(t, artifact.ValidateAPIReference(reference))
	requireRule(t, violations, validation.RuleSchema)
	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	_, err = artifact.ParseAPIReference(data)
	requireRule(t, requireValidationErrors(t, err), validation.RuleSchema)
}

func TestAPIReferenceRejectsLegacySignatureShape(t *testing.T) {
	for name, signature := range map[string]string{
		"string parameter": `{"parameters":["value"]}`,
		"documentation":    `{"parameters":[],"documentation":"Legacy prose."}`,
	} {
		t.Run(name, func(t *testing.T) {
			data := []byte(`{"schemaVersion":1,"id":"acme/archive","version":"1.0.0","namespaces":[{"name":"ARCHIVE","functions":[{"name":"READ","signatures":[` + signature + `]}]}]}`)
			requireValidationErrors(t, func() error {
				_, err := artifact.ParseAPIReference(data)

				return err
			}())
		})
	}
}

func TestAPIReferenceRejectsIncompleteParameterDocumentation(t *testing.T) {
	for _, mutate := range []func(*artifact.APIParameter){
		func(parameter *artifact.APIParameter) { parameter.Description = "" },
		func(parameter *artifact.APIParameter) { parameter.Type = "" },
	} {
		reference := validAPIReference()
		parameter := &reference.Namespaces[1].Functions[0].Signatures[1].Parameters[0]
		mutate(parameter)
		requireValidationErrors(t, artifact.ValidateAPIReference(reference))
	}
}

func TestAPIReferenceRejectsDuplicateParameterNames(t *testing.T) {
	reference := validAPIReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[1]
	signature.Parameters = append(signature.Parameters, signature.Parameters[0])
	requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.RuleDuplicate)
}

func TestAPIReferenceRejectsBlankStructuredMetadata(t *testing.T) {
	for _, mutate := range []func(*artifact.APIFunctionSignature){
		func(signature *artifact.APIFunctionSignature) { signature.Description = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Parameters[0].Type = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Parameters[0].Description = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Return.Type = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Return.Description = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Throws[0].Error = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Throws[0].Description = " \t" },
		func(signature *artifact.APIFunctionSignature) { signature.Deprecated = " \t" },
	} {
		reference := validAPIReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[1]
		signature.Description = "Reads archive paths."
		signature.Return = &artifact.APIReturn{Type: "Any", Description: "Archive content."}
		mutate(signature)
		requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.RuleSchema)
	}
}

func TestAPIReferenceRejectsMultilineTypeExpressions(t *testing.T) {
	for _, mutate := range []func(*artifact.APIFunctionSignature){
		func(signature *artifact.APIFunctionSignature) { signature.Parameters[0].Type = "String\nBinary" },
		func(signature *artifact.APIFunctionSignature) { signature.Return.Type = "String\rBinary" },
		func(signature *artifact.APIFunctionSignature) { signature.Throws[0].Error = "Parse\nError" },
	} {
		reference := validAPIReference()
		signature := &reference.Namespaces[1].Functions[0].Signatures[1]
		signature.Return = &artifact.APIReturn{Type: "Any", Description: "Archive content."}
		mutate(signature)
		requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.RulePattern)
	}
}

func TestAPIReferencePreservesRepeatedThrows(t *testing.T) {
	reference := validAPIReference()
	signature := &reference.Namespaces[1].Functions[0].Signatures[1]
	signature.Throws = append(signature.Throws, artifact.APIThrownError{Error: "ReadError", Description: "Another read failure."})
	if err := artifact.ValidateAPIReference(reference); err != nil {
		t.Fatalf("validate repeated throws: %v", err)
	}

	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := artifact.ParseAPIReference(data)
	if err != nil {
		t.Fatalf("parse repeated throws: %v", err)
	}

	got := parsed.Namespaces[1].Functions[0].Signatures[1].Throws
	if len(got) != 2 || got[0].Description != "An archive path cannot be read." || got[1].Description != "Another read failure." {
		t.Fatalf("throws order differs: %#v", got)
	}
}
