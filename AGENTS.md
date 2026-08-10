# AGENTS.md

This file is the canonical operating guide for coding agents working in this
repository. It applies to the Ferret specifications, schemas, validation
libraries, and `ferret-spec` CLI. If repository documentation conflicts with
this file, prefer `go.mod`, `Makefile`, and `.github/workflows/ci.yml` for the
toolchain, commands, and CI behavior, and prefer the checked-in schemas and
tests for contract behavior.

## Repo snapshot

* Module path: `github.com/MontFerret/specs`
* Go version: 1.25+
* Primary purpose: define portable, machine-readable contracts for the Ferret
  ecosystem and provide offline Go validation for those contracts.
* Authored JSON Schemas live under `schemas/` and use JSON Schema Draft 2020-12.
* Public Go packages live under `pkg/`; shared implementation-only helpers live
  under `internal/`.
* `cmd/ferret-spec` is the offline manifest validation CLI.
* `scripts/` contains formatting and GitHub Pages artifact helpers.
* The default Make target is `make check`, which runs formatting checks, vet,
  and tests.

## Architectural mental model

Specs turns authored contracts into offline validation APIs.

Primary validation flow:

```text
authored schema
    -> embedded schema resource
    -> strict document decoding
    -> JSON Schema validation
    -> typed Go value
    -> semantic validation
    -> validated value or structured violations
```

Programmatically constructed values enter at the typed Go value boundary, are
converted to a schema-compatible document, and then pass through schema and
semantic validation.

Agents should reason about changes by contract layer and ownership boundary:

* Document shape belongs in JSON Schema.
* Public in-memory representations belong in the matching `pkg/` package.
* Constraints that JSON Schema cannot express clearly belong in semantic
  validation in the matching package.
* Shared validation vocabulary and error structure belong in `pkg/validation`.
* Strict JSON document mechanics belong in `internal/jsondocument`.
* Distribution identity messages shared by manifest and Registry validation
  belong in `internal/registryidentity`.
* Transport, Git, filesystem layout, publication, and cross-document behavior
  do not belong in this repository.

## Canonical invariants

* Checked-in files under `schemas/` are the source of truth for schema behavior.
* Every schema declares Draft 2020-12 and has a canonical ID under
  `https://schemas.ferretlang.org/`.
* Validation must remain offline. Embedded validators must not fetch schema
  references from the network.
* Contract objects are closed unless the schema explicitly defines a map-like
  extension point. Unknown fields should fail rather than become silent data.
* Parsing accepts exactly one document and rejects duplicate object keys.
  Registry source documents and Registry artifacts are JSON-only; module
  manifests may be supplied as YAML or JSON.
* Public ingestion APIs validate by default. Do not return partially decoded or
  schema-only values from `Parse`, `Load`, or `LoadFile` entry points.
* Schema validation precedes conversion to a public typed value, and semantic
  validation runs after conversion.
* Structured validation errors use `pkg/validation`. Preserve the scope, rule
  strings, RFC 6901 JSON Pointer paths, messages, JSON shape, and deterministic
  violation ordering when behavior is not intentionally changing.
* `name` and Registry `owner`/`name` fields are canonical lowercase
  distribution identities. They are distinct from the independent,
  case-sensitive FQL runtime namespace.
* Canonical distribution identities are rejected when mis-cased; validators do
  not silently normalize persisted identities.
* `ferret.yaml` is the canonical repository-level Module Manifest filename.
  JSON remains an accepted explicitly supplied serialization, not a second
  repository filename convention.
* `plugin/v1.json` and `registry/v1.json` are reserved placeholders and must
  reject every document until those contracts are intentionally defined.
* Generated Registry artifact types describe portable wire documents and their
  same-document semantics. They do not define how a registry is stored,
  generated, navigated, or published.

## Ownership boundary with the wider Ferret ecosystem

This repository owns:

* portable JSON Schemas and stable schema identifiers;
* public Go wire types that correspond to those schemas;
* strict parsing and offline schema validation;
* semantic validation that depends only on one supplied document or typed
  value;
