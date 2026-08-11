package api

const (
	// SchemaVersion is the supported Ferret API Reference schema version.
	SchemaVersion = 1

	// SchemaV1 is the canonical schema ID for Ferret API Reference v1.
	SchemaV1 = "https://schemas.ferretlang.org/registry/artifact/api/v1.json"

	// IndexSchemaVersion is the supported Ferret API Reference Index schema version.
	IndexSchemaVersion = 1

	// IndexSchemaV1 is the canonical schema ID for Ferret API Reference Index v1.
	IndexSchemaV1 = "https://schemas.ferretlang.org/registry/artifact/api-index/v1.json"
)

type (
	// Index discovers immutable API Reference artifacts by version.
	Index struct {
		// SchemaVersion identifies the API Reference Index wire contract.
		SchemaVersion int `json:"schemaVersion"`
		// Latest is the greatest stable version, omitted when only prereleases exist.
		Latest string `json:"latest,omitempty"`
		// Versions contains every published API Reference in descending semantic-version order.
		Versions []IndexVersion `json:"versions"`
	}

	// IndexVersion links one semantic version to its authoritative API Reference location.
	IndexVersion struct {
		// Version is the immutable API Reference version.
		Version string `json:"version"`
		// Href is the URI reference consumers follow to load the API Reference.
		Href string `json:"href"`
	}

	// Reference contains the statically derived Ferret-facing API for one module version.
	Reference struct {
		// SchemaVersion identifies the API Reference wire contract.
		SchemaVersion int `json:"schemaVersion"`
		// ID is the canonical lowercase owner/module coordinate.
		ID string `json:"id"`
		// Version is the immutable module version described by this reference.
		Version string `json:"version"`
		// Namespaces contains the registered Ferret namespaces in deterministic order.
		Namespaces []Namespace `json:"namespaces"`
	}

	// Namespace contains the functions registered in one Ferret namespace.
	// An empty name identifies the global namespace.
	Namespace struct {
		// Name is the case-sensitive Ferret namespace or an empty global namespace.
		Name string `json:"name"`
		// Functions contains the registered functions in deterministic order.
		Functions []Function `json:"functions"`
	}

	// Function contains every registered overload for one Ferret function name.
	Function struct {
		// Name is the case-sensitive Ferret function name.
		Name string `json:"name"`
		// Signatures contains every distinct fixed or variadic signature.
		Signatures []Signature `json:"signatures"`
	}

	// Signature describes one fixed-arity or variadic Ferret function definition.
	Signature struct {
		// Parameters contains the Ferret-facing parameters in call order.
		Parameters []Parameter `json:"parameters"`
		// Variadic reports whether registration exposes a variadic signature.
		Variadic bool `json:"variadic,omitempty"`
		// Description contains ordinary declaration prose without structured annotations.
		Description string `json:"description,omitempty"`
		// Return describes the Ferret-facing result when one is documented.
		Return *Return `json:"return,omitempty"`
		// Throws contains documented Ferret-visible failures in source order.
		Throws []Throw `json:"throws,omitempty"`
		// Deprecated contains the structured deprecation message when present.
		Deprecated string `json:"deprecated,omitempty"`
	}

	// Parameter describes one Ferret-facing function parameter.
	Parameter struct {
		// Name is the Ferret-facing parameter name.
		Name string `json:"name"`
		// Type is an opaque Ferret type expression preserved as authored.
		Type string `json:"type,omitempty"`
		// Description explains the parameter's Ferret-facing meaning.
		Description string `json:"description,omitempty"`
	}

	// Return describes a documented Ferret-facing function result.
	Return struct {
		// Type is an opaque Ferret type expression preserved as authored.
		Type string `json:"type"`
		// Description explains the result's Ferret-facing meaning.
		Description string `json:"description"`
	}

	// Throw describes one documented Ferret-visible function failure.
	Throw struct {
		// Error is an opaque Ferret error expression preserved as authored.
		Error string `json:"error"`
		// Description explains when the failure is visible to a Ferret caller.
		Description string `json:"description"`
	}

	// Documentation contains the Ferret-facing metadata parsed from one declaration comment.
	Documentation struct {
		// Description contains ordinary prose with structured annotation lines removed.
		Description string
		// Parameters contains authored parameters in declaration order.
		Parameters []Parameter
		// Return contains the single authored return value when present.
		Return *Return
		// Throws contains every authored failure in declaration order.
		Throws []Throw
		// Deprecated contains the authored structured deprecation message.
		Deprecated string
	}
)
