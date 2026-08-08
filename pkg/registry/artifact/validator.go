package artifact

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	gomodule "golang.org/x/mod/module"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/internal/registryidentity"
	registryspec "github.com/MontFerret/specs/pkg/registry"
	ferretschemas "github.com/MontFerret/specs/schemas"
)

var (
	artifactSchemasOnce sync.Once
	artifactSchemas     map[string]*jsonschema.Schema
	artifactSchemasErr  error
)

func ValidateRootIndex(index *RootIndex) error {
	return validateArtifact(index, RootSchemaV1, validateRootIndexSemantics)
}

func ValidateModuleIndex(index *ModuleIndex) error {
	return validateArtifact(index, ModuleIndexSchemaV1, validateModuleIndexSemantics)
}

func ValidateModuleDocument(document *ModuleDocument) error {
	return validateArtifact(document, ModuleSchemaV1, validateModuleDocumentSemantics)
}

func ValidateVersionDocument(document *VersionDocument) error {
	return validateArtifact(document, VersionSchemaV1, validateVersionDocumentSemantics)
}

func ValidateCategoryIndex(index *CategoryIndex) error {
	return validateArtifact(index, CategoryIndexSchemaV1, validateCategoryIndexSemantics)
}

func ValidateCategoryDocument(document *CategoryDocument) error {
	return validateArtifact(document, CategorySchemaV1, validateCategoryDocumentSemantics)
}

func ValidatePluginIndex(index *PluginIndex) error {
	return validateArtifact(index, PluginIndexSchemaV1, noSemanticValidation[PluginIndex])
}

func validateArtifact[T any](value *T, schemaID string, semantic func(*T) error) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode registry artifact: %w", err)
	}

	document, err := jsondocument.Decode(data)
	if err != nil {
		return fmt.Errorf("decode encoded registry artifact: %w", err)
	}

	if err := validateDocument(schemaID, document); err != nil {
		return err
	}

	return semantic(value)
}

func validateDocument(schemaID string, document any) error {
	if version, ok := artifactSchemaVersion(document); ok && version > 0 && version != SchemaVersion {
		return &UnsupportedVersionError{Version: version}
	}

	schema, err := compiledArtifactSchema(schemaID)
	if err != nil {
		return err
	}

	if err := schema.Validate(document); err != nil {
		var validationErr *jsonschema.ValidationError
		if !errors.As(err, &validationErr) {
			return fmt.Errorf("validate registry artifact schema: %w", err)
		}

		return newValidationErrors(flattenSchemaErrors(validationErr, schemaID))
	}

	return nil
}

func artifactSchemaVersion(document any) (int, bool) {
	object, ok := document.(map[string]any)
	if !ok {
		return 0, false
	}

	number, ok := object["schemaVersion"].(json.Number)
	if !ok {
		return 0, false
	}

	value, err := strconv.Atoi(number.String())
	if err != nil {
		return 0, false
	}

	return value, true
}

func compiledArtifactSchema(schemaID string) (*jsonschema.Schema, error) {
	artifactSchemasOnce.Do(compileArtifactSchemas)
	if artifactSchemasErr != nil {
		return nil, artifactSchemasErr
	}

	schema, exists := artifactSchemas[schemaID]
	if !exists {
		return nil, fmt.Errorf("registry artifact schema %q is not registered", schemaID)
	}

	return schema, nil
}

func compileArtifactSchemas() {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})

	err := fs.WalkDir(ferretschemas.FS, ".", func(schemaPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() || !strings.HasSuffix(schemaPath, ".json") {
			return nil
		}

		data, err := ferretschemas.FS.ReadFile(schemaPath)
		if err != nil {
			return err
		}

		document, err := jsondocument.Decode(data)
		if err != nil {
			return fmt.Errorf("decode embedded schema %q: %w", schemaPath, err)
		}

		object, ok := document.(map[string]any)
		if !ok {
			return fmt.Errorf("embedded schema %q is not an object", schemaPath)
		}

		id, ok := object["$id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("embedded schema %q has no $id", schemaPath)
		}

		if err := compiler.AddResource(id, document); err != nil {
			return fmt.Errorf("register embedded schema %q: %w", schemaPath, err)
		}

		return nil
	})
	if err != nil {
		artifactSchemasErr = err
		return
	}

	artifactSchemas = make(map[string]*jsonschema.Schema, len(artifactSchemaIDs()))
	for _, schemaID := range artifactSchemaIDs() {
		schema, err := compiler.Compile(schemaID)
		if err != nil {
			artifactSchemasErr = fmt.Errorf("compile embedded registry artifact schema %q: %w", schemaID, err)
			return
		}

		artifactSchemas[schemaID] = schema
	}
}