* shared validation errors, rule identifiers, and JSON Pointer construction;
* the portable output contract of `ferret-spec`.

This repository does not own:

* Registry checkout layout or filesystem traversal;
* remote Git access, tag or commit inspection, credentials, or transport;
* cross-document Registry consistency and navigation;
* Registry distribution generation or publication history policy;
* module installation, dependency resolution, or build behavior;
* hosted schema DNS, Pages settings, or other external infrastructure.

Those behaviors belong to consumers such as Barn or the Ferret CLI. Do not move
them into Specs merely because Specs defines a document they consume.

## Package and asset map

Agents should begin with the package or asset whose responsibility owns the
requested behavior.

### Canonical schemas and embedding

* `schemas/`
    * Owns every authored, published, or reserved JSON Schema.
    * `common/` contains reusable primitives such as identifiers, namespaces,
      semantic versions, version ranges, SPDX expressions, and URLs.
    * `module/` contains the Module Manifest contract.
    * `registry/` contains Registry source-record contracts.
    * `registry/artifact/` contains generated distribution wire contracts.
    * Paths and `$id` values must stay aligned.
* `schemas/embed.go`
    * Exposes the canonical schema set through `schemas.FS` for offline use.
    * The `go:embed` patterns must include every supported or reserved schema.
    * Do not add a schema without embedding and compilation coverage.
* `schemas/embed_test.go`
    * Verifies Draft 2020-12 declarations, canonical IDs, offline compilation,
      reserved placeholder behavior, and important embedding expectations.

### Public validation packages

* `pkg/module`
    * Owns Module Manifest v1 types, constants, JSON/YAML parsing, schema
      validation, and module-local semantic rules.
    * `LoadFile`, `Load`, and `Parse` return only fully validated manifests.
    * `Validate` applies the same schema and semantic contract to constructed
      values.
    * Owns the `ferret.yaml` filename constant and the public schema ID.
* `pkg/registry`
    * Owns Registry Module Manifest and Version Record source contracts.
    * Parses exactly one strict JSON document and validates it offline.
    * Owns same-record rules such as identity spelling, public repository URL
      shape, normalized source paths, strict versions, Git tag form, commit
      shape, and publication timestamp shape.
    * Does not inspect repositories or prove that a tag resolves to a commit.
* `pkg/registry/artifact`
    * Owns Registry artifact schema IDs, wire types, parsing, validation, and
      same-document reference rules.
    * Keeps schema version handling explicit; unsupported positive versions use
      `UnsupportedVersionError` rather than a shared validation violation.
    * Does not own generation, hosting, route resolution, or consistency across
      separate artifact files.
* `pkg/validation`
    * Owns `Scope`, `Rule`, `Violation`, `Errors`, `NewErrors`, and
      `JSONPointer`.
    * `NewErrors` returns `nil` for no violations and sorts violations by path,
      rule, then message.
    * An empty JSON Pointer identifies the document root.
    * Add shared rule identifiers here instead of duplicating domain-local rule
      strings.

### Internal helpers

* `internal/jsondocument`
    * Owns strict decoding of exactly one JSON value, duplicate-key rejection,
      number preservation, and conversion from a decoded document into a typed
      value.
    * Keep it format-neutral within JSON; YAML support remains module-specific.
* `internal/registryidentity`
    * Owns shared user-facing messages for canonical Registry and distribution
      identity failures.
    * It is implementation support, not a public validation API.

### CLI, documentation, fixtures, and automation

* `cmd/ferret-spec`
    * Owns command parsing, input orchestration, text rendering, stable JSON
      reports, and exit-code behavior.
    * It delegates manifest validation to `pkg/module`; it must not duplicate
      schema or semantic rules.
* `docs/`
    * Owns human-facing contract documentation that supplements the schemas.
    * Documentation must agree with current schemas and public Go behavior.
* `testdata/`
    * Owns canonical valid and invalid fixtures used to demonstrate observable
      contract behavior.
* `scripts/`
    * Owns substantial POSIX shell logic for Go/JSON formatting checks and
      Pages artifact construction.
    * Keep direct Go commands and Make target composition in `Makefile`.
