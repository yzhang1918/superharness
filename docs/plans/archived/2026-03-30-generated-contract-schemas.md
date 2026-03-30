---
template_version: 0.2.0
created_at: "2026-03-30T23:45:08+08:00"
source_type: issue
source_refs:
    - '#72'
---

# Make harness contracts discoverable through generated schemas

## Goal

Make the current harness command and local-artifact contracts discoverable as
checked-in JSON Schema files that users, docs, and downstream tools can inspect
without reverse-engineering Go structs or prose specs.

This slice should reduce field-level source-of-truth drift by treating Go
contract types as the canonical definition for JSON payload shapes, exporting
schemas from those types into the repository, and updating spec docs to point
readers at the generated schema artifacts rather than duplicating field tables
by hand. The slice must not change existing command behavior or rewrite the
workflow model.

## Scope

### In Scope

- Identify the user-facing JSON surfaces that should be published in a first
  schema set, including stateful command results, command JSON inputs, and
  command-owned local JSON artifacts.
- Introduce a single Go-owned contract layer that existing command
  implementations can use or alias without changing their runtime behavior.
- Add an automated schema-generation path that exports checked-in JSON Schema
  files under `docs/schemas/`.
- Update tracked docs so specs explain semantics and workflow rules while
  linking to the generated schemas for field-level detail.
- Add reproducibility and drift checks so schema files stay aligned with the Go
  contract definitions over time.

### Out of Scope

- Changing the meaning, sequencing, or validation behavior of existing harness
  commands.
- Redesigning the v0.2 workflow, canonical node model, or tracked plan
  lifecycle.
- Converting the markdown tracked-plan document itself into a JSON-schema-first
  artifact.
- Normalizing legacy output shapes such as lifecycle command envelopes in the
  same slice unless that falls out as a no-behavior-change refactor needed to
  share contract types.

## Acceptance Criteria

- [x] A checked-in schema set exists under `docs/schemas/` for the selected
      first-class JSON surfaces, and those files are generated from Go-owned
      contract types rather than handwritten as a separate field-definition
      source.
- [x] The generated schema set covers at least the current stateful command
      result envelopes plus the JSON input or artifact shapes that users or
      tooling are expected to inspect directly, without changing the current
      wire behavior of those commands.
- [x] `docs/specs/cli-contract.md` and related spec entry points describe
      command semantics and workflow rules while pointing readers to the schema
      files for exact field-level structure instead of restating those fields in
      prose tables.
- [x] A single automated regeneration path exists, and repository validation
      fails when the checked-in schema files drift from the Go contract
      definitions.
- [x] Focused automated coverage proves the generator output is reproducible and
      that representative command payloads or fixtures still satisfy the
      exported schemas.

## Deferred Items

- Publishing schemas as release assets or serving them from a versioned hosted
  docs site.
- Replacing all runtime JSON decoding with schema-driven validation in the same
  slice.
- Any follow-up contract cleanup that intentionally changes current command
  shapes to remove legacy fields or improve consistency.

## Work Breakdown

### Step 1: Define the Go-owned public contract surface and schema generator

- Done: [x]

#### Objective

Establish a single Go contract layer for the selected public JSON surfaces and
add the generator that can export those contracts as JSON Schema files.

#### Details

This step should answer the key source-of-truth question in code: which Go
types now define the public JSON surfaces, and how existing command packages
reuse or reference those types without changing command behavior. The selected
surface should include the shared result-envelope building blocks where they
already exist, command-specific payloads that users submit directly, and local
artifact records that users or tools are expected to inspect. The generator
should be repo-local and deterministic so later CI or tests can rerun it
without hand edits.

#### Expected Files

- `go.mod`
- `internal/contracts/`
- `cmd/schemagen/main.go`
- `internal/status/service.go`
- `internal/review/service.go`
- `internal/evidence/service.go`
- `internal/lifecycle/service.go`
- `internal/plan/lint.go`

#### Validation

- The repository has one clear Go-owned location for the exported command or
  artifact contracts selected for this slice.
- Running the schema generator produces deterministic output for those
  contracts without requiring manual edits to the generated files.
- Existing command behavior remains unchanged aside from any harmless import or
  type-sharing refactors needed to point implementations at the shared
  contracts.

#### Execution Notes

