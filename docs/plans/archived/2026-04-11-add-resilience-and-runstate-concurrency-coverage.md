---
template_version: 0.2.0
created_at: "2026-04-11T21:26:48+08:00"
source_type: issue
source_refs:
    - '#37'
    - '#56'
size: L
---

# Add resilience and runstate concurrency coverage for archive, reopen, evidence, and status

## Goal

Close `#37` and `#56` with targeted test coverage that proves the repository
fails safely under malformed local state and remains coherent under realistic
overlapping runstate command flows.

This slice should not devolve into "more tests" in the abstract. It should add
the missing repo-level evidence that the harness handles deterministic failure
cases in `tests/resilience/` and that archive, reopen, evidence submission, and
status still honor the runstate contract under deterministic command
interleavings. If implementation reveals that the current production seams do
not support those tests cleanly, make the smallest behavior-preserving changes
needed to expose the intended contracts rather than widening the scope into a
state-model redesign.

## Scope

### In Scope

- Add a dedicated `tests/resilience/` package with deterministic repository-level
  coverage for the highest-value failure cases called out by `#37`.
- Cover corrupted or malformed `.local/harness/current-plan.json` and confirm
  the affected commands fail safely with conservative summaries and no
  accidental state mutation.
- Cover missing or malformed review and evidence artifacts that `status` or
  adjacent read paths must treat conservatively rather than silently accepting
  as clean state.
- Cover at least one archive or reopen rollback-family failure at the
  repository level when local state or path moves are incomplete or invalid.
- Add one or more deterministic integration-style tests for realistic command
  interleavings around archive, reopen, evidence submission, and status, aimed
  at the broader concurrency contract described by `#56`.
- Extend `tests/support/` only where shared repo-level helpers materially
  improve readability or determinism for the new resilience and concurrency
  scenarios.
- Update tracked coverage notes if needed so the repository makes it clear that
  resilience coverage is no longer an open broad gap after this slice.
- If execution uncovers leftover scope that prevents honest issue closure,
  record it explicitly as a follow-up issue instead of leaving the gap implicit.

### Out of Scope

- Fuzzing or property-style parser coverage from `#36`.
- Broad new happy-path lifecycle E2E scenarios beyond the deterministic
  resilience and concurrency cases needed for `#37` and `#56`.
- Redesigning state-lock semantics, cache policy, archive semantics, or the
  evidence model unless a narrow behavior-preserving seam is required to make
  the new tests deterministic.
- Nondeterministic stress, long-running race harnesses, or flaky timing-based
  concurrency infrastructure.
- Rewriting existing package-local rollback tests merely to mirror them at the
  repo level when they do not add new contract evidence.

## Acceptance Criteria

- [x] `tests/resilience/` exists and `go test ./tests/resilience -count=1`
      passes with deterministic repository-level cases that directly satisfy the
      failure-path intent of `#37`.
- [x] The resilience suite covers malformed or corrupted
      `.local/harness/current-plan.json`, missing or malformed review/evidence
      artifacts, and at least one archive or reopen rollback-family safety
      case where the command must fail conservatively without leaving the
      worktree in a misleading state.
- [x] The new resilience assertions prove safe failure behavior, not only that
      a command returns non-zero; the tests pin summaries, warnings, pointer or
      artifact preservation, and any required rollback outcomes that define the
      contract.
- [x] `go test ./tests/e2e -count=1` passes with at least one deterministic
      integration-style scenario that exercises realistic overlapping command
      patterns around archive, reopen, evidence submission, and status for the
      same plan, satisfying the broader contract requested by `#56`.
- [x] The concurrency-focused assertions prove CLI-level runstate coherence:
      overlapping commands either serialize or fail clearly, status stays
      conservative, and no stale or cross-revision evidence is mistaken for the
      current archived candidate during the tested interleavings.
- [x] Any minimal production or helper changes introduced for testability stay
      behavior-preserving and are covered by the new repo-level scenarios plus
      any focused package-level regression tests needed for the touched code.
- [x] The resulting tracked docs or closeout notes make it defensible to close
      `#37` and `#56` without a hidden "more resilience/concurrency coverage
      later" bucket; if anything material remains, it is moved into an explicit
      follow-up issue before archive.

## Deferred Items

- `#36`: evaluate and, if warranted, add fuzz or property-style coverage for
  parsing-heavy harness paths such as plan linting, review artifacts, and
  evidence payload decoding.
- Any later expansion into broader stress or race-style infrastructure beyond
  the deterministic command interleavings needed to close `#56`.

## Work Breakdown

### Step 1: Define the repo-level gap targets and fixture strategy

- Done: [x]

#### Objective

