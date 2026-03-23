---
template_version: 0.2.0
created_at: "2026-03-23T00:00:00+08:00"
source_type: issue
source_refs:
    - '#6'
    - '#33'
---

# Expand repo-level E2E lifecycle and merge handoff coverage

## Goal

Expand the repo-level binary-driven test suites so the remaining `#6` work is
no longer "add more integration coverage" in the abstract. After this slice,
the only uncovered follow-up under `#6` should be the dedicated
`tests/resilience/` package and any later fuzzing for parsing-heavy paths.

This slice should also close `#33` by adding smoke coverage for the
`scripts/install-dev-harness` branch where the chosen install directory is
already on `PATH`. The new coverage should prove the installer verifies the
PATH-resolved wrapper with `command -v harness` and exercises the follow-up
`harness --help` probe through that resolved wrapper, not only by calling the
wrapper file directly.

Because the workflow graph is finite at the transition-family level, this
slice should also leave behind a durable E2E transition-coverage report that
maps scenario tests to the normative state-transition matrix. That report
should answer "what do the E2E tests cover today?" without creating a second
workflow spec that can drift from the existing contracts. As part of that
work, execution should actively proofread `docs/specs/state-transitions.md`
against the real CLI/state behavior; if the new E2E scenarios reveal missing,
invalid, or underspecified transitions, this slice should correct the tracked
specs instead of leaving the drift implicit in tests only.

## Scope

### In Scope

- Add smoke coverage for the installer's "target install dir already on PATH"
  branch, including the `command -v harness` verification path and the
  follow-up `harness --help` probe through a PATH-resolved wrapper.
- Add a test-owned E2E transition-coverage matrix/report that maps scenario
  tests to the normative transitions in `docs/specs/state-transitions.md` and
  calls out any intentionally deferred gaps.
- Treat the transition-coverage work as a proofread pass over
  `docs/specs/state-transitions.md` and adjacent state docs; if execution
  exposes missing, invalid, or ambiguous transition rules, update the tracked
  specs as part of this slice.
- Extend `tests/support/` only where it materially improves readability or
  reduces duplication for the new repo-level lifecycle and handoff scenarios.
- Add multiple real-binary `tests/e2e/` scenarios that cover:
  - bounded loop coverage for repeated step-review and finalize-review repair
    cycles
  - `reopen --mode new-step` semantics on a canonical three-step plan,
    including the pre-new-step pending state and the first reopened step once
    it is added
  - archive and reopen lifecycle roundtrips
  - archived-candidate handoff through publish, CI, and sync evidence until
    `harness status` resolves `execution/finalize/await_merge`
  - merge confirmation and post-merge cleanup through
    `harness land --pr ...` and `harness land complete`
- Reuse the existing review-workflow E2E as the pre-archive path instead of
  replacing it with a monolithic one-test-does-everything suite.
- Make behavior-preserving testability adjustments only when the new smoke or
  E2E suites expose gaps that cannot be covered cleanly otherwise.

### Out of Scope

- Adding `tests/resilience/` or deterministic failure-injection scenarios in
  this slice.
- Adding fuzz tests or property-based parsing coverage in this slice.
- Attempting full unbounded route enumeration across infinite loop-capable
  workflows; this slice should use explicit loop budgets on a canonical plan
  shape instead.
- Reworking package-local `internal/*` tests to share repo-level helpers when
  that deduplication is not required for the new binary-driven suites.
- Redesigning CLI contracts or tracked plan semantics beyond narrowly scoped
  fixes required by regressions that the new suites expose.

## Acceptance Criteria

- [x] `tests/smoke/install_dev_harness_test.go` includes a passing case for the
      on-PATH installer branch that proves the installed wrapper is the command
      resolved by `command -v harness` and that the installer's verification
      reaches `harness --help` through that PATH-resolved wrapper.
- [x] `go test ./tests/e2e -count=1` passes with dedicated real-binary
      scenarios for archive/reopen, archived-candidate evidence handoff, and
      land/cleanup in addition to the existing review-workflow coverage.
- [x] The new E2E assertions pin the durable artifacts and state transitions
      that matter for lifecycle correctness, including tracked-plan path moves,
      current-plan pointer updates, revision bumps on reopen, evidence artifact
      persistence, merge-ready status transitions, and idle-after-land
      restoration.
