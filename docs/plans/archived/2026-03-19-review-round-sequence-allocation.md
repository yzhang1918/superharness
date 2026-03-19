---
status: archived
lifecycle: awaiting_merge_approval
revision: 1
template_version: 0.1.0
created_at: "2026-03-19T23:40:00+08:00"
updated_at: "2026-03-19T23:51:41+08:00"
source_type: issue
source_refs:
    - '#9'
---

# Allocate review round IDs from the max existing sequence

## Goal

Make review round ID allocation robust when plan-local review history is sparse
or mixed with legacy timestamp-based directories. Starting a new review round
should always pick the next compact numeric sequence without reusing an older
round ID that could overwrite existing artifacts.

This slice should keep the current v0.1 one-active-round model intact while
hardening only the sequence allocator and the tests that define its behavior.

## Scope

### In Scope

- Replace directory-count-based review round allocation with logic that scans
  existing compact round IDs and allocates from the maximum observed sequence.
- Ignore legacy timestamp-based review directory names when computing the next
  compact numeric sequence.
- Add or update tests that cover sparse compact history, legacy timestamp
  history, and mixed-directory cases.
- Clarify the repository-level review workflow in `AGENTS.md` so review-stage
  execution explicitly requires spawned reviewer subagents and strict adherence
  to the repo-local review skills.

### Out of Scope

- Changing the review manifest, ledger, or aggregate artifact schema.
- Modeling overlapping active review rounds or changing aggregate behavior.
- General refactors of review services unrelated to round ID allocation.

## Acceptance Criteria

- [x] Starting a review round after sparse compact history like
      `review-001-delta` and `review-003-full` allocates `review-004-<kind>`
      rather than reusing `review-003-<kind>` or filling gaps.
- [x] Legacy timestamp-based review directories do not affect compact sequence
      allocation.
- [x] Automated tests cover sparse and mixed review-history layouts.
- [x] `AGENTS.md` explicitly states that review orchestration must use spawned
      reviewer subagents and follow the repo-local skill rules rather than
      improvised inline review.

## Deferred Items

- Late aggregate protection for older rounds remains deferred to #7.
- Broader review-state or workflow changes remain out of scope for this slice.

## Work Breakdown

### Step 1: Harden compact review round sequence discovery

- Status: completed

#### Objective

Replace the current directory-count heuristic with parsing logic that finds the
highest existing compact review round sequence for the current plan.

#### Details

The allocator should recognize only compact directory names shaped like
`review-<NNN>-<kind>`, ignore other directories, and continue returning
`review-001-<kind>` when no compact history exists. Keep the change localized
to review-round allocation unless a small helper extraction improves clarity.

#### Expected Files

- `internal/review/service.go`

#### Validation

- Unit tests prove the allocator uses the max compact sequence instead of raw
  directory count.
- Existing review-service tests continue to pass after the allocator change.

#### Execution Notes

Replaced the directory-count allocator with compact round parsing in
`internal/review/service.go`. The new logic scans plan-local review
directories, recognizes only `review-<NNN>-<kind>` names, tracks the maximum
observed numeric sequence, and returns `max + 1`. Legacy timestamp-based
directory names are now ignored instead of affecting compact sequence
allocation.

#### Review Notes

Focused package validation and the full Go test suite both passed after the
allocator change. The archive-gating full review outcome is recorded below in
this plan's `Review Summary`.

### Step 2: Lock behavior with sparse-history tests

- Status: completed

#### Objective

Add focused tests that capture sparse compact history and mixed legacy history
so future refactors cannot regress round allocation.

#### Details

Cover at least one case with gaps in compact round numbering and one case where
legacy timestamp-based round directories coexist with compact IDs. Prefer
service-level tests unless a lower-level helper test is clearly smaller and
more direct.

#### Expected Files

- `internal/review/service_test.go`

#### Validation

- New tests fail against the old directory-count allocator and pass with the
  new behavior.
- `go test ./internal/review` passes.

#### Execution Notes

