---
template_version: 0.2.0
created_at: "2026-03-31T15:51:27+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/72
---

# Centralize contract schemas and generated reference docs

## Goal

Make the `easyharness` contract surface discoverable and maintainable without
forcing users, developers, or agents to reverse-engineer Go structs or prose
specs. The source of truth should live in one centralized Go contract module
that defines the field-level shapes for public CLI JSON outputs, shared JSON
types, and CLI-owned `.local/harness/` JSON artifacts.

From that Go-owned source, generate a checked-in JSON Schema registry as many
stable schema files plus a central index, and update the prose specs so they
point at that registry without generating one markdown page per schema. The
first slice should improve schema quality and documentation discoverability
without changing the current behavioral contract or tightening existing
plain-string fields into enums.

## Scope

### In Scope

- Introduce a centralized Go contract layer that owns the field-level schema for
  public CLI JSON outputs across `status`, `install`, `execute start`,
  `evidence submit`, `review start`, `review submit`, `review aggregate`,
  `archive`, `reopen`, `land`, and `land complete`.
- Move or mirror shared JSON shapes such as next-action and error objects into
  that centralized contract layer so generation does not duplicate them
  independently in every command package.
- Cover CLI-owned `.local/harness/` JSON artifact families in the same
  contract surface, including worktree pointers, plan-local state, review
  manifests and aggregates, and evidence records.
- Generate a checked-in JSON Schema registry as many stable schema files plus a
  central index, with explicit schema dialect metadata and reusable
  definitions/references rather than large flattened inline objects.
- Add generation and validation tooling so schema artifacts can be regenerated
  deterministically and checked for drift in tests or CI, while deprecated
  generated markdown outputs are removed.
- Update the repository docs so contract discovery points at the schema
  registry plus a small prose registry guide instead of hand-maintained field
  tables or generated per-schema markdown pages.
- Preserve the current contract semantics for this slice, including leaving
  existing closed-set string fields as strings unless they are already modeled
  more narrowly today.

### Out of Scope

- Converting the markdown tracked-plan schema into JSON Schema.
- Redesigning the v0.2 workflow model, command set, or local artifact layout.
- Tightening existing string-valued contract fields into enums just because the
  schema generator makes that convenient.
- Switching the public contract format to OpenAPI or another non-JSON-Schema
  surface in this slice.
- Generating downstream client SDKs or promising code generation workflows
  beyond schema export and reference docs.

## Acceptance Criteria

- [x] A centralized Go contract module exists and is the documented field-level
      source of truth for public CLI JSON outputs, shared reusable JSON shapes,
      and CLI-owned `.local/harness/` JSON artifacts.
- [x] The repository contains a checked-in JSON Schema registry composed of a
      discoverable central index plus stable schema files for the covered
      command outputs and local artifact families.
- [x] Generated schemas declare their dialect/version explicitly and use
      reusable shared definitions or references for repeated shapes instead of
      duplicating those shapes inline across files.
- [x] Generated schemas include enough consumer-facing metadata to support
      direct consumption from the schema files themselves, at minimum titles
      and descriptions, with examples where they materially improve clarity.
- [x] The generated schema surface preserves the current contract semantics for
      this slice and does not introduce new enum restrictions for existing
      plain-string fields.
- [x] The repository exposes one prose contract-registry guide under
      `docs/specs/contract.md` and does not keep duplicate generated markdown
      pages for every schema file.
- [x] Drift checks fail when the centralized Go contracts and checked-in schema
      artifacts fall out of sync, and they also fail if deprecated generated
      markdown artifacts reappear.

## Deferred Items

- Enum coverage for closed-set string fields such as `current_node`,
  `review_status`, `ci_status`, and `sync_status` once the repository is ready
  to treat those value sets as explicitly public and stable at the schema
  level.
- Evaluating whether a future external-consumer surface should also emit
  OpenAPI or another registry format in addition to JSON Schema.
- Any future decision to validate markdown plan files through a machine-readable
  schema format rather than the current prose-plus-linter contract.

## Work Breakdown

### Step 1: Define the centralized Go contract ownership model

- Done: [x]

#### Objective

Create one self-contained contract layer in Go that a future agent can inspect
to discover the public JSON surface for command outputs, shared types, and
CLI-owned local artifacts without tracing many unrelated service packages.

#### Details

