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
			reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0] = "file path"
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
					artifact.APIFunctionSignature{Parameters: []string{"other"}},
				)
			},
		},
		{
			name: "variadic signature",
			mutate: func(reference *artifact.APIReference) {
				reference.Namespaces[1].Functions[0].Signatures = append(
					reference.Namespaces[1].Functions[0].Signatures,
					artifact.APIFunctionSignature{Parameters: []string{"values"}, Variadic: true},
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

func TestAPIReferenceVariadicSignatureHasOneParameter(t *testing.T) {
	reference := validAPIReference()
	reference.Namespaces[1].Functions[0].Signatures[1].Parameters = []string{"first", "rest"}
	requireRule(t, requireValidationErrors(t, artifact.ValidateAPIReference(reference)), validation.RuleSchema)
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
	reference.Namespaces[1].Functions[0].Signatures[0].Parameters[0] = "_"
	violations := requireValidationErrors(t, artifact.ValidateAPIReference(reference))
	requireRule(t, violations, validation.RuleSchema)
	data, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	_, err = artifact.ParseAPIReference(data)
	requireRule(t, requireValidationErrors(t, err), validation.RuleSchema)
}
