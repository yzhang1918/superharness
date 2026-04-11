---
template_version: 0.2.0
created_at: "2026-04-11T23:42:00+08:00"
source_type: issue
source_refs:
    - '#57'
size: S
---

# Stabilize review and state lock coordination without widening mutation scope

## Goal

Close `#57` by turning the current review/state locking behavior into an
explicit, shared contract instead of leaving lock ordering and contention
handling split across ad hoc review command implementations.

This slice should preserve the current concurrency surface rather than redesign
it. `state.json` is now a thin control-plane artifact rather than a
`current_node` cache, so the work should align the code and specs with that
reality while making it harder for future review-path changes to introduce lock
ordering regressions or hidden scope creep.

## Scope

### In Scope

- Confirm and codify the current split responsibilities among review mutation
  locking, state mutation locking, and timeline mutation locking.
- Refactor the review command paths that mutate both review artifacts and
  `state.json` so they acquire locks through one shared, review-local
  orchestration entrypoint with a fixed ordering contract.
- Keep the ordering contract explicit and centralized for the dual-lock review
  paths so future commands do not re-decide the sequence independently.
- Preserve the current behavior boundary where `review start` and
  `review aggregate` coordinate both review artifacts and `state.json`, while
  `review submit` remains review-only.
- Update tracked specs to reflect the current role of `state.json` as a
  control-plane runtime artifact and to document the review/state lock
  relationship in the narrowest durable way that future agents can follow.
- Add focused regression coverage for lock acquisition ordering, contention
  behavior, and the boundary that keeps `review submit` out of state mutation
  locking.

### Out of Scope

- Replacing the separate review and state mutation locks with one broader
  plan-local lock primitive.
- Expanding `review submit` to acquire the state mutation lock.
- Broadening lock coverage for unrelated command families beyond the current
  review start and aggregate paths.
- Changing the runtime role of `state.json`, reintroducing a persisted
  `current_node`, or redesigning status resolution.
- Folding timeline event serialization into this issue or introducing new lock
  files.

## Acceptance Criteria

- [x] `#57` is satisfiable through a narrow implementation that keeps distinct
      review and state mutation semantics while removing duplicated dual-lock
      orchestration from review command code.
- [x] The dual-lock review paths use one shared entrypoint that fixes the
      acquisition order and keeps contention failures fail-fast and
      understandable without silently widening lock scope.
- [x] `review submit` continues to operate without a state mutation lock, and
      tests make that boundary explicit so later refactors do not accidentally
      serialize it behind `state.json` writes.
- [x] Tracked specs and nearby code comments describe `state.json` as the
      plan-local control-plane artifact it is today and explain the narrow
      review/state coordination contract without implying a future-wide lock
      merge.
- [x] Focused automated tests cover the shared review-path locking contract and
      prove that the refactor does not introduce new contention failures,
      deadlock-prone ordering drift, or behavior regressions for normal review
      flows.

## Deferred Items

- Any future decision to converge review and state serialization into a single
  broader lock primitive after a separate design discussion.
- Any later cleanup that generalizes lock orchestration across timeline or
  other non-review mutation surfaces.

## Work Breakdown

### Step 1: Pin the current mutation surfaces and the narrow lock contract

- Done: [x]

#### Objective

Turn the post-issue-51 implementation reality into an explicit contract for
this slice so execution can close `#57` without accidentally broadening what is
serialized.

#### Details

Start by grounding the plan in the current code, not the older assumption that
`state.json` caches workflow nodes. The intended outcome is a clear written
boundary: `review start` and `review aggregate` are the only review commands in
scope for shared review-plus-state coordination, `review submit` stays
review-only, and timeline locking remains a separate concern. Any spec wording
that still sounds like "state mutation" means "all workflow state" should be
tightened so future agents can tell that this issue is about lock orchestration
discipline, not a broad concurrency redesign.

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- optional nearby docs if one additional tracked note is needed to keep the
  contract honest

#### Validation

