package validation

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewErrorsReturnsNilWithoutViolations(t *testing.T) {
	if err := NewErrors(ScopeManifest, nil); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestErrorsFormatEveryScope(t *testing.T) {
	for _, test := range []struct {
		name  string
		scope Scope
		want  string
	}{
		{name: "manifest", scope: ScopeManifest, want: "manifest validation failed at /name (schema): invalid name"},
		{name: "registry", scope: ScopeRegistry, want: "registry validation failed at /name (schema): invalid name"},
		{name: "registry artifact", scope: ScopeRegistryArtifact, want: "registry artifact validation failed at /name (schema): invalid name"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := NewErrors(test.scope, []Violation{{Path: "/name", Rule: RuleSchema, Message: "invalid name"}})
			if err == nil || err.Error() != test.want {
				t.Fatalf("error = %q, want %q", err, test.want)
			}
		})
	}
}

func TestErrorsFormatDocumentRoot(t *testing.T) {
	err := NewErrors(ScopeManifest, []Violation{{Rule: RuleDecode, Message: "invalid document"}})
	want := "manifest validation failed at document root (decode): invalid document"
	if err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestNewErrorsSortsViolations(t *testing.T) {
	err := NewErrors(ScopeRegistry, []Violation{
		{Path: "/z", Rule: RuleSchema, Message: "last"},
		{Path: "/a", Rule: RuleSemVer, Message: "second rule"},
		{Path: "/a", Rule: RuleSchema, Message: "second message"},
		{Path: "/a", Rule: RuleSchema, Message: "first message"},
	})

	var validationErr *Errors
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *Errors, got %T: %v", err, err)
	}

	want := []Violation{
		{Path: "/a", Rule: RuleSchema, Message: "first message"},
		{Path: "/a", Rule: RuleSchema, Message: "second message"},
		{Path: "/a", Rule: RuleSemVer, Message: "second rule"},
		{Path: "/z", Rule: RuleSchema, Message: "last"},
	}
	if !reflect.DeepEqual(validationErr.Violations, want) {
		t.Fatalf("violations = %#v, want %#v", validationErr.Violations, want)
	}

	if got, want := err.Error(), "registry validation failed with 4 violations"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNilErrorsUsesGenericMessage(t *testing.T) {
	var err *Errors
	if got, want := err.Error(), "validation failed"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestJSONPointer(t *testing.T) {
	for _, test := range []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "root", want: ""},
		{name: "tokens", parts: []string{"dependencies", "0", "module"}, want: "/dependencies/0/module"},
		{name: "escaping", parts: []string{"a/b", "~value"}, want: "/a~1b/~0value"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := JSONPointer(test.parts...); got != test.want {
				t.Fatalf("JSONPointer() = %q, want %q", got, test.want)
			}
		})
	}
}