func artifactSchemaIDs() []string {
	return []string{
		RootSchemaV1,
		ModuleIndexSchemaV1,
		ModuleSchemaV1,
		VersionSchemaV1,
		CategoryIndexSchemaV1,
		CategorySchemaV1,
		PluginIndexSchemaV1,
	}
}

func flattenSchemaErrors(validationErr *jsonschema.ValidationError, schemaID string) []Violation {
	if len(validationErr.Causes) > 0 {
		violations := make([]Violation, 0, len(validationErr.Causes))
		for _, cause := range validationErr.Causes {
			violations = append(violations, flattenSchemaErrors(cause, schemaID)...)
		}

		return violations
	}

	rule := RuleSchema
	message := "document does not match the registry artifact schema"
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
	if rule == Rule("pattern") {
		if identityMessage := artifactIdentityMessage(schemaID, validationErr.InstanceLocation); identityMessage != "" {
			message = identityMessage
		}
	}

	return []Violation{{
		Path:    jsonPointer(validationErr.InstanceLocation...),
		Rule:    rule,
		Message: message,
	}}
}

func artifactIdentityMessage(schemaID string, location []string) string {
	switch schemaID {
	case ModuleIndexSchemaV1, CategorySchemaV1:
		if len(location) == 3 && location[0] == "modules" && location[2] == "id" {
			return registryidentity.CoordinateMessage
		}
	case ModuleSchemaV1:
		if len(location) != 1 {
			return ""
		}
		switch location[0] {
		case "id":
			return registryidentity.CoordinateMessage
		case "owner":
			return registryidentity.OwnerMessage
		case "name":
			return registryidentity.ModuleNameMessage
		}
	case VersionSchemaV1:
		if len(location) == 1 && location[0] == "id" {
			return registryidentity.CoordinateMessage
		}
	}

	return ""
}

func validateRootIndexSemantics(index *RootIndex) error {
	violations := validateReferenceMap(index.Artifacts, "artifacts")
	return newValidationErrors(violations)
}

func validateModuleIndexSemantics(index *ModuleIndex) error {
	return newValidationErrors(validateModuleReferences(index.Modules, "modules"))
}

func validateModuleDocumentSemantics(document *ModuleDocument) error {
	violations := make([]Violation, 0)
	if document.ID != document.Owner+"/"+document.Name {
		violations = append(violations, Violation{
			Path:    "/id",
			Rule:    RuleIdentity,
			Message: "module ID must equal owner/name",
		})
	}

	seen := make(map[string]struct{}, len(document.Versions))
	latestFound := document.Latest == ""
	for index, version := range document.Versions {
		if _, exists := seen[version.Version]; exists {
			violations = append(violations, Violation{
				Path:    jsonPointer("versions", strconv.Itoa(index), "version"),
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("version %q is duplicated", version.Version),
			})
		}
		seen[version.Version] = struct{}{}

		if version.Version == document.Latest {
			latestFound = true
		}

		_, offset := version.PublishedAt.Zone()
		if version.PublishedAt.IsZero() || offset != 0 {
			violations = append(violations, Violation{
				Path:    jsonPointer("versions", strconv.Itoa(index), "publishedAt"),
				Rule:    RuleTimestamp,
				Message: "publication timestamp must be a non-zero UTC time",
			})
		}

		if err := validateReference(version.Href); err != nil {
			violations = append(violations, Violation{
				Path:    jsonPointer("versions", strconv.Itoa(index), "href"),
				Rule:    RuleReference,
				Message: err.Error(),
			})
		}
	}

	if !latestFound {
		violations = append(violations, Violation{
			Path:    "/latest",
			Rule:    RuleIdentity,
			Message: fmt.Sprintf("latest version %q is not listed", document.Latest),
		})
	}

	return newValidationErrors(violations)
}