- [x] The E2E suite includes bounded coverage for the loop-prone workflow
      families that are easiest to get wrong: at least one step-review rerun,
      at least one finalize-review repair/rerun, partial publish handoff that
      remains in `execution/finalize/publish`, `land` status that remains
      stable before `harness land complete`, and a `reopen --mode new-step`
      scenario on a canonical three-step plan.
- [x] The E2E suite treats rollback and repair transitions as first-class
      coverage targets rather than incidental side effects of happy-path
      scenarios, including explicit assertions on the source node that is being
      invalidated or repaired before the workflow advances again.
- [x] The `reopen --mode new-step` scenario proves the exact status semantics
      that have been easy to misread: the reopened candidate first stays in a
      pending finalize/new-step state until a new unfinished step is added, and
      once that extra step exists the work resumes at that new step's
      `implement` node with the same review/advance behavior as an ordinary
      current step.
- [x] Reopen coverage makes the source-node semantics explicit: at minimum one
      `reopen --mode finalize-fix` case is asserted from archived publish
      handoff and one `reopen --mode new-step` case is asserted from either
      archived publish or merge-ready `await_merge`; if one origin is judged
      semantically redundant after proofread, the coverage report must say so
      explicitly instead of leaving the omission implicit.
- [x] A tracked E2E transition-coverage report exists, is anchored to the
      existing state-transition spec rather than redefining it, and makes it
      easy to answer how many scenario families and transition families the
      repo-level E2E suite currently covers.
- [x] Any missing, invalid, or ambiguous transition rules discovered while
      building the new E2E suite are corrected in the tracked state docs, and
      the coverage report calls out the resulting source-of-truth alignment.
- [x] The resulting tracked plan and closeout notes make the remaining `#6`
      follow-up explicit as resilience coverage plus fuzzing only, with no
      additional broad "more E2E later" bucket left ambiguous.

## Deferred Items

- Add `tests/resilience/` coverage for deterministic failure cases such as
  corrupted current-plan markers, missing review/evidence artifacts, and
  archive rollback failures.
- Evaluate and, if warranted, add fuzz coverage for parsing-heavy paths such
  as plan linting, review artifacts, and evidence payload decoding.

## Work Breakdown

### Step 1: Define bounded transition coverage reporting and proofread state docs

- Done: [x]

#### Objective

Define how repo-level E2E coverage is reported against the state machine,
proofread the tracked transition docs against the real CLI behavior, then add
or refine the shared helpers needed to keep the lifecycle and merge-handoff
scenarios readable without scattering plan-rewrite, payload-writing, or
status/assertion boilerplate across multiple test files.

#### Details

Do not create a second normative workflow spec. The authoritative workflow
contract remains `docs/specs/state-transitions.md`; this step should instead
add a report or machine-readable matrix that maps tests to those existing
transitions and highlights what is still intentionally deferred. Use a
canonical three-step plan shape plus explicit loop budgets so the report can
describe bounded route coverage without pretending infinite workflows are
exhaustively enumerable. At minimum, document the intended budgets for: one
step-review repair loop, one finalize repair loop, progressive publish/CI/sync
handoff while status stays in `execution/finalize/publish`, one `land` cleanup
stability check before completion, and one `reopen --mode new-step`
consumption that grows the plan from three tracked steps to four. If that
work exposes drift between the tracked specs and actual status/lifecycle
behavior, update the tracked docs in this step before expanding the tests.
The coverage report should call out rollback families separately from
straight-line progress families so it is obvious whether review/reopen repair
paths are actually covered or merely implied by larger scenarios.
Keep `tests/support/` narrowly focused on binary-driven repository suites.
Prefer helpers that encode repeated fixture-shaping or JSON/assertion patterns
already needed by more than one E2E scenario. Avoid pushing command semantics
behind opaque abstractions; the test bodies should still read like explicit
workflow transcripts.

#### Expected Files

- `docs/testing/e2e-transition-coverage.md`
- `docs/specs/state-transitions.md`
- optional adjacent state docs such as `docs/specs/state-model.md` or
  `docs/specs/cli-contract.md` if the proofread reveals drift
- `tests/e2e/coverage_test.go`
- `tests/support/plan.go`
- `tests/support/repo.go`
- `tests/support/assert.go`
- optional new `tests/e2e/*_test.go` helper file if the package needs
  scenario-local helpers instead of widening `tests/support/`

#### Validation

- The tracked report explains E2E coverage in terms of scenario tests,
  canonical nodes, transition families, and the chosen bounded-loop model,
  while citing
  `docs/specs/state-transitions.md` as the source of truth.
