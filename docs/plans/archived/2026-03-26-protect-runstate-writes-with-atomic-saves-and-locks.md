---
template_version: 0.2.0
created_at: "2026-03-26T22:54:50+08:00"
source_type: issue
source_refs:
    - '#51'
---

# Protect runstate writes with atomic saves and state locks

## Goal

Prevent local harness runstate files from becoming corrupted when multiple
commands write them close together or when a write is interrupted mid-flight.

This slice hardens the CLI-owned persistence layer by making `state.json` and
`current-plan.json` writes atomic, and by serializing plan-local `state.json`
mutations behind a shared fail-fast lock. It intentionally keeps `harness
status` writing its thin cache on every run rather than changing the cache
policy in the same slice.

## Scope

### In Scope

- Make `internal/runstate` persist `state.json` with same-directory temp-file
  replacement instead of direct overwrite writes.
- Apply the same atomic write behavior to `current-plan.json` so the local
  runstate write strategy is consistent across CLI-owned JSON files.
- Add a shared per-plan state-mutation lock for command paths that rewrite
  `state.json`, with clear fail-fast errors when another state mutation is
  already in progress.
- Route `status`, `evidence`, `lifecycle`, and any other plan-local
  `state.json` writer through the shared state lock without collapsing review
  mutation locking into the same primitive.
- Add deterministic regression coverage for atomic-save helpers and lock
  contention on representative command flows.
- Update tracked docs/specs so the atomic-save and fail-fast state-lock
  behavior is explicit for future agents and maintainers.

### Out of Scope

- Changing `harness status` into a read-only command or skipping writes when
  the cached node is unchanged.
- Replacing the existing review-mutation lock with a broader unified mutation
  lock for every local plan artifact.
- Adding background retries, waiting behavior, or automatic backoff when the
  state lock is already held.
- Broad resilience work beyond the targeted runstate-write corruption and
  contention coverage needed for `#51`.

## Acceptance Criteria

- [x] `state.json` and `current-plan.json` are written via atomic replacement
      rather than direct `os.WriteFile` overwrites.
- [x] Commands that mutate a plan-local `state.json` fail fast with a clear,
      user-facing error when another state mutation for the same plan is
      already in progress.
- [x] Focused regression tests cover at least one lock-contention path and the
      persistence helper behavior needed to keep runstate JSON parseable after
      interrupted or overlapping saves.
- [x] The tracked CLI/state-model docs describe the new atomic-save and
      per-plan state-lock expectations for CLI-owned runstate files.

## Deferred Items

- Whether `harness status` should later skip cache writes when the resolved
  node is unchanged.
- Whether review locking and state locking should later converge on a single
  broader mutation primitive.
- Whether additional evidence/reopen/archive E2E concurrency coverage is worth
  adding after the targeted regression tests land.

## Work Breakdown

### Step 1: Add atomic runstate persistence helpers

- Done: [x]

#### Objective

Centralize CLI-owned JSON persistence in `internal/runstate` so both
`state.json` and `current-plan.json` use atomic same-directory replacement.

#### Details

Introduce a reusable helper that writes marshaled JSON to a temp file in the
destination directory and renames it into place only after the content is
fully staged. Preserve existing file locations and payload shapes so the slice
changes durability semantics without redesigning the runstate model.

#### Expected Files

- `internal/runstate/state.go`
- `internal/runstate/state_test.go`

#### Validation

- New helper-level tests prove the atomic persistence path produces parseable
  JSON files at the canonical runstate locations.
- Existing runstate callers continue loading the same JSON schema after the
  helper swap.

#### Execution Notes

Added an atomic JSON persistence helper in `internal/runstate/state.go` and
switched both `SaveState` and `saveCurrentPlan` to use same-directory temp-file
replacement plus `fsync` before rename. Added `internal/runstate/state_test.go`
to verify both `state.json` and `current-plan.json` are rewritten to the exact
expected JSON payload without stale trailing content from older writes.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This helper-only precursor was completed together with
Steps 2 and 3 as one tightly coupled runstate-hardening slice, so a separate
Step 1 review would have duplicated the later broader risk scan.

### Step 2: Serialize plan-local state mutations with a shared fail-fast lock

- Done: [x]

#### Objective

Ensure every command path that rewrites `state.json` acquires the same per-plan
state lock before persisting local runstate.

#### Details

Add a plan-local lock helper separate from the existing review-mutation lock so
`status`, `evidence`, and lifecycle flows can fail fast when another
state-mutation command already owns the lock for the same plan. Keep the review
lock in place for review-specific sequencing rather than broadening this slice
into a repository-wide mutation-lock refactor.

#### Expected Files

- `internal/runstate/state.go`
- `internal/status/service.go`
- `internal/evidence/service.go`
- `internal/lifecycle/service.go`
- `internal/review/service.go`

#### Validation

- Representative state-writing commands return a clear contention error when
  the shared state lock is already held for the target plan.
- Normal single-command flows still update `state.json` successfully after the
  lock integration.

#### Execution Notes

Added a shared per-plan state-mutation lock in `internal/runstate/state.go` and
wired `status`, `evidence`, lifecycle commands, and review start/aggregate
through it so their `state.json` read-modify-write paths are serialized and
fail fast on contention. Lifecycle and evidence loaders now acquire the lock
before loading local state, and `status` now refuses to refresh the
`current_node` cache when another command is already mutating plan-local state.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 is the main behavior change, but it remained
entangled with Step 3's contention coverage and doc contract updates, so a
separate step-bound review before those protections landed would have been
misleading duplication.

### Step 3: Lock the behavior with regression coverage and docs