func validateVersionDocumentSemantics(document *VersionDocument) error {
	violations := make([]Violation, 0)

	owner, name, _ := strings.Cut(document.ID, "/")
	manifest := &registryspec.ModuleManifest{
		Schema: registryspec.ModuleManifestSchemaV1,
		Owner:  owner,
		Name:   name,
		Source: registryspec.Source{
			Repository: document.Source.Repository,
			Path:       document.Source.Path,
		},
	}
	violations = appendRegistryViolations(violations, registryspec.ValidateModuleManifest(manifest), "")

	record := &registryspec.VersionRecord{
		Schema:  registryspec.VersionRecordSchemaV1,
		Version: document.Version,
		Tag:     "v" + document.Version,
		Commit:  document.Source.Commit,
	}
	violations = appendRegistryViolations(violations, registryspec.ValidateVersionRecord(record), "source")

	if err := gomodule.Check(document.Package.Path, "v"+document.Version); err != nil {
		violations = append(violations, Violation{
			Path:    "/package/path",
			Rule:    RulePackagePath,
			Message: err.Error(),
		})
	}

	for name, value := range document.Links {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			violations = append(violations, Violation{
				Path:    jsonPointer("links", name),
				Rule:    RuleReference,
				Message: "link must be an absolute HTTPS URL without credentials",
			})
		}
	}

	violations = append(violations, validateReferenceMap(document.Content, "content")...)

	return newValidationErrors(violations)
}

func validateCategoryIndexSemantics(index *CategoryIndex) error {
	violations := make([]Violation, 0)
	seen := make(map[string]struct{}, len(index.Categories))
	for itemIndex, category := range index.Categories {
		if _, exists := seen[category.ID]; exists {
			violations = append(violations, Violation{
				Path:    jsonPointer("categories", strconv.Itoa(itemIndex), "id"),
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("category ID %q is duplicated", category.ID),
			})
		}
		seen[category.ID] = struct{}{}

		if err := validateReference(category.Href); err != nil {
			violations = append(violations, Violation{
				Path:    jsonPointer("categories", strconv.Itoa(itemIndex), "href"),
				Rule:    RuleReference,
				Message: err.Error(),
			})
		}
	}

	return newValidationErrors(violations)
}

func validateCategoryDocumentSemantics(document *CategoryDocument) error {
	return newValidationErrors(validateModuleReferences(document.Modules, "modules"))
}

func validateModuleReferences(entries []ModuleIndexEntry, field string) []Violation {
	violations := make([]Violation, 0)
	seen := make(map[string]struct{}, len(entries))
	for index, entry := range entries {
		if _, exists := seen[entry.ID]; exists {
			violations = append(violations, Violation{
				Path:    jsonPointer(field, strconv.Itoa(index), "id"),
				Rule:    RuleDuplicate,
				Message: fmt.Sprintf("module ID %q is duplicated", entry.ID),
			})
		}
		seen[entry.ID] = struct{}{}

		if err := validateReference(entry.Href); err != nil {
			violations = append(violations, Violation{
				Path:    jsonPointer(field, strconv.Itoa(index), "href"),
				Rule:    RuleReference,
				Message: err.Error(),
			})
		}
	}

	return violations
}

func validateReferenceMap(references map[string]string, field string) []Violation {
	violations := make([]Violation, 0)
	for name, value := range references {
		if err := validateReference(value); err != nil {
			violations = append(violations, Violation{
				Path:    jsonPointer(field, name),
				Rule:    RuleReference,
				Message: err.Error(),
			})
		}
	}

	return violations
}

func validateReference(value string) error {
	reference, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("reference is not a valid URI reference")
	}

	if reference.User != nil || reference.RawQuery != "" || reference.Fragment != "" {
		return fmt.Errorf("reference must not contain credentials, a query, or a fragment")
	}

	return nil
}

func appendRegistryViolations(violations []Violation, err error, prefix string) []Violation {
	if err == nil {
		return violations
	}

	var validationErr *registryspec.ValidationErrors
	if !errors.As(err, &validationErr) {
		return append(violations, Violation{Path: jsonPointer(prefix), Rule: RuleSource, Message: err.Error()})
	}

	for _, violation := range validationErr.Violations {
		path := violation.Path
		if prefix != "" {
			if path == "/commit" {
				path = jsonPointer(prefix, "commit")
			} else if path != "/version" {
				path = jsonPointer(prefix) + path
			}
		}

		violations = append(violations, Violation{
			Path:    path,
			Rule:    RuleSource,
			Message: violation.Message,
		})
	}

	return violations
}

func noSemanticValidation[T any](*T) error {
	return nil
}

type offlineLoader struct{}

func (offlineLoader) Load(schemaURL string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", schemaURL)
}