* `.github/workflows/ci.yml`
    * Owns the authoritative CI job layout and Pages deployment gates.
    * Build, validation, test-matrix, and race jobs must remain independently
      visible unless a workflow change explicitly redesigns them.

## Where to start by task

* Add or change a document field:
    * edit the owning schema;
    * update the corresponding public Go type;
    * add or update valid, invalid, strict-parsing, and round-trip coverage;
    * update semantic validation only when the new rule is not purely
      structural;
    * update README or `docs/` when the public contract changes.
* Add or change a semantic rule:
    * keep the rule in the owning `pkg/` package;
    * use shared rule vocabulary and JSON Pointer construction;
    * return all independent violations that can be reported in one pass;
    * test path, rule, message, scope, and deterministic ordering.
* Add a new schema family or version:
    * choose a canonical versioned path and matching `$id`;
    * update `schemas/embed.go` if existing patterns do not cover it;
    * add public constants/types/parsers only when a Go consumption surface is
      required;
    * extend offline compilation and Pages artifact coverage.
* Change parsing behavior:
    * preserve exactly-one-document and duplicate-key rejection;
    * keep JSON number handling stable;
    * validate before typed conversion;
    * test malformed, duplicate, trailing-document, unknown-field, and nil
      cases.
* Change Registry source contracts:
    * work in `schemas/registry` and `pkg/registry`;
    * keep Git and filesystem verification outside Specs;
    * preserve the distinction between source-record validation and repository
      truth.
* Change Registry artifact contracts:
    * work in `schemas/registry/artifact` and `pkg/registry/artifact`;
    * keep wire types and schemas coordinated;
    * distinguish unsupported schema versions from malformed v1 documents;
    * limit semantic validation to the current artifact document.
* Change `ferret-spec` behavior:
    * inspect `cmd/ferret-spec/main.go` and its tests;
    * preserve output-stream, report-order, aggregate-status, and exit-code
      contracts;
    * continue processing every supplied input unless argument parsing itself
      fails.
* Change schema publication:
    * inspect `scripts/build-pages.sh` and the deploy job together;
    * keep the build deterministic and restricted to schema JSON files;
    * verify the generated artifact in a fresh temporary destination.

## Schema design and compatibility rules

* Schema paths are versioned by major version. Do not repurpose a versioned ID
  for an incompatible contract.
* Within v1, an updated validator must continue accepting documents that were
  valid under an earlier v1 contract unless the task explicitly authorizes a
  breaking correction and release strategy.
* Because objects are closed, adding an optional field can still be rejected by
  older embedded validators. Treat additive schema changes as coordinated
  ecosystem changes, not as automatically safe wire evolution.
* Keep object closure explicit with `additionalProperties: false` where the
  object is not intentionally map-like.
* Prefer references to canonical `common/` schemas over copying patterns and
  constraints across schemas.
* Use JSON Schema for document shape, required members, scalar constraints,
  formats, and closed-object behavior. Use Go semantic validation for rules
  such as duplicate logical entries, identity relationships, namespace
  containment, semantic version range parsing, and SPDX expression validation.
* Preserve authored JSON key order when editing schemas. Formatting uses `jq`
  with two-space indentation and must not become a semantic rewrite.
* Do not add compatibility aliases, legacy fields, silent normalization, or
  permissive fallback parsing unless the task explicitly defines that public
  compatibility contract.

## Validation error contract

Validation failures are observable public behavior.

* Use `*validation.Errors` for schema and semantic violations so callers can
  inspect them with `errors.As`.
* Keep operational failures, internal failures, and unsupported artifact
  versions distinguishable from document violations.
* Use RFC 6901 JSON Pointers. Build paths from unescaped tokens with
  `validation.JSONPointer`; do not hand-roll pointer escaping.
* Reuse existing rule strings when the meaning is unchanged.
* Messages should explain the violated contract without exposing
  implementation details from the schema library.
* Preserve multi-violation reporting when validators can safely identify
  independent failures in one pass.
* Tests should assert structured path, rule, and message behavior, not only the
  flattened `error.Error()` string.