- Done: [x]

#### Objective

Make the new durability and contention behavior explicit through focused tests
and tracked contract updates.

#### Details

Add deterministic regression coverage around lock contention and runstate
writing, favoring package-level tests over broad infrastructure fuzzing. Update
the relevant tracked docs so future agents know that CLI-owned runstate writes
are atomic and that plan-local `state.json` mutation now uses a fail-fast
state lock.

#### Expected Files

- `internal/status/service_test.go`
- `internal/evidence/service_test.go`
- `internal/lifecycle/service_test.go`
- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`

#### Validation

- Focused package-level tests for the touched services pass with the new
  lock-contention and persistence coverage.
- `go test ./...` passes once the slice is ready for broader verification.

#### Execution Notes

Added contention tests in `internal/status/service_test.go`,
`internal/evidence/service_test.go`, and `internal/lifecycle/service_test.go`
that hold the new state lock and assert the commands fail clearly. Updated
`docs/specs/state-model.md` and `docs/specs/cli-contract.md` to document
atomic CLI-owned runstate writes and the shared fail-fast per-plan state lock.
Validated with `go test ./internal/runstate ./internal/status ./internal/evidence ./internal/lifecycle ./internal/review`
and `go test ./...`. Finalize review `review-001-full` then found a locked-plan
TOCTOU gap plus two regression gaps, so the slice added
`plan.DetectCurrentPathLocked`, review state-lock contention tests, and a
rename-failure atomic-save test before rerunning validation.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 3 only added the deterministic contention coverage
and spec wording needed to lock in the completed Step 2 behavior, so a
separate closeout review would have duplicated the finalize review that follows
immediately after the full slice.

## Validation Strategy

- Add helper-level tests for atomic runstate persistence because the main risk
  is malformed local JSON after interrupted or overlapping writes, not parser
  behavior itself.
- Add focused service-level contention tests that hold the new state lock and
  assert representative commands fail fast without mutating local state.
- Run targeted package tests during implementation, then `go test ./...`
  before archive so the persistence changes are validated against broader CLI
  expectations.

## Risks

- Risk: Introducing a new shared state lock could deadlock or create confusing
  interaction with existing review locking if command ordering is inconsistent.
  - Mitigation: Keep the new lock narrowly scoped to `state.json` mutation,
    preserve the existing review lock semantics, and add contention tests that
    exercise representative command entry points.
- Risk: Atomic-save refactoring could accidentally change file permissions,
  directories, or payload formatting relied on by existing tests.
  - Mitigation: Reuse the existing file paths and JSON marshaling code, and add
    direct persistence tests around the canonical runstate files before wiring
    callers to the helper.

## Validation Summary

- Added helper-level persistence coverage in `internal/runstate/state_test.go`
  for exact JSON replacement plus a rename-failure rollback case that preserves
  the original file.
- Added fail-fast state-lock contention coverage for `status`, `evidence`,
  lifecycle, and review state-writer entry points.
- Added `internal/plan/current_test.go` coverage for the locked-plan-stem guard
  that prevents mixing one plan's document with another plan's `state.json`
  after lock acquisition.
- Validated the repaired candidate with `go test ./internal/plan ./internal/runstate ./internal/status ./internal/evidence ./internal/lifecycle ./internal/review`
  and `go test ./...`.

## Review Summary

- Finalize review `review-001-full` found three blocking issues: the initial
  lock-integration still had a plan-selection TOCTOU gap, review start/aggregate
  lacked direct state-lock contention tests, and atomic-save coverage did not
  exercise rename-failure rollback.
- The repair slice added `plan.DetectCurrentPathLocked`, extended review tests
  to cover shared state-lock contention, and added a rename-failure
  `writeJSONAtomic` regression.
- Follow-up finalize review `review-002-full` passed clean across the
  `correctness`, `tests`, and `docs_consistency` slots with no findings.

## Archive Summary

- Archived At: 2026-03-26T23:18:23+08:00
- Revision: 1
- PR: not created yet; publish evidence should record the PR URL after archive.
- Ready: `review-002-full` passed clean, acceptance criteria are satisfied, and
  the validation evidence listed above makes the candidate ready for
  `harness archive`.
- Merge Handoff: Archive the plan, commit the tracked move plus the code/doc
  changes, push the branch, open or update the PR, and record publish/CI/sync
  evidence before treating the candidate as merge-ready.

## Outcome Summary

### Delivered

- Hardened CLI-owned runstate persistence so both `state.json` and
  `current-plan.json` use atomic same-directory replacement instead of direct
  overwrite writes.
- Added a shared fail-fast per-plan state-mutation lock and wired the main
  `state.json` writers through it, including `status`, `evidence`, lifecycle
  flows, and review start/aggregate.
- Closed the locked-plan TOCTOU gap by requiring post-lock plan detection to
  resolve to the same plan stem, and documented the resulting runstate
  durability contract in the tracked specs.
- Added deterministic regression coverage for atomic-save rollback, locked-plan
  detection, and shared state-lock contention across the touched command paths.

### Not Delivered

- `harness status` still rewrites the thin cache on every successful run even
  when the resolved node is unchanged.
- Review sequencing still uses a separate `.review-mutation.lock`; this slice
  did not collapse review and state serialization into one broader primitive.
- This slice added targeted regression coverage, not the broader
  archive/evidence/status concurrency scenarios suggested as future hardening.

### Follow-Up Issues

- #56 Add broader concurrency coverage for archive, reopen, evidence, and
  status runstate updates
- #57 Evaluate whether review and state mutations should share one lock
  primitive
- #58 Consider skipping no-op current_node cache writes in harness status
