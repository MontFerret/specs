package registry_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MontFerret/specs/pkg/registry"
)

const commitSHA1 = "0123456789abcdef0123456789abcdef01234567"

func TestValidModuleManifest(t *testing.T) {
	for _, sourcePath := range []string{"", "modules/archive", "modules/資料"} {
		manifest := validModuleManifest()
		manifest.Source.Path = sourcePath
		if err := registry.ValidateModuleManifest(manifest); err != nil {
			t.Errorf("path %q: %v", sourcePath, err)
		}
	}
}

func TestInvalidOwnerAndModuleName(t *testing.T) {
	for name, mutate := range map[string]func(*registry.ModuleManifest){
		"owner": func(manifest *registry.ModuleManifest) { manifest.Owner = "MontFerret" },
		"name":  func(manifest *registry.ModuleManifest) { manifest.Name = "-archive" },
	} {
		t.Run(name, func(t *testing.T) {
			manifest := validModuleManifest()
			mutate(manifest)
			requireValidationErrors(t, registry.ValidateModuleManifest(manifest))
		})
	}
}

func TestRepositoryURLValidation(t *testing.T) {
	valid := []string{
		"https://github.com/MontFerret/contrib.git",
		"https://gitlab.example.org/group/repository",
		"https://codeberg.org:443/owner/repository.git",
	}
	for _, repository := range valid {
		manifest := validModuleManifest()
		manifest.Source.Repository = repository
		if err := registry.ValidateModuleManifest(manifest); err != nil {
			t.Errorf("expected %q to be valid: %v", repository, err)
		}
	}

	invalid := []string{
		"http://github.com/MontFerret/contrib.git",
		"ssh://git@github.com/MontFerret/contrib.git",
		"https://github.com",
		"https://user@example.org/repository.git",
		"https://example.org/repository.git?ref=main",
		"https://example.org/repository.git#readme",
	}
	for _, repository := range invalid {
		manifest := validModuleManifest()
		manifest.Source.Repository = repository
		requireViolation(t, requireValidationErrors(t, registry.ValidateModuleManifest(manifest)), "/source/repository")
	}
}

func TestSourcePathValidation(t *testing.T) {
	for _, sourcePath := range []string{"/module", "module/", "module//nested", "module/../other", "./module", `module\nested`, "module/\x00nested"} {
		manifest := validModuleManifest()
		manifest.Source.Path = sourcePath
		requireViolation(t, requireValidationErrors(t, registry.ValidateModuleManifest(manifest)), "/source/path")
	}
}

func TestValidVersionRecord(t *testing.T) {
	for _, test := range []struct {
		version string
		tag     string
		commit  string
	}{
		{version: "1.2.0", tag: "archive/v1.2.0", commit: commitSHA1},
		{version: "1.3.0-beta.1+build.7", tag: "v1.3.0-beta.1", commit: strings.Repeat("a", 64)},
	} {
		record := validVersionRecord()
		record.Version, record.Tag, record.Commit = test.version, test.tag, test.commit
		if err := registry.ValidateVersionRecord(record); err != nil {
			t.Errorf("expected %#v to be valid: %v", test, err)
		}
	}
}

func TestInvalidVersionRecordValues(t *testing.T) {
	for name, mutate := range map[string]func(*registry.VersionRecord){
		"semver":       func(record *registry.VersionRecord) { record.Version = "v1.2.0" },
		"missing tag":  func(record *registry.VersionRecord) { record.Tag = "" },
		"invalid tag":  func(record *registry.VersionRecord) { record.Tag = "archive..release" },
		"short commit": func(record *registry.VersionRecord) { record.Commit = "0123456" },
		"upper commit": func(record *registry.VersionRecord) { record.Commit = strings.Repeat("A", 40) },
	} {
		t.Run(name, func(t *testing.T) {
			record := validVersionRecord()
			mutate(record)
			requireValidationErrors(t, registry.ValidateVersionRecord(record))
		})
	}
}