## CLI output contract

The CLI has both human-facing and machine-facing output surfaces.

* `ferret-spec validate module` requires explicit file arguments; it does not
  discover manifests implicitly.
* `-` reads standard input and may appear at most once.
* Text mode writes valid results to standard output and violations or
  operational errors to standard error.
* JSON mode writes exactly one versioned report to standard output. Do not mix
  progress, diagnostics, or informational prose into machine-readable stdout.
* Results remain in supplied input order, while violations within a result use
  the deterministic shared ordering.
* Status values are `valid`, `invalid`, and `error`.
* Exit code `0` means all inputs are valid, `1` means at least one input is
  invalid, and `2` means usage, I/O, output, or internal failure.
* Operational errors take aggregate precedence over invalid results, but all
  supplied inputs are processed.

## Generated and published assets

* `schemas/` is authored source. A generated Pages directory is an artifact,
  not an independent source tree; recreate it with `scripts/build-pages.sh`
  rather than editing it by hand.
* `scripts/build-pages.sh` requires a fresh destination, copies only schema JSON
  files, and emits a deterministic sorted index.
* The GitHub Pages deploy runs only from `main` after build, validation, test,
  and race jobs succeed.
* Repository validation is offline. Do not make tests depend on the public
  schema host being reachable.
* Do not report hosted URLs or deployment state as verified solely because the
  checked-in schemas and Pages artifact build succeed.

## Public API rules

* Treat `schemas.FS` and every exported symbol under `pkg/` as API-sensitive.
* Do not export new symbols merely to share implementation details across
  packages; prefer an owning public package or an `internal/` helper according
  to the boundary map.
* Keep schema constants, typed fields, JSON/YAML tags, parsing APIs, and
  validation behavior coordinated.
* Public parse/load APIs should return validated values or an error, never a
  partially valid value.
* Preserve established package names and public types unless the task
  explicitly authorizes a breaking migration.

## Go type and file structure rules

These rules are mandatory unless the task explicitly requires otherwise.

* Do not define multiple method-bearing structs in the same `.go` file.
* Prefer declaring a method-bearing struct as a standalone
  `type Name struct { ... }`.
* A method-bearing struct should usually live in its own file, named after the
  primary type or responsibility whenever practical, for example:
    * `application.go` for `application`;
    * `report.go` for `validationReport`;
    * `errors.go` for `Errors`.
* Grouped `type ( ... )` declarations are allowed for interfaces, passive
  data-only structs, and small related helper or value types from one narrow
  concern.
* A grouped declaration may contain exactly one method-bearing struct when it
  is the only behavioral type in the file and the other types are passive
  helpers from the same concern.
* Do not use grouped declarations to hide multiple substantial behavioral
  types.
* If a helper gains methods and would create a second method-bearing struct in
  the file, extract it immediately.
* Keep methods with their struct unless there is a strong, explicit reason to
  split by concern.
* Do not place a new method-bearing struct in an existing file merely because
  it compiles.

Allowed:

```go
type (
	validationStatus string

	validationReport struct {
		Status  validationStatus
		Results []validationResult
	}

	validationResult struct {
		Source string
		Status validationStatus
	}
)
```

Avoid:

```go
type (
	application struct {
		// ...
	}

	validationReport struct {
		// ...
	}
)
```

The goal is to keep behavioral ownership obvious while still allowing passive,
closely related wire and report types to remain together.

## Function and method ownership rules

These rules are mandatory unless the task explicitly requires otherwise.

* A file centered on a method-bearing type contains the type, closely related
  constants, its methods, and constructors only.
* Do not mix unrelated package-level helpers into a type-centered file.
* Constructors are the normally allowed package-level functions in
  type-centered files; unrelated package functions belong elsewhere.
* If logic belongs to the primary type, implement it as a method.
* If logic is genuinely package-level, place it in a helper-focused file.
* Prefer package-level functions only when there is no natural owning type or
  the behavior is genuinely package-level.
* A file containing methods plus non-constructor package-level functions is
  usually a structure violation and should be refactored.
