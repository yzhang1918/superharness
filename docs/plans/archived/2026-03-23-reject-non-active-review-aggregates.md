---
template_version: 0.2.0
created_at: "2026-03-23T23:10:00+08:00"
source_type: issue
source_refs:
    - '#7'
---

# Reject non-active review aggregate updates

## Goal

Prevent late aggregation from an older review round from rewriting
`state.active_review_round` and regressing `harness status` or archive
readiness back to stale review state.

This slice keeps the current v0.1 single-active-round model and enforces it at
`harness review aggregate` time only. It intentionally does not make
`harness review start` reject overlapping rounds yet, because the repository
still lacks an explicit cancel, abandon, or supersede flow for broken rounds.

## Scope

### In Scope

- Reject `harness review aggregate --round <round-id>` unless `<round-id>`
  matches the current active review round for the executing plan.
- Leave current local review state untouched when a non-active round tries to
  aggregate late.
- Add focused regression tests for the stale older-round path and the normal
  active-round success path.
- Update the CLI contract docs so the active-round-only aggregate behavior is
  explicit for future agents and maintainers.

### Out of Scope

- Rejecting `harness review start` when another review round is already active.
- Adding a cancel, abandon, or supersede workflow for broken review rounds.
- Modeling multiple simultaneous active review rounds in local state.
- Adding fuzz or resilience infrastructure beyond the targeted deterministic
  regression coverage needed for this bug.

## Acceptance Criteria

- [x] `harness review aggregate` succeeds only when `--round` matches the
      current `state.active_review_round.round_id` for the executing plan.
- [x] Aggregating an older round after a newer round has started returns a
      clear error, does not overwrite `state.active_review_round`, and does not
      persist a new aggregate result for the rejected request.
- [x] Focused package-level contract/regression tests cover both the normal
      active-round aggregate path and the stale older-round rejection path.
- [x] `docs/specs/cli-contract.md` documents that `review aggregate` is only
      valid for the current active round in the v0.1 review model.

## Deferred Items

- Whether `harness review start` should later reject overlapping rounds once
  the CLI has an explicit supersede or abandon mechanism.
- Whether the project needs a separate repair-oriented command for historical
  aggregate backfill instead of overloading `review aggregate`.

## Work Breakdown

### Step 1: Enforce active-round-only aggregate validation

- Done: [x]

#### Objective

Teach `review aggregate` to refuse stale or non-current round IDs before any
aggregate artifact or state rewrite can occur.

#### Details

Validate the requested round against the current executing plan state and
return a clear command error when the round is not the active one. Preserve
the current active review round and do not write a new aggregate artifact for a
rejected stale request. Keep `review start` behavior unchanged in this slice so
operators are not stranded when a round becomes unusable before aggregation.

#### Expected Files

- `internal/review/service.go`

#### Validation

- The command rejects non-active round IDs before mutating local review state.
- The active-round success path still produces aggregate output and state
  updates for the current round.

#### Execution Notes

Added an early active-round guard in `internal/review/service.go` so
`harness review aggregate` now rejects any round ID other than the current
`state.active_review_round.round_id` before loading manifests or writing an
aggregate artifact. This preserves the in-flight active round when an older
round tries to aggregate late and keeps `review start` behavior unchanged for
now because the CLI still lacks an explicit abandon or supersede path.
Follow-up review surfaced a remaining inter-process race, so the final
implementation also serializes `review start` and `review aggregate` with an
OS-backed plan-local file lock before either command reads or rewrites review
state. Finalize review then tightened the flow one more time by carrying the
locked plan path through subsequent plan loading so the command always mutates
the same plan it locked.

#### Review Notes

`review-001-delta` found that the initial active-round guard only validated a
snapshot of local state, so `Aggregate` was updated to revalidate the active
round before persistence. `review-002-delta` then identified that validation
and persistence still interleaved non-atomically with `review start`, so the
command pair was serialized with a shared review-mutation lock. `review-003-delta`
found that a directory sentinel could leave a stale lock after process death,
which was replaced with an OS-backed file lock. `review-004-delta` passed clean
with no findings.

### Step 2: Lock the behavior with regression coverage and docs

- Done: [x]

#### Objective

Make the new aggregate guard durable through focused tests and contract text.

#### Details

Add deterministic package-level contract/regression tests for the older-round
late
aggregate sequence and for the normal current-round aggregate path. Document
the guard in the CLI contract so future agents do not assume `review aggregate`
can be used as a historical repair or backfill tool. This is primarily a
package-level contract/regression slice, not fuzzing or general resilience
work; an
additional E2E test is optional only if command-wiring coverage proves useful
after implementation.