Turn `#37` and `#56` into a concrete repo-level test matrix, then add only the
shared helpers or coverage-note updates needed to keep the new scenarios
deterministic and reviewable.

#### Details

Start by mapping the issue text against what the repository already covers in
package-local tests so execution does not waste time re-proving the same
rollback branches at a different layer. The outcome of this step should be a
small explicit matrix of repo-level gaps: malformed current-plan pointer,
degraded artifact reads, one rollback-family scenario, and one or more
realistic command interleavings around archive/reopen/evidence/status. If the
existing `docs/testing/e2e-transition-coverage.md` wording still presents
resilience work as an unresolved broad follow-up after this slice lands, update
 that tracked note as part of this step or the final step so repository-visible
coverage expectations stay honest. Keep helper changes narrow and transparent;
the test bodies should still read like real command transcripts.

#### Expected Files

- `docs/testing/e2e-transition-coverage.md`
- `tests/support/repo.go`
- `tests/support/run.go`
- `tests/support/assert.go`
- optional new `tests/support/*` helper files if shared deterministic fixture
  setup is needed for the resilience suite

#### Validation

- The planned repo-level scenarios are explicitly narrower than the package-level
  rollback matrix and clearly tied to the issue closure goals.
- Any helper additions reduce duplication without hiding which CLI commands or
  files the new tests are asserting against.
- If tracked coverage docs are touched, they continue to describe the source of
  truth accurately and no longer leave resilience coverage as an ambiguous
  future bucket once the rest of this plan is complete.

#### Execution Notes

Mapped the repo-level closure targets against existing package-local coverage,
then kept the shared fixture expansion narrow: `tests/support/repo.go` now adds
`Workspace.WriteFile`, and `docs/testing/e2e-transition-coverage.md` now records
that the earlier resilience/concurrency follow-up is covered by the new suites
instead of remaining an ambiguous deferred bucket.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Fixture strategy, helper scope, and coverage-note
updates are tightly coupled to the implemented resilience and concurrency
scenarios, so a separate Step 1 delta review would be misleading duplication.

### Step 2: Add deterministic resilience coverage for malformed local state

- Done: [x]

#### Objective

Create `tests/resilience/` and prove the CLI fails safely when core local
workflow state or artifacts are corrupted, missing, or semantically invalid.

#### Details

Favor generated temporary repositories and small targeted malformed payloads
over large checked-in fixtures. At minimum, cover a corrupted
`current-plan.json` case that affects `status` or another current-plan consumer,
one missing or malformed review/evidence artifact case that must surface a
warning or degraded summary instead of a false clean state, and one
archive/reopen rollback-family case whose safety property is easier to trust
when exercised through the real binary. The assertions need to prove more than
"command failed": pin the conservative user-facing summary, any warning or
error paths that matter, and the persisted file or pointer state that must
remain intact after the failure.

#### Expected Files

- `tests/resilience/current_plan_test.go`
- `tests/resilience/artifact_failures_test.go`
- `tests/resilience/archive_reopen_rollback_test.go`
- `tests/support/repo.go`
- `tests/support/assert.go`
- optional focused package-level tests if a small seam or fallback path must be
  tightened in production code

#### Validation

- `go test ./tests/resilience -count=1` passes.
- The resilience suite proves conservative failure behavior for malformed local
  state and artifact degradation without relying on flaky timing or global test
  ordering.
- Any touched production fallback paths remain covered by focused package-level
  tests in addition to the repo-level binary assertions.

#### Execution Notes

Added `tests/resilience/` with deterministic real-binary coverage for malformed
`current-plan.json`, malformed historical review/evidence artifacts that must
keep `status` conservative, and archive/reopen rollback-family failures where
the current plan pointer and on-disk plan paths must recover cleanly.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The resilience suite is part of the same integrated
repo-level closure slice as Step 1 and Step 3, so branch-level finalize review
is the trustworthy review surface.

### Step 3: Add deterministic integration-style concurrency coverage for runstate-dense workflows

- Done: [x]

#### Objective

Prove the broader concurrency contract behind `#56` with realistic overlapping
command patterns around archive, reopen, evidence submission, and status.

#### Details

These tests should sit above the existing package-level lock-contention checks
by exercising real command sequences against the same plan and asserting
end-to-end coherence. Favor deterministic orchestration such as explicit lock
ownership, staged command ordering, or injected overlap points rather than
timing races. Good target patterns include: status attempting to refresh while
another command owns the state-mutation lock; evidence submission plus status
or reopen against an archived candidate; or archive/reopen transitions whose
revision-sensitive evidence lookup must remain coherent across the interleaving.
If a minimal behavior-preserving seam is needed to expose a deterministic hook,
keep it local to the touched service and back it with focused package tests.
This step should also finish any coverage-note updates needed so closeout can
honestly say that the remaining open follow-up is parser fuzz/property work,
not a vague concurrency bucket.

