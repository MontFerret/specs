# Ferret Specifications

[![CI](https://github.com/MontFerret/specs/actions/workflows/ci.yml/badge.svg)](https://github.com/MontFerret/specs/actions/workflows/ci.yml)

This repository is the canonical source for machine-readable specifications and
validation libraries used across the Ferret ecosystem.

## Module Manifest v1

Ferret Module Manifest v1 describes what a module is and which runtime surface
it exposes. It does not describe how the module is built, published, installed,
or resolved.

Every module root has at most one repository-level manifest. Its canonical
filename is `ferret.yaml`. The validation library can also parse explicitly
supplied JSON documents, but JSON is not a second repository filename
convention.

The canonical schema is:

```text
https://schemas.ferretlang.org/module/v1.json
```

The schema uses JSON Schema Draft 2020-12 and references reusable components
under `https://schemas.ferretlang.org/common/`. The checked-in schemas are the
source of truth; publishing the schema host is managed separately.

A minimal YAML manifest is:

```yaml
$schema: https://schemas.ferretlang.org/module/v1.json
name: montferret/sqlite
namespace: DB::SQLITE
version: 1.0.0
description: Provides SQLite database access for Ferret queries.
license: Apache-2.0
documentation: https://docs.montferret.dev/modules/sqlite/
repository:
  url: https://github.com/MontFerret/contrib
  directory: modules/db/sqlite
```

`name` is the canonical lowercase `owner/module` distribution identity used by
registries and dependency declarations. Validators reject mixed-case spelling;
they do not silently lowercase it. `namespace` is the independent,
case-sensitive runtime identity used by Ferret scripts. Namespace segments
follow the normal FQL identifier grammar and are not restricted to uppercase.
An exported namespace must equal or descend from the manifest's runtime
namespace.

`repository.url` identifies the source repository. Monorepo modules set the
optional normalized relative `repository.directory`; standalone modules omit
it. The legacy repository URL string is not part of this corrected v1
contract.

`description` is a concise, single-line registry and CLI summary; detailed
guidance belongs at the required `documentation` URL. `authors`, when present,
is a non-empty array whose entries require `name` and may include `email` and
`url`. Authorship does not confer registry ownership or publishing authority.

`dependencies` lists required runtime module coordinates and npm-compatible
version ranges as descriptive metadata. Duplicate coordinates and direct
self-dependencies are invalid; this specification does not install or resolve
them. `exports` groups functions, types, and constants by FQL namespace and
lists dialects at module level. Duplicate exports and namespaces outside the
module's primary namespace tree are invalid.

Version constraints use npm-compatible semantic version ranges. Prerelease
versions do not satisfy a range unless the range explicitly includes a
prerelease comparator.

All manifest URLs are absolute HTTPS URLs. License values are full SPDX license
expressions. Unknown properties are rejected so spelling mistakes do not become
silent metadata.

See [Module Documentation v1](docs/module-documentation-v1.md) for the
documentation contract referenced by `documentation`.

## Go validation library

The Go package targets Go 1.25 and validates JSON and YAML manifests through
three stages:

1. decode exactly one document;
2. validate it against the embedded JSON Schema;
3. apply semantic rules such as npm range parsing, SPDX validation, duplicate
   detection, and export namespace containment.

All ingestion functions return only fully validated manifests. Structured
failures use the shared `pkg/validation` package:

```go
manifest, err := module.LoadFile(module.ManifestFilename)
if err != nil {
    var validationErr *validation.Errors
    if errors.As(err, &validationErr) {
        for _, violation := range validationErr.Violations {
            log.Printf("%s: %s", violation.Path, violation.Message)
        }
    }
	
    return err
}
```

The package exposes `LoadFile`, `Load`, and `Parse` for serialized documents and
`Validate` for programmatically constructed manifests. Validation never fetches
schemas over the network.

## Manifest validation CLI

Install a release-pinned copy of `ferret-spec` with:

```sh
go install github.com/MontFerret/specs/cmd/ferret-spec@v1.1.0
```

The CLI uses the same embedded schemas and semantic checks as `pkg/module`, so
validation remains offline. Validate one or more JSON or YAML module manifests
by passing their paths explicitly:

```sh
ferret-spec validate module ferret.yaml
ferret-spec validate module modules/http/ferret.yaml modules/html/ferret.yaml
```

Use `-` once to read a manifest from standard input:

```sh
ferret-spec validate module - < ferret.yaml
```

Text output is the default. Valid inputs are reported on standard output, while
violations and operational errors are reported on standard error. For portable
CI integration, `--format json` writes one versioned report to standard output:

```sh
ferret-spec validate module --format json ferret.yaml
```

The JSON report contains `formatVersion`, `kind`, aggregate `status`, and one
ordered result per input. Status values are `valid`, `invalid`, and `error`.
JSON Pointer paths, stable rule identifiers, and messages are preserved in each
invalid result.

Exit codes are:

- `0` when every input is valid;
- `1` when at least one manifest is invalid;
- `2` for command usage, file I/O, or internal errors.

Operational errors take precedence over invalid results, but every supplied
input is processed. A version-pinned CI step can therefore install and invoke
the validator directly:

```sh
go install github.com/MontFerret/specs/cmd/ferret-spec@v1.1.0
ferret-spec validate module ferret.yaml
```

## Versioning

Schema paths are versioned by major version. Within v1, changes must remain
backward compatible: an updated v1 validator must continue to accept documents
that were valid under an earlier v1 schema. Because objects are closed, an older
embedded validator is not guaranteed to accept fields introduced by a later v1
schema.

## Registry v1

Ferret Registry v1 defines the reviewed records used by a Git-backed module
registry. A registry module manifest identifies an owner/name coordinate and an
anonymous HTTPS Git source; a version record pins a strict semantic version and
Git tag to an exact commit.

Registry `owner` and `name` values use canonical lowercase spelling. Wherever a
generated or consumed artifact stores an explicit module `id`, it must equal the
exact `${owner}/${name}` value. Mixed-case source records and artifacts are
invalid rather than normalized, and this distribution contract does not alter
the independent case-sensitive Ferret runtime namespace.

The canonical schemas are:

```text
https://schemas.ferretlang.org/registry/module/v1.json
https://schemas.ferretlang.org/registry/version/v1.json
```

The `pkg/registry` package parses and validates both JSON document types without
network access.

Generated Registry artifact v1 is split by document role:

```text
https://schemas.ferretlang.org/registry/artifact/root/v1.json
https://schemas.ferretlang.org/registry/artifact/module-index/v1.json
https://schemas.ferretlang.org/registry/artifact/module/v1.json
https://schemas.ferretlang.org/registry/artifact/version/v1.json
https://schemas.ferretlang.org/registry/artifact/api/v1.json
https://schemas.ferretlang.org/registry/artifact/category-index/v1.json
https://schemas.ferretlang.org/registry/artifact/category/v1.json
https://schemas.ferretlang.org/registry/artifact/plugin-index/v1.json
```

The `pkg/registry/artifact` package owns the corresponding wire types, strict
JSON parsing, and local validation. Registry-checkout layout, remote Git
inspection, publication history, distribution generation, hosting, and
cross-document navigation remain responsibilities of the registry
implementation rather than these portable contracts.

The API Reference artifact describes each registered function signature with
ordered Ferret-facing parameter objects. A parameter always has a name and may
also carry a Ferret type expression and description. Signatures may include
prose, a return value, ordered visible failures, and a deprecation message.
Ferret type and error expressions are opaque strings rather than Go types.

The older registry placeholder at `/registry/v1.json` and the plugin v1 file
remain reserved and deliberately reject every document.

## Development

Run the default formatting, vet, and test checks with:

```sh
make check
```

JSON formatting requires `jq`. It uses two-space indentation and preserves the
authored key order.

Other common operations include:

```sh
make build           # Build all packages.
make test-race       # Run tests with the race detector.
make fmt             # Format Go and JSON files.
make fmt-json        # Format only JSON files.
make fmt-json-check  # Check JSON formatting without changing files.
make tidy            # Update module metadata.
make mod-check       # Check module metadata without changing it.
make coverage        # Write coverage.out.
make clean           # Clear the test cache and coverage profile.
make help            # List every available target.
```
