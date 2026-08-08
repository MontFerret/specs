package artifact_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/MontFerret/specs/pkg/registry/artifact"
)

const commitSHA1 = "0123456789abcdef0123456789abcdef01234567"

func TestArtifactRoundTrips(t *testing.T) {
	root := validRootIndex()
	root.Artifacts["advisories"] = "/advisories/index.json"
	roundTrip(t, root, artifact.ParseRootIndex)

	moduleIndex := validModuleIndex()
	roundTrip(t, moduleIndex, artifact.ParseModuleIndex)

	moduleDocument := validModuleDocument()
	roundTrip(t, moduleDocument, artifact.ParseModuleDocument)

	versionDocument := validVersionDocument()
	versionDocument.Content["readme"] = "./README.md"
	roundTrip(t, versionDocument, artifact.ParseVersionDocument)

	categoryIndex := validCategoryIndex()
	roundTrip(t, categoryIndex, artifact.ParseCategoryIndex)

	categoryDocument := validCategoryDocument()
	roundTrip(t, categoryDocument, artifact.ParseCategoryDocument)

	pluginIndex := &artifact.PluginIndex{SchemaVersion: artifact.SchemaVersion, Plugins: []json.RawMessage{}}
	roundTrip(t, pluginIndex, artifact.ParsePluginIndex)
}

func TestEmptyAndPrereleaseOnlyIndexesAreValid(t *testing.T) {
	if err := artifact.ValidateModuleIndex(&artifact.ModuleIndex{
		SchemaVersion: artifact.SchemaVersion,
		Modules:       []artifact.ModuleIndexEntry{},
	}); err != nil {
		t.Fatal(err)
	}

	if err := artifact.ValidateCategoryIndex(&artifact.CategoryIndex{
		SchemaVersion: artifact.SchemaVersion,
		Categories:    []artifact.CategoryIndexEntry{},
	}); err != nil {
		t.Fatal(err)
	}

	index := validModuleIndex()
	index.Modules[0].Latest = ""
	if err := artifact.ValidateModuleIndex(index); err != nil {
		t.Fatal(err)
	}

	document := validModuleDocument()
	document.Latest = ""
	if err := artifact.ValidateModuleDocument(document); err != nil {
		t.Fatal(err)
	}
}

