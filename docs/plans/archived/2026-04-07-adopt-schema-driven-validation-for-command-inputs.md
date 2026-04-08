---
template_version: 0.2.0
created_at: "2026-04-07T23:36:44+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/75
---

# Adopt schema-driven validation for structured JSON command inputs

## Goal

Make the checked-in input schemas under `schema/inputs/` executable contract
enforcement for the structured JSON command-input surfaces that already rely on
them as documentation. This slice should move `harness review start`,
`harness review submit`, and `harness evidence submit` away from plain JSON
decoding plus fully handwritten structural validation and onto one reusable
runtime schema-validation path.

The work should keep the existing Go-owned contract layer under
`internal/contracts/` as the only field-level source of truth. It should also
keep command-specific semantic validation where the schema intentionally does
not express business rules, rather than trying to force all workflow rules into
JSON Schema.

## Scope

### In Scope

- Introduce a runtime JSON Schema validation path for structured CLI command
  inputs, backed by the existing generated input schemas under `schema/inputs/`.
- Add one explicit runtime validation dependency rather than rebuilding a second
  handwritten schema system inside the CLI.
- Apply schema-driven validation to `harness review start`,
  `harness review submit`, and `harness evidence submit`.
- Preserve or refine the current command error reporting so structural input
  failures remain actionable and machine-readable.
- Retain handwritten semantic checks only for business rules that the schema
  intentionally does not represent.
- Add focused tests that lock expected validation behavior and catch
  schema/runtime drift for these command inputs.

### Out of Scope

- Validating internal `.local/harness/*` artifact reads or other read-model
  decode paths such as `status`, `runstate`, `reviewui`, or `timeline`.
- Changing lifecycle or other command-result wire shapes.
- Converging remaining lifecycle-oriented outputs onto the v0.2 shared envelope.
- Reworking the broader workflow/state model or the tracked-plan schema.

## Acceptance Criteria

- [x] A reusable runtime validation path exists for generated command-input
      schemas and uses the existing Go-owned contract/schema surface as its only
      schema source of truth.
- [x] `harness review start` validates review spec payloads against the generated
      schema before command-specific semantic checks run.
- [x] `harness review submit` validates reviewer submission payloads against the
      generated schema before command-specific semantic checks run.
- [x] `harness evidence submit` validates each supported evidence payload against
      the matching generated schema before command-specific semantic checks run.
- [x] Structural errors such as missing required fields, wrong JSON types, or
      unknown properties fail consistently with actionable command errors.
- [x] Existing handwritten validation is reduced to business rules not already
      enforced by the schema surface.
- [x] Focused automated tests cover valid payloads, representative structural
      failures, and dependency/schema integration for the affected commands.

## Deferred Items

- Extending schema-driven validation to harness-owned artifact read paths under
  `.local/harness/`.
- Applying the same runtime validation machinery to any future structured
  command-input surfaces beyond review and evidence.
- Any output-contract cleanup for lifecycle-oriented command results.

## Work Breakdown

### Step 1: Add a reusable runtime schema-validation path for command inputs

- Done: [x]

#### Objective

Choose and integrate one runtime JSON Schema validator so command-input
validation can execute against the generated input schemas without adding a
second schema-definition surface.

#### Details

This step should define the ownership boundary clearly: `internal/contracts`
and the generated `schema/inputs/*.schema.json` files remain the schema source
of truth, while the new runtime validation layer only loads and executes that
surface. The implementation should also decide how command code selects the
correct input schema and how schema-validation failures become the existing
command error vocabulary rather than raw library errors.

#### Expected Files

- `go.mod`
- `go.sum`
- new runtime validation package(s) under `internal/`
- `schema/index.json` only if integration requires stable lookup metadata

#### Validation

- The repository builds with the chosen runtime validator dependency.
- Focused tests prove the runtime validator can load and execute the generated
  input schemas used by this slice.
- The implementation does not introduce a second handwritten schema-definition
  layer inside command code.

#### Execution Notes

Integrated `github.com/santhosh-tekuri/jsonschema/v6` as the runtime
validator and added a new shared `internal/inputschema` package that validates
raw command-input payloads against the generated input schemas. The final
runtime path executes against a contract-sync-generated
`internal/inputschema/generated_schemas.go` artifact, so built binaries carry
the same schema bytes that are written to `schema/inputs/*.schema.json`
without depending on a source checkout at runtime. Focused Red/Green/Refactor
coverage landed in `internal/inputschema` first, then contract-sync now owns
both the checked-in JSON schemas and the generated Go schema-byte map from the
same registry-driven generation pass.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The shared validator foundation is tightly coupled to
the command-boundary wiring completed in Step 2, so reviewing Step 1 in
isolation would be artificially narrow.

### Step 2: Wire review and evidence commands onto schema-driven input validation

- Done: [x]

#### Objective

Replace handwritten structural validation at the affected command-input
boundaries with the shared schema-validation path while preserving the semantic
checks that still belong in Go code.

#### Details

`harness review start`, `harness review submit`, and `harness evidence submit`
should all validate the raw JSON payload against the matching generated schema
before proceeding to business-rule checks such as inferred-step rules or
status-specific requirements that the current schemas intentionally leave to
code. This step should simplify redundant structural checks where the schema now
enforces the same requirement and keep the command error shape stable enough for
existing callers and tests.

#### Expected Files

- `internal/review/service.go`
- `internal/evidence/service.go`
- `internal/cli/app.go` if shared error shaping needs CLI-layer support
- focused tests under `internal/review/`, `internal/evidence/`, or
  `internal/cli/`

#### Validation