- The tracked plan and touched specs make it clear why this slice is standard
  rather than lightweight and why timeline and lock unification remain outside
  scope.
- A future agent reading only the plan plus the updated specs can explain which
  commands are supposed to take which locks and why `review submit` is
  intentionally excluded from state locking.

#### Execution Notes

Documented the narrowed runtime model in `docs/specs/state-model.md` and
`docs/specs/cli-contract.md`: `state.json` remains a control-plane artifact,
timeline locking stays separate, `review start` and `review aggregate` are the
dual-lock review paths, and `review submit` stays review-only.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 was tightly coupled to the Step 2 helper
extraction and Step 3 regression coverage, so a separate step-bound review
would have duplicated the later branch-level finalize review.

### Step 2: Centralize dual-lock review-path orchestration without changing lock coverage

- Done: [x]

#### Objective

Refactor the review service so the commands that mutate both review artifacts
and `state.json` share one explicit locking helper with fixed ordering and
stable contention behavior.

#### Details

Keep the change narrow and behavior-preserving. The goal is not to invent a new
global locking abstraction; it is to remove duplicated sequencing logic from
`review start` and `review aggregate` so later edits cannot drift in lock
ordering or error wording. If a small helper in `internal/review/` or
`internal/runstate/` improves clarity, use it, but do not leak this slice into
other command families or change the existing mutation surfaces. Normal review
flows should continue to produce the same durable artifacts and summaries after
the refactor.

#### Expected Files

- `internal/review/service.go`
- optional supporting review-local helper files if that makes the orchestration
  clearer
- optional `internal/runstate/state.go` only if a tiny shared helper is the
  cleanest narrow seam

#### Validation

- `review start` and `review aggregate` no longer hand-roll separate dual-lock
  orchestration.
- The central helper keeps the lock order explicit and does not widen locking
  for `review submit` or unrelated command families.
- Normal review-path behavior and artifact persistence remain unchanged aside
  from the intended implementation consolidation.

#### Execution Notes

Added a shared review-local helper in `internal/review/service.go` so
`review start` and `review aggregate` now acquire review and state locks
through one fixed-order entrypoint instead of hand-rolling the sequencing in
two places. This was a behavior-preserving refactor, so strict red/green TDD
was not practical; the safety proof comes from the new focused regression
coverage added alongside the refactor.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The helper extraction is only trustworthy when read
together with the spec updates and regression tests, so finalize review is the
right review surface for this step.

### Step 3: Lock the contract in with focused regression coverage

- Done: [x]

#### Objective

Prove that the narrower shared review-path orchestration closes `#57` without
introducing new issues in contention handling or mutation scope.

#### Details

Favor deterministic tests over broad concurrency infrastructure. Cover at least
the shared dual-lock entrypoint behavior for the review paths that touch
`state.json`, a contention scenario that proves fail-fast behavior still
surfaces cleanly, and the boundary that leaves `review submit` outside state
locking. If the refactor exposes the need for tiny test seams, keep them local
to the touched review or runstate code and back them with package-level tests
rather than widening into repository-level orchestration work.

#### Expected Files

- `internal/review/service_test.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- optional focused tests in `internal/runstate/state_test.go`
- optional nearby spec or comment touch-ups if the tests reveal an ambiguity

#### Validation

- Focused automated tests for the touched packages pass.
- The new assertions would catch accidental lock-order drift, accidental state
  locking of `review submit`, or confusing contention behavior in the shared
  review-path helper.
- Closeout can honestly say `#57` is resolved by codifying the current dual-lock
  model rather than deferring the real risk to another hidden follow-up.

#### Execution Notes

Extended `internal/review/service_test.go` with focused lock-contract tests:
`review start` and `review aggregate` now prove they prefer the review lock
when both locks are held, and the initial service-level `review submit` test
proved the intended lock boundary. Finalize review then caught a real CLI-layer
gap in `internal/cli/app.go`: the wrapper was still taking a locked pre-submit
status snapshot. The repair switched that snapshot to the unlocked status path
and added a CLI regression test in `internal/cli/app_test.go` so the runtime
boundary now matches the documented contract.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The added regression coverage exists to support the same
small integrated slice reviewed at finalize time, so a separate step-closeout
round would be redundant.

