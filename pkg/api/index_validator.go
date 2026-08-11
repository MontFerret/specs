package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
	ferretschemas "github.com/MontFerret/specs/schemas"
)

const indexSchemaPath = "registry/artifact/api-index/v1.json"

var (
	indexSchemaOnce sync.Once
	indexSchemaV1   *jsonschema.Schema
	indexSchemaErr  error
)

// ValidateIndex validates a programmatically constructed Ferret API Reference Index.
func ValidateIndex(index *Index) error {
	data, err := json.Marshal(index)
	if err != nil {
		return fmt.Errorf("encode Ferret API Reference Index: %w", err)
	}

	document, err := jsondocument.Decode(data)
	if err != nil {
		return fmt.Errorf("decode encoded Ferret API Reference Index: %w", err)
	}

	if err := validateIndexDocument(document); err != nil {
		return err
	}

	return validateIndexSemantics(index)
}

func validateIndexDocument(document any) error {
	if version, ok := documentSchemaVersion(document); ok && version > 0 && version != IndexSchemaVersion {
		return &UnsupportedVersionError{Version: version}
	}

	schema, err := compiledIndexSchema()
	if err != nil {
		return err
	}

	if err := schema.Validate(document); err != nil {
		var validationErr *jsonschema.ValidationError
		if !errors.As(err, &validationErr) {
			return fmt.Errorf("validate Ferret API Reference Index schema: %w", err)
		}

		return validation.NewErrors(validation.ScopeRegistryArtifact, flattenSchemaErrors(validationErr))
	}

	return nil
}

func compiledIndexSchema() (*jsonschema.Schema, error) {
	indexSchemaOnce.Do(compileIndexSchema)

	return indexSchemaV1, indexSchemaErr
}

func compileIndexSchema() {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})

	for schemaID, resourcePath := range map[string]string{
		"https://schemas.ferretlang.org/common/semver.json": "common/semver.json",
		IndexSchemaV1: indexSchemaPath,
	} {
		data, err := ferretschemas.FS.ReadFile(resourcePath)
		if err != nil {
			indexSchemaErr = fmt.Errorf("read embedded schema %q: %w", resourcePath, err)

			return
		}

		document, err := jsondocument.Decode(data)
		if err != nil {
			indexSchemaErr = fmt.Errorf("decode embedded schema %q: %w", resourcePath, err)

			return
		}

		if err := compiler.AddResource(schemaID, document); err != nil {
			indexSchemaErr = fmt.Errorf("register embedded schema %q: %w", resourcePath, err)

			return
		}
	}

	indexSchemaV1, indexSchemaErr = compiler.Compile(IndexSchemaV1)
	if indexSchemaErr != nil {
		indexSchemaErr = fmt.Errorf("compile embedded Ferret API Reference Index schema %q: %w", IndexSchemaV1, indexSchemaErr)
	}
}

func validateIndexSemantics(index *Index) error {
	violations := make([]validation.Violation, 0)
	seenVersions := make(map[string]struct{}, len(index.Versions))
	seenHrefs := make(map[string]struct{}, len(index.Versions))
	var greatestStable string

	for position, entry := range index.Versions {
		entryPath := validation.JSONPointer("versions", strconv.Itoa(position))

		if _, exists := seenVersions[entry.Version]; exists {
			violations = append(violations, validation.Violation{
				Path:    entryPath + "/version",
				Rule:    validation.RuleDuplicate,
				Message: fmt.Sprintf("version %q is duplicated", entry.Version),
			})
		}

		seenVersions[entry.Version] = struct{}{}

		if _, exists := seenHrefs[entry.Href]; exists {
			violations = append(violations, validation.Violation{
				Path:    entryPath + "/href",
				Rule:    validation.RuleDuplicate,
				Message: fmt.Sprintf("href %q is duplicated", entry.Href),
			})
		}

		seenHrefs[entry.Href] = struct{}{}

		version, err := semver.StrictNewVersion(entry.Version)
		if err != nil {
			continue
		}

		if version.Prerelease() == "" && (greatestStable == "" || compareIndexVersions(entry.Version, greatestStable) < 0) {
			greatestStable = entry.Version
		}

		if position > 0 && compareIndexVersions(index.Versions[position-1].Version, entry.Version) > 0 {
			violations = append(violations, validation.Violation{
				Path:    entryPath + "/version",
				Rule:    validation.RuleSemVer,
				Message: "versions must be ordered by descending semantic version",
			})
		}
	}

	switch {
	case greatestStable == "" && index.Latest != "":
		violations = append(violations, validation.Violation{
			Path:    "/latest",
			Rule:    validation.RuleSemVer,
			Message: "latest must be omitted when the index contains only prereleases",
		})
	case greatestStable != "" && index.Latest == "":
		violations = append(violations, validation.Violation{
			Path:    "/latest",
			Rule:    validation.RuleSemVer,
			Message: fmt.Sprintf("latest must identify the greatest stable version %q", greatestStable),
		})
	case greatestStable != "" && index.Latest != greatestStable:
		violations = append(violations, validation.Violation{
			Path:    "/latest",
			Rule:    validation.RuleSemVer,
			Message: fmt.Sprintf("latest must identify the greatest stable version %q", greatestStable),
		})
	}

	return validation.NewErrors(validation.ScopeRegistryArtifact, violations)
}

func compareIndexVersions(left, right string) int {
	leftVersion, leftErr := semver.StrictNewVersion(left)
	rightVersion, rightErr := semver.StrictNewVersion(right)
	if leftErr != nil || rightErr != nil {
		return strings.Compare(right, left)
	}

	if leftVersion.GreaterThan(rightVersion) {
		return -1
	}

	if leftVersion.LessThan(rightVersion) {
		return 1
	}

	return strings.Compare(right, left)
}
