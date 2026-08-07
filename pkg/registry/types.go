// Package registry parses and validates Ferret Registry v1 documents.
package registry

import "time"

const (
	// ModuleManifestSchemaV1 is the canonical Registry Module Manifest v1 schema identifier.
	ModuleManifestSchemaV1 = "https://schemas.ferretlang.org/registry/module/v1.json"
	// VersionRecordSchemaV1 is the canonical Registry Version Record v1 schema identifier.
	VersionRecordSchemaV1 = "https://schemas.ferretlang.org/registry/version/v1.json"
)

// ModuleManifest identifies a module in a registry and its public Git source.
type ModuleManifest struct {
	Schema string `json:"$schema"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`
	Source Source `json:"source"`
}

// Source identifies the public Git repository and optional module root.
type Source struct {
	Repository string `json:"repository"`
	Path       string `json:"path,omitempty"`
}

// VersionRecord pins one module version to an immutable Git release identity.
type VersionRecord struct {
	Schema      string     `json:"$schema"`
	Version     string     `json:"version"`
	Tag         string     `json:"tag"`
	Commit      string     `json:"commit"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
}
