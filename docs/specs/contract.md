# Contract Registry

## Purpose

This document defines how `easyharness` publishes its field-level JSON
contract surface.

The machine-readable public artifact is the checked-in JSON Schema registry
under [`schema/`](../../schema/), with [`schema/index.json`](../../schema/index.json)
as the discovery entrypoint.

## Source of Truth

The field-level source of truth lives in the Go-owned contract module under
`internal/contracts/`.

`scripts/sync-contract-artifacts` reflects that Go-owned surface into the
checked-in schema registry. `scripts/sync-contract-artifacts --check` fails
when the registry drifts from the current Go contract definitions.

## What the Registry Covers

The schema registry currently covers:

- public CLI JSON command results
- JSON command inputs such as review and evidence payloads
- shared reusable JSON shapes
- CLI-owned `.local/harness/` JSON artifacts

The registry does not cover the markdown tracked-plan schema.

## Documentation Policy

We intentionally do not generate one markdown file per schema.

Earlier generated markdown pages mostly restated the schema files without
adding enough meaning to justify the extra maintenance surface. The repository
therefore keeps:

- prose specs in `docs/specs/` for workflow meaning, ownership, and contract
  boundaries
- checked-in JSON Schema files in `schema/` for field-level structure

When a contract needs extra human explanation beyond what the schema can carry,
add that explanation to the relevant prose spec instead of generating a second
field table.

## Compatibility Notes

The registry is meant to make the current contract discoverable, not to tighten
it opportunistically.

In particular, existing public string fields remain plain strings unless the
repository explicitly decides to promote a value set into a narrower public
contract later.
