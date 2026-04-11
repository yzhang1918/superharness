---
template_version: 0.2.0
created_at: "2026-04-11T21:38:59+08:00"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/36
size: S
---

# Close Issue 36 With Focused Parsing Coverage

## Goal

Close `#36` by adding focused fuzz or property-style coverage for the
highest-value parsing-heavy harness paths without expanding into deterministic
resilience work, repo-level lifecycle E2E, or broader concurrency hardening.

This slice should make a clear repository-level judgment about which candidate
paths deserve fuzz/property investment now. The expected closeout is: plan
markdown parsing and command-input schema decoding gain meaningful new
coverage, while review artifact readers and historical evidence-record readers
are explicitly judged out of scope for this issue because their current
deterministic coverage is already stronger than the remaining risk reduction
available from a first fuzzing pass.

## Scope

### In Scope

- Add package-level Go fuzz tests for the plan markdown parsing surface in
  `internal/plan`, centered on `LintFile`, `LoadFile`, and the shared parsing
  helpers they exercise.
- Add property-style or seed-based invariants in `internal/plan` that check
  stable relationships between linting and document loading on canonical plan
  inputs.
- Add focused fuzz or property-style coverage for `internal/inputschema`
  normalization logic, especially JSON-pointer rendering, quoted-property
  extraction, and parent-issue pruning.
- Keep evidence command coverage aligned with the schema layer where that helps
  prove schema-derived error paths still surface correctly through a real
  command entrypoint.
- Leave an execution trail strong enough that archive or issue closeout can say
  `#36` was intentionally closed after evaluating plan lint, review artifacts,
  and evidence payload decoding rather than only touching one of them.

### Out of Scope

- `tests/resilience/`, deterministic failure-path coverage, or any work owned
  by `#37`.
- Repo-level lifecycle E2E coverage, fixture expansion, or changes under
  `tests/support/`.
- Broader concurrency or lock-behavior coverage owned by `#56`.
- Deep fuzzing of `internal/reviewui` artifact recovery or `internal/evidence`
  historical record loading beyond what is needed to justify why those readers
  are not the primary targets for closing `#36`.
- Schema redesigns, command-shape changes, or new issue/follow-up creation.

## Acceptance Criteria

- [x] `internal/plan` has new package-level fuzz or property-style coverage for
      parsing-heavy plan inputs, and the targeted surfaces do not panic when
      fed arbitrary data plus seeded canonical plan examples.
- [x] Canonical valid-plan seeds assert at least one stable plan invariant such
      as: lint success and document loading stay aligned, current-step
      detection remains deterministic, or archive-readiness helpers stay
      coherent after successful parsing.
- [x] `internal/inputschema` has new fuzz or property-style coverage for path
      rendering and validation-error normalization, including nested-array
      paths, quoted-property extraction, and parent-issue pruning behavior.
- [x] Any touched `internal/evidence` regression coverage stays narrowly tied
      to schema-decoding behavior and does not expand into resilience-style
      malformed-artifact recovery.
- [x] The slice documents, through plan execution notes and closeout, that
      review artifact readers and historical evidence-record readers were
      evaluated but were not required additions for closing `#36`.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Add focused fuzz coverage for plan markdown parsing

- Done: [x]

#### Objective

Introduce high-signal fuzz or property-style tests in `internal/plan` that
exercise the mixed YAML and Markdown parsing surface without widening into
repo-level fixtures or resilience infrastructure.

#### Details

Target the shared parser surface behind `LintFile` and `LoadFile`, not a new
parallel helper API. Seed the fuzz corpus with valid rendered plans plus a
small number of invalid structured variants so the engine learns both success
and failure shapes. Prefer invariants that stay useful under future template
evolution, such as no panic, stable error/result shape expectations for seeded
examples, and coherence between successful lint and successful document load
for canonical valid plans.

#### Expected Files

- `internal/plan/lint_test.go`
- `internal/plan/document_test.go`
- `internal/plan/*_test.go`

#### Validation