Updated the legacy-history test so it asserts that a lone timestamp-based
directory still yields `review-001-<kind>`. Added a sparse-history test that
mixes compact rounds with a numbering gap plus a legacy timestamp directory and
asserts that the next round uses the highest compact sequence plus one.

#### Review Notes

`go test ./internal/review` and `go test ./...` both passed with the new test
coverage in place.

### Step 3: Clarify mandatory reviewer-subagent orchestration

- Status: completed

#### Objective

Update `AGENTS.md` so the repository contract explicitly requires spawned
reviewer subagents during review orchestration and directs the controller agent
to follow the repo-local skill rules strictly.

#### Details

Capture the rule at the repo-contract layer instead of leaving it only in the
skill pack or in chat. The guidance should make clear that the controller
stays in `harness-execute`, reviewer work belongs to spawned agents using
`harness-reviewer`, and aggregation must wait for all reviewer slots to finish
per the skill references.

#### Expected Files

- `AGENTS.md`

#### Validation

- `AGENTS.md` clearly states the mandatory reviewer-subagent requirement and
  points execution detail back to the repo-local skills.
- `harness plan lint` passes after the plan update.

#### Execution Notes

Updated `AGENTS.md` with a dedicated `Review Execution` section that makes the
reviewer-subagent requirement explicit at the repo-contract layer. The new
guidance states that the controller remains in `harness-execute`, reviewer work
belongs to spawned `harness-reviewer` subagents, and aggregation must wait for
verified reviewer submissions instead of using an improvised controller-only
review flow.

#### Review Notes

`harness plan lint` passed after the scope update, and `review-001-full`
passed with no findings after spawned reviewer subagents covered correctness,
tests, and docs consistency.

## Validation Strategy

- Run focused review package tests while iterating on the allocator.
- Run the full Go test suite before closeout because review round IDs influence
  status and lifecycle flows outside `internal/review`.

## Risks

- Risk: The allocator could accidentally stop recognizing valid compact review
  directories and reset sequence allocation.
  - Mitigation: Keep the accepted directory pattern explicit and cover mixed
    history with targeted tests.
- Risk: A helper refactor could broaden the change surface and create
  unintended review behavior changes.
  - Mitigation: Keep the change minimal and validate against the existing full
    test suite.

## Validation Summary

Validated the slice with `go test ./internal/review` during implementation and
`go test ./...` before closeout. Additional dogfood validation covered a real
`harness review start -> reviewer subagent submit -> aggregate` cycle for
`review-001-full`, which passed cleanly with no findings.

## Review Summary

`review-001-full` ran as a full archive-gating review with three spawned
reviewer subagents covering `correctness`, `tests`, and `docs_consistency`.
Each reviewer submitted through `harness review submit`, the controller
verified and closed every reviewer subagent before aggregation, and the round
aggregated to `pass` with no findings.

## Archive Summary

- Archived At: 2026-03-19T23:51:41+08:00
- Revision: 1
- PR: not opened yet (branch `codex/review-round-sequence-max`)
- Ready: Review round ID allocation now advances from the maximum existing
  compact sequence, ignores legacy timestamp-based review directories, ships
  targeted sparse-history tests, and records the reviewer-subagent default at
  the repo-contract layer in `AGENTS.md`.
- Merge Handoff: Commit and push the archived plan move plus code changes, open
  a PR for branch `codex/review-round-sequence-max`, then let checks rerun
  before asking for merge approval.

## Outcome Summary

### Delivered

Delivered a review-round allocator that scans existing compact round IDs and
allocates from the maximum observed sequence instead of raw directory count,
which prevents sparse local history from reusing an existing round ID.
Legacy timestamp-based review directories no longer influence compact sequence
allocation, and focused tests now cover both legacy-only and sparse mixed
history layouts. This slice also tightened the repo contract so review
orchestration explicitly defaults to spawned reviewer subagents that follow the
repo-local review skills.

### Not Delivered

Late aggregate protection for older rounds remains deferred to #7, and this
slice did not change the broader review-state model beyond round-sequence
allocation and the repo-level review-orchestration wording.

### Follow-Up Issues

- #7 late aggregate protection for older rounds