#### Expected Files

- `tests/e2e/runstate_concurrency_test.go`
- `tests/e2e/helpers_test.go`
- `tests/support/run.go`
- `internal/status/service_test.go`
- `internal/evidence/service_test.go`
- `internal/lifecycle/service_test.go`
- optional touched production files such as `internal/status/service.go`,
  `internal/evidence/service.go`, or `internal/lifecycle/service.go` if a
  deterministic test seam is required
- `docs/testing/e2e-transition-coverage.md`

#### Validation

- `go test ./tests/e2e -count=1` passes with the new deterministic concurrency
  scenario coverage.
- Targeted package tests for any seam or touched service continue to pass.
- The resulting repo-level assertions demonstrate CLI-level coherence under the
  tested interleavings and do not merely restate helper-level lock behavior.

#### Execution Notes

Added `tests/e2e/runstate_concurrency_test.go` to prove a real archive ->
await-merge -> reopen -> re-archive -> evidence handoff loop at revision 2.
The scenario explicitly proves stale revision-1 evidence stays ignored after
reopen and that `status` plus `evidence submit` fail clearly while the shared
state lock is held.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The deterministic concurrency scenario depends on the
same fixture strategy and documentation closeout as the resilience suite, so
branch-level finalize review is the trustworthy review surface.

## Validation Strategy

- Run `go test ./tests/resilience -count=1` throughout development because the
  new suite is the main closure evidence for `#37`.
- Run `go test ./tests/e2e -count=1` while building the deterministic
  concurrency scenarios so the broader contract behind `#56` is exercised
  through the real binary.
- If execution adds or tightens any deterministic hooks in production code, run
  the focused package suites for the touched services in addition to the repo-level
  tests.
- Run `go test ./...` before archive so the added resilience and concurrency
  coverage is validated against the full repository.

## Risks

- Risk: The repo-level scenarios could duplicate existing package-local
  rollback coverage without adding new issue-closing evidence.
  - Mitigation: Keep Step 1 explicit about the repo-level gap matrix and prefer
    degraded-read, pointer-integrity, and command-interleaving contracts that
    package-local tests do not already prove.
- Risk: Deterministic concurrency tests could drift into flaky timing-based
  orchestration.
  - Mitigation: Use explicit lock ownership or staged overlap points instead of
    sleeps or racey goroutine timing.
- Risk: A real issue-closing gap may appear only after implementation starts,
  leaving hidden residual scope.
  - Mitigation: Record any remaining material gap as an explicit follow-up issue
    before archive rather than quietly weakening the issue-closure claim.

## Validation Summary

- Added deterministic repo-level resilience coverage in
  `tests/resilience/` for malformed current-plan pointers, malformed
  review/evidence artifacts, and archive/reopen rollback-family failures.
- Added deterministic repo-level concurrency coverage in
  `tests/e2e/runstate_concurrency_test.go` for archive, reopen, evidence, and
  status interleavings across revisions.
- Validated with `go test ./tests/resilience -count=1`,
  `go test ./tests/e2e -count=1`, and `go test ./... -count=1`.

## Review Summary

- Finalize review `review-001-full` passed with no blocking or non-blocking
  findings across the `correctness`, `tests`, and `docs_consistency` slots.
- Reviewer submissions confirmed that the new repo-level tests, coverage note,
  and tracked plan state agree on the intended issue-closure scope for `#37`
  and `#56`.

## Archive Summary

- Archived At: 2026-04-11T21:51:24+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate now covers the deterministic resilience and runstate
  interleavings needed to close `#37` and `#56`, and the full repository test
  suite passed before archive.
- Merge Handoff: Commit and push the archived plan move, then record publish,
  CI, and sync evidence for the archived candidate before treating it as truly
  waiting for merge approval.

## Outcome Summary

### Delivered

- Added `tests/resilience/` coverage for malformed current-plan pointers,
  degraded artifact reads, and archive/reopen rollback-family safety cases.
- Added a deterministic `tests/e2e/runstate_concurrency_test.go` scenario that
  proves stale evidence stays revision-scoped after reopen and that lock
  contention fails clearly for `status` and `evidence submit`.
- Updated `docs/testing/e2e-transition-coverage.md` so resilience and broader
  runstate follow-up no longer appear as an ambiguous deferred gap.

### Not Delivered

NONE.

### Follow-Up Issues

- `#36` remains the explicit follow-up for fuzz or property-style coverage on
  parsing-heavy paths such as plan linting, review artifacts, and evidence
  payload decoding.
