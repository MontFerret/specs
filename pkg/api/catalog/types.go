package catalog

const (
	// SchemaVersion is the supported API Catalog schema version.
	SchemaVersion = 1

	// SchemaV1 is the canonical schema ID for API Catalog v1.
	SchemaV1 = "https://schemas.ferretlang.org/registry/artifact/api-catalog/v1.json"
)

type (
	// Catalog describes the presentation structure paired with one API Reference.
	Catalog struct {
		SchemaVersion int        `json:"schemaVersion"`
		ID            string     `json:"id"`
		Version       string     `json:"version"`
		Categories    []Category `json:"categories"`
	}

	// Category groups Ferret functions for documentation and navigation.
	Category struct {
		ID          string        `json:"id"`
		Title       string        `json:"title"`
		Description string        `json:"description"`
		Functions   []FunctionRef `json:"functions"`
	}

	// FunctionRef identifies one API Reference function without redefining it.
	FunctionRef struct {
		Namespace string `json:"namespace"`
		Name      string `json:"name"`
	}
)
