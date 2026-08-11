package api_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/MontFerret/specs/pkg/api"
	"github.com/MontFerret/specs/pkg/validation"
)

func TestIndexParsingIsStrict(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate key":       `{"schemaVersion":1,"schemaVersion":1,"versions":[{"version":"1.0.0","href":"./versions/1.0.0/api.json"}]}`,
		"trailing document":   `{"schemaVersion":1,"versions":[{"version":"1.0.0","href":"./versions/1.0.0/api.json"}]} {}`,
		"unknown field":       `{"schemaVersion":1,"versions":[{"version":"1.0.0","href":"./versions/1.0.0/api.json"}],"extra":true}`,
		"unknown entry field": `{"schemaVersion":1,"latest":"1.0.0","versions":[{"version":"1.0.0","href":"./versions/1.0.0/api.json","extra":true}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := api.ParseIndex([]byte(data))
			requireValidationErrors(t, err)
		})
	}
}

func TestIndexRequiresGreatestStableVersionAsLatest(t *testing.T) {
	index := &api.Index{
		SchemaVersion: api.IndexSchemaVersion,
		Latest:        "2.0.0",
		Versions: []api.IndexVersion{
			{Version: "2.0.0", Href: "./versions/2.0.0/api.json"},
			{Version: "1.0.0", Href: "./versions/1.0.0/api.json"},
		},
	}

	if err := api.ValidateIndex(index); err != nil {
		t.Fatalf("validate stable index: %v", err)
	}

	index.Latest = "1.0.0"
	requireRule(t, requireValidationErrors(t, api.ValidateIndex(index)), validation.RuleSemVer)
}

func TestIndexRoundTrip(t *testing.T) {
	index := validIndex()

	data, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := api.ParseIndex(data)
	if err != nil {
		t.Fatalf("parse index: %v", err)
	}

	if !reflect.DeepEqual(parsed, index) {
		t.Fatalf("parsed index = %#v, want %#v", parsed, index)
	}
}

func TestIndexAllowsPrereleaseOnlyVersionsWithoutLatest(t *testing.T) {
	index := &api.Index{
		SchemaVersion: api.IndexSchemaVersion,
		Versions: []api.IndexVersion{
			{Version: "2.0.0-alpha.2", Href: "./versions/2.0.0-alpha.2/api.json"},
			{Version: "2.0.0-alpha.1", Href: "./versions/2.0.0-alpha.1/api.json"},
		},
	}

	if err := api.ValidateIndex(index); err != nil {
		t.Fatalf("validate prerelease-only index: %v", err)
	}
}

func TestIndexRejectsInvalidMembers(t *testing.T) {
	for _, test := range []struct {
		name   string
		rule   validation.Rule
		mutate func(*api.Index)
	}{
		{name: "invalid version", rule: validation.RulePattern, mutate: func(index *api.Index) { index.Versions[0].Version = "latest" }},
		{name: "empty href", rule: "minLength", mutate: func(index *api.Index) { index.Versions[0].Href = "" }},
		{name: "invalid href", rule: "format", mutate: func(index *api.Index) { index.Versions[0].Href = "https://[invalid" }},
		{name: "query href", rule: validation.RulePattern, mutate: func(index *api.Index) { index.Versions[0].Href += "?download=1" }},
		{name: "fragment href", rule: validation.RulePattern, mutate: func(index *api.Index) { index.Versions[0].Href += "#api" }},
		{name: "duplicate version", rule: validation.RuleDuplicate, mutate: func(index *api.Index) { index.Versions[1].Version = index.Versions[0].Version }},
		{name: "duplicate href", rule: validation.RuleDuplicate, mutate: func(index *api.Index) { index.Versions[1].Href = index.Versions[0].Href }},
		{name: "wrong order", rule: validation.RuleSemVer, mutate: func(index *api.Index) { index.Versions[0], index.Versions[1] = index.Versions[1], index.Versions[0] }},
		{name: "missing latest", rule: validation.RuleSemVer, mutate: func(index *api.Index) { index.Latest = "" }},
		{name: "prerelease latest", rule: validation.RuleSemVer, mutate: func(index *api.Index) { index.Latest = "2.0.0-alpha.1" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			index := validIndex()
			test.mutate(index)
			requireRule(t, requireValidationErrors(t, api.ValidateIndex(index)), test.rule)
		})
	}
}

func TestIndexUsesRawVersionTieBreakerForBuildMetadata(t *testing.T) {
	index := &api.Index{
		SchemaVersion: api.IndexSchemaVersion,
		Latest:        "2.0.0+build.2",
		Versions: []api.IndexVersion{
			{Version: "2.0.0+build.2", Href: "./versions/2.0.0+build.2/api.json"},
			{Version: "2.0.0+build.1", Href: "./versions/2.0.0+build.1/api.json"},
		},
	}

	if err := api.ValidateIndex(index); err != nil {
		t.Fatalf("validate build metadata ordering: %v", err)
	}
}

func TestIndexUnsupportedVersion(t *testing.T) {
	data := []byte(`{"schemaVersion":2,"versions":[{"version":"1.0.0","href":"./versions/1.0.0/api.json"}]}`)
	_, err := api.ParseIndex(data)

	var versionErr *api.UnsupportedVersionError
	if !errors.As(err, &versionErr) || versionErr.Version != 2 {
		t.Fatalf("error = %T %v, want UnsupportedVersionError for version 2", err, err)
	}
}

func TestIndexValidationRejectsNil(t *testing.T) {
	requireValidationErrors(t, api.ValidateIndex(nil))
}

func validIndex() *api.Index {
	return &api.Index{
		SchemaVersion: api.IndexSchemaVersion,
		Latest:        "2.0.0",
		Versions: []api.IndexVersion{
			{Version: "2.0.0", Href: "./versions/2.0.0/api.json"},
			{Version: "2.0.0-alpha.1", Href: "./versions/2.0.0-alpha.1/api.json"},
		},
	}
}
