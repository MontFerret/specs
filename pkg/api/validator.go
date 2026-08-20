package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/MontFerret/specs/internal/jsondocument"
	"github.com/MontFerret/specs/internal/registryidentity"
	"github.com/MontFerret/specs/pkg/validation"
	ferretschemas "github.com/MontFerret/specs/schemas"
)

const schemaPath = "registry/artifact/api/v1.json"

var (
	apiSchemaOnce sync.Once
	apiSchemaV1   *jsonschema.Schema
	apiSchemaErr  error
)

// Validate validates a programmatically constructed Ferret API Reference.
func Validate(reference *Reference) error {
	data, err := json.Marshal(reference)
	if err != nil {
		return fmt.Errorf("encode Ferret API Reference: %w", err)
	}

	document, err := jsondocument.Decode(data)
	if err != nil {
		return fmt.Errorf("decode encoded Ferret API Reference: %w", err)
	}

	if err := validateDocument(document); err != nil {
		return err
	}

	return validateSemantics(reference)
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
			return fmt.Errorf("validate Ferret API Reference schema: %w", err)
		}

		return validation.NewErrors(validation.ScopeRegistryArtifact, flattenSchemaErrors(validationErr))
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
	apiSchemaOnce.Do(compileSchema)

	return apiSchemaV1, apiSchemaErr
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
			apiSchemaErr = fmt.Errorf("read embedded schema %q: %w", resourcePath, err)

			return
		}

		document, err := jsondocument.Decode(data)
		if err != nil {
			apiSchemaErr = fmt.Errorf("decode embedded schema %q: %w", resourcePath, err)

			return
		}

		if err := compiler.AddResource(schemaID, document); err != nil {
			apiSchemaErr = fmt.Errorf("register embedded schema %q: %w", resourcePath, err)

			return
		}
	}

	apiSchemaV1, apiSchemaErr = compiler.Compile(SchemaV1)
	if apiSchemaErr != nil {
		apiSchemaErr = fmt.Errorf("compile embedded Ferret API Reference schema %q: %w", SchemaV1, apiSchemaErr)
	}
}

func validateSemantics(reference *Reference) error {
	violations := make([]validation.Violation, 0)
	seenNamespaces := make(map[string]struct{}, len(reference.Namespaces))

	for namespaceIndex, namespace := range reference.Namespaces {
		namespacePath := validation.JSONPointer("namespaces", strconv.Itoa(namespaceIndex))
		if _, exists := seenNamespaces[namespace.Name]; exists {
			violations = append(violations, validation.Violation{
				Path:    namespacePath + "/name",
				Rule:    validation.RuleDuplicate,
				Message: fmt.Sprintf("namespace %q is duplicated", namespace.Name),
			})
		}

		seenNamespaces[namespace.Name] = struct{}{}
		seenFunctions := make(map[string]struct{}, len(namespace.Functions))

		for functionIndex, function := range namespace.Functions {
			functionPath := namespacePath + "/functions/" + strconv.Itoa(functionIndex)
			if _, exists := seenFunctions[function.Name]; exists {
				violations = append(violations, validation.Violation{
					Path:    functionPath + "/name",
					Rule:    validation.RuleDuplicate,
					Message: fmt.Sprintf("function %q is duplicated in namespace %q", function.Name, namespace.Name),
				})
			}

			seenFunctions[function.Name] = struct{}{}
			seenSignatures := make(map[string]struct{}, len(function.Signatures))

			for signatureIndex, signature := range function.Signatures {
				signaturePath := functionPath + "/signatures/" + strconv.Itoa(signatureIndex)
				seenParameters := make(map[string]struct{}, len(signature.Parameters))

				for parameterIndex, parameter := range signature.Parameters {
					parameterPath := signaturePath + "/parameters/" + strconv.Itoa(parameterIndex)
					if parameter.Name == "_" {
						violations = append(violations, validation.Violation{
							Path:    parameterPath + "/name",
							Rule:    validation.RuleSchema,
							Message: `parameter name "_" is reserved for generated argument names`,
						})
					}

					if _, exists := seenParameters[parameter.Name]; exists {
						violations = append(violations, validation.Violation{
							Path:    parameterPath + "/name",
							Rule:    validation.RuleDuplicate,
							Message: fmt.Sprintf("parameter %q is duplicated", parameter.Name),
						})
					}

					seenParameters[parameter.Name] = struct{}{}

					if parameter.Description != "" && strings.TrimSpace(parameter.Description) == "" {
						violations = append(violations, validation.Violation{
							Path:    parameterPath + "/description",
							Rule:    validation.RuleSchema,
							Message: "parameter description must not be blank",
						})
					}
				}

				if signature.Description != "" && strings.TrimSpace(signature.Description) == "" {
					violations = append(violations, validation.Violation{
						Path:    signaturePath + "/description",
						Rule:    validation.RuleSchema,
						Message: "signature description must not be blank",
					})
				}

				if signature.Return != nil {
					if strings.TrimSpace(signature.Return.Description) == "" {
						violations = append(violations, validation.Violation{
							Path:    signaturePath + "/return/description",
							Rule:    validation.RuleSchema,
							Message: "return description must not be blank",
						})
					}
				}

				for throwIndex, thrown := range signature.Throws {
					throwPath := signaturePath + "/throws/" + strconv.Itoa(throwIndex)
					if strings.TrimSpace(thrown.Error) == "" {
						violations = append(violations, validation.Violation{
							Path:    throwPath + "/error",
							Rule:    validation.RuleSchema,
							Message: "thrown error must not be blank",
						})
					}

					if strings.TrimSpace(thrown.Description) == "" {
						violations = append(violations, validation.Violation{
							Path:    throwPath + "/description",
							Rule:    validation.RuleSchema,
							Message: "thrown error description must not be blank",
						})
					}
				}

				if signature.Deprecated != "" && strings.TrimSpace(signature.Deprecated) == "" {
					violations = append(violations, validation.Violation{
						Path:    signaturePath + "/deprecated",
						Rule:    validation.RuleSchema,
						Message: "deprecation description must not be blank",
					})
				}

				key := strconv.Itoa(len(signature.Parameters))
				if signature.Variadic {
					key = "variadic"
				}

				if _, exists := seenSignatures[key]; exists {
					violations = append(violations, validation.Violation{
						Path:    signaturePath,
						Rule:    validation.RuleDuplicate,
						Message: fmt.Sprintf("function %q has more than one %s signature", function.Name, key),
					})
				}

				seenSignatures[key] = struct{}{}
			}
		}
	}

	return validation.NewErrors(validation.ScopeRegistryArtifact, violations)
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
	message := "document does not match the registry artifact schema"

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

	if rule == validation.RulePattern && len(validationErr.InstanceLocation) == 1 && validationErr.InstanceLocation[0] == "id" {
		message = registryidentity.CoordinateMessage
	}

	return []validation.Violation{{
		Path:    validation.JSONPointer(validationErr.InstanceLocation...),
		Rule:    rule,
		Message: message,
	}}
}

type offlineLoader struct{}

func (offlineLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema loading is disabled: %s", url)
}

func (offlineLoader) ReadFile(url string) ([]byte, error) {
	return nil, &fs.PathError{Op: "read", Path: url, Err: fs.ErrNotExist}
}