- Any spec drift exposed by the bounded coverage model is corrected in tracked
  docs before later implementation steps build on the wrong assumptions.
- The repo-level helpers make the new lifecycle and handoff tests shorter
  without hiding which real `harness` commands are under test.
- Any helper changes are validated by the new `tests/e2e` scenarios that
  consume them plus `go test ./tests/e2e -count=1`.

#### Execution Notes

Added the tracked coverage report at `docs/testing/e2e-transition-coverage.md`
and expanded `tests/e2e/coverage_test.go` into a machine-checked catalog that
now fails if the maintained canonical transition catalog loses bounded
scenario coverage or drifts from `docs/specs/state-transitions.md`.
Integrated the incoming proofread updates in `docs/specs/state-model.md` and
`docs/specs/state-transitions.md`, then added shared E2E helpers plus
`AppendStepBeforeValidationStrategy` so reopen/new-step scenarios can express
the real command transcript without duplicating boilerplate. Validated with
`go test ./tests/e2e -count=1` and later again with `go test ./... -count=1`.
After finalize review `review-001-full` flagged that the catalog/spec sync
claim was too weakly enforced, repaired `coverage_test.go` to parse the
tracked transition matrix and state-preserving list directly, then reran
`go test ./tests/e2e -count=1` and `go test ./... -count=1`.
After finalize review `review-002-full` flagged missing explicit review-node
self-loop assertions and over-strong wording about catalog-driven coverage
regressions, added submission-time `status` assertions in the shared review
helpers and clarified the report/plan wording before rerunning
`go test ./tests/e2e -count=1` and `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Steps 1-5 were intentionally implemented as one
integrated repo-level E2E expansion, so per-step delta review would have been
misleading; a branch-level finalize review will cover the combined spec,
report, smoke, and E2E changes.

### Step 2: Cover review loops and reopen-new-step semantics

- Done: [x]

#### Objective

Add bounded real-binary E2E coverage for the loop-prone workflow families that
most need clarification: step-review reruns, finalize repair/reruns, and
`reopen --mode new-step` on a canonical three-step plan.

#### Details

This step should prove at least one step-local review failure followed by a
clean rerun, at least one finalize-review failure followed by repair and a
later passing finalize review, and the tricky `new-step` reopen path the user
called out explicitly. For `new-step`, use a three-step plan that fully
completes, reopens after archive or merge-ready handoff, confirms the
intermediate status before a new step is added, then adds step 4 and proves
the reopened work resumes at `execution/step-4/implement`. The test should
then show that the new current step behaves like an ordinary planned step:
without review it remains at `implement`, and after a clean step review it can
advance through the same derived transitions as any other current step.
Part of this step is to answer the question "after `reopen --mode new-step`,
does later status behave exactly like an ordinary planned `step-k/implement`
case?" with explicit assertions rather than inference.

#### Expected Files

- `tests/e2e/review_repair_loop_test.go`
- `tests/e2e/reopen_new_step_test.go`
- `tests/support/plan.go`
- `tests/support/repo.go`
- `tests/support/assert.go`

#### Validation

- `go test ./tests/e2e -count=1` passes with assertions on repeated
  `implement <-> review` behavior, finalize repair/rerun behavior, the
  intermediate `new-step` pending state, and the later `step-4/implement`
  behavior after the new unfinished step is added.

#### Execution Notes

Added `tests/e2e/review_repair_loop_test.go` for step-review and finalize
repair reruns, plus `tests/e2e/reopen_new_step_test.go` for the canonical
three-step `reopen --mode new-step` flow through pending finalize repair,
step-4 creation, and resumed ordinary step behavior. Extended shared helpers
to keep the scenario bodies readable while still driving the real binary.
Validated with `go test ./tests/e2e -count=1` and `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The loop and reopen coverage landed in the same batch
as the surrounding transition-matrix and handoff work, so a separate step-only
delta review would not have matched the actual review surface; finalize review
will cover the integrated branch.

### Step 3: Cover installer PATH verification and finalize-fix archive/reopen

- Done: [x]

#### Objective

Close `#33` and add a focused lifecycle E2E for the archive/reopen roundtrip
that returns an archived candidate to active execution through the
`finalize-fix` path.

#### Details

