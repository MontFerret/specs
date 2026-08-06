// Package schemas exposes the canonical Ferret schemas as an embedded file
// system so consumers can validate documents without network access.
package schemas

import "embed"

// FS contains every published or reserved schema in this module.
//
// Paths are relative to this package directory, for example
// "module-manifest/v1.json".
//
//go:embed common/*.json module-manifest/*.json plugin-manifest/*.json registry-entry/*.json
var FS embed.FS
