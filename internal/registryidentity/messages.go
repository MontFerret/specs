// Package registryidentity contains shared validation diagnostics for Ferret
// distribution identities.
package registryidentity

const (
	// CoordinateMessage describes the canonical owner/name spelling contract.
	CoordinateMessage = "module identity must use canonical lowercase owner/name spelling; each segment must start and end with a lowercase letter or digit"
	// OwnerMessage describes the canonical Registry owner spelling contract.
	OwnerMessage = "registry owner must use canonical lowercase spelling"
	// ModuleNameMessage describes the canonical Registry module name spelling contract.
	ModuleNameMessage = "registry module name must use canonical lowercase spelling"
)