Added a new `internal/contracts/` package that centralizes the exported JSON
contract types for status, lifecycle, review, evidence, plan lint, and
runstate artifacts without changing the current wire shapes. Existing service
and runstate packages now alias those shared contract types instead of owning
independent copies.

Added `cmd/schemagen` plus `contracts.GenerateSchemaFiles`, and pinned
`github.com/google/jsonschema-go` in `go.mod` so the repository can export
checked-in JSON Schemas directly from the Go-owned contract layer.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This contract-layer extraction is tightly coupled to the
generated schema publishing and drift-check work in Steps 2 and 3, so a
separate Step 1 review would have duplicated the later review pass over the
fully wired slice.

### Step 2: Publish checked-in schemas and point specs at them

- Done: [x]

#### Objective

Generate the initial checked-in schema set and make the tracked docs use those
artifacts as the field-level reference surface.

#### Details

Keep the docs focused on semantics, ownership, workflow, and non-obvious
constraints. They should link to the generated schema files instead of copying
field lists into prose. The checked-in schema set should be organized so a cold
reader can find the relevant command result, command input, or local artifact
schema without reading generator code first.

#### Expected Files

- `docs/schemas/`
- `docs/specs/index.md`
- `docs/specs/cli-contract.md`
- `README.md`

#### Validation

- A reader starting from the specs can navigate directly to the generated schema
  files for exact field-level structure.
- The schema file names and layout are clear enough that downstream tooling can
  consume them without guessing which file corresponds to which command surface.
- No spec doc reintroduces a second handwritten field-definition source for the
  same JSON surface.

#### Execution Notes

Generated the first checked-in schema set under `docs/schemas/` for command
results, structured command inputs, and command-owned local JSON artifacts.
Added `docs/schemas/index.md` and updated the CLI/spec entry points so docs now
link to the generated schema files for field-level detail instead of growing
more handwritten field tables.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only becomes meaningful together with the
generator and drift-check wiring from Steps 1 and 3, so a standalone review at
this boundary would have been an incomplete contract scan.

### Step 3: Automate regeneration and drift checks

- Done: [x]

#### Objective

Make schema generation part of the normal repository workflow so checked-in
schemas stay synchronized with the Go contract layer.

#### Details

Add one obvious regeneration entry point, such as `go generate` support, a repo
script, or both, and wire repository validation to fail when generated schema
artifacts are stale. Favor the lightest mechanism that fits the current repo:
`go test ./...` already runs in CI, so the drift check can live in tests, CI,
or both, as long as a contributor gets a clear failure and a single command to
repair it.

#### Expected Files

- `scripts/update-schemas`
- `.github/workflows/ci.yml`
- `internal/contracts/`
- `tests/`

#### Validation

- There is one documented command for regenerating the schema files locally.
- CI or test validation catches schema drift deterministically.
- Focused automated coverage proves representative generated schemas are
  reproducible and that representative command outputs or fixtures still conform
  to them.

#### Execution Notes

Added a single regeneration path through `scripts/update-schemas` plus a
`go:generate` directive on `internal/contracts/schemas.go`. Added contract
tests that compare regenerated schemas to the checked-in files and validate
representative payloads against those exported schemas. CI now runs
`scripts/update-schemas --check` before `go test ./...`.

Follow-up review repairs made the schema index generated from the same contract
registry as the `.schema.json` files, restored full-tree drift checking, and
made regeneration replace stale files so renames or removals do not leave dead
schemas behind.

#### Review Notes

Step-closeout review `review-001-delta` found one important issue: the schema
index could drift because it was hand-maintained and excluded from the
drift-check path. Follow-up review `review-002-delta` then found one more
important issue: regeneration did not remove stale files after schema renames
or deletions. Repaired both problems by generating `docs/schemas/index.md`
from the contract registry and by clearing the output tree before regeneration.
Follow-up review `review-003-delta` reran clean with a pass decision.

## Validation Strategy

- Run `harness plan lint` on the tracked plan before execution starts.
- Add focused Go tests around the new contract or schema-generation package so
  generator output and schema loading are deterministic.
- Validate representative command outputs or stored fixture payloads against the
  exported schemas without changing runtime command semantics.
- Run `go test ./...` once the slice is ready so the new generator, shared
  types, and drift checks are exercised in the existing repository test path.