func TestParsingIsStrict(t *testing.T) {
	for name, data := range map[string]string{
		"duplicate key":     `{"schemaVersion":1,"schemaVersion":1,"artifacts":{"categories":"/categories.json","modules":"/modules/index.json","plugins":"/plugins/index.json"}}`,
		"trailing document": `{"schemaVersion":1,"artifacts":{"categories":"/categories.json","modules":"/modules/index.json","plugins":"/plugins/index.json"}} {}`,
		"unknown field":     `{"schemaVersion":1,"artifacts":{"categories":"/categories.json","modules":"/modules/index.json","plugins":"/plugins/index.json"},"extra":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := artifact.ParseRootIndex([]byte(data))
			requireValidationErrors(t, err)
		})
	}
}

func TestUnsupportedSchemaVersionIsTyped(t *testing.T) {
	_, err := artifact.ParseRootIndex([]byte(`{"schemaVersion":2,"artifacts":{"categories":"/categories.json","modules":"/modules/index.json","plugins":"/plugins/index.json"}}`))
	var versionErr *artifact.UnsupportedVersionError
	if !errors.As(err, &versionErr) || versionErr.Version != 2 {
		t.Fatalf("expected unsupported version 2, got %T: %v", err, err)
	}
}

func TestCanonicalKeysAndReservedPluginsAreEnforced(t *testing.T) {
	root := validRootIndex()
	delete(root.Artifacts, artifact.ArtifactKeyPlugins)
	requireValidationErrors(t, artifact.ValidateRootIndex(root))

	version := validVersionDocument()
	delete(version.Content, artifact.ContentKeyDocumentationHTML)
	requireValidationErrors(t, artifact.ValidateVersionDocument(version))

	plugins := &artifact.PluginIndex{
		SchemaVersion: artifact.SchemaVersion,
		Plugins:       []json.RawMessage{json.RawMessage(`{"id":"acme/plugin"}`)},
	}
	requireValidationErrors(t, artifact.ValidatePluginIndex(plugins))

	category := validCategoryDocument()
	category.Modules = []artifact.ModuleIndexEntry{}
	requireValidationErrors(t, artifact.ValidateCategoryDocument(category))
}

func TestSameDocumentSemantics(t *testing.T) {
	t.Run("duplicate module IDs", func(t *testing.T) {
		index := validModuleIndex()
		duplicate := index.Modules[0]
		duplicate.Href = "/modules/acme/archive/duplicate.json"
		index.Modules = append(index.Modules, duplicate)
		requireRule(t, requireValidationErrors(t, artifact.ValidateModuleIndex(index)), artifact.RuleDuplicate)
	})

	t.Run("module identity and latest", func(t *testing.T) {
		document := validModuleDocument()
		document.ID = "acme/other"
		document.Latest = "2.0.0"
		validationErr := requireValidationErrors(t, artifact.ValidateModuleDocument(document))
		requireRule(t, validationErr, artifact.RuleIdentity)
	})

	t.Run("duplicate versions", func(t *testing.T) {
		document := validModuleDocument()
		duplicate := document.Versions[0]
		duplicate.Href = "/modules/acme/archive/versions/1.0.0/duplicate.json"
		document.Versions = append(document.Versions, duplicate)
		requireRule(t, requireValidationErrors(t, artifact.ValidateModuleDocument(document)), artifact.RuleDuplicate)
	})

	t.Run("zero publication timestamp", func(t *testing.T) {
		document := validModuleDocument()
		document.Versions[0].PublishedAt = time.Time{}
		requireRule(t, requireValidationErrors(t, artifact.ValidateModuleDocument(document)), artifact.RuleTimestamp)
	})

	t.Run("package path", func(t *testing.T) {
		document := validVersionDocument()
		document.Package.Path = "example.com/archive/v2"
		requireRule(t, requireValidationErrors(t, artifact.ValidateVersionDocument(document)), artifact.RulePackagePath)
	})

	t.Run("source", func(t *testing.T) {
		document := validVersionDocument()
		document.Source.Repository = "https://user@example.com/archive.git"
		requireRule(t, requireValidationErrors(t, artifact.ValidateVersionDocument(document)), artifact.RuleSource)
	})
}

func TestInvalidArtifactReferencesAreRejected(t *testing.T) {
	for _, reference := range []string{"", "./index.json?preview=1", "./index.json#section", "https://user@example.com/index.json"} {
		root := validRootIndex()
		root.Artifacts[artifact.ArtifactKeyModules] = reference
		if err := artifact.ValidateRootIndex(root); err == nil {
			t.Fatalf("expected reference %q to be rejected", reference)
		}
	}
}

func TestNilArtifactsReturnStructuredErrors(t *testing.T) {
	requireValidationErrors(t, artifact.ValidateRootIndex(nil))
	requireValidationErrors(t, artifact.ValidateModuleIndex(nil))
	requireValidationErrors(t, artifact.ValidateModuleDocument(nil))
	requireValidationErrors(t, artifact.ValidateVersionDocument(nil))
	requireValidationErrors(t, artifact.ValidateCategoryIndex(nil))
	requireValidationErrors(t, artifact.ValidateCategoryDocument(nil))
	requireValidationErrors(t, artifact.ValidatePluginIndex(nil))
}

func validRootIndex() *artifact.RootIndex {
	return &artifact.RootIndex{
		SchemaVersion: artifact.SchemaVersion,
		Artifacts: map[string]string{
			artifact.ArtifactKeyCategories: "/categories.json",
			artifact.ArtifactKeyModules:    "/modules/index.json",
			artifact.ArtifactKeyPlugins:    "/plugins/index.json",
		},
	}
}

func validModuleIndex() *artifact.ModuleIndex {
	return &artifact.ModuleIndex{
		SchemaVersion: artifact.SchemaVersion,
		Modules: []artifact.ModuleIndexEntry{{
			ID:     "acme/archive",
			Latest: "1.0.0",
			Href:   "/modules/acme/archive/index.json",
		}},
	}
}

func validModuleDocument() *artifact.ModuleDocument {
	return &artifact.ModuleDocument{
		SchemaVersion: artifact.SchemaVersion,
		ID:            "acme/archive",
		Owner:         "acme",
		Name:          "archive",
		Description:   "Archive support.",
		Latest:        "1.0.0",
		Versions: []artifact.ModuleDocumentVersion{{
			Version:     "1.0.0",
			PublishedAt: time.Date(2026, time.August, 7, 21, 54, 12, 0, time.UTC),
			Href:        "/modules/acme/archive/versions/1.0.0/index.json",
		}},
	}
}

func validVersionDocument() *artifact.VersionDocument {
	return &artifact.VersionDocument{
		SchemaVersion: artifact.SchemaVersion,
		ID:            "acme/archive",
		Version:       "1.0.0",
		Description:   "Archive support.",
		Namespace:     "ARCHIVE",
		Ferret:        ">=2.0.0 <3.0.0",
		License:       "Apache-2.0",
		Links:         map[string]string{"homepage": "https://example.com/archive"},
		Source: artifact.VersionSource{
			Repository: "https://example.com/archive.git",
			Path:       "modules/archive",
			Commit:     commitSHA1,
		},
		Package: artifact.VersionPackage{Path: "example.com/archive"},
		Content: map[string]string{
			artifact.ContentKeyDocumentation:     "./docs.md",
			artifact.ContentKeyDocumentationHTML: "./docs.html",
		},
	}
}

func validCategoryIndex() *artifact.CategoryIndex {
	return &artifact.CategoryIndex{
		SchemaVersion: artifact.SchemaVersion,
		Categories: []artifact.CategoryIndexEntry{{
			ID:    "data-formats",
			Name:  "Data Formats",
			Count: 1,
			Href:  "/categories/data-formats.json",
		}},
	}
}

func validCategoryDocument() *artifact.CategoryDocument {
	return &artifact.CategoryDocument{
		SchemaVersion: artifact.SchemaVersion,
		Category: artifact.CategorySummary{
			ID:   "data-formats",
			Name: "Data Formats",
		},
		Modules: validModuleIndex().Modules,
	}
}

func roundTrip[T any](t *testing.T, value *T, parse func([]byte) (*T, error)) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := parse(data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(parsed, value) {
		t.Fatalf("round trip differs:\ngot  %#v\nwant %#v", parsed, value)
	}
}

func requireValidationErrors(t *testing.T, err error) *artifact.ValidationErrors {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr *artifact.ValidationErrors
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *artifact.ValidationErrors, got %T: %v", err, err)
	}

	return validationErr
}

func requireRule(t *testing.T, validationErr *artifact.ValidationErrors, rule artifact.Rule) {
	t.Helper()
	for _, violation := range validationErr.Violations {
		if violation.Rule == rule {
			return
		}
	}

	var rules []string
	for _, violation := range validationErr.Violations {
		rules = append(rules, string(violation.Rule))
	}
	t.Fatalf("missing rule %q in %s", rule, strings.Join(rules, ", "))
}
