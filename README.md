# Ferret Specifications

[![CI](https://github.com/MontFerret/specs/actions/workflows/ci.yml/badge.svg)](https://github.com/MontFerret/specs/actions/workflows/ci.yml)

This repository is the canonical source for machine-readable specifications and
validation libraries used across the Ferret ecosystem.

## Module Manifest v1

Ferret Module Manifest v1 describes what a module is and which runtime surface
it exposes. It does not describe how the module is built, published, installed,
or resolved.

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
```

`name` is the distribution identity used by registries and dependency
declarations. `namespace` is the independent, case-sensitive runtime identity
used by Ferret scripts. An exported namespace must equal or descend from the
manifest's runtime namespace.

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

All ingestion functions return only fully validated manifests:

```go
manifest, err := module.LoadFile("ferret-module.yaml")
if err != nil {
    var validationErr *module.ValidationErrors
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

The canonical schemas are:

```text
https://schemas.ferretlang.org/registry/module/v1.json
https://schemas.ferretlang.org/registry/version/v1.json
```

The `pkg/registry` package parses and validates both JSON document types without
network access. Registry-checkout layout, remote Git inspection, publication
history, and generated catalogs remain responsibilities of the registry
implementation rather than these portable contracts.

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
