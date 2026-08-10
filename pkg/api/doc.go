// Package api defines portable Ferret-facing API metadata.
//
// Go signatures describe how a host calls module code, but they do not always
// describe the FQL parameters, return values, failures, and deprecation state
// exposed to Ferret users. This package provides the shared API Reference v1
// wire model and parses the structured documentation used to supply that
// Ferret-facing metadata.
//
// ParseDocumentation accepts normalized documentation-body text. Callers that
// inspect Go source are responsible for removing line or block comment markers
// and for attaching parser line numbers to source positions.
package api
