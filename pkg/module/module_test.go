package module_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/goccy/go-yaml"

	"github.com/MontFerret/specs/pkg/module"
	"github.com/MontFerret/specs/pkg/validation"
)

const fixtureRoot = "../../testdata/module-manifest"

func TestValidFixtures(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join(fixtureRoot, "valid", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no valid fixtures found")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			t.Parallel()
			if _, err := module.LoadFile(fixture); err != nil {
				t.Fatalf("expected valid fixture: %v", err)
			}
		})
	}
}

func TestInvalidFixtures(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join(fixtureRoot, "invalid", "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no invalid fixtures found")
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(filepath.Base(fixture), func(t *testing.T) {
			t.Parallel()
			_, err := module.LoadFile(fixture)
			if err == nil {
				t.Fatal("expected fixture to be invalid")
			}

			var validationErr *validation.Errors
			if !errors.As(err, &validationErr) {
				t.Fatalf("expected structured validation errors, got %T: %v", err, err)
			}
			if len(validationErr.Violations) == 0 {
				t.Fatal("expected at least one violation")
			}
			for _, violation := range validationErr.Violations {
				if violation.Rule == "" || violation.Message == "" {
					t.Fatalf("incomplete violation: %#v", violation)
				}
			}
		})
	}
}

func TestJSONAndYAMLDecodeIdentically(t *testing.T) {
	jsonManifest, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "minimal.json"))
	if err != nil {
		t.Fatal(err)
	}

	yamlManifest, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(jsonManifest, yamlManifest) {
		t.Fatalf("JSON and YAML differ:\nJSON: %#v\nYAML: %#v", jsonManifest, yamlManifest)
	}
}

func TestCanonicalManifestFilename(t *testing.T) {
	if module.ManifestFilename != "ferret.yaml" {
		t.Fatalf("manifest filename = %q, want ferret.yaml", module.ManifestFilename)
	}
}

func TestPublicIngestionAPIsValidateByDefault(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "valid", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := module.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := module.Load(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	fromFile, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(parsed, loaded) || !reflect.DeepEqual(parsed, fromFile) {
		t.Fatal("public ingestion APIs returned different manifests")
	}
	if err := module.Validate(parsed); err != nil {
		t.Fatalf("validate parsed manifest: %v", err)
	}
}

func TestLoadPropagatesReaderFailure(t *testing.T) {
	_, err := module.Load(failingReader{})
	if err == nil || !strings.Contains(err.Error(), "read module manifest") {
		t.Fatalf("expected reader failure, got %v", err)
	}
}

func TestSerializationRoundTrips(t *testing.T) {
	manifest, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "complete.json"))
	if err != nil {
		t.Fatal(err)
	}

	jsonData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := module.Parse(jsonData)
	if err != nil {
		t.Fatal(err)
	}

	yamlData, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := module.Parse(yamlData)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(manifest, fromJSON) || !reflect.DeepEqual(manifest, fromYAML) {
		t.Fatal("serialized manifest did not round-trip")
	}
}

func TestMultipleSemanticViolations(t *testing.T) {
	_, err := module.LoadFile(filepath.Join(fixtureRoot, "invalid", "multiple-violations.yaml"))
	validationErr := requireValidationErrors(t, err)

	want := map[string]validation.Rule{
		"/compatibility/ferret":   validation.RuleVersionRange,
		"/dependencies/0/version": validation.RuleVersionRange,
		"/dependencies/1/module":  validation.RuleDuplicate,
		"/license":                validation.RuleSPDX,
	}
	if len(validationErr.Violations) != len(want) {
		t.Fatalf("expected %d violations, got %#v", len(want), validationErr.Violations)
	}
	for _, violation := range validationErr.Violations {
		if want[violation.Path] != violation.Rule {
			t.Fatalf("unexpected violation: %#v", violation)
		}
	}
}

func TestDuplicateExportsReportEveryDuplicateKind(t *testing.T) {
	_, err := module.LoadFile(filepath.Join(fixtureRoot, "invalid", "duplicate-exports.yaml"))
	validationErr := requireValidationErrors(t, err)

	wantPaths := map[string]bool{
		"/exports/dialects/1":               false,
		"/exports/namespaces/0/constants/0": false,
		"/exports/namespaces/0/functions/1": false,
		"/exports/namespaces/1/name":        false,
	}
	for _, violation := range validationErr.Violations {
		if _, ok := wantPaths[violation.Path]; ok && violation.Rule == validation.RuleDuplicate {
			wantPaths[violation.Path] = true
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("missing duplicate violation at %s: %#v", path, validationErr.Violations)
		}
	}
}

func TestNamespaceScopeUsesSegmentBoundary(t *testing.T) {
	manifest := minimalManifest()
	manifest.Exports = &module.Exports{Namespaces: []module.NamespaceExport{{
		Name:      "DB::SQLITE_EXTRA",
		Functions: []string{"OPEN"},
	}}}

	validationErr := requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/exports/namespaces/0/name", validation.RuleNamespaceScope)
}

