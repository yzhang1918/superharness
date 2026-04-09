---
template_version: 0.2.0
created_at: "2026-04-09T09:36:00+08:00"
source_type: direct_request
source_refs:
    - '#58'
---

# Shrink `state.json` to control-plane-only runtime state

## Goal

Reduce `state.json` from a mixed cache-and-control artifact into a small
command-owned control-plane file that only preserves runtime facts that cannot
be reconstructed cheaply and safely from tracked plans or append-only local
artifacts.

This slice removes cache-style fields such as `current_node`, plan path/stem
pointers, and evidence pointers from `state.json`, keeps only the control
surface needed to coordinate execution/review/reopen/land flows, and teaches
`harness status` plus other readers to resolve the rest from the current plan,
review artifacts, and evidence records directly.

## Scope

### In Scope

- Redefine the plan-local `state.json` contract as a control-plane-only
  artifact rather than a latest-resolution cache.
- Remove cache-only fields from the runstate schema and code paths:
  `current_node`, `plan_path`, `plan_stem`, `latest_evidence`, `latest_ci`,
  `sync`, and `latest_publish`.
- Keep the control-plane fields that still represent cross-command runtime
  state: `execution_started_at`, `revision`, `reopen`,
  `active_review_round`, and `land`.
- Stop `harness status` from rewriting `state.json` during read-only status
  resolution.
- Teach evidence and status readers to discover the latest publish/ci/sync
  facts from append-only evidence artifacts instead of state-file pointers.
- Update specs, schemas, and tests so future agents can understand the
  shrunken runtime model without discovery-chat context.

### Out of Scope

- Removing `state.json` entirely.
- Removing `.local/harness/current-plan.json` or its last-landed handoff role.
- Redesigning review artifacts, evidence artifact formats, or the broader
  current-plan selection model.
- Adding compatibility shims that preserve the old cache fields alongside the
  new control-plane-only shape.

## Acceptance Criteria

- [x] `schema/artifacts/local-state.schema.json` and
      `internal/contracts/runstate.go` describe `state.json` using only the
      retained control-plane fields.
- [x] `harness status` no longer writes `state.json` just to cache the latest
      resolved node, plan path, or plan stem.
- [x] Archived publish/ci/sync readiness is still resolved correctly when the
      latest evidence must be discovered from append-only evidence records
      rather than state-file pointers.
- [x] Review, reopen, execute-start, and land flows still preserve the runtime
      coordination they need after the cache fields are removed.
- [x] Focused regression coverage proves the new model works without relying on
      removed state-file caches, and the tracked specs describe the new split
      clearly.

## Deferred Items

- Whether `execution_started_at` should eventually move out of `state.json`
  into its own append-only execution artifact.
- Whether the current active review round can later be derived fully from
  review artifacts rather than being retained in `state.json`.
- Whether a later slice should remove `state.json` entirely after the remaining
  control-plane fields have dedicated artifact homes.

## Work Breakdown

### Step 1: Redefine the runstate contract around control-plane-only fields

- Done: [x]

#### Objective

Make the `state.json` schema and normative docs describe only the runtime
fields that must survive across command boundaries.

#### Details

Update the tracked contracts so a cold reader can see that `state.json` is no
longer the place where `harness status` caches the latest resolved node or
latest evidence pointers. The retained fields should match the actual
cross-command coordination surfaces that still need persistence: execution
start, active review context, reopen mode/revision baseline, plan-local
revision, and in-progress land bookkeeping. `current-plan.json` should remain
the worktree-level source for current-plan and last-landed pointers.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- `schema/artifacts/local-state.schema.json`
- `internal/contracts/runstate.go`

#### Validation

- The docs and schema agree on the retained field set for `state.json`.
- Any generated or checked-in contract tests continue to pass after the schema
  and contract updates.

#### Execution Notes

Shrank the tracked `runstate.State` contract, local-state schema, and state
specs down to control-plane-only fields. Removed `current_node`,
plan path/stem pointers, and evidence pointer caches from the normative model
so the retained runtime surface now matches the fields still required for
execute-start, active review coordination, reopen bookkeeping, revision
tracking, and in-progress land state.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This contract-definition slice was implemented as part
of one coupled repo-wide state-model refactor, so finalize review covers the
integrated candidate rather than a standalone step-closeout round.

Reviewed the schema/spec/code contract together by re-running
`scripts/sync-contract-artifacts --check` and updating package + e2e tests
that previously assumed cache fields still existed in `state.json`.

### Step 2: Remove cache-field writes and derive runtime facts from artifacts

- Done: [x]

#### Objective

Refactor runtime readers and mutators so `state.json` is no longer treated as
the storage location for cached node/evidence/path answers.

#### Details

Delete the `harness status` cache-refresh path entirely rather than replacing
it with another no-op cache layer. Update lifecycle, review, and evidence
writers so they only persist the retained control-plane fields. Replace
state-based evidence pointers with direct latest-record discovery from the
append-only evidence directories. Any consumers that previously consulted
`state.current_node` or state-held plan/evidence pointers should instead rely
on retained control-plane fields or direct artifact inspection.

#### Expected Files

- `internal/status/service.go`
- `internal/evidence/service.go`
- `internal/lifecycle/service.go`
- `internal/review/service.go`
- `internal/reviewui/service.go`
- `internal/plan/runtime.go`

#### Validation

- `harness status` resolves the same workflow nodes without persisting a cache
  write.
- Publish/ci/sync readiness still resolves correctly after evidence is
  discovered from artifacts instead of `state.json`.
- Lifecycle and review commands still persist the control-plane fields they
  own without writing removed cache fields back into state.

#### Execution Notes