- `go test ./internal/plan`
- `go test -fuzz=Fuzz -run=^$ ./internal/plan` for a bounded fuzz pass

#### Execution Notes

Added `internal/plan/fuzz_test.go` with canonical seed properties that keep
`LintFile` and `LoadFile` aligned across active, archived, and archived
lightweight plans; added a tracked-plan corpus check that every repository plan
that lints cleanly also loads cleanly; and added a bounded file-based fuzz
target asserting `LintFile` success implies `LoadFile` success while document
helper methods stay panic-free on arbitrary inputs. Validated with
`go test ./internal/plan` and
`go test -run=^$ -fuzz=FuzzLintFileAndLoadFileAgreement -fuzztime=2s ./internal/plan`.
After `review-001-delta`, tightened the fuzz baseline so exact canonical seeds
must continue linting and loading cleanly, closing the one-way alignment gap
the reviewer called out.

#### Review Notes

`review-001-delta` passed with one non-blocking correctness finding: the
original fuzz target only asserted `LintFile` success implied `LoadFile`
success. Fixed that gap by requiring exact canonical seeds in the fuzz baseline
to continue linting and loading cleanly, then reran
`go test ./internal/plan` and
`go test -run=^$ -fuzz=FuzzLintFileAndLoadFileAgreement -fuzztime=2s ./internal/plan`.

### Step 2: Cover schema-driven input decoding and keep issue closeout bounded

- Done: [x]

#### Objective

Add focused fuzz or property-style coverage for command-input schema
normalization in `internal/inputschema`, then keep any supporting evidence
tests tightly limited to proving those normalized errors still reach a real
command surface.

#### Details

Concentrate on the path-shaping logic that easyharness owns locally:
`renderInstanceLocation`, `propertiesFromValidationMessage`,
`renderIssueDetails`, and `pruneParentIssues`. Use schema-backed seeds so the
tests stay grounded in real command inputs instead of arbitrary generated
structures. If `internal/evidence/service_test.go` needs small updates, keep
them at the command-boundary level and do not broaden into malformed historical
record loading or status-side conservative behavior. During execution, record
the explicit judgment that `internal/reviewui` already has strong deterministic
coverage for malformed and partial artifacts, so it is not the first fuzzing
target needed to close `#36`.

#### Expected Files

- `internal/inputschema/validator_test.go`
- `internal/inputschema/*_test.go`
- `internal/evidence/service_test.go`

#### Validation

- `go test ./internal/inputschema ./internal/evidence`
- `go test -fuzz=Fuzz -run=^$ ./internal/inputschema` for a bounded fuzz pass

#### Execution Notes

Added `internal/inputschema/fuzz_test.go` in the package-under-test so the
slice can exercise unexported normalization helpers directly. The new coverage
adds deterministic helper properties for JSON-pointer rendering,
quoted-property splitting, and parent-issue pruning, plus a schema-backed fuzz
target that checks `Validate` never returns empty or de-normalized issue paths
and never leaks parent-child path pairs after pruning. Kept `internal/evidence`
unchanged and revalidated it only as a command-boundary consumer of the schema
layer. This step intentionally did not deepen `internal/reviewui` or
historical evidence-record fuzzing because existing deterministic malformed and
partial-artifact tests already cover that recovery path more strongly than a
first bounded fuzz pass would. Validated with
`go test ./internal/inputschema`,
`go test -run=^$ -fuzz=FuzzValidateNormalizesIssuePaths -fuzztime=2s ./internal/inputschema`,
and `go test ./internal/inputschema ./internal/evidence`.

#### Review Notes

`review-002-delta` passed cleanly with no findings after inspecting the new
`internal/inputschema` helper properties, schema-backed fuzz target, generated
schemas, and the command-boundary evidence consumers. The round confirmed this
step materially advances `#36` while staying bounded away from `reviewui`,
`tests/resilience`, `tests/support`, and historical evidence-reader hardening.

## Validation Strategy

- Use package-level `go test` runs for the touched units instead of repo-level
  E2E suites.
