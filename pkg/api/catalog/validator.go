package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/pkg/validation"
	ferretschemas "github.com/MontFerret/specs/schemas"
)

const schemaPath = "registry/artifact/api-catalog/v1.json"

var (
	catalogSchemaOnce sync.Once
	catalogSchemaV1   *jsonschema.Schema
	catalogSchemaErr  error
)

// Validate validates a programmatically constructed API Catalog.
func Validate(catalog *Catalog) error {
	data, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("encode API Catalog: %w", err)
	}

	document, err := jsondocument.Decode(data)
	if err != nil {
		return fmt.Errorf("decode encoded API Catalog: %w", err)
	}

	if err := validateDocument(document); err != nil {
		return err
	}

	return validateSemantics(catalog)
}

func validateDocument(document any) error {
	if version, ok := documentSchemaVersion(document); ok && version > 0 && version != SchemaVersion {
		return &UnsupportedVersionError{Version: version}
	}

	schema, err := compiledSchema()
	if err != nil {
		return err
	}

	if err := schema.Validate(document); err != nil {
		var validationErr *jsonschema.ValidationError
		if !errors.As(err, &validationErr) {
			return fmt.Errorf("validate API Catalog schema: %w", err)
		}

		return validation.NewErrors(validation.ScopeAPICatalog, flattenSchemaErrors(validationErr))
	}

	return nil
}

func documentSchemaVersion(document any) (int, bool) {
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

func compiledSchema() (*jsonschema.Schema, error) {
	catalogSchemaOnce.Do(compileSchema)

	return catalogSchemaV1, catalogSchemaErr
}

func compileSchema() {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})

	for schemaID, resourcePath := range map[string]string{
		"https://schemas.ferretlang.org/common/identifier.json": "common/identifier.json",
		"https://schemas.ferretlang.org/common/namespace.json":  "common/namespace.json",
		"https://schemas.ferretlang.org/common/semver.json":     "common/semver.json",
		SchemaV1: schemaPath,
	} {
		data, err := ferretschemas.FS.ReadFile(resourcePath)
		if err != nil {
			catalogSchemaErr = fmt.Errorf("read embedded schema %q: %w", resourcePath, err)

			return
		}

		document, err := jsondocument.Decode(data)
		if err != nil {
			catalogSchemaErr = fmt.Errorf("decode embedded schema %q: %w", resourcePath, err)

			return
		}

		if err := compiler.AddResource(schemaID, document); err != nil {
			catalogSchemaErr = fmt.Errorf("register embedded schema %q: %w", resourcePath, err)

			return
		}
	}

	catalogSchemaV1, catalogSchemaErr = compiler.Compile(SchemaV1)
	if catalogSchemaErr != nil {
		catalogSchemaErr = fmt.Errorf("compile embedded API Catalog schema %q: %w", SchemaV1, catalogSchemaErr)
	}
}

func validateSemantics(catalog *Catalog) error {
	violations := make([]validation.Violation, 0)
	seenCategories := make(map[string]struct{}, len(catalog.Categories))
	seenFunctions := make(map[FunctionRef]string)

	for categoryIndex, category := range catalog.Categories {
		categoryPath := validation.JSONPointer("categories", strconv.Itoa(categoryIndex))
		if _, exists := seenCategories[category.ID]; exists {
			violations = append(violations, validation.Violation{
				Path:    categoryPath + "/id",
				Rule:    validation.RuleDuplicate,
				Message: fmt.Sprintf("category ID %q is duplicated", category.ID),
			})
		}

		seenCategories[category.ID] = struct{}{}

		if strings.TrimSpace(category.Title) == "" {
			violations = append(violations, validation.Violation{
				Path:    categoryPath + "/title",
				Rule:    validation.RuleSchema,
				Message: "category title must not be blank",
			})
		}

		if strings.TrimSpace(category.Description) == "" {
			violations = append(violations, validation.Violation{
				Path:    categoryPath + "/description",
				Rule:    validation.RuleSchema,
				Message: "category description must not be blank",
			})
		}

		for functionIndex, function := range category.Functions {
			functionPath := validation.JSONPointer("categories", strconv.Itoa(categoryIndex), "functions", strconv.Itoa(functionIndex))
			if previousCategory, exists := seenFunctions[function]; exists {
				violations = append(violations, validation.Violation{
					Path:    functionPath,
					Rule:    validation.RuleDuplicate,
					Message: fmt.Sprintf("function %q is already assigned to category %q", qualifiedName(function), previousCategory),
				})
			}

			seenFunctions[function] = category.ID

			if functionIndex > 0 && compareFunctionRefs(category.Functions[functionIndex-1], function) >= 0 {
				violations = append(violations, validation.Violation{
					Path:    functionPath,
					Rule:    validation.RuleSchema,
					Message: "category functions must be sorted by namespace and name in ascending lexical order",
				})
			}
		}
	}

	return validation.NewErrors(validation.ScopeAPICatalog, violations)
}

func compareFunctionRefs(left, right FunctionRef) int {
	if compared := strings.Compare(left.Namespace, right.Namespace); compared != 0 {
		return compared
	}

	return strings.Compare(left.Name, right.Name)
}

func qualifiedName(function FunctionRef) string {
	if function.Namespace == "" {
		return function.Name
	}

	return function.Namespace + "::" + function.Name
}

func flattenSchemaErrors(validationErr *jsonschema.ValidationError) []validation.Violation {
	if len(validationErr.Causes) > 0 {
		violations := make([]validation.Violation, 0, len(validationErr.Causes))

		for _, cause := range validationErr.Causes {
			violations = append(violations, flattenSchemaErrors(cause)...)
		}

		return violations
	}

	rule := validation.RuleSchema
	message := "document does not match the API Catalog schema"

	if validationErr.ErrorKind != nil {
		keywordPath := validationErr.ErrorKind.KeywordPath()
		if len(keywordPath) > 0 {
			rule = validation.Rule(keywordPath[len(keywordPath)-1])
		}

		output := validationErr.BasicOutput()
		if output.Error != nil {
			message = output.Error.String()
		}
	}

	return []validation.Violation{{
		Path:    validation.JSONPointer(validationErr.InstanceLocation...),
		Rule:    rule,
		Message: message,
	}}
}