* Keep related package-level functions together by responsibility. For example,
  JSON Pointer construction belongs in `json_pointer.go`, not in the file
  centered on `Errors`.
* Do not fragment a cohesive behavioral type across many files merely to make
  individual files shorter.

## Comment rules for functions and methods

* Do not comment every function or method by default.
* Exported functions and methods should usually have doc comments, especially
  when they define parsing, validation, wire, error, or CLI contracts.
* Comment unexported functions and methods only when they carry non-obvious
  semantics, invariants, side effects, ownership, cleanup, ordering, or failure
  behavior.
* Explain intent, contracts, invariants, or observable behavior rather than
  restating names and signatures.
* Prefer comments about strict decoding, offline behavior, error structure, and
  compatibility over implementation narration.
* Avoid comment wallpaper. Dense, meaningful comments are better than
  mechanical documentation.

Preferred:

```go
// JSONPointer builds an RFC 6901 JSON Pointer from unescaped reference tokens.
func JSONPointer(parts ...string) string
```

Preferred for a behavioral contract:

```go
// UnsupportedVersionError reports a positive artifact schema version other than v1.
type UnsupportedVersionError struct {
	Version int
}
```

Avoid:

```go
// Error returns an error string.
func (e *Errors) Error() string
```

## Go control-flow spacing rules

These rules are mandatory for handwritten Go code. Blank lines should separate
logical units and make control-flow boundaries visually obvious.

### Immediate producer and check

A declaration, assignment, function call, type assertion, lookup, parse
operation, or similar statement should remain directly adjacent to a following
`if` when that `if` immediately checks or consumes the produced value.

Preferred:

```go
document, err := jsondocument.Decode(data)
if err != nil {
	return nil, err
}
```

Preferred:

```go
validationErr, ok := err.(*validation.Errors)
if !ok {
	return err
}
```

Do not insert a blank line between the producer and its immediate error, nil,
bounds, boolean, lookup, or type-assertion check.

### Separation from preceding logic

If an immediate producer-and-check unit follows another statement or logical
unit, separate it from the preceding code with a blank line.

Preferred:

```go
compiler.AssertFormat()

schema, err := compiler.Compile(schemaID)
if err != nil {
	return nil, err
}
```

No leading blank line is needed when the producer begins the enclosing block:

```go
func parse(data []byte) (any, error) {
	document, err := jsondocument.Decode(data)
	if err != nil {
		return nil, err
	}

	return document, nil
}
```

### Consecutive control-flow blocks

Separate independent `if` statements with a blank line.

Avoid:

```go
if manifest == nil {
	return ErrInvalid
}
if manifest.Name == "" {
	return ErrMissingName
}
```

Prefer:

```go
if manifest == nil {
	return ErrInvalid
}

if manifest.Name == "" {
	return ErrMissingName
}
```

This applies even when both decisions are short.

### Statements after control flow

Add a blank line after a completed `if`, `for`, or `switch` block before a
separate statement or logical unit.

Avoid:

```go
if err != nil {
	return err
}
violations = append(violations, next...)
```

Prefer:

```go
if err != nil {
	return err
}

violations = append(violations, next...)
```

### Local type declarations

Local types declared inside functions are allowed, but should be deliberate.

Prefer a local type when it is small, passive, method-free, used by only that
function, and makes a local algorithm easier to understand:

```go
func validateMembers(...) []validation.Violation {
	type memberList struct {
		name   string
		values []string
	}

	// Function-local validation state follows.
}
```

Prefer a package-level unexported type when it represents a meaningful domain
concept, is used across a substantial function, may gain behavior, is shared by
nearby helpers, or is clearer at package scope. Do not promote a tiny throwaway
struct merely for consistency, and do not keep a meaningful concept local only
to avoid adding a package-level type.

## JSON Schema, JSON, and fixture style

These rules apply to authored schemas and JSON fixtures.

* Use two-space indentation through the existing `jq`-backed Make targets.
  Do not manually reserialize individual files with a different formatter.
* Preserve authored key order; do not alphabetize schema or fixture objects.
* Order top-level schema keywords consistently: `$schema`, `$id`, `title`,
  `description`, `type`, object closure, `required`, `properties`, then `$defs`
  or other supporting constraints.
