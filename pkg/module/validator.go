package module

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/Masterminds/semver/v3"
	"github.com/github/go-spdx/v2/spdxexp"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/MontFerret/specs/internal/registryidentity"
	ferretschemas "github.com/MontFerret/specs/schemas"
)

const manifestSchemaPath = "module/v1.json"

var (
	compileOnce      sync.Once
	compiledManifest *jsonschema.Schema
	compileErr       error
)

// Validate validates a programmatically constructed module manifest.
func Validate(manifest *Manifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("encode module manifest: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var document any
	if err := decoder.Decode(&document); err != nil {
		return fmt.Errorf("decode encoded module manifest: %w", err)
	}

	if err := validateSchema(document); err != nil {
		return err
	}

	return validateSemantics(manifest)
}

func validateSchema(document any) error {
	schema, err := moduleSchema()
	if err != nil {
		return err
	}

	err = schema.Validate(document)
	if err == nil {
		return nil
	}

	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return fmt.Errorf("validate module manifest schema: %w", err)
	}

	return newValidationErrors(flattenSchemaErrors(validationErr))
}

func flattenSchemaErrors(validationErr *jsonschema.ValidationError) []Violation {
	if len(validationErr.Causes) > 0 {
		violations := make([]Violation, 0, len(validationErr.Causes))

		for _, cause := range validationErr.Causes {
			violations = append(violations, flattenSchemaErrors(cause)...)
		}

		return violations
	}

	rule := RuleSchema
	message := "document does not match the module manifest schema"

	if validationErr.ErrorKind != nil {
		keywordPath := validationErr.ErrorKind.KeywordPath()
		if len(keywordPath) > 0 {
			rule = Rule(keywordPath[len(keywordPath)-1])
		}

		output := validationErr.BasicOutput()
		if output.Error != nil {
			message = output.Error.String()
		}
	}
	if rule == Rule("pattern") && isDistributionIdentityLocation(validationErr.InstanceLocation) {
		message = registryidentity.CoordinateMessage
	}

	return []Violation{{
		Path:    jsonPointer(validationErr.InstanceLocation...),
		Rule:    rule,
		Message: message,
	}}
}

func isDistributionIdentityLocation(location []string) bool {
	if len(location) == 1 {
		return location[0] == "name"
	}

	return len(location) == 3 && location[0] == "dependencies" && location[2] == "module"
}

func moduleSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})

		resources := map[string]string{
			"https://schemas.ferretlang.org/common/identifier.json":    "common/identifier.json",
			"https://schemas.ferretlang.org/common/namespace.json":     "common/namespace.json",
			"https://schemas.ferretlang.org/common/semver.json":        "common/semver.json",
			"https://schemas.ferretlang.org/common/version-range.json": "common/version-range.json",
			"https://schemas.ferretlang.org/common/spdx-license.json":  "common/spdx-license.json",
			"https://schemas.ferretlang.org/common/url.json":           "common/url.json",
			SchemaV1: manifestSchemaPath,
		}

		for schemaURL, path := range resources {
			document, err := readSchema(path)
			if err != nil {
				compileErr = err
				return
			}

			if err := compiler.AddResource(schemaURL, document); err != nil {
				compileErr = fmt.Errorf("register embedded schema %q: %w", path, err)

				return
			}
		}

		compiledManifest, compileErr = compiler.Compile(SchemaV1)
		if compileErr != nil {
			compileErr = fmt.Errorf("compile embedded module manifest schema: %w", compileErr)
		}
	})

	return compiledManifest, compileErr
}

func readSchema(path string) (any, error) {
	data, err := ferretschemas.FS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read embedded schema %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode embedded schema %q: %w", path, err)
	}

	return document, nil
}

