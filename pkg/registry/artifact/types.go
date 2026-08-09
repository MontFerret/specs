// Package artifact defines the portable wire contract for generated Ferret
// Registry distribution artifacts.
package artifact

import (
	"encoding/json"
	"time"
)

const (
	// SchemaVersion is the supported generated artifact schema version.
	SchemaVersion = 1

	RootSchemaV1          = "https://schemas.ferretlang.org/registry/artifact/root/v1.json"
	ModuleIndexSchemaV1   = "https://schemas.ferretlang.org/registry/artifact/module-index/v1.json"
	ModuleSchemaV1        = "https://schemas.ferretlang.org/registry/artifact/module/v1.json"
	VersionSchemaV1       = "https://schemas.ferretlang.org/registry/artifact/version/v1.json"
	APISchemaV1           = "https://schemas.ferretlang.org/registry/artifact/api/v1.json"
	CategoryIndexSchemaV1 = "https://schemas.ferretlang.org/registry/artifact/category-index/v1.json"
	CategorySchemaV1      = "https://schemas.ferretlang.org/registry/artifact/category/v1.json"
	PluginIndexSchemaV1   = "https://schemas.ferretlang.org/registry/artifact/plugin-index/v1.json"

	ArtifactKeyCategories = "categories"
	ArtifactKeyModules    = "modules"
	ArtifactKeyPlugins    = "plugins"

	ContentKeyDocumentation     = "documentation"
	ContentKeyDocumentationHTML = "documentationHtml"
	ContentKeyAPI               = "api"
)

type (
	// RootIndex discovers the artifact indexes in a generated distribution.
	RootIndex struct {
		SchemaVersion int               `json:"schemaVersion"`
		Artifacts     map[string]string `json:"artifacts"`
	}

	// ModuleIndex lists compact references to every registered module.
	ModuleIndex struct {
		SchemaVersion int                `json:"schemaVersion"`
		Modules       []ModuleIndexEntry `json:"modules"`
	}

	// ModuleIndexEntry is the compact module representation shared by indexes.
	ModuleIndexEntry struct {
		ID     string `json:"id"`
		Latest string `json:"latest,omitempty"`
		Href   string `json:"href"`
	}

	// CategoryIndex lists the available registry categories.
	CategoryIndex struct {
		SchemaVersion int                  `json:"schemaVersion"`
		Categories    []CategoryIndexEntry `json:"categories"`
	}

	// CategoryIndexEntry is one category discovery reference.
	CategoryIndexEntry struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Count int    `json:"count"`
		Href  string `json:"href"`
	}

	// CategoryDocument lists the modules belonging to one category.
	CategoryDocument struct {
		SchemaVersion int                `json:"schemaVersion"`
		Category      CategorySummary    `json:"category"`
		Modules       []ModuleIndexEntry `json:"modules"`
	}

	// CategorySummary identifies one category.
	CategorySummary struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	// PluginIndex reserves the generated plugin discovery surface.
	PluginIndex struct {
		SchemaVersion int               `json:"schemaVersion"`
		Plugins       []json.RawMessage `json:"plugins"`
	}

	// ModuleDocument contains public metadata and versions for one module.
	ModuleDocument struct {
		SchemaVersion int                     `json:"schemaVersion"`
		ID            string                  `json:"id"`
		Owner         string                  `json:"owner"`
		Name          string                  `json:"name"`
		Description   string                  `json:"description"`
		Latest        string                  `json:"latest,omitempty"`
		Versions      []ModuleDocumentVersion `json:"versions"`
	}

	// ModuleDocumentVersion references one immutable version artifact.
	ModuleDocumentVersion struct {
		Version     string    `json:"version"`
		PublishedAt time.Time `json:"publishedAt"`
		Href        string    `json:"href"`
	}

	// VersionDocument contains immutable public metadata for one module version.
	VersionDocument struct {
		SchemaVersion int               `json:"schemaVersion"`
		ID            string            `json:"id"`
		Version       string            `json:"version"`
		Description   string            `json:"description"`
		Namespace     string            `json:"namespace"`
		Ferret        string            `json:"ferret,omitempty"`
		License       string            `json:"license"`
		Links         map[string]string `json:"links,omitempty"`
		Source        VersionSource     `json:"source"`
		Package       VersionPackage    `json:"package"`
		Content       map[string]string `json:"content"`
	}

	// VersionPackage identifies the installable package for a module version.
	VersionPackage struct {
		Path string `json:"path"`
	}

	// VersionSource identifies the immutable Git source for a module version.
	VersionSource struct {
		Repository string `json:"repository"`
		Path       string `json:"path,omitempty"`
		Commit     string `json:"commit"`
	}

	// APIReference contains the statically derived Ferret-facing API for one module version.
	APIReference struct {
		SchemaVersion int            `json:"schemaVersion"`
		ID            string         `json:"id"`
		Version       string         `json:"version"`
		Namespaces    []APINamespace `json:"namespaces"`
	}

	// APINamespace contains the functions registered in one Ferret namespace.
	// An empty name identifies the global namespace.
	APINamespace struct {
		Name      string        `json:"name"`
		Functions []APIFunction `json:"functions"`
	}

	// APIFunction contains every registered overload for one Ferret function name.
	APIFunction struct {
		Name       string                 `json:"name"`
		Signatures []APIFunctionSignature `json:"signatures"`
	}

	// APIFunctionSignature describes one fixed-arity or variadic function definition.
	APIFunctionSignature struct {
		Parameters    []string `json:"parameters"`
		Variadic      bool     `json:"variadic,omitempty"`
		Documentation string   `json:"documentation,omitempty"`
	}
)
