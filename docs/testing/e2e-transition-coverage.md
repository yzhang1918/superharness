# E2E Transition Coverage

## Purpose

This document reports what the repo-level E2E suite explicitly covers today.
It is not a second workflow spec.

The normative workflow contract remains:

- `docs/specs/state-transitions.md`
- `docs/specs/state-model.md`

The machine-readable catalog that backs this report lives in:

- `tests/e2e/coverage_test.go`

## Coverage Model

The v0.2 transition matrix currently has 27 transition families after the
`new-step` proofread that makes the intermediate `execution/finalize/fix`
state explicit before a new unfinished step is added.

The bounded repo-level E2E suite now reaches all 10 canonical workflow nodes:

- `idle`
- `plan`
- `execution/step-<n>/implement`
- `execution/step-<n>/review`
- `execution/finalize/review`
- `execution/finalize/fix`
- `execution/finalize/archive`
- `execution/finalize/publish`
- `execution/finalize/await_merge`
- `land`

This report uses a bounded route model instead of pretending infinite
loop-capable workflows can be exhaustively enumerated. The current bounded
model is:

- canonical plan shape: 3 planned steps before any reopened follow-up
- one step-review repair loop
- one finalize repair loop
- progressive publish evidence while status remains in
  `execution/finalize/publish`
- one `land` stability check before `harness land complete`
- one `reopen --mode new-step` consumption that grows the plan from 3 steps to
  4

## Current Explicit Coverage

The repo-level suite now has 8 scenario families:

- `review_workflow`
  - `TestReviewWorkflowWithBuiltBinary`
  - covers `idle -> plan`, `plan -> execute`, clean step review, and clean
    finalize review into archive closeout
- `review_repair_loop`
  - `TestReviewRepairLoopsWithBuiltBinary`
  - covers one step-review repair loop and one finalize-review repair loop
- `archive_reopen_finalize_fix`
  - `TestArchiveReopenFinalizeFixWithBuiltBinary`
  - covers archive handoff plus `publish -> finalize/fix` via
    `reopen --mode finalize-fix`
- `reopen_new_step`
  - `TestReopenNewStepWithBuiltBinary`
  - covers `publish -> finalize/fix` via `reopen --mode new-step`, the
    pending `new-step` wait state, and the later resume at the new
    `step-k/implement`
- `publish_handoff`
  - `TestPublishHandoffWithBuiltBinary`
  - covers publish self-loops and the later `publish -> await_merge`
- `land_workflow`
  - `TestLandWorkflowWithBuiltBinary`
  - covers `await_merge -> land -> idle` including the `land` self-loop
- `await_merge_reopen_finalize_fix`
  - `TestAwaitMergeReopenFinalizeFixWithBuiltBinary`
  - covers the merge-ready rollback origin
    `await_merge -> finalize/fix` via `reopen --mode finalize-fix`
- `await_merge_reopen_new_step`
  - `TestAwaitMergeReopenNewStepWithBuiltBinary`
  - covers the merge-ready rollback origin
    `await_merge -> finalize/fix` via `reopen --mode new-step`, then the
    derived resume at the first newly added step

So the current explicit repo-level E2E coverage is:

- scenarios: 8
- canonical nodes: 10 / 10
- transition families: 27 / 27 in the maintained bounded scenario catalog

`tests/e2e/coverage_test.go` now enforces both halves of the bounded catalog:

- every canonical transition family must be covered by at least one scenario
- the canonical family list must stay in sync with
  `docs/specs/state-transitions.md`, including the state-preserving updates

That means future transition-spec edits and catalog drift now fail fast instead
of silently drifting.

This is still a maintained transition-coverage matrix, not auto-derived
executable branch coverage. The repo-level E2E assertions are the evidence for
the mapped transitions; the catalog is the durable index that says which
scenario proves which transition family.

## Covered Transition Families

All 27 canonical transition families from
`docs/specs/state-transitions.md` are now covered by at least one repo-level
real-binary scenario:

- `idle_to_plan`
- `plan_self`
- `plan_to_step_implement`
- `step_implement_self`
- `step_implement_to_review`
- `step_implement_to_next_step_implement`
- `step_implement_to_finalize_review`
- `step_review_self`
- `step_review_to_step_implement_clean`
- `step_review_to_step_implement_repair`
- `finalize_review_self`
- `finalize_review_to_finalize_fix`
- `finalize_review_to_finalize_archive`
- `finalize_fix_self`
- `finalize_fix_to_finalize_review`
- `finalize_fix_to_new_step_implement`
- `finalize_archive_self`
- `finalize_archive_to_publish`
- `publish_self`
- `publish_to_await_merge`
- `publish_to_finalize_fix`
- `publish_to_finalize_fix_new_step`
- `await_merge_to_land`
- `await_merge_to_finalize_fix`
- `await_merge_to_finalize_fix_new_step`
- `land_self`
- `land_to_idle`

## Remaining E2E Gaps

There are no remaining gaps at the bounded transition-family level for the
repo-level E2E suite.

The earlier lifecycle-expansion slice intentionally deferred resilience and
broader runstate-interleaving follow-up. Those adjacent gaps are now covered by:

- `tests/resilience/` for malformed current-plan pointers, degraded
  review/evidence artifact reads, and archive/reopen rollback-family safety
  cases
- `tests/e2e/runstate_concurrency_test.go` for deterministic archive, reopen,
  evidence, and status interleavings around revision-scoped archived evidence
  and fail-fast lock contention

The remaining follow-up after those adjacent suites is:

- fuzzing or property-style coverage for parsing-heavy paths
- unbounded route enumeration beyond the documented loop budgets

## Proofread Notes

The tracked state docs were corrected before the new E2E scenarios were added:

- `reopen --mode new-step` no longer claims an immediate jump from archived
  publish/await-merge states to `execution/step-<m>/implement`
- the intermediate `execution/finalize/fix` state is now explicit until the
  first reopened step exists
- `execution/finalize/fix -> execution/step-<m>/implement` is now documented
  as a derived transition from tracked plan edits
- `execution/finalize/fix -> execution/finalize/fix` is now documented as a
  state-preserving update while finalize repair or new-step preparation
  continues