The installer smoke should model the common case where the target wrapper
directory is already on `PATH`, then assert the script's success output and
PATH-resolved verification behavior. The lifecycle E2E should exercise a real
archive candidate through `harness archive`, confirm archived status/handoff
guidance, reopen with `finalize-fix`, and assert that the plan returns to
active execution with the tracked-plan file, revision, and next actions
updated correctly. This step should make the `execution/finalize/publish ->
execution/finalize/fix` rollback explicit so reopen-from-archived-handoff is
not left implicit in broader end-to-end coverage.

#### Expected Files

- `tests/smoke/install_dev_harness_test.go`
- `tests/e2e/archive_reopen_test.go`
- `tests/support/plan.go`
- `tests/support/assert.go`

#### Validation

- `go test ./tests/smoke -count=1` passes with the new installer branch
  coverage.
- `go test ./tests/e2e -count=1` passes with archive/reopen assertions on the
  archived plan path, reopened active plan path, current-plan pointer, and
  reopened revision/state facts for the `finalize-fix` mode.

#### Execution Notes

Closed `#33` by adding the on-PATH installer smoke in
`tests/smoke/install_dev_harness_test.go`, and added
`tests/e2e/archive_reopen_test.go` to assert archive path moves,
`reopen --mode finalize-fix`, revision bumps, and restored active-plan
pointers. Reused the shared binary-driven helpers rather than widening lower
level packages. Validated with `go test ./tests/smoke -count=1`,
`go test ./tests/e2e -count=1`, and `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The installer smoke and archive/reopen assertions are
part of the same integrated branch slice, so finalize review is a better fit
than a late isolated delta review for this step.

### Step 4: Cover archived-candidate evidence handoff to await-merge

- Done: [x]

#### Objective

Add real-binary E2E coverage for the post-archive handoff path so repo-level
tests prove how publish, CI, and sync evidence move an archived candidate from
`execution/finalize/publish` to `execution/finalize/await_merge`.

#### Details

This scenario should start from a real archived candidate, submit realistic
JSON payloads through `harness evidence submit`, and assert both the append-only
artifacts and the canonical `harness status` node progression after each
domain. It should explicitly pin the publish self-loop by showing that partial
evidence still leaves the candidate in `execution/finalize/publish` until the
full merge-ready set is recorded. Keep the scenario focused on the happy-path
handoff rather than folding resilience or invalid-payload cases into this
slice.

#### Expected Files

- `tests/e2e/publish_handoff_test.go`
- `tests/support/repo.go`
- `tests/support/assert.go`
- optional helper updates under `tests/support/`

#### Validation

- `go test ./tests/e2e -count=1` passes with assertions that publish, CI, and
  sync evidence are written under `.local/harness/plans/<plan-stem>/evidence/`
  and that `harness status` advances from archived publish handoff to
  `execution/finalize/await_merge` only when the merge-ready evidence set is
  complete.

#### Execution Notes

Added `tests/e2e/publish_handoff_test.go` to prove publish and CI self-loops
under `execution/finalize/publish`, then the transition to
`execution/finalize/await_merge` only after sync evidence arrives. Also added
`tests/e2e/await_merge_reopen_test.go` so merge-ready rollback origins are
covered explicitly for both `finalize-fix` and `new-step`. Validated with
`go test ./tests/e2e -count=1` and `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Merge-handoff and merge-ready reopen coverage share the
same branch-level risk surface as the rest of this E2E expansion, so the
meaningful review point is finalize review on the whole batch.

### Step 5: Cover land entry and post-merge cleanup

- Done: [x]

#### Objective

Add real-binary E2E coverage for `harness land --pr ...` and
`harness land complete` so the repo-level suite covers the last happy-path
workflow boundary after merge approval.

#### Details

This scenario should begin from a merge-ready archived candidate, record land
entry with a realistic PR URL (and optional commit if it improves the
assertions), verify `land` status guidance, and prove the `land -> land`
stability the spec describes by checking that status remains in `land` while
cleanup is still outstanding. It should then run `harness land complete` and
assert the resulting idle worktree markers. The assertions should cover the
local state written for land entry/completion plus the current-plan marker
updates that allow `harness status` to return to `idle` while still preserving
last-landed context.

#### Expected Files

- `tests/e2e/land_workflow_test.go`
- `tests/support/assert.go`
- `tests/support/repo.go`
- optional helper updates under `tests/support/`

#### Validation

- `go test ./tests/e2e -count=1` passes with assertions on land-entry output,
  land cleanup state, `current-plan.json` updates, and the final idle status.
- `go test ./... -count=1` passes so the expanded repo-level suites do not
  regress the package-local contracts they build on.