* Keep `required` and `properties` entries in the same logical contract order as
  the corresponding Go fields and human-facing examples when practical.
* Place reusable definitions in a narrowly named `$defs` entry or a canonical
  schema under `schemas/common`; do not copy a pattern merely to avoid a
  reference.
* Use local fragment references or canonical embedded schema IDs so validation
  remains offline.
* Keep object closure explicit. Put `additionalProperties: false` near the
  object `type`; use schema-valued `additionalProperties` only for intentional
  map surfaces.
* Keep fixtures readable and focused. A negative fixture should normally
  isolate one behavior unless it intentionally exercises multi-violation
  aggregation.
* Keep JSON fixtures aligned with the public serialized field order. Do not add
  fields solely to exercise Go zero values.
* JSON does not support comments. Put contract rationale in schema
  `description` fields, Go comments, README, or `docs/` as appropriate.
* When a schema path or ID changes, update its `$id`, references, embedding,
  public constants, tests, and documentation together.

## POSIX shell style

These rules apply to scripts under `scripts/`.

* Use `#!/bin/sh` and portable POSIX shell syntax; do not introduce Bash-only
  arrays, conditionals, process substitution, or function syntax.
* Start scripts with `set -eu` unless a documented workflow requires different
  failure handling.
* Quote parameter expansions and command substitutions unless intentional word
  splitting is required and documented.
* Use `printf` for predictable output; do not rely on implementation-specific
  `echo` behavior.
* Use descriptive, task-specific lowercase names for script state. Reserve
  uppercase names for environment or tool overrides such as `GOFMT`, `JQ`, and
  `JSON_INDENT`.
* Never repurpose common environment variables such as `HOME` or
  `CODEX_HOME`.
* Check required tools explicitly and return a distinct nonzero status when a
  prerequisite is missing.
* Create temporary files with `mktemp`, register cleanup with `trap`, and clean
  up on both success and failure.
* Validate destructive or replacement targets before writing. Reject empty,
  root, current-directory, or pre-existing destinations when the operation
  requires a fresh target.
* Format or replace source files through a temporary file, and replace the
  original only after the producing command succeeds.
* Preserve command failures and send diagnostics to standard error. Do not
  silently convert a failed formatter, copy, or comparison into success.
* Keep substantial shell logic in scripts and direct tool invocations plus Make
  target composition in `Makefile`.

## Verification guide

Choose checks in proportion to the changed contract, and run the repository's
authoritative targets rather than inventing parallel command sequences.

Standard checks:

```sh
make fmt
make check
make build
make mod-check
make test-race
```

Additional expectations:

* Schema changes should exercise offline compilation, embedding, valid and
  invalid examples, unknown fields, boundaries, semantic rules, and typed
  round trips where applicable.
* Parser changes should cover duplicate keys, trailing documents, malformed
  input, numeric handling, and operational reader/file errors.
* Validation changes should cover nil constructed values and multiple
  independent violations where applicable.
* CLI changes should verify exact stdout/stderr placement, JSON shape, input
  ordering, aggregate status, and exit codes.
* Pages or publication changes should run `scripts/build-pages.sh` with a new
  path beneath a temporary directory and verify that schema files are copied
  byte-for-byte and the index is sorted.
* Documentation-only changes need factual review and `git diff --check`; Go
  tests are unnecessary when no executable or contract behavior changed.
* JSON formatting checks require `jq`. Keep missing-tool and malformed-input
  failures distinct from product failures.

## Response style

When assisting with this repository, keep responses practical, concise, and
engineering-focused.

* Lead with the outcome or contract impact.
* Use short sections and bullets for decisions, trade-offs, and follow-up work.
* Explain ownership and compatibility implications for schema or API changes.
* Prefer focused diffs and snippets over full-file dumps.
* Call out intentional wire, validation, CLI, or compatibility changes in the
  final summary.
* Distinguish code failures from missing tools, restricted caches, network
  limitations, and external deployment state.
* Avoid repeating the same context in multiple sections of a response.
