package registry

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"unicode"

	"github.com/Masterminds/semver/v3"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	ferretschemas "github.com/MontFerret/specs/schemas"
)

const (
	moduleSchemaPath  = "registry/module/v1.json"
	versionSchemaPath = "registry/version/v1.json"
)

var (
	moduleSchemaOnce  sync.Once
	moduleSchemaV1    *jsonschema.Schema
	moduleSchemaErr   error
	versionSchemaOnce sync.Once
	versionSchemaV1   *jsonschema.Schema
	versionSchemaErr  error
)

// ValidateModuleManifest validates a programmatically constructed Registry Module Manifest.
func ValidateModuleManifest(manifest *ModuleManifest) error {
	document, err := encodedDocument(manifest)
	if err != nil {
		return fmt.Errorf("encode registry module manifest: %w", err)
	}
	if err := validateModuleDocument(document); err != nil {
		return err
	}
	return validateModuleSemantics(manifest)
}

// ValidateVersionRecord validates a programmatically constructed Registry Version Record.
func ValidateVersionRecord(record *VersionRecord) error {
	document, err := encodedDocument(record)
	if err != nil {
		return fmt.Errorf("encode registry version record: %w", err)
	}
	if err := validateVersionDocument(document); err != nil {
		return err
	}
	return validateVersionSemantics(record)
}

func encodedDocument(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	return document, nil
}

func validateModuleDocument(document any) error {
	schema, err := compiledModuleSchema()
	if err != nil {
		return err
	}
	return validateSchema(schema, document)
}

func validateVersionDocument(document any) error {
	schema, err := compiledVersionSchema()
	if err != nil {
		return err
	}
	return validateSchema(schema, document)
}

func validateSchema(schema *jsonschema.Schema, document any) error {
	err := schema.Validate(document)
	if err == nil {
		return nil
	}
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return fmt.Errorf("validate registry schema: %w", err)
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
	message := "document does not match the registry schema"
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
	return []Violation{{
		Path:    jsonPointer(validationErr.InstanceLocation...),
		Rule:    rule,
		Message: message,
	}}
}

func compiledModuleSchema() (*jsonschema.Schema, error) {
	moduleSchemaOnce.Do(func() {
		moduleSchemaV1, moduleSchemaErr = compileSchema(ModuleManifestSchemaV1, map[string]string{
			ModuleManifestSchemaV1: moduleSchemaPath,
		})
	})
	return moduleSchemaV1, moduleSchemaErr
}

func compiledVersionSchema() (*jsonschema.Schema, error) {
	versionSchemaOnce.Do(func() {
		versionSchemaV1, versionSchemaErr = compileSchema(VersionRecordSchemaV1, map[string]string{
			"https://schemas.ferretlang.org/common/semver.json": "common/semver.json",
			VersionRecordSchemaV1:                               versionSchemaPath,
		})
	})
	return versionSchemaV1, versionSchemaErr
}

func compileSchema(schemaID string, resources map[string]string) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})
	for id, schemaPath := range resources {
		document, err := readSchema(schemaPath)
		if err != nil {
			return nil, err
		}
		if err := compiler.AddResource(id, document); err != nil {
			return nil, fmt.Errorf("register embedded schema %q: %w", schemaPath, err)
		}
	}
	schema, err := compiler.Compile(schemaID)
	if err != nil {
		return nil, fmt.Errorf("compile embedded registry schema %q: %w", schemaID, err)
	}
	return schema, nil
}

func readSchema(schemaPath string) (any, error) {
	data, err := ferretschemas.FS.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded schema %q: %w", schemaPath, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode embedded schema %q: %w", schemaPath, err)
	}
	return document, nil
}

type offlineLoader struct{}

func (offlineLoader) Load(schemaURL string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", schemaURL)
}

func validateModuleSemantics(manifest *ModuleManifest) error {
	violations := make([]Violation, 0, 2)
	if err := validateRepositoryURL(manifest.Source.Repository); err != nil {
		violations = append(violations, Violation{Path: jsonPointer("source", "repository"), Rule: RuleRepositoryURL, Message: err.Error()})
	}
	if manifest.Source.Path != "" {
		if err := validateSourcePath(manifest.Source.Path); err != nil {
			violations = append(violations, Violation{Path: jsonPointer("source", "path"), Rule: RuleSourcePath, Message: err.Error()})
		}
	}
	return newValidationErrors(violations)
}

func validateVersionSemantics(record *VersionRecord) error {
	violations := make([]Violation, 0, 2)
	if _, err := semver.StrictNewVersion(record.Version); err != nil {
		violations = append(violations, Violation{Path: jsonPointer("version"), Rule: RuleSemVer, Message: "version must be a strict Semantic Versioning 2.0.0 version"})
	}
	if err := validateTag(record.Tag); err != nil {
		violations = append(violations, Violation{Path: jsonPointer("tag"), Rule: RuleTag, Message: err.Error()})
	}
	return newValidationErrors(violations)
}

func validateRepositoryURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("repository must be a valid absolute HTTPS URL")
	}
	if parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("repository must be a valid absolute HTTPS URL with a repository path")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("repository URL must not contain credentials, a query, or a fragment")
	}
	return nil
}

func validateSourcePath(value string) error {
	if strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "\\") {
		return fmt.Errorf("source path must be a normalized relative slash-separated path")
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("source path must not contain empty, current-directory, or parent-directory segments")
		}
		for _, character := range segment {
			if unicode.IsControl(character) {
				return fmt.Errorf("source path must not contain control characters")
			}
		}
	}
	return nil
}

func validateTag(value string) error {
	if value == "" || strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.HasSuffix(value, ".") || strings.Contains(value, "//") {
		return fmt.Errorf("tag must be a valid Git tag name")
	}
	if strings.Contains(value, "..") || strings.Contains(value, "@{") {
		return fmt.Errorf("tag must be a valid Git tag name")
	}
	for _, character := range value {
		if unicode.IsControl(character) || strings.ContainsRune(" ~^:?*[\\", character) {
			return fmt.Errorf("tag must be a valid Git tag name")
		}
	}
	for _, component := range strings.Split(value, "/") {
		if strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
			return fmt.Errorf("tag must be a valid Git tag name")
		}
	}
	return nil
}