#### Expected Files

- `internal/review/service_test.go`
- `docs/specs/cli-contract.md`

#### Validation

- `go test ./internal/review` passes with the new regression coverage.
- `go test ./...` passes once the slice is ready for broader verification.

#### Execution Notes

Added a package-level regression test in `internal/review/service_test.go`
that reproduces the stale older-round aggregate sequence and asserts the
request fails without rewriting local state or writing `aggregate.json`.
Updated `docs/specs/cli-contract.md` so the v0.1 single-active-round model now
states that `review aggregate` only applies to the current active round.
Added lock-behavior tests covering both `review start` and `review aggregate`
when another review mutation is already in progress. Validated with
`go test ./internal/review` and `go test ./...`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only tightened package-level regression
coverage and CLI contract wording around the already-reviewed Step 1 behavior
change, so a separate closeout review would have duplicated the same narrow
risk scan.

## Validation Strategy

- Primary coverage should be a focused package-level contract/regression test in
  `internal/review/service_test.go`, because the bug is a deterministic review
  state transition error rather than a fuzzing or infrastructure-hardening
  problem.
- Run `go test ./internal/review` during implementation and `go test ./...`
  before archive so the aggregate guard is validated both locally and against
  broader repository expectations.
- If implementation changes expose a command-wiring gap that unit coverage
  cannot express clearly, add one narrow E2E regression only for that missing
  seam rather than broadening the slice by default.

## Risks

- Risk: Strict aggregate validation could block a legitimate recovery path for
  a broken or abandoned review round.
  - Mitigation: Keep `review start` unchanged in this slice, return actionable
    errors that point back to the current active round, and defer explicit
    supersede or abandon flow design to follow-up work.

## Validation Summary

- Reproduced the stale-round bug with a temporary harness workspace by starting
  a newer review round and then aggregating the older round, which confirmed
  the original state-regression path before implementation.
- Added package-level regression coverage in
  `internal/review/service_test.go` for stale older-round rejection plus
  mutation-lock contention on both `review start` and `review aggregate`.
- Validated the shipped candidate with `go test ./internal/review` and
  `go test ./...`.

## Review Summary

- Step-closeout review progressed through `review-001-delta` to
  `review-004-delta`. The first three rounds surfaced real correctness gaps:
  snapshot-only validation, non-atomic review-start interleaving, and crash-
  stale directory locks. Each finding was repaired and rerun until
  `review-004-delta` passed clean.
- Finalize review progressed through `review-005-full` and `review-006-full`.
  `review-005-full` caught a TOCTOU gap between lock acquisition and later plan
  detection, and `review-006-full` passed clean after the command flow began
  carrying the locked plan path through review start and aggregate.

## Archive Summary

- Archived At: 2026-03-23T23:45:05+08:00
- Revision: 1
candidate is ready for `harness archive`.
- PR: not created yet; publish evidence should record the PR URL after archive.
- Ready: `review-006-full` passed as the structural `pre_archive` gate, the
  acceptance criteria are satisfied, and the remaining work is the tracked
  archive move plus post-archive publish/CI/sync evidence.
- Merge Handoff: After archive, commit and push the archived plan move plus the
  tracked code and doc changes, open the PR, record publish/CI/sync evidence,
  and keep deferred follow-up scope visible in `#38` and `#39`.

## Outcome Summary

### Delivered

- Hardened `harness review aggregate` so it only accepts the current active
  review round and rejects stale older rounds before persisting aggregate or
  state changes.
- Added review-mutation serialization for `review start` and
  `review aggregate`, first to close the review-start race and then to make the
  lock crash-safe and plan-path-consistent under finalize review feedback.
- Added package-level regression coverage for stale older-round rejection and
  mutation-lock contention, and updated the CLI contract to document that
  historical aggregate repair is out of scope for `review aggregate`.

### Not Delivered

- This slice did not make `review start` reject overlapping rounds on its own;
  that remains deferred until the CLI has an explicit supersede or abandon
  workflow for broken review rounds.
- This slice did not add a dedicated historical aggregate backfill command or
  workflow; `review aggregate` remains intentionally scoped to the current
  active round.

### Follow-Up Issues

- #38 Decide whether historical review aggregate backfill needs a dedicated
  repair flow
- #39 Add explicit supersede or abandon flow for broken review rounds
