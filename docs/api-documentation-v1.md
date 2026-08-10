# Ferret API Documentation v1

Ferret API documentation describes the Ferret-facing behavior of a registered
function. It is intentionally separate from its Go signature: registration can
rename a function, expose a different parameter model, or present Ferret types
that do not correspond one-to-one with Go types.

The canonical Go implementation is `github.com/MontFerret/specs/pkg/api`.
`api.ParseDocumentation` accepts a normalized documentation body without Go
comment delimiters (`//`, `/*`, and `*/`) or leading block-comment `*` markers.
Source analyzers remain responsible for comment normalization and for attaching
file, declaration, and source-position context to an error.

## Grammar

The supported annotations are:

```text
@param <name> {<type>} <description>
@return {<type>} <description>
@throws {<error>} <description>
@deprecated <description>
```

An annotation is recognized only when its tag begins at the first character of
a line and the tag is followed by whitespace or the end of the line. The
supported set is exactly `@param`, `@return`, `@throws`, and `@deprecated`.
Unknown tags and indented supported tags remain ordinary prose.

Parameter names use ASCII letters, digits, and underscores, must begin with a
letter or underscore, and must not be `_`. A documentation body must not repeat
a parameter name and may contain at most one `@return` and one `@deprecated`.
Every `@throws` annotation is retained in source order.

Type and error expressions are opaque, nonblank, single-line strings. Their
bytes inside the outer braces are preserved exactly, including spaces and
nested braces. The parser does not define a type-expression language or AST.
Descriptions are required and must be nonblank. A JSDoc-style dash separator,
such as `@param value {String} - Input.`, is invalid.

Structured annotations are single-line declarations. Continuation lines and
additional JSDoc tags are not supported.

## Prose and deprecation

Ordinary prose is preserved separately in `api.Documentation.Description`.
Valid annotation lines are removed while paragraph boundaries remain intact.
When structured `@deprecated` metadata exists, paragraphs beginning with the
standard Go `Deprecated:` marker are removed from the prose to avoid rendering
the same notice twice. Without `@deprecated`, those paragraphs remain prose.

For example, this normalized body:

```text
Decode decodes XML content into a normalized document object.

@param data {String|Binary} XML content.
@return {Object} Normalized XML document.
@throws {ParseError} XML input is malformed.
@deprecated Use Parse instead.
```

produces a description plus one ordered parameter, one return value, one thrown
error, and a deprecation message. The same values map directly to
`api.Signature` when a source analyzer constructs an API Reference.

## Errors

`api.ParseDocumentation` performs syntax and semantic validation. It returns a
`*api.DocumentationError` with:

- a stable `Kind`;
- a one-based `Line` within the normalized body;
- the original `Annotation` line; and
- a human-readable `Detail` containing the expected grammar or conflict.

The stable kinds distinguish malformed annotations, duplicate parameters,
multiple returns, and multiple deprecations. Callers should inspect the typed
error and add their own source context instead of parsing its message.

## API Reference v1

The public wire document uses the closed JSON Schema at:

```text
https://schemas.ferretlang.org/registry/artifact/api/v1.json
```

Its canonical embedded schema path is
`schemas/registry/artifact/api/v1.json`. `api.Parse` strictly decodes and fully
validates serialized documents; `api.Validate` validates constructed
`api.Reference` values. Both preserve Registry-artifact validation scope and
structured violations from `pkg/validation`.

Compatible additions to the wire contract remain schema version 1. An
incompatible wire change requires a new schema version and schema ID. The
structured documentation grammar may evolve independently when its normalized
output remains representable by the current wire contract.
