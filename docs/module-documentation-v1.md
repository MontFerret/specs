# Ferret Module Documentation v1

## Status

This document defines the documentation contract for registry-compatible
Ferret modules using the canonical repository manifest `ferret.yaml` and
Module Manifest v1. A module root contains at most one such manifest.

The manifest's required `documentation` field must be an absolute HTTPS URL for
the module's canonical long-form documentation. Long-form documentation does
not belong in the manifest; the manifest `description` remains a short,
one-sentence summary.

## Required sections

Canonical module documentation must contain the following section headings:

### Overview

Explain the module's purpose, the problem it addresses, and the runtime
namespace or namespaces it exposes. Keep build and publishing mechanics out of
the overview unless they are necessary for using the module.

### Installation

Explain how a user makes the module available to their Ferret environment.
Installation commands may vary by host or distribution channel.

### Quick Start

Provide a minimal working Ferret example that demonstrates the module's primary
use case. Include any configuration required for the example to run.

### API Reference

Document exported namespaces, functions, types, constants, and dialects. For
functions, include parameters, return values, errors, and a usage example where
those details are not self-evident.

## Additional sections

Authors may add any other sections useful to their module, such as
Configuration, Security, Examples, Migration, Troubleshooting, or Changelog.
Additional sections must not replace or rename the four required sections.

## Validation

This contract is descriptive in v1. The specification repository does not
fetch, parse, or validate module documentation, and it does not parse README
files or generate documentation.