## Risks

- Risk: The slice could accidentally create a fourth source of truth by keeping
  the old per-package structs, the new shared contract types, handwritten docs,
  and generated schemas all alive at once.
  - Mitigation: Treat the shared Go contract layer as the only field-level
    source, generate schemas from it, and update docs to link to those schemas
    instead of duplicating field tables.
- Risk: A cleanup refactor could quietly change command payload shapes while
  trying to centralize contract types.
  - Mitigation: Keep acceptance criteria explicit about no behavior changes and
    add schema/fixture validation that proves current command shapes still hold.
- Risk: Schema generation could become cumbersome if contributors have to
  remember multiple ad hoc commands.
  - Mitigation: Provide one obvious regeneration path and enforce drift through
    deterministic repository validation.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-30-generated-contract-schemas.md`
  passed after each major tracked-plan update.
- `scripts/update-schemas` and `scripts/update-schemas --check` now pass with
  generated `.schema.json` files plus the generated `docs/schemas/index.md`.
- Focused package validation passed for the extracted contract layer and schema
  automation: `go test ./internal/contracts ./internal/status
  ./internal/evidence ./internal/review ./internal/lifecycle ./internal/plan
  ./internal/runstate ./internal/cli`.
- Repository-level validation passed with `go test ./...`.
- Follow-up repair validation after review findings passed with
  `go test ./internal/contracts ./internal/cli ./tests/smoke -count=1`, plus
  the final full-suite run of `go test ./...`.
- After reopening in `finalize-fix` mode for sync freshness, the branch merged
  `origin/main` cleanly and reran `go test ./...` on revision 2.

## Review Summary

- Step-closeout review `review-001-delta` found one important issue: the schema
  index was hand-maintained and excluded from drift checking.
- Follow-up review `review-002-delta` found one important issue: regeneration
  left stale generated files behind after schema removals or renames.
- Follow-up review `review-003-delta` reran the bounded repair clean with a
  pass decision.
- Finalize full review `review-004-full` passed on `correctness`, `tests`, and
  `docs_consistency` with no findings.
- After sync-driven reopen to revision 2, finalize delta review
  `review-005-delta` found one blocker: the reopened plan still carried stale
  reopen sentinel markers in the durable summary sections.
- Follow-up finalize delta review `review-006-delta` reran clean with a pass
  decision after the revision 2 durable summary sections were refreshed.

## Archive Summary

- Archived At: 2026-03-31T00:52:59+08:00
- Revision: 2
- PR: https://github.com/catu-ai/easyharness/pull/76
- Ready: The revision 2 candidate merged the narrow `origin/main` baseline
  change cleanly, reran `go test ./...`, and passed `review-006-delta` as the
  finalize repair review, so the reopened candidate is ready to archive again.
- Merge Handoff: Re-archive the plan, push the updated branch to PR #76, and
  refresh publish, CI, and sync evidence until status reaches merge approval
  again.

## Outcome Summary

### Delivered

- Added a new Go-owned `internal/contracts` layer for command results, command
  inputs, and command-owned local JSON artifacts without changing the existing
  wire behavior.
- Added `cmd/schemagen`, `scripts/update-schemas`, and a `go:generate`
  directive so the repository can regenerate checked-in schemas from the shared
  contract registry.
- Generated the initial schema set under `docs/schemas/`, including command
  result envelopes, structured inputs, runstate artifacts, review artifacts,
  and evidence artifacts.
- Made `docs/schemas/index.md` generated from the same registry as the schema
  files so schema discovery and drift checking stay in sync.
- Added contract-level regression tests plus CI drift checking for generated
  schemas, including stale-file cleanup coverage.
- Updated README and spec entry points to reference generated schemas instead
  of growing more handwritten field-definition docs.

### Not Delivered

- Generated schemas are not yet published outside the repository checkout or
  attached to release artifacts.
- Runtime schema-driven validation was intentionally deferred; the current
  candidate keeps the existing Go validation paths.
- Remaining legacy/public-shape cleanup, including any future lifecycle-result
  convergence work, was intentionally deferred to a follow-up slice.

### Follow-Up Issues

- #74: Publish generated schema references outside the repository.
- #75: Evaluate schema-driven validation and remaining contract cleanup.