func TestSchemaErrorsExposePointerAndKeyword(t *testing.T) {
	_, err := module.LoadFile(filepath.Join(fixtureRoot, "invalid", "missing-documentation.yaml"))
	validationErr := requireValidationErrors(t, err)
	requireViolation(t, validationErr, "", validation.Rule("required"))

	_, err = module.LoadFile(filepath.Join(fixtureRoot, "invalid", "unknown-property.yaml"))
	validationErr = requireValidationErrors(t, err)
	requireViolation(t, validationErr, "", validation.Rule("additionalProperties"))
}

func TestDecodeRejectsDuplicateKeysAndMultipleDocuments(t *testing.T) {
	_, err := module.LoadFile(filepath.Join(fixtureRoot, "invalid", "duplicate-key.yaml"))
	validationErr := requireValidationErrors(t, err)
	requireViolation(t, validationErr, "", validation.RuleDecode)

	data, err := os.ReadFile(filepath.Join(fixtureRoot, "valid", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\n---\nname: second/document\n")...)
	_, err = module.Parse(data)
	validationErr = requireValidationErrors(t, err)
	requireViolation(t, validationErr, "", validation.RuleDecode)
}

func TestValidateNilManifestReturnsStructuredError(t *testing.T) {
	validationErr := requireValidationErrors(t, module.Validate(nil))
	requireViolation(t, validationErr, "", validation.Rule("type"))
}

func TestOptionalFieldsRemainUnset(t *testing.T) {
	manifest, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "minimal.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Authors != nil || manifest.Links != nil || manifest.Compatibility != nil ||
		manifest.Dependencies != nil || manifest.Keywords != nil || manifest.Categories != nil ||
		manifest.Exports != nil || manifest.Repository != nil {
		t.Fatalf("optional defaults were inserted: %#v", manifest)
	}
}

func TestStructuredRepositoryRoundTrip(t *testing.T) {
	manifest, err := module.LoadFile(filepath.Join(fixtureRoot, "valid", "complete.json"))
	if err != nil {
		t.Fatal(err)
	}

	want := &module.Repository{
		URL:       "https://github.com/MontFerret/contrib",
		Directory: "modules/db/sqlite",
	}
	if !reflect.DeepEqual(manifest.Repository, want) {
		t.Fatalf("repository = %#v, want %#v", manifest.Repository, want)
	}
}

func TestLegacyRepositoryStringIsRejected(t *testing.T) {
	_, err := module.LoadFile(filepath.Join(fixtureRoot, "invalid", "legacy-repository-string.yaml"))
	validationErr := requireValidationErrors(t, err)
	requireViolation(t, validationErr, "/repository", validation.Rule("type"))
}

func TestSchemaBoundaries(t *testing.T) {
	manifest := minimalManifest()
	manifest.Description = strings.Repeat("é", 200)
	if err := module.Validate(manifest); err != nil {
		t.Fatalf("200-character Unicode description should be valid: %v", err)
	}

	manifest.Description += "é"
	validationErr := requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/description", validation.Rule("maxLength"))

	manifest = minimalManifest()
	manifest.Documentation = "http://docs.montferret.dev/modules/http/"
	validationErr = requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/documentation", validation.Rule("pattern"))

	manifest = minimalManifest()
	manifest.Repository = &module.Repository{URL: "https://"}
	validationErr = requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/repository/url", validation.Rule("pattern"))

	manifest = minimalManifest()
	manifest.Authors = []module.Author{{Name: "Maintainer", Email: "not-an-email"}}
	validationErr = requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/authors/0/email", validation.Rule("format"))
}

func TestRepositoryDirectoryNormalization(t *testing.T) {
	for _, directory := range []string{".", "..", "modules/./sqlite", "modules/../sqlite"} {
		manifest := minimalManifest()
		manifest.Repository = &module.Repository{
			URL:       "https://github.com/MontFerret/contrib",
			Directory: directory,
		}

		validationErr := requireValidationErrors(t, module.Validate(manifest))
		requireViolation(t, validationErr, "/repository/directory", validation.RuleRepositoryDirectory)
	}
}

func TestDistributionIdentityRequiresCanonicalLowercase(t *testing.T) {
	if err := module.Validate(minimalManifest()); err != nil {
		t.Fatalf("canonical lowercase identity should be valid: %v", err)
	}

	const message = "module identity must use canonical lowercase owner/name spelling; each segment must start and end with a lowercase letter or digit"
	for _, name := range []string{
		"MONTFERRET/archive",
		"MontFerret/archive",
		"montferret/ARCHIVE",
		"montferret/Archive",
	} {
		t.Run(name, func(t *testing.T) {
			manifest := minimalManifest()
			manifest.Name = name

			validationErr := requireValidationErrors(t, module.Validate(manifest))
			requireViolationDetails(t, validationErr, "/name", validation.Rule("pattern"), message)

			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			_, err = module.Parse(data)
			validationErr = requireValidationErrors(t, err)
			requireViolationDetails(t, validationErr, "/name", validation.Rule("pattern"), message)
		})
	}

	manifest := minimalManifest()
	manifest.Dependencies = []module.Dependency{{Module: "MontFerret/archive", Version: "^1.0.0"}}
	validationErr := requireValidationErrors(t, module.Validate(manifest))
	requireViolationDetails(t, validationErr, "/dependencies/0/module", validation.Rule("pattern"), message)
}

func TestNamespaceCasingFollowsFQL(t *testing.T) {
	for _, namespace := range []string{"db::sqlite", "Db::SQLite", "DB::SQLITE"} {
		manifest := minimalManifest()
		manifest.Namespace = namespace
		manifest.Exports = &module.Exports{Namespaces: []module.NamespaceExport{{
			Name:      namespace,
			Functions: []string{"OPEN"},
		}}}

		if err := module.Validate(manifest); err != nil {
			t.Errorf("namespace %q should be valid: %v", namespace, err)
		}
	}
}

func TestNPMRangeExamples(t *testing.T) {
	for _, value := range []string{"^2.0.0", "~2.4.0", ">=2.0.0 <3.0.0", "2.x"} {
		manifest := minimalManifest()
		manifest.Compatibility = &module.Compatibility{Ferret: value}
		manifest.Dependencies = []module.Dependency{{Module: "montferret/yaml", Version: value}}

		if err := module.Validate(manifest); err != nil {
			t.Errorf("range %q should be valid: %v", value, err)
		}
	}
}

func TestSelfDependency(t *testing.T) {
	manifest := minimalManifest()
	manifest.Dependencies = []module.Dependency{{Module: manifest.Name, Version: "^1.0.0"}}

	validationErr := requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/dependencies/0/module", validation.RuleSelfDependency)
}

func TestSPDXExpressions(t *testing.T) {
	valid := []string{
		"Apache-2.0 OR MIT",
		"GPL-2.0-only WITH Bison-exception-2.2",
		"LicenseRef-Proprietary",
	}
	for _, expression := range valid {
		manifest := minimalManifest()
		manifest.License = expression
		if err := module.Validate(manifest); err != nil {
			t.Errorf("expected %q to be valid: %v", expression, err)
		}
	}

	manifest := minimalManifest()
	manifest.License = "MIT MAYBE Apache-2.0"
	validationErr := requireValidationErrors(t, module.Validate(manifest))
	requireViolation(t, validationErr, "/license", validation.RuleSPDX)
}

func TestNPMRangePrereleaseContract(t *testing.T) {
	stableRange, err := semver.NewConstraint("^2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	prerelease, err := semver.StrictNewVersion("2.1.0-beta.1")
	if err != nil {
		t.Fatal(err)
	}
	if stableRange.Check(prerelease) {
		t.Fatal("prerelease unexpectedly satisfied a range without a prerelease comparator")
	}

	explicitRange, err := semver.NewConstraint(">=2.1.0-beta.1 <2.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if !explicitRange.Check(prerelease) {
		t.Fatal("prerelease did not satisfy an explicitly requested prerelease range")
	}
}

func minimalManifest() *module.Manifest {
	return &module.Manifest{
		Schema:        module.SchemaV1,
		Name:          "montferret/http",
		Namespace:     "NET::HTTP",
		Version:       "1.0.0",
		Description:   "Provides HTTP client functions for Ferret queries.",
		License:       "Apache-2.0",
		Documentation: "https://docs.montferret.dev/modules/http/",
	}
}

func requireValidationErrors(t *testing.T, err error) *validation.Errors {
	t.Helper()
	if err == nil {
		t.Fatal("expected validation error")
	}

	var validationErr *validation.Errors
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *validation.Errors, got %T: %v", err, err)
	}

	return validationErr
}

func requireViolation(t *testing.T, validationErr *validation.Errors, path string, rule validation.Rule) {
	t.Helper()
	for _, violation := range validationErr.Violations {
		if violation.Path == path && violation.Rule == rule {
			return
		}
	}

	t.Fatalf("missing violation path=%q rule=%q in %#v", path, rule, validationErr.Violations)
}

func requireViolationDetails(t *testing.T, validationErr *validation.Errors, path string, rule validation.Rule, message string) {
	t.Helper()
	for _, violation := range validationErr.Violations {
		if violation.Path == path && violation.Rule == rule && violation.Message == message {
			return
		}
	}

	t.Fatalf("missing violation path=%q rule=%q message=%q in %#v", path, rule, message, validationErr.Violations)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