#### Execution Notes

Added `tests/e2e/land_workflow_test.go` to cover
`await_merge -> land -> idle`, including the `land` self-loop before
`harness land complete` and the preserved last-landed metadata after cleanup.
Kept the scenario binary-driven and validated the whole repository with
`go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This last workflow boundary depends on the same shared
helpers and merge-handoff fixtures as Step 4, so the integrated finalize
review is the reliable review boundary for the branch.

## Validation Strategy

- Run `harness plan lint` on this tracked plan before execution starts and
  whenever scope wording changes.
- Keep the repo-level suites directly runnable with
  `go test ./tests/smoke -count=1` and `go test ./tests/e2e -count=1` while
  implementation is in progress.
- Treat route coverage as bounded rather than exhaustive: use the canonical
  three-step plan plus the documented loop budgets when deciding whether the
  transition matrix is adequately covered.
- Before archive, run `go test ./... -count=1` so the expanded smoke and E2E
  layers still agree with the lower-level package tests.

## Risks

- Risk: The new repo-level scenarios could become brittle or unreadable if they
  duplicate too much fixture-shaping and state inspection logic.
  - Mitigation: Extract only the repeated helper patterns that are already
    shared across scenarios, and keep each test focused on one workflow
    boundary rather than one giant end-to-end script.
- Risk: The tracked transition docs may not fully match the subtle
  `reopen --mode new-step` and loop semantics that status currently resolves,
  which could lead to tests codifying the wrong contract.
  - Mitigation: Treat mismatches as first-class findings in Step 1, correct
    the tracked specs before locking the coverage matrix, and make the new E2E
    scenarios assert the agreed behavior explicitly.
- Risk: E2E setup could accidentally bypass the CLI-owned contracts it claims
  to protect by seeding too much command-owned state directly.
  - Mitigation: Drive every transition under test through the real built
    `harness` binary, and use direct file shaping only for tracked-plan content
    or inert input fixtures that the CLI is expected to consume.

## Validation Summary

Validated the expanded bounded transition-coverage slice with repeated repo-level
and repository-wide runs:

- `go test ./tests/smoke -count=1`
- `go test ./tests/e2e -count=1`
- `go test ./... -count=1`

The final green runs include the added on-PATH installer smoke, the review
repair loops, archive/reopen coverage for both `finalize-fix` and `new-step`,
publish/await-merge handoff, land cleanup, and the spec-synced bounded
transition catalog checks.

## Review Summary

Finalize review required three rounds:

- `review-001-full`
  - raised one blocking finding that the catalog/spec sync claim was too weakly
    enforced
  - repaired by parsing the tracked transition matrix and state-preserving list
    directly in `tests/e2e/coverage_test.go`
- `review-002-full`
  - raised two blocking findings: missing explicit review-node self-loop
    assertions and over-strong wording about catalog-driven coverage
    regressions
  - repaired by adding submission-time `status` assertions in the shared review
    helpers and clarifying the report/plan wording
- `review-003-full`
  - passed clean with no blocking or non-blocking findings

## Archive Summary

- Archived At: 2026-03-23T21:34:27+08:00
- Revision: 1
- PR: NONE. The candidate has not been committed, pushed, or opened as a PR yet.
- Ready: The tracked plan is archive-ready locally after the clean finalize
  review and green validation runs.
- Merge Handoff: After archive, commit the archive move, push the branch, open
  or update the PR, and record publish, CI, and sync evidence before treating
  the candidate as merge-ready.

## Outcome Summary

### Delivered

- Closed `#33` with smoke coverage for the installer branch where the selected
  install directory is already on `PATH`.
- Expanded repo-level binary-driven E2E coverage to eight scenario families
  that cover all 10 canonical nodes and all 27 bounded transition families in
  the maintained transition catalog.
- Added explicit rollback and repair coverage for step review, finalize review,
  archived publish reopen, merge-ready reopen, `reopen --mode new-step`,
  publish self-loops, and land cleanup.
- Proofread and aligned the tracked state docs for `new-step` semantics so the
  pending finalize-fix state is explicit until the first reopened unfinished
  step exists.

### Not Delivered

- `tests/resilience/` coverage for deterministic failure-injection or corrupted
  state scenarios.
- Fuzz or property-style coverage for parsing-heavy paths.

### Follow-Up Issues

- `#6`
  - remaining follow-up scope is now limited to `tests/resilience/` plus later
    fuzz/property coverage
