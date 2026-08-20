package artifact

import (
	"errors"

	ferretapi "github.com/MontFerret/specs/pkg/api"
)

// APISchemaV1 is the canonical Ferret API Reference v1 schema ID.
// Deprecated: use api.SchemaV1.
const APISchemaV1 = ferretapi.SchemaV1

const (
	// APITypeKindNamed identifies an open named Ferret semantic type.
	// Deprecated: use api.TypeKindNamed.
	APITypeKindNamed = ferretapi.TypeKindNamed
	// APITypeKindUnion identifies an ordered choice of semantic types.
	// Deprecated: use api.TypeKindUnion.
	APITypeKindUnion = ferretapi.TypeKindUnion
	// APITypeKindList identifies a homogeneous list element type.
	// Deprecated: use api.TypeKindList.
	APITypeKindList = ferretapi.TypeKindList
)

type (
	// APITypeKind identifies one recursive Ferret API type variant.
	// Deprecated: use api.TypeKind.
	APITypeKind = ferretapi.TypeKind
	// APIType describes a recursive Ferret-facing semantic type.
	// Deprecated: use api.Type.
	APIType = ferretapi.Type
	// APIReference is the Ferret API Reference v1 document.
	// Deprecated: use api.Reference.
	APIReference = ferretapi.Reference
	// APINamespace is one Ferret API namespace.
	// Deprecated: use api.Namespace.
	APINamespace = ferretapi.Namespace
	// APIFunction is one Ferret API function and its overloads.
	// Deprecated: use api.Function.
	APIFunction = ferretapi.Function
	// APIFunctionSignature is one Ferret API function signature.
	// Deprecated: use api.Signature.
	APIFunctionSignature = ferretapi.Signature
	// APIParameter is one Ferret-facing function parameter.
	// Deprecated: use api.Parameter.
	APIParameter = ferretapi.Parameter
	// APIReturn is one documented Ferret-facing result.
	// Deprecated: use api.Return.
	APIReturn = ferretapi.Return
	// APIThrownError is one documented Ferret-visible failure.
	// Deprecated: use api.Throw.
	APIThrownError = ferretapi.Throw
)

// ParseAPIReference parses and validates a Ferret API Reference artifact.
// Deprecated: use api.Parse.
func ParseAPIReference(data []byte) (*APIReference, error) {
	reference, err := ferretapi.Parse(data)

	return reference, compatibilityAPIError(err)
}

// ValidateAPIReference validates a programmatically constructed Ferret API Reference.
// Deprecated: use api.Validate.
func ValidateAPIReference(reference *APIReference) error {
	return compatibilityAPIError(ferretapi.Validate(reference))
}

func compatibilityAPIError(err error) error {
	var versionErr *ferretapi.UnsupportedVersionError
	if errors.As(err, &versionErr) {
		return &UnsupportedVersionError{Version: versionErr.Version}
	}

	return err
}
