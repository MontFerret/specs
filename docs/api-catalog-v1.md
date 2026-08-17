# API Catalog v1

The API Catalog is presentation metadata paired with one immutable Ferret API
Reference artifact. Its canonical schema is:

```text
https://schemas.ferretlang.org/registry/artifact/api-catalog/v1.json
```

The API Reference remains the canonical description of callable Ferret API
semantics. The catalog supplies only ordered documentation categories whose
members reference API functions by namespace and name. The Standard Library is
its first publisher, and official modules may use the same contract in the
future.

Categories such as `arrays`, `io`, and `testing` may group global and namespaced
functions together. They are not Ferret namespaces and do not make names such
as `math::abs` callable. An empty namespace references a global function; a
nonempty namespace uses the same qualified namespace grammar as API Reference
v1.

`pkg/api/catalog` strictly parses one JSON document and validates it using
the embedded schema without network access. It also enforces unique category
IDs, unique function membership across categories, lexical function ordering,
and strict structured function references. Consumers remain responsible for
checking that the catalog and API artifacts have matching identity and version
and that every referenced namespace and function exists.