func validateSemantics(manifest *Manifest) error {
	violations := make([]Violation, 0)

	if _, err := semver.StrictNewVersion(manifest.Version); err != nil {
		violations = append(violations, Violation{
			Path:    jsonPointer("version"),
			Rule:    RuleSemVer,
			Message: "version must be a strict Semantic Versioning 2.0.0 version",
		})
	}

	if manifest.Compatibility != nil {
		violations = appendRangeViolation(
			violations,
			jsonPointer("compatibility", "ferret"),
			manifest.Compatibility.Ferret,
		)
	}

	seenDependencies := make(map[string]struct{}, len(manifest.Dependencies))
	for i, dependency := range manifest.Dependencies {
		path := jsonPointer("dependencies", strconv.Itoa(i))
		violations = appendRangeViolation(violations, path+"/version", dependency.Version)

		if _, exists := seenDependencies[dependency.Module]; exists {
			violations = append(violations, Violation{
				Path:    path + "/module",
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("dependency %q is declared more than once", dependency.Module),
			})
		} else {
			seenDependencies[dependency.Module] = struct{}{}
		}

		if dependency.Module == manifest.Name {
			violations = append(violations, Violation{
				Path:    path + "/module",
				Rule:    RuleSelfDependency,
				Message: fmt.Sprintf("module %q must not depend on itself", manifest.Name),
			})
		}
	}

	if manifest.Repository != nil && manifest.Repository.Directory != "" && !validRepositoryDirectory(manifest.Repository.Directory) {
		violations = append(violations, Violation{
			Path:    jsonPointer("repository", "directory"),
			Rule:    RuleRepositoryDirectory,
			Message: "repository directory must be a normalized relative slash-separated path",
		})
	}

	if valid, _ := spdxexp.ValidateLicenses([]string{manifest.License}); !valid {
		violations = append(violations, Violation{
			Path:    jsonPointer("license"),
			Rule:    RuleSPDX,
			Message: "license must be a valid SPDX license expression",
		})
	}

	if manifest.Exports != nil {
		violations = append(violations, validateExports(manifest.Namespace, manifest.Exports)...)
	}

	return newValidationErrors(violations)
}

func validRepositoryDirectory(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}

		for _, character := range segment {
			if unicode.IsControl(character) {
				return false
			}
		}
	}

	return true
}

func appendRangeViolation(violations []Violation, path, value string) []Violation {
	if _, err := semver.NewConstraint(value); err != nil {
		return append(violations, Violation{
			Path:    path,
			Rule:    RuleVersionRange,
			Message: "version must be a valid npm-compatible semantic version range",
		})
	}

	return violations
}

func validateExports(root string, exports *Exports) []Violation {
	violations := make([]Violation, 0)
	seenNamespaces := make(map[string]struct{}, len(exports.Namespaces))

	for i, namespace := range exports.Namespaces {
		path := jsonPointer("exports", "namespaces", strconv.Itoa(i))
		if _, exists := seenNamespaces[namespace.Name]; exists {
			violations = append(violations, Violation{
				Path:    path + "/name",
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("namespace %q is exported more than once", namespace.Name),
			})
		} else {
			seenNamespaces[namespace.Name] = struct{}{}
		}

		if namespace.Name != root && !strings.HasPrefix(namespace.Name, root+"::") {
			violations = append(violations, Violation{
				Path:    path + "/name",
				Rule:    RuleNamespaceScope,
				Message: fmt.Sprintf("exported namespace %q must equal or descend from module namespace %q", namespace.Name, root),
			})
		}

		violations = append(violations, validateMembers(path, namespace)...)
	}

	seenDialects := make(map[string]struct{}, len(exports.Dialects))
	for i, dialect := range exports.Dialects {
		if _, exists := seenDialects[dialect]; exists {
			violations = append(violations, Violation{
				Path:    jsonPointer("exports", "dialects", strconv.Itoa(i)),
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("dialect %q is exported more than once", dialect),
			})
		} else {
			seenDialects[dialect] = struct{}{}
		}
	}

	return violations
}

func validateMembers(basePath string, namespace NamespaceExport) []Violation {
	type memberList struct {
		name   string
		values []string
	}

	lists := []memberList{
		{name: "functions", values: namespace.Functions},
		{name: "types", values: namespace.Types},
		{name: "constants", values: namespace.Constants},
	}
	seen := make(map[string]string)
	violations := make([]Violation, 0)

	for _, list := range lists {
		for i, member := range list.values {
			if firstKind, exists := seen[member]; exists {
				violations = append(violations, Violation{
					Path:    basePath + "/" + list.name + "/" + strconv.Itoa(i),
					Rule:    RuleDuplicate,
					Message: fmt.Sprintf("member %q duplicates an export in %s", member, firstKind),
				})
			} else {
				seen[member] = list.name
			}
		}
	}

	return violations
}
