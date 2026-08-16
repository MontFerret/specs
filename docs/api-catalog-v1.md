# API Catalog v1

The API Catalog is presentation metadata paired with one immutable Ferret API
Reference artifact. Its canonical schema is:

```text
https://schemas.ferretlang.org/registry/artifact/api-catalog/v1.json
```

The API Reference remains the canonical description of callable Ferret API
semantics. The catalog supplies only ordered documentation categories and the
real top-level namespace roots that navigation should expose. The Standard
Library is its first publisher, and official modules may use the same contract
in the future.

Categories such as `arrays`, `math`, and `strings` group global functions for
documentation. They are not Ferret namespaces and do not make names such as
`math::abs` callable.

`pkg/api/catalog` strictly parses one JSON document and validates it using
the embedded schema without network access. It also enforces unique category
IDs, unique function membership across categories, lexical function ordering,
and lexical namespace-root ordering. Consumers remain responsible for checking
that the catalog and API artifacts have matching identity and version and that
their functions and namespaces agree.
