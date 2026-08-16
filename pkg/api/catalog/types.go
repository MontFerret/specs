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
		SchemaVersion  int        `json:"schemaVersion"`
		ID             string     `json:"id"`
		Version        string     `json:"version"`
		Categories     []Category `json:"categories"`
		NamespaceRoots []string   `json:"namespaceRoots"`
	}

	// Category groups global Ferret functions for documentation and navigation.
	Category struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Functions   []string `json:"functions"`
	}
)