- Each affected command rejects structurally invalid payloads before semantic
  workflow validation runs.
- Commands still enforce the business rules that are intentionally outside the
  schema surface.
- Automated tests cover valid payloads, missing required fields, wrong types,
  and unknown properties for the affected commands.

#### Execution Notes

Replaced handwritten structural decoding/validation at the `review start`,
`review submit`, and `evidence submit` entry points with the shared schema
validation path while preserving Go-side semantic checks such as status- and
workflow-specific rules. Added focused service and CLI tests that now reject
unknown properties, wrong JSON types, malformed JSON, and other structural
input failures with stable machine-readable paths. Validation:
`go test ./internal/inputschema ./internal/review ./internal/evidence
./internal/cli ./internal/contracts ./internal/contractsync` and
`scripts/sync-contract-artifacts --check`. After the first delta review raised
coverage gaps, the slice added required-field review tests, a CLI
`review submit` schema-failure test, and publish/sync evidence negative-path
tests before rerunning the same focused validation set. Finalize follow-up
repairs then added built-binary coverage in `tests/e2e/review_workflow_test.go`
for schema-invalid `review start` input and tightened `internal/inputschema`
so repeated required-field failures keep field-level paths instead of collapsing
to the parent object.

#### Review Notes

`review-001-delta` requested changes because the first pass still missed
required-field review coverage, a CLI `review submit` schema-failure path, and
non-CI evidence negative-path coverage. Follow-up fixes added those tests.
`review-004-delta` then closed the step-bound review gate with a clean pass.
Finalize review `review-005-full` requested changes because the first schema
loader still depended on the easyharness source tree and collapsed repeated
required-field errors to parent paths. Follow-up repair `review-006-delta`
passed after moving runtime loading onto contract-sync-generated schema bytes
and splitting multi-required-field errors back to field-level paths.
`review-007-full` then requested one more tests-only follow-up to add a
built-binary invalid-input schema-validation check, and `review-008-full`
passed cleanly after that E2E coverage landed.

## Validation Strategy

- Run focused unit tests for the new runtime validation package and the affected
  review/evidence command packages.
- Re-run contract-sync coverage if the runtime integration relies on schema
  lookup metadata or other generated contract artifacts.
- Run targeted CLI tests that prove command error payloads remain actionable for
  invalid structured input.

## Risks

- Risk: Runtime schema validation could report library-shaped errors that are
  noisier or less actionable than the current command-specific validation.
  - Mitigation: Normalize validation failures into the existing command error
    vocabulary and cover representative failure cases in tests.
- Risk: The runtime validator could drift from the generated schema surface or
  load schemas in a way that accidentally creates a second source of truth.
  - Mitigation: Resolve schemas directly from the checked-in/generated contract
    artifacts and add tests around schema lookup and execution.
- Risk: Removing handwritten structural checks could accidentally drop business
  rules that are not expressible in the current schemas.
  - Mitigation: Review each existing validation branch explicitly and retain
    only the semantic rules that still belong in Go code.

## Validation Summary

Validated the schema-driven cutover with focused package and binary coverage:
`go test ./internal/inputschema ./internal/review ./internal/evidence
./internal/cli ./internal/contractsync`, `scripts/sync-contract-artifacts
--check`, `go test ./tests/smoke -run TestInstallDevHarnessDefaultsToUserLocalBin`,
and `go test ./tests/e2e -run TestReviewWorkflow`. The final E2E pass now
includes a built-binary invalid-input `review start` assertion so packaging and
generated-schema loading regressions fail outside in-process tests too.

## Review Summary

Step-bound review `review-004-delta` passed for Step 2 after the initial test
coverage gaps were repaired. Finalize review `review-005-full` surfaced two
real correctness issues in the first schema-loading implementation plus one
coverage gap; bounded repair review `review-006-delta` passed after the loader
and field-path fixes landed. A subsequent finalize reread `review-007-full`
found one remaining built-binary invalid-input coverage gap, and the final full
finalize review `review-008-full` passed with no findings.

## Archive Summary

- Archived At: 2026-04-08T08:55:15+08:00
- Revision: 1
- PR: NONE. Publish evidence should record the PR URL after archive.
- Ready: Acceptance criteria are satisfied, Step 2 and finalize review gates are
  clean, and the final candidate passed `review-008-full` after repairing the
  runtime schema loader, repeated-required-field path precision, and built-binary
  invalid-input coverage.
- Merge Handoff: Run `harness archive`, commit the tracked archive move plus the
  code, test, and closeout-summary changes, push branch `codex/issue-75`, open
  or refresh the PR, and record publish/CI/sync evidence until `harness status`
  reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added `internal/inputschema` as a reusable runtime schema-validation layer for
  command inputs backed by generated contract artifacts.
- Switched `harness review start`, `harness review submit`, and
  `harness evidence submit` onto schema-first validation while preserving
  command-specific semantic checks in Go.
- Extended contract-sync so the same generation pass now produces both the
  checked-in JSON schemas and an owned Go schema-byte map for built-binary
  runtime validation.
- Added focused service, CLI, validator, smoke, and built-binary E2E coverage
  for malformed JSON, missing required fields, wrong types, unknown properties,
  repeated required-field errors, and packaged-binary invalid-input behavior.

### Not Delivered

- No schema-driven validation was added to internal `.local/harness/*` artifact
  loads or other read-model decode paths.
- No lifecycle or other output-contract cleanup was attempted in this slice.

### Follow-Up Issues

- [#110](https://github.com/catu-ai/easyharness/issues/110) Converge remaining
  lifecycle-oriented outputs on the v0.2 shared envelope.