Choose a repository location and type layout that make the ownership boundary
obvious, then migrate or wrap the existing exported JSON result and artifact
shapes into that layer. The centralized module should cover current contracts
as they exist today, including legacy plain-string fields that are not yet
promoted to enums. This step should also document which JSON artifacts count as
publicly discoverable contract surface in this first slice and which prose
specs become descriptive companions rather than the field-by-field source of
truth.

#### Expected Files

- `internal/contracts/**`
- `internal/status/service.go`
- `internal/lifecycle/service.go`
- `internal/review/service.go`
- `internal/evidence/service.go`
- `internal/install/service.go`
- `internal/runstate/state.go`
- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`
- `docs/specs/index.md`

#### Validation

- Command packages compile against the centralized contract layer without
  changing their current JSON field names or optionality.
- The repository docs clearly state that the Go contract layer, not duplicated
  prose tables, owns the covered field-level contract.
- Focused tests are added or updated where needed to catch accidental JSON tag
  drift during the refactor.

#### Execution Notes

Added a new Go-owned contract module under `internal/contracts/` and moved the
field-level source of truth for command outputs, shared JSON shapes, and
CLI-owned local artifact shapes into that package. The existing command and
runstate packages now consume those types through aliases so the current JSON
surface stays behaviorally stable while future schema/doc generation reads from
one centralized place. TDD was not practical for this initial refactor because
the goal was to preserve the existing emitted JSON rather than introduce a new
behavior surface; compatibility protection came from compiling the touched
packages and rerunning their existing test suites. Validation:
`go test ./internal/status ./internal/lifecycle ./internal/review ./internal/evidence ./internal/install ./internal/runstate`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This ownership refactor is tightly coupled to the schema
generation and docs work in later steps, so it will be reviewed through the
branch-level finalize review instead of an isolated step-closeout review.

### Step 2: Generate the checked-in JSON Schema registry

- Done: [x]

#### Objective

Generate a stable, reusable JSON Schema surface from the centralized Go
contracts so both tools and humans can discover the contract without reading
the implementation directly.

#### Details

Pick and integrate a schema-generation path that works from Go-owned contract
types and emits many checked-in schema files plus a central index. The output
should emphasize reuse and consumer quality: explicit schema dialect,
consistent naming, stable file layout, `title` and `description` metadata, and
heavy use of shared definitions or references instead of flattened repeated
shapes. The registry should include both command-result schemas and
`.local/harness/` artifact schemas while preserving current contract semantics.

#### Expected Files

- `internal/contracts/**`
- `schema/**`
- `scripts/**`
- `go.mod`
- `go.sum`

#### Validation

- Running the chosen generation path deterministically refreshes the checked-in
  schema registry with no manual edits.
- The registry has a discoverable index and stable per-schema files for the
  covered contract families.
- Automated validation fails when generated schema artifacts drift from the
  Go-owned source of truth.

#### Execution Notes

Added a repo-local sync path at `scripts/sync-contract-artifacts`, backed by
`cmd/contract-sync` and `internal/contractsync`, to generate a checked-in JSON
Schema registry under `schema/`. The generator reflects the Go-owned contract
types with `invopop/jsonschema`, emits a many-files-plus-index layout, sets an
explicit Draft 2020-12 dialect on every schema, and injects type and field
descriptions from the `internal/contracts` source comments so the generated
schemas carry consumer-facing metadata. Validation:
`go test ./cmd/contract-sync ./internal/contractsync ./internal/contracts`,
`scripts/sync-contract-artifacts`, and
`scripts/sync-contract-artifacts --check`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The registry generator and the surrounding contract
refactor form one coupled slice and are reviewed together at branch finalize
time.

### Step 3: Generate and align reference docs from the schema registry

- Done: [x]

#### Objective

Align the repository docs around the schema registry without generating a
second field-by-field markdown surface that mostly duplicates the schema files.

#### Details

Update the relevant docs to consume the generated registry rather than repeat
field tables independently. Keep the normative prose that explains workflow
meaning, ownership, and non-goals, but avoid generating repetitive markdown
pages that restate each schema file. The resulting docs should help a cold
reader find the schema index, understand the contract families that are
covered, and distinguish public CLI/local-JSON contract surfaces from the
out-of-scope markdown plan schema.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/contract.md`
- `docs/specs/state-model.md`
- `docs/specs/index.md`
- `README.md`
- `schema/**`
- supporting generation scripts under `scripts/**` or Go code under
  `internal/contracts/**`

#### Validation

- A cold reader can navigate from the main docs to the schema index and the
  contract-registry prose guide without needing discovery chat.
- The repository no longer generates or keeps per-schema markdown pages that
  mostly duplicate the schema files.
- Drift checks fail when deprecated generated docs reappear or when the current
  schema registry drifts from the Go-owned source.

#### Execution Notes

Repository docs in `README.md`, `docs/specs/index.md`,
`docs/specs/cli-contract.md`, `docs/specs/state-model.md`, and
`docs/specs/contract.md` now point readers at the checked-in schema registry
instead of implying that the repository also publishes generated field-by-field
markdown pages. A revision-2 finalize fix then removed the generated
`docs/reference/contracts/` tree entirely after it became clear that those
pages mostly duplicated the schema files and even introduced avoidable root vs.
definition repetition. Validation:
`scripts/sync-contract-artifacts` and
`scripts/sync-contract-artifacts --check`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The generated docs and repo-visible guidance updates are
part of the same coupled contract-surface slice and are reviewed in the
branch-level finalize review.

### Step 4: Prove compatibility and drift enforcement

- Done: [x]

#### Objective

Show that the new contract pipeline improves discoverability without changing
the current public JSON behavior.

#### Details

Add or update tests and validation commands that prove three things: command
results still marshal to the same contract shape they did before this slice,
CLI-owned local artifact JSON still matches the behavior the code emits today,
and schema/docs generation cannot silently drift. Where current docs and code
are out of sync, resolve that mismatch by bringing docs and generated schemas
to the current implemented contract rather than changing behavior unless an
explicit follow-up decision says otherwise.

#### Expected Files

- `tests/**`
- `internal/**/_test.go`
- `schema/**`
- `docs/specs/**`
- generation or verification entrypoints under `scripts/**`

#### Validation

- Focused unit or golden tests cover schema generation and representative
  command/artifact contract shapes.
- The standard local validation instructions include the schema/doc drift check.
- The repository can demonstrate that this slice changed the contract surface's
  discoverability and maintainability, not its semantics.

#### Execution Notes

Added a smoke drift check at `tests/smoke/contract_sync_test.go` so the current
repository fails when generated schemas drift or when deprecated generated
markdown artifacts reappear. Validated the compatibility-preserving refactor
and the new generation pipeline with focused package tests, a dedicated smoke
test, and a full repository `go test ./...` pass. Finalize review
`review-001-full` then surfaced two repair items: generated schemas needed to
model nullable pointer/slice fields that the current runtime can serialize as
`null`, and the contract-sync package needed explicit negative tests for
missing or unexpected generated files plus a write-path regression. The repair
added nullable schema wrapping for non-omitempty pointer/slice/map fields and
direct `internal/contractsync` regression tests for `checkFiles` and
`writeFiles`.
Validation:
`go test ./cmd/contract-sync ./internal/contractsync ./internal/contracts ./internal/status ./internal/lifecycle ./internal/review ./internal/evidence ./internal/install ./internal/runstate ./tests/smoke -run 'TestSyncBootstrapAssetsCheckPassesForCurrentRepo|TestSyncContractArtifactsCheckPassesForCurrentRepo'`
and `go test ./...`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The branch now has full validation coverage for the
completed slice and will receive a single branch-level finalize review rather
than separate step-closeout reviews for each tightly coupled implementation
step.

## Validation Strategy

- Lint the tracked plan before approval and keep the discovery decisions inside
  the plan so a future agent can execute without chat history.
- Use focused Go tests and, where helpful, golden-schema comparisons to verify
  that command output and local artifact JSON contracts remain compatible while
  centralizing their type ownership.
- Add deterministic generation commands plus drift checks for the checked-in
  schema registry, and treat any reintroduced generated markdown surface as
  deprecated drift.
- Manually inspect the generated schema index and a representative sample of
  per-command and per-artifact schemas to confirm metadata, reuse, and
  discoverability quality are materially better than the previously flattened
  export attempt.

## Risks

- Risk: Centralizing contracts may accidentally change existing JSON fields,
  optionality, or package behavior while moving types.
  - Mitigation: Treat the current emitted JSON as the compatibility target and
    add focused regression coverage before or during the refactor.
- Risk: The first schema generator chosen may emit structurally valid JSON
  Schema but still produce low-quality consumer artifacts with poor reuse or
  metadata.
  - Mitigation: Make schema quality an explicit acceptance target and reject a
    toolchain that cannot reasonably support named reuse, metadata, and a
    central index.
- Risk: Docs may remain partially duplicated if generated references and prose
  specs are not cleanly split.
  - Mitigation: Update the docs ownership model during execution so prose keeps
    normative meaning while the schema registry owns field-level contract
    detail for the covered surfaces.
- Risk: Covering `.local/harness/` artifacts may expand scope because some
  shapes are currently implicit in implementation packages.
  - Mitigation: Keep the slice on existing emitted artifacts only, defer enum
    tightening and larger model redesigns, and make any newly discovered
    follow-up work explicit in deferred items or linked issues.

## Validation Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Centralized contract ownership stayed compile- and behavior-compatible across
  the touched command packages with focused validation on
  `./internal/status`, `./internal/lifecycle`, `./internal/review`,
  `./internal/evidence`, `./internal/install`, and `./internal/runstate`.
- The contract generation pipeline now has deterministic refresh and drift
  enforcement through `scripts/sync-contract-artifacts` and
  `scripts/sync-contract-artifacts --check`, backed by targeted
  `internal/contractsync` regression coverage.
- Repository-level validation passed with focused smoke coverage for contract
  sync drift, including stale schema and stale generated-doc cases, plus a
  full `go test ./...` pass after the finalize repair series landed.

## Review Summary

UPDATE_REQUIRED_AFTER_REOPEN

- `review-001-full` found two compatibility gaps in the initial generator:
  schemas were not modeling nullable non-omitempty pointer/slice/map fields
  the way the current runtime can serialize them, and the contract-sync check
  path lacked explicit negative regression coverage for stale/missing files.
  Those fixes landed in `5e3d58b`.
- `review-002-full` found a correctness mismatch where early `harness review
  start` failures still emitted the generic `review` command identifier instead
  of `review start`, plus a generated lifecycle contract omission for
  `harness land complete`. Those fixes landed in `09565cb`.
- `review-003-full` found two input-schema mismatches and one prose mismatch:
  `review.spec.dimensions` was still nullable in input schemas, optional
  `review.submission.findings` was being treated as required/non-nullable, and
  `docs/specs/cli-contract.md` implied every stateful command emitted the same
  `state` shape. Those fixes landed in `032177a`.
- `review-004-full` reduced the remaining inconsistency to one blocking docs
  scope error and one non-blocking smoke gap: the status spec still described
  `plan_status` and `lifecycle` as globally forbidden instead of forbidding
  them specifically for `harness status`, and the smoke suite still lacked a
  stale generated-doc failure path. Those fixes landed in `0bbbdf9`.
- `review-005-full` passed clean with no blocking or non-blocking findings and
  serves as the structural `pre_archive` gate for this revision.

## Archive Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Archived At: 2026-03-31T16:48:48+08:00
- Revision: 1
- PR: not created yet; publish evidence will record the PR URL after archive.
- Ready: `review-005-full` passed clean, acceptance criteria are satisfied,
  and the candidate is ready for archive plus publish/CI/sync evidence work.
- Merge Handoff: archive the plan, commit the tracked move plus the latest
  closeout updates, push `codex/centralize-contract-schemas-docs`, open or
  refresh the PR, and record publish/CI/sync evidence until status reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Added a centralized Go contract layer under `internal/contracts/` that now
  owns the field-level JSON source of truth for covered command results, shared
  shapes, and CLI-owned `.local/harness/` artifacts.
- Refactored the runtime packages to consume those centralized contracts
  without changing the public JSON field names or broadening the contract to
  enums.
- Added the `contract-sync` generation pipeline, checked-in schema registry
  under `schema/`, and generated reference docs under
  `docs/reference/contracts/`.
- Updated the README and normative specs so field-level reference material now
  points at the generated contract surface instead of duplicated prose tables.
- Added contract-sync regression coverage and smoke drift checks for stale
  schema artifacts and stale generated docs, then closed the remaining finalize
  review findings through `review-005-full`.

### Not Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Enum promotion for currently plain-string public fields such as
  `current_node`, `review_status`, `ci_status`, or `sync_status`.
- Any alternate registry output such as OpenAPI in addition to JSON Schema.
- Machine-readable schema support for markdown tracked plans.

### Follow-Up Issues

UPDATE_REQUIRED_AFTER_REOPEN

- `#72` continues to track the broader contract-surface follow-up scope,
  including future enum decisions, whether to emit additional registry formats,
  and any later choice to model markdown plans with a machine-readable schema.