## Validation Strategy

- Run `harness plan lint` on this plan before approval.
- During execution, run targeted Go tests for the touched review and runstate
  packages, adding any narrow spec or comment checks needed to keep the
  lock-contract wording aligned with the code.
- Prefer deterministic lock-contention tests over timing-sensitive concurrent
  stress.

## Risks

- Risk: A refactor intended to centralize ordering could accidentally widen lock
  scope or change the user-visible contention behavior.
  - Mitigation: Keep the helper narrowly owned by the review paths that already
    take both locks and pin the current boundaries with focused tests.
- Risk: Spec cleanup could overstate the contract and imply broader lock
  unification than the code actually delivers.
  - Mitigation: Phrase the docs around the current review/state coordination
    behavior and explicitly list lock unification as deferred work.

## Validation Summary

Validated the narrowed lock contract with focused package tests:
`go test ./internal/review ./internal/cli -count=1`.

The review-focused suite now covers review-versus-state lock preference when
both locks are held, plus the boundary that keeps `review submit` off
state-mutation locking at both the service and CLI wrapper layers.

## Review Summary

Finalize review proceeded in three rounds:

- `review-001-full` found one important docs/runtime mismatch: `review submit`
  still acquired the state lock indirectly through a locked pre-submit status
  snapshot in the CLI wrapper.
- `review-002-delta` rechecked the narrow repair that switched that snapshot to
  the unlocked status path and passed with no findings.
- `review-003-full` reran full finalize review for revision `1` and passed with
  no findings, restoring archive readiness for the candidate.

## Archive Summary

- Archived At: 2026-04-12T00:00:34+08:00
- Revision: 1
The archived candidate closes `#57` by keeping the existing split between
review, state, and timeline mutation surfaces while making the review/state
interaction explicit and harder to regress.

Durable closeout now lives in tracked specs plus focused regression tests:

- `docs/specs/state-model.md` and `docs/specs/cli-contract.md` explain the
  current control-plane role of `state.json`, the separate timeline lock, and
  the fixed review-then-state acquisition order for dual-lock review paths.
- `internal/review/service.go` centralizes the dual-lock orchestration for
  `review start` and `review aggregate`.
- `internal/review/service_test.go` and `internal/cli/app_test.go` pin the
  intended lock boundaries, including the repaired CLI `review submit` path.
- PR: NONE. The candidate has not been pushed or opened as a PR yet.
- Ready: The branch is archive-ready locally after the repaired CLI boundary,
  the clean `review-003-full` finalize review, and focused
  `go test ./internal/review ./internal/cli -count=1` validation.
- Merge Handoff: Archive the plan, commit the archive move and closeout notes,
  push `codex/issue-57-lock-coordination`, open or update the PR, and record
  publish, CI, and sync evidence before treating the candidate as waiting for
  merge approval.

## Outcome Summary

### Delivered

- Closed `#57` with a narrow implementation instead of a broader lock-model
  redesign.
- Kept `review submit` outside the plan-local state lock path in actual runtime
  behavior, not just in service-level code or docs wording.
- Added focused regression coverage for lock-order preference, state-lock
  contention, and the repaired CLI wrapper behavior.
- Left the review/state/timeline split explicit and documented so future work
  can evolve from a truthful baseline.

### Not Delivered

- No broader convergence to a single plan-local mutation lock primitive.
- No general lock-orchestration cleanup outside the current review start,
  review submit, and review aggregate surfaces.

### Follow-Up Issues

- No new GitHub follow-up issue was created in this slice. Any future decision
  to unify review and state locking should begin as a separate design request
  rather than being treated as unresolved debt inside `#57`.
- Timeline-lock cleanup likewise remains intentionally out of scope unless a
  later request broadens the concurrency model beyond this issue.