func TestParsingRejectsMissingAndUnknownFields(t *testing.T) {
	for _, document := range []string{
		`{"$schema":"https://schemas.ferretlang.org/registry/version/v1.json","version":"1.0.0","commit":"` + commitSHA1 + `"}`,
		`{"$schema":"https://schemas.ferretlang.org/registry/version/v1.json","version":"1.0.0","tag":"v1.0.0"}`,
		`{"$schema":"https://schemas.ferretlang.org/registry/version/v1.json","version":"1.0.0","tag":"v1.0.0","commit":"` + commitSHA1 + `","extra":true}`,
		`{"$schema":"https://schemas.ferretlang.org/registry/version/v2.json","version":"1.0.0","tag":"v1.0.0","commit":"` + commitSHA1 + `"}`,
	} {
		if _, err := registry.ParseVersionRecord([]byte(document)); err == nil {
			t.Fatalf("expected document to fail: %s", document)
		}
	}
}

func TestParsingRejectsDuplicateKeysAndTrailingDocuments(t *testing.T) {
	duplicate := `{"$schema":"https://schemas.ferretlang.org/registry/version/v1.json","version":"1.0.0","version":"2.0.0","tag":"v1.0.0","commit":"` + commitSHA1 + `"}`
	_, err := registry.ParseVersionRecord([]byte(duplicate))
	requireRule(t, requireValidationErrors(t, err), registry.RuleDecode)

	valid, err := json.Marshal(validVersionRecord())
	if err != nil {
		t.Fatal(err)
	}
	_, err = registry.ParseVersionRecord(append(valid, []byte(` {}`)...))
	requireRule(t, requireValidationErrors(t, err), registry.RuleDecode)
}

func TestParsingAndLoadingRoundTrip(t *testing.T) {
	manifest := validModuleManifest()
	manifest.Source.Path = "modules/archive"
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := registry.ParseModuleManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := registry.LoadModuleManifest(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	filePath := filepath.Join(directory, "manifest.json")
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	fromFile, err := registry.LoadModuleManifestFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, loaded) || !reflect.DeepEqual(parsed, fromFile) || !reflect.DeepEqual(parsed, manifest) {
		t.Fatalf("round trip differs: %#v %#v %#v", parsed, loaded, fromFile)
	}
}

func TestValidateNilReturnsStructuredErrors(t *testing.T) {
	requireValidationErrors(t, registry.ValidateModuleManifest(nil))
	requireValidationErrors(t, registry.ValidateVersionRecord(nil))
}

func validModuleManifest() *registry.ModuleManifest {
	return &registry.ModuleManifest{
		Schema: registry.ModuleManifestSchemaV1,
		Owner:  "montferret",
		Name:   "archive",
		Source: registry.Source{Repository: "https://github.com/MontFerret/contrib.git"},
	}
}

func validVersionRecord() *registry.VersionRecord {
	return &registry.VersionRecord{
		Schema:  registry.VersionRecordSchemaV1,
		Version: "1.2.0",
		Tag:     "archive/v1.2.0",
		Commit:  commitSHA1,
	}
}

func requireValidationErrors(t *testing.T, err error) *registry.ValidationErrors {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error")
	}
	var validationErr *registry.ValidationErrors
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *registry.ValidationErrors, got %T: %v", err, err)
	}
	return validationErr
}

func requireViolation(t *testing.T, validationErr *registry.ValidationErrors, path string) {
	t.Helper()
	for _, violation := range validationErr.Violations {
		if violation.Path == path {
			return
		}
	}
	t.Fatalf("missing violation at %q in %#v", path, validationErr.Violations)
}

func requireRule(t *testing.T, validationErr *registry.ValidationErrors, rule registry.Rule) {
	t.Helper()
	for _, violation := range validationErr.Violations {
		if violation.Rule == rule {
			return
		}
	}
	t.Fatalf("missing rule %q in %#v", rule, validationErr.Violations)
}