- Run bounded Go fuzz passes only in `internal/plan` and `internal/inputschema`
  so the work stays isolated from `tests/resilience/`, `tests/support/`, and
  other worktrees handling `#37` plus `#56`.
- Treat closeout as incomplete unless the final validation record can explain
  why review artifact readers and historical evidence readers were evaluated
  but not made primary fuzz targets for this issue.

## Risks

- Risk: Fuzz targets around file-based plan parsing can become flaky or too
  coupled to temporary filesystem setup.
  - Mitigation: Seed from deterministic tempdir fixtures and keep invariants
    focused on no-panic and stable parser relationships rather than brittle
    exact-error text for random inputs.
- Risk: The slice could drift into resilience or malformed-artifact hardening
  already separated into `#37`.
  - Mitigation: Keep all new work inside package tests for `internal/plan`,
    `internal/inputschema`, and narrowly scoped evidence regressions.
- Risk: Closing `#36` could look premature if the plan does not explicitly
  justify why review artifact readers were not fuzzed.
  - Mitigation: Make that judgment explicit in execution and archive summaries
    and only archive once the added coverage plus rationale would let a cold
    reviewer understand the decision.

## Validation Summary

- Added `internal/plan/fuzz_test.go` with canonical active, archived, and
  archived lightweight plan seeds; a tracked-plan corpus lint/load agreement
  property; and a bounded fuzz target covering arbitrary plan-file inputs.
- Added `internal/inputschema/fuzz_test.go` with helper-level path
  normalization properties plus a schema-backed fuzz target for `Validate`.
- Validation runs:
  - `go test ./internal/plan`
  - `go test -run=^$ -fuzz=FuzzLintFileAndLoadFileAgreement -fuzztime=2s ./internal/plan`
  - `go test ./internal/inputschema`
  - `go test -run=^$ -fuzz=FuzzValidateNormalizesIssuePaths -fuzztime=2s ./internal/inputschema`
  - `go test ./internal/plan ./internal/inputschema ./internal/evidence`

## Review Summary

- `review-001-delta`
  - passed with one non-blocking correctness finding about one-way lint/load
    agreement in the initial fuzz target
  - repaired by requiring exact canonical seeds in the fuzz baseline to
    continue linting and loading cleanly
- `review-002-delta`
  - passed clean with no findings on the `internal/inputschema` coverage and
    bounded issue-closure rationale
- `review-003-full`
  - passed clean with no findings across correctness, tests, and docs
    consistency for the full candidate
- The final candidate is archive-ready after the clean full review and green
  validation runs.

## Archive Summary

- Archived At: 2026-04-11T21:55:34+08:00
- Revision: 1
- PR: NONE. The candidate has not been pushed or opened as a PR yet.
- Ready: The branch is archive-ready locally after the clean finalize review
  and focused parsing-coverage validation runs.
- Merge Handoff: Archive the plan, commit the archive move, push
  `codex/close-issue-36-parsing-coverage`, open or update the PR, and record
  publish, CI, and sync evidence before treating the candidate as waiting for
  merge approval.

## Outcome Summary

### Delivered

- Added focused parsing-heavy fuzz/property coverage in `internal/plan` that
  exercises plan-file parsing across canonical seeds, tracked-plan corpus
  inputs, and bounded arbitrary plan content.
- Added focused parsing-heavy fuzz/property coverage in `internal/inputschema`
  for JSON-pointer rendering, quoted-property extraction, parent-issue
  pruning, and schema-backed validation-error normalization.
- Documented and validated the bounded closure rationale for `#36`: keep
  `internal/evidence` at the command-boundary consumer level and rely on the
  repository's existing deterministic malformed-artifact tests in
  `internal/reviewui` rather than widening this slice into resilience work.

### Not Delivered

- No new fuzz target was added for `internal/reviewui` artifact recovery.
- No new fuzz target was added for historical evidence-record loading in
  `internal/evidence`.
- No deterministic resilience coverage, repo-level lifecycle E2E expansion, or
  `tests/support/` work was added in this slice.

### Follow-Up Issues

NONE
