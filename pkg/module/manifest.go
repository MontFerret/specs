// Package module parses and validates Ferret Module Manifest documents.
package module

// SchemaV1 is the canonical schema identifier for Ferret Module Manifest v1.
const SchemaV1 = "https://schemas.ferretlang.org/module/v1.json"

// Manifest describes the distribution and runtime surface of a Ferret module.
// Version and version-range fields retain their original textual form.
type Manifest struct {
	Schema        string            `json:"$schema" yaml:"$schema"`
	Name          string            `json:"name" yaml:"name"`
	Namespace     string            `json:"namespace" yaml:"namespace"`
	Version       string            `json:"version" yaml:"version"`
	Description   string            `json:"description" yaml:"description"`
	License       string            `json:"license" yaml:"license"`
	Authors       []Author          `json:"authors,omitempty" yaml:"authors,omitempty"`
	Documentation string            `json:"documentation" yaml:"documentation"`
	Repository    string            `json:"repository,omitempty" yaml:"repository,omitempty"`
	Links         map[string]string `json:"links,omitempty" yaml:"links,omitempty"`
	Compatibility *Compatibility    `json:"compatibility,omitempty" yaml:"compatibility,omitempty"`
	Dependencies  []Dependency      `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Keywords      []string          `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Categories    []string          `json:"categories,omitempty" yaml:"categories,omitempty"`
	Exports       *Exports          `json:"exports,omitempty" yaml:"exports,omitempty"`
}

// Author identifies a module author and optional contact information.
type Author struct {
	Name  string `json:"name" yaml:"name"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
	URL   string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Compatibility declares the supported Ferret runtime versions.
type Compatibility struct {
	Ferret string `json:"ferret" yaml:"ferret"`
}

// Dependency declares a required runtime dependency.
type Dependency struct {
	Module  string `json:"module" yaml:"module"`
	Version string `json:"version" yaml:"version"`
}

// Exports describes the namespaces and dialects exposed by a module.
type Exports struct {
	Namespaces []NamespaceExport `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	Dialects   []string          `json:"dialects,omitempty" yaml:"dialects,omitempty"`
}

// NamespaceExport describes members exposed in one Ferret namespace.
type NamespaceExport struct {
	Name      string   `json:"name" yaml:"name"`
	Functions []string `json:"functions,omitempty" yaml:"functions,omitempty"`
	Types     []string `json:"types,omitempty" yaml:"types,omitempty"`
	Constants []string `json:"constants,omitempty" yaml:"constants,omitempty"`
}