Removed cache-style `state.json` writes from `status`, lifecycle, review, and
evidence mutators. `harness status` now resolves workflow nodes without
rewriting local state, and archived publish/ci/sync readiness now loads the
latest evidence directly from append-only evidence records instead of
state-file pointers.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Runtime writer cleanup and artifact-driven evidence
reads landed together with the contract/test updates, so finalize review covers
the cohesive candidate instead of a separate step-closeout round.

Reviewed the runtime readers/writers against lifecycle coverage for execute,
review, reopen, archive, land, status, review UI, and CLI/e2e flows so the
remaining state fields are only the ones still used as control-plane inputs.

### Step 3: Lock the model with regression coverage and cleanup docs

- Done: [x]

#### Objective

Prove the control-plane-only model end to end and remove stale references to
the old cache semantics.

#### Details

Add focused regression tests that fail if status reintroduces no-op
`state.json` rewrites, if archived readiness still secretly depends on
state-held evidence pointers, or if retained review/reopen/land flows regress
after the field removal. Sweep the touched docs and tests for outdated
descriptions of `state.json` as a thin latest-node cache so future agents do
not rediscover the old mental model.

#### Expected Files

- `internal/status/service_test.go`
- `internal/evidence/service_test.go`
- `internal/lifecycle/service_test.go`
- `internal/review/service_test.go`
- `internal/reviewui/service_test.go`
- `README.md`

#### Validation

- New and updated package tests cover the removal of cache-field writes and the
  artifact-driven evidence resolution path.
- `go test ./...` passes once the slice is complete.

#### Execution Notes

Updated regression coverage across status, evidence, lifecycle, review UI,
runstate, timeline, UI, CLI, and E2E tests so the suite now locks in the
control-plane-only state model. The tracked specs were also rewritten to stop
describing `state.json` as a latest-node/evidence cache. Follow-up finalize
review repairs then tightened the removed-key assertions to include
`plan_path`/`plan_stem`, added multi-record latest-evidence regression tests,
updated generated contract wording that still called local state a cache, and
scoped artifact-driven evidence lookup to the current revision so reopen does
not reuse stale publish/ci/sync records. A later review follow-up also added
raw on-disk `state.json` assertions on mutating unit/e2e paths so writer
commands cannot silently reintroduce removed cache keys.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Regression-locking this slice depended on the integrated
repo-wide change set, so finalize review covers the resulting candidate rather
than a separate step-closeout round.

Validated the regression net with focused package runs and a full `go test ./...`
pass after the state-contract cleanup, including the review workflow E2E that
reads the persisted local state artifact directly. Finalize review rounds
`review-001-full` and `review-002-full` surfaced and drove fixes for residual
docs/test gaps plus reopen revision scoping in artifact-backed evidence lookup.
`review-003-full` then tightened writer-path raw-state assertions for removed
cache keys on archive/e2e mutating paths.

## Validation Strategy

- Start with focused package tests around the runstate readers/writers because
  the main risk is accidentally removing a control-plane dependency that a
  later command still needs.
- Add a direct regression that proves consecutive `harness status` reads stop
  mutating `state.json` when nothing else changes.
- Re-run the broader Go test suite once the targeted status/evidence/lifecycle
  flows are green.

## Risks

- Risk: Removing fields that look like cache may accidentally drop a hidden
  control-plane dependency in review, reopen, or land flows.
  - Mitigation: Keep the retained-field list explicit in Step 1 and add
    package-level regression coverage for each retained command path in Step 3.
- Risk: Direct evidence discovery could pick the wrong record if the selection
  rules are underspecified.
  - Mitigation: Define and test one deterministic “latest evidence” rule in
    code and specs before deleting the old state-pointer path.

## Validation Summary

- `go build ./...`
- `scripts/sync-contract-artifacts --check`
- `go test ./internal/evidence ./internal/reviewui ./internal/review ./internal/lifecycle ./internal/status -count=1`
- `go test ./internal/cli ./internal/runstate ./internal/timeline ./internal/ui -count=1`
- `go test ./tests/e2e -count=1`
- `go test ./... -count=1`
- `go test ./internal/evidence ./internal/status ./internal/lifecycle -count=1`
- `go test ./internal/cli ./internal/runstate ./internal/timeline ./internal/ui ./tests/e2e -count=1`
- `scripts/sync-contract-artifacts`
- `scripts/sync-contract-artifacts --check`

## Review Summary

Finalize review `review-004-full` passed after iterative repair rounds
`review-001-full` through `review-003-full` surfaced and drove the remaining
docs/test/revision-scoping fixes. The final reviewer pass found no remaining
correctness, test-coverage, or docs-consistency blockers for the
control-plane-only `state.json` model.

## Archive Summary

- Archived At: 2026-04-09T10:21:51+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate has a passing finalize review and is ready to archive,
  then move through publish evidence toward merge approval.
- Merge Handoff: Archive the plan, record publish/CI/sync evidence for the
  archived candidate, and wait for explicit human merge approval before land.

## Outcome Summary

### Delivered

- Reframed `state.json` as a control-plane-only artifact and removed cached
  node/path/evidence pointer fields from contracts, schemas, and runtime
  writers.
- Stopped `harness status` from mutating `state.json` on read-only resolution
  and switched publish/CI/sync lookup to artifact-driven discovery.
- Locked the new model with package, CLI, UI, timeline, lifecycle, and E2E
  regression coverage, including reopen revision scoping and raw on-disk
  `state.json` assertions for mutating paths.

### Not Delivered

NONE.

### Follow-Up Issues

- Consider whether `execution_started_at` should move into its own durable
  execution-start artifact once the control-plane-only `state.json` slice is
  stable.
