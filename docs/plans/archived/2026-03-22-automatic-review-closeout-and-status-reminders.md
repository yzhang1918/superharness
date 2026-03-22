---
template_version: 0.2.0
created_at: "2026-03-22T00:00:00+08:00"
source_type: issue
source_refs:
    - '#22'
    - '#28'
---

# Clarify automatic review closeout and status reminders

## Goal

Clarify the controller's review-discipline contract so a future agent can tell
when step-closeout review must happen, when finalize review must start
automatically, and when routine review progression should proceed without
asking the human to micromanage.

Add a deterministic `harness status` reminder layer for missed step-closeout
review. The reminder should surface only after a completed step is missing a
qualifying clean step-closeout review or an explicit
`NO_STEP_REVIEW_NEEDED: <reason>` marker, while keeping ordinary "this slice
may now be reviewable" guidance in `next_actions` instead of heuristic
warnings.

## Scope

### In Scope

- Tighten `AGENTS.md`, `harness-execute`, and the execute references so the
  controller owns routine review progression, runs `harness status` at explicit
  checkpoints, and does not stop to ask the human for permission before
  ordinary step-closeout or finalize review.
- Update the normative specs for step-closeout review and `harness status` so
  the repository explicitly distinguishes:
  - routine review guidance in `next_actions`
  - workflow-discipline exceptions in `warnings`
  - the explicit `NO_STEP_REVIEW_NEEDED: <reason>` suppression marker in
    step-local `Review Notes`
- Teach `harness status` to warn when an already completed earlier step lacks a
  qualifying clean `step_closeout` review, while keeping the current node
  stable even if the warning is first noticed during a later step or finalize
  closeout.
- Add focused tests for the new status warnings and suppression behavior.
- After reopen feedback, tighten the reminder logic so it respects the latest
  step-closeout round for a step title, avoids suggesting a second review while
  an active review round is already in progress, and keeps plan/execute skills
  proactive about requesting explicit subagent approval before review
  orchestration needs it.
- After later follow-up feedback, make archived-candidate reminder guidance
  reopen-aware, rebuild readiness summaries after reminder debt is attached,
  and consume `reopen --mode new-step` once the first new step lands so later
  finalize-time fixes do not keep proliferating extra steps.
- Remove the ad hoc review-discipline postmortem now that its durable guidance
  is tracked in specs, skills, tests, and issue history.

### Out of Scope

- Adding a hard execution gate that prevents agents from editing tracked plan
  markdown directly or marking a step done before review.
- Introducing a new command-owned step-closeout command or another new CLI
  surface just for review discipline.
- Heuristically warning during `execution/step-<n>/implement` that the current
  slice might be ready for review before a step is actually marked done.
- Reworking finalize node transitions so later discovery of a missing earlier
  step review rewinds the node back to `execution/step-<i>/review`.

## Acceptance Criteria

- [x] `AGENTS.md`, `harness-execute`, and the execute references make it clear
      that the controller must run `harness status` at routine execution
      checkpoints, automatically start step-closeout or finalize review when
      the workflow calls for it, and only pause for blockers, scope changes, or
      explicit merge approval.
- [x] The specs clearly define that a completed step is review-complete when it
      has either a clean `step_closeout` review (`delta` by default, `full`
      allowed when the slice needs a broader pass) or a
      `NO_STEP_REVIEW_NEEDED: <reason>` marker in `Review Notes`.
- [x] `harness status` keeps ordinary review prompts in `next_actions`, but
      emits `warnings` once an already completed earlier step is missing
      qualifying step-closeout review evidence; the warning remains informative
      during later-step and finalize nodes without forcing a node rollback.
- [x] Focused Go tests cover missing earlier-step review warnings, finalize-time
      warning behavior, suppression via `NO_STEP_REVIEW_NEEDED`, and a clean
      reviewed step that does not warn.
- [x] Historical step-closeout reminder satisfaction is based on the latest
      round for a step target, so a later non-clean closeout round can make the
      reminder reappear.
- [x] `harness status` does not suggest starting a fresh review when the
      current node already has an in-flight review round that must be
      aggregated first.
- [x] `AGENTS.md`, `harness-plan`, and `harness-execute` tell the controller to
      request explicit subagent authorization during plan approval when review
      subagents are likely, and to ask immediately as a fallback if execution
      reaches a point where reviewer subagents are required but authorization is
      still missing.
- [x] The temporary postmortem at
      `docs/postmortems/2026-03-22-review-discipline-postmortem.md` is removed
      once its durable lessons are preserved elsewhere.
- [x] When archived `publish` or `await_merge` status discovers missing
      step-closeout debt, `next_actions` tells the controller to reopen the
      candidate before repairing it instead of suggesting an invalid direct
      review start.
- [x] `harness status` rebuilds finalize/archive/archived summaries after
      missing-closeout reminders are attached so it does not claim archive- or
      merge-readiness while warning that earlier closeout is still incomplete.
- [x] `reopen --mode new-step` is treated as consumed once the first reopened
      step has been added, so later finalize-time findings can be repaired
      in-place instead of forcing another new unfinished step by default.

## Deferred Items

- Revisit whether missed step-closeout review should eventually become a harder
  archive or execution gate instead of a reminder-only contract.
- Consider adding a dedicated retrospective-review workflow or command if
  reminder-only status guidance proves too soft in practice.

## Work Breakdown

### Step 1: Tighten controller review-discipline guidance

- Done: [x]

#### Objective

Update the durable docs and skills so a cold reader can see when the controller
must run status, start step-closeout review, start finalize review, and avoid
routine stop-and-ask pauses.

#### Details

Fold the discovery decisions into the tracked docs instead of leaving them in
chat. The guidance should explicitly say the controller should run
`harness status` at start/resume, before marking a step done, after each review
aggregate, and before relying on finalize progression. It should also
distinguish routine review progression from real escalation conditions.

#### Expected Files

- `AGENTS.md`
- `.agents/skills/harness-execute/SKILL.md`
- `.agents/skills/harness-execute/references/step-inner-loop.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`

#### Validation

- A cold reader can tell when step-closeout review must happen versus when
  finalize review must start automatically.
- The docs clearly call for routine `harness status` checkpoints instead of
  assuming the controller will remember them from chat.

#### Execution Notes

Updated `AGENTS.md`, `harness-execute`, `step-inner-loop.md`, and
`review-orchestration.md` so the controller-owned review flow is explicit:
routine `harness status` checkpoints are named directly, ordinary
step-closeout/finalize review no longer asks the human for permission, and
`NO_STEP_REVIEW_NEEDED: <reason>` is documented as the step-local exception.
Validated the wording by rereading the affected files together to make sure the
checkpoint list, review-start rules, and non-goals stay aligned.

#### Review Notes

`review-001-delta` passed clean with `docs_consistency` and `agent_ux` slots.
The reviewers agreed the controller checkpoints, routine review ownership, and
human-escalation boundaries are explicit enough for a future controller to
follow without relying on discovery chat.

### Step 2: Define the status reminder contract

- Done: [x]

#### Objective

Write the normative spec updates for missing step-closeout review reminders,
including the explicit suppression marker and the split between `next_actions`
and `warnings`.

#### Details

Capture the accepted direction precisely:
- status should not guess whether the current in-progress slice is reviewable
- status should warn only after a completed earlier step is missing review
  discipline
- later-step or finalize warnings should keep the current node stable
- a clean `step_closeout` review may be `delta` or `full`
- `Review Notes` may suppress the warning with
  `NO_STEP_REVIEW_NEEDED: <reason>`

#### Expected Files

- `docs/specs/state-model.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- The specs define when a completed step counts as review-complete and where
  the suppression marker belongs.
- The status contract distinguishes ordinary guidance from true reminder
  warnings without relying on hidden discovery context.

#### Execution Notes

Updated the normative specs so the repository now defines review-complete step
closeout as either a clean `step_closeout` review or an explicit
`NO_STEP_REVIEW_NEEDED: <reason>` marker. The status contract now reserves
`warnings` for recoverable ambiguity and missed-closeout reminders, keeps
ordinary review prompts in `next_actions`, and clarifies that later-step or
finalize warnings should not force a node rollback. The review-start contract
also now says `step_closeout` targets should use the tracked step title for
deterministic status matching.

#### Review Notes

`review-002-delta` passed clean with `correctness` and `docs_consistency`
slots. The reviewers agreed the specs now use compatible terminology for
review-complete step closeout, `NO_STEP_REVIEW_NEEDED`, stable-node late
warnings, and the `next_actions` versus `warnings` split.

### Step 3: Implement and test reminder warnings

- Done: [x]

#### Objective

Teach `harness status` to emit deterministic warnings and next actions for
missing earlier step-closeout review, then cover the behavior with focused Go
tests.

#### Details

The implementation should inspect completed steps that precede the current
workflow position, determine whether each one has a qualifying clean
`step_closeout` review or an explicit `NO_STEP_REVIEW_NEEDED` marker, and then
surface the earliest unresolved miss in `next_actions` plus compact summary
warnings. When the warning is first discovered during finalize, status should
remain in finalize and use warnings rather than forcing the node back to an
earlier step.

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`

#### Validation

- `go test ./internal/status -count=1`
- The new tests prove:
  - no warning for a clean completed step
  - warning while working on a later step after an earlier done step missed
    step-closeout review
  - warning while already in finalize closeout
  - no warning when `Review Notes` contains
    `NO_STEP_REVIEW_NEEDED: <reason>`

#### Execution Notes

Implemented reminder-only step-closeout warning logic in
`internal/status/service.go`. Status now scans historical `step_closeout`
review artifacts, accepts either clean review evidence or
`NO_STEP_REVIEW_NEEDED: <reason>`, warns only for completed earlier steps in
later-step or finalize review/archive nodes, and prepends repair-first
`next_actions` without changing the resolved node. Added focused coverage in
`internal/status/service_test.go` for clean `full` step closeout, later-step
warnings, finalize warnings, and marker-based suppression, then validated the
slice with `go test ./internal/status -count=1`.

Finalize review then exposed two real follow-up gaps in the same slice: the
first finalize test did not actually pin historical closeout lookup, and the
reminder logic dropped away after archive into `execution/finalize/publish` and
`execution/finalize/await_merge`. Tightened the finalize assertion to prove the
clean Step 2 artifact suppresses warnings, extended reminder coverage across
all `execution/finalize/*` nodes, added archived publish/await-merge coverage,
and updated the archived CLI fixture to use explicit
`NO_STEP_REVIEW_NEEDED: ...` closeout so unrelated reopen tests keep using
review-complete data.

#### Review Notes

`review-003-delta` passed clean with `correctness` and `tests` slots. The
reviewers confirmed the reminder-only status behavior stays deterministic:
clean historical `full` step-closeout review satisfies the contract, later-step
and finalize warnings stay informative without rewinding the node, and
`NO_STEP_REVIEW_NEEDED: <reason>` suppresses the warning as specified.
Subsequent full finalize rounds `review-004-full` and `review-005-full`
surfaced one real gap each; both findings were fixed and `review-006-full`
passed clean across `correctness`, `tests`, and `docs_consistency`.

### Step 4: Address reopened feedback and proactive subagent approval

- Done: [x]

#### Objective

Fix the unresolved PR feedback on the reminder implementation, then tighten the
planning/execution guidance so controller agents ask for explicit subagent
authorization early instead of stalling once reviewer subagents are required.

#### Details

The reopened scope has four concrete outcomes:
- historical `step_closeout` satisfaction must follow the latest round for a
  step target, not any older pass
- warning-driven repair actions must not suggest a new review while another
  review round is already active
- plan approval should proactively request explicit subagent authorization when
  later execution is likely to need reviewer subagents, with an execute-time
  fallback if that approval is still missing
- the one-off postmortem file should be deleted now that the durable contract
  lives in tracked docs, skills, tests, and GitHub issues

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `internal/status/service_internal_test.go`
- `AGENTS.md`
- `.agents/skills/harness-plan/SKILL.md`
- `.agents/skills/harness-execute/SKILL.md`
- `docs/postmortems/2026-03-22-review-discipline-postmortem.md` (delete)

#### Validation

- `harness plan lint docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`
- `go test ./internal/status -count=1`
- `go test ./...`
- PR review threads for the two unresolved `internal/status/service.go`
  findings are replied to and resolved after the fixes land

#### Execution Notes

Updated `internal/status/service.go` so historical `step_closeout` satisfaction
follows the latest round for a target instead of any older pass, and warning
repair guidance no longer suggests starting a second review while a `.../review`
node already has an in-flight round to aggregate. Added focused regression
coverage in `internal/status/service_test.go` for latest-round-wins behavior
and both step-review/finalize-review "do not start another round" cases.

The first reopened step-closeout review surfaced two more correctness edges in
that same area, and both are now fixed in the same slice: a newer in-flight
`step_closeout` round now supersedes an older pass instead of leaving stale
closeout satisfaction behind, and the in-flight repair guidance now tells the
controller to aggregate the active round first instead of still implying a
second review should start immediately. Revalidated the tightened logic with
`go test ./internal/status -count=1`.

The second reopened review pass exposed two narrower fallback cases, which are
now also covered: duplicate-review suppression now applies whenever any active
review round is still in flight, even if status had to fall back to an
`/implement` node, and an unreadable newer `step_closeout` manifest now
conservatively invalidates older pass evidence instead of leaving it
authoritative. Revalidated those repairs with `go test ./internal/status -count=1`
and `go test ./...`.

The third reopened review pass narrowed the unreadable-manifest behavior one
step further: the reminder scan no longer uses a global unreadable-history
watermark that can unsatisfy unrelated steps. Instead, unreadable history is
only allowed to override a target when status can still conservatively bind the
round back to that same step, and the regression coverage now proves that one
step's broken review artifact does not resurrect reminders for another step's
clean closeout.

Extended the controller guidance in `AGENTS.md`, `harness-plan/SKILL.md`, and
`harness-execute/SKILL.md` so plan approval should proactively request explicit
reviewer-subagent authorization when later execution is likely to need it, with
an execute-time fallback if that approval is still missing. Removed
`docs/postmortems/2026-03-22-review-discipline-postmortem.md` now that the
durable lessons are captured in tracked docs, tests, and issues. Validated the
reopened slice with `harness plan lint ...` and `go test ./...`.

The fourth reopened review pass (`review-010-delta`) caught one last
unreadable-history hole and one remaining documentation gap. The reminder scan
now treats a newer unreadable historical review round with no mappable target
as a conservative watermark, so an older clean pass can no longer suppress the
step-closeout warning after unknown newer evidence appears. Added a regression
test for that unknown-target case and tightened `harness-execute/SKILL.md` so
the execute-time fallback at a reviewer-subagent boundary is stated as an
explicit controller rule, not just implied by earlier docs. Revalidated the
repair with `go test ./internal/status -count=1` and `go test ./...`.

The next reopened review pass (`review-011-delta`) asked for one more focused
test around the new unreadable-history rescue path. Added
`internal/status/service_internal_test.go` as a same-package regression that
directly exercises `loadSatisfiedStepCloseoutTargets` when the active
`reviewCtx` must recover an unreadable current round whose aggregate target
cannot be mapped back to a tracked step. Revalidated the slice with
`go test ./internal/status -count=1` and `go test ./...`.

#### Review Notes

`review-007-delta` through `review-011-delta` each surfaced a narrower
reopened defect or coverage gap in the reminder implementation: latest-round
closeout semantics, in-flight duplicate-review suppression, unreadable-history
conservatism, explicit execute-time subagent-approval fallback wording, and
same-package coverage for the active-`reviewCtx` unreadable-history rescue
path. Each finding was repaired in the same tracked slice, with focused and
full Go validation rerun after every fix. `review-012-delta` then passed clean
across correctness, tests, and docs-consistency, so Step 4 now has a clean
step-closeout review.

### Step 5: Refresh revision 2 archive-facing summaries

- Done: [x]

#### Objective

Update the tracked plan's closeout sections so revision 2 no longer carries
reopen placeholders or stale revision 1 archive metadata before the next
finalize review judges archive readiness.

#### Details

This cleanup is limited to the tracked plan:
- replace every `UPDATE_REQUIRED_AFTER_REOPEN` placeholder in the archive-facing
  sections with revision 2 content
- refresh `Validation Summary`, `Review Summary`, `Archive Summary`,
  `Outcome Summary`, and `Follow-Up Issues` so they describe the current reopen
  state instead of the pre-reopen archive
- keep the revision 2 narrative accurate even though the new archive timestamp
  will only exist after the candidate is re-archived

#### Expected Files

- `docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`

#### Validation

- `harness plan lint docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`

#### Execution Notes

Refreshed the plan's archive-facing sections so revision 2 no longer carries
`UPDATE_REQUIRED_AFTER_REOPEN` placeholders or stale revision 1 archive
metadata into finalize review. The refreshed summaries now describe the current
reopened candidate: revision 1 was archived earlier on the same branch, but the
active revision 2 candidate supersedes that archive state until the branch is
re-archived after a clean finalize review. Revalidated the tracked plan with
`harness plan lint ...` and refreshed the reopen-era validation baseline with
`go test ./internal/status -count=1`, `go test ./internal/cli -count=1`, and
`go test ./...`.

#### Review Notes

`review-014-delta` ran a docs-consistency closeout focused on the refreshed
archive-facing sections and passed clean with no findings, confirming that the
revision 2 summaries no longer carry stale placeholders or obsolete revision 1
metadata as current state.

### Step 6: Synchronize reopened acceptance criteria

- Done: [x]

#### Objective

Bring the top-level acceptance criteria back into sync with the completed
reopened work so finalize review no longer sees the tracked plan claiming the
revision 2 scope is still pending.

#### Details

This final reopen cleanup should:
- mark the four reopen-era acceptance criteria complete now that the Step 4 and
  Step 5 work is done and their review rounds passed
- keep the acceptance criteria aligned with the refreshed archive-facing
  summaries and the actual durable changes landed in code, specs, skills, and
  docs
- avoid changing substantive scope; this is a synchronization update only

#### Expected Files

- `docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`

#### Validation

- `harness plan lint docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`

#### Execution Notes

Marked the four reopen-era acceptance criteria complete so the top-level plan
contract now matches the landed revision 2 work, the refreshed archive-facing
summaries, and the clean step-closeout reviews for Steps 4 and 5. Revalidated
the tracked plan with `harness plan lint ...`.

#### Review Notes

`review-016-delta` ran a docs-consistency closeout on the reopened acceptance
criteria sync and passed clean with no findings, confirming that the top-level
checkboxes now match the completed revision 2 work and refreshed plan
summaries.

### Step 7: Consume reopened new-step mode and repair archived reminder guidance

- Done: [x]

#### Objective

Fix the remaining archived/reminder correctness issues from PR review and make
`reopen --mode new-step` stop forcing extra steps once the first reopened step
has already landed.

#### Details

This revision 3 slice has three concrete outcomes:
- archived `publish` and `await_merge` reminder guidance should tell the
  controller to reopen before attempting step-closeout repair
- readiness summaries should be recomputed after reminder debt is attached so
  archived/finalize states do not still claim archive- or merge-readiness while
  warnings say otherwise
- once the first new step has been added after `reopen --mode new-step`, later
  finalize-time findings should no longer force another new unfinished step by
  default

#### Expected Files

- `internal/status/service.go`
- `internal/status/service_test.go`
- `docs/specs/state-model.md`
- `docs/specs/plan-schema.md`

#### Validation

- `harness plan lint docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`
- `go test ./internal/status -count=1`
- `go test ./...`
- the two unresolved PR review threads about archived reminder repair guidance
  and stale summaries are replied to and resolved after the fixes land

#### Execution Notes

Updated `internal/status/service.go` so missing-closeout reminder guidance now
rebuilds finalize/archive/archived summaries instead of still claiming
archive-ready or merge-ready status, and archived `publish` / `await_merge`
actions now tell the controller to `harness reopen --mode finalize-fix` before
repairing earlier missing step-closeout debt. Tightened the `new-step`
reopen-mode handling so the requirement is only pending before the first
reopened step exists; once that step lands, later finalize-time findings no
longer force another new unfinished step by default.

Added regression coverage in `internal/status/service_test.go` for archived
reopen-aware next actions, reminder-aware summary rebuilding, consumed
`new-step` behavior after later findings, and the mixed case where
missing-closeout warnings are still visible but the initial `new-step` cue
must remain dominant until the first reopened step is added. Updated
`docs/specs/state-model.md` and `docs/specs/plan-schema.md` so the documented
reopen transition matches the new consumed-after-first-step state behavior.

The first Step 7 delta review (`review-018-delta`) asked for one more explicit
regression around the active unreadable-manifest rescue path. Added
`TestLoadSatisfiedStepCloseoutTargetsUsesActiveInFlightReviewContextForUnreadableCurrentRound`
to `internal/status/service_internal_test.go` so an unreadable current-round
manifest with no aggregate still overrides an older pass via the active
`reviewCtx` fallback. Revalidated with `go test ./internal/status -count=1`
and `go test ./...`.

Validation:
- `go test ./internal/status -count=1`
- `go test ./internal/lifecycle -run 'TestReopenNewStepRecordsModeAndStatusCue|TestArchiveMovesPlanAndUpdatesPointers|TestReopenMarkersMustBeClearedBeforeRearchive' -count=1`
- `go test ./...`
- `scripts/install-dev-harness --force`
- `harness status`

#### Review Notes

`review-018-delta` found one important tests gap: the same-package coverage did
not make the active in-flight unreadable-manifest rescue path explicit enough.
Added a focused regression in `internal/status/service_internal_test.go` for
that branch, reran the status package plus full Go test suite, and then
reran step closeout.

`review-019-delta` passed clean with `tests` and `docs_consistency`. Step 7
now has explicit regression coverage for the active in-flight unreadable
manifest rescue path, and the step-local notes match the follow-up repair.

## Validation Strategy

- Run `harness plan lint` before execution starts and after any material scope
  update to this tracked plan.
- During execution, keep doc-only changes readable with direct file review and
  validate status behavior with `go test ./internal/status -count=1`.
- Before archive, run at least the focused status package tests and any broader
  Go test coverage needed by the touched files.

## Risks

- Risk: The reminder logic could misclassify older review history and create
  noisy warnings for already clean steps.
  - Mitigation: Reuse the existing structural review metadata path, accept
    either clean `delta` or clean `full` `step_closeout` review, and cover the
    no-warning path in tests.
- Risk: The docs could still leave too much room for controller interpretation
  around when to run status or start review.
  - Mitigation: Add explicit controller checkpoints and spell out that routine
    review progression is controller-owned once the plan is approved.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-22-automatic-review-closeout-and-status-reminders.md`
- `go test ./internal/status -count=1`
- `go test ./internal/lifecycle -run 'TestReopenNewStepRecordsModeAndStatusCue|TestArchiveMovesPlanAndUpdatesPointers|TestReopenMarkersMustBeClearedBeforeRearchive' -count=1`
- `go test ./...`
- `scripts/install-dev-harness --force`
- `harness status`
- Revision 3 kept the focused status package, targeted lifecycle coverage, and
  the full Go suite green while fixing the archived reopen guidance, consumed
  `new-step` semantics, the active in-flight unreadable-manifest rescue path,
  the finalize archive-closeout reminder assertions, and the mixed-debt
  finalize/archive next-action overlay so reminder guidance no longer hides
  blocker-specific or ordinary finalize repair follow-up.
- The same revision 3 validation loop now also proves that archived
  `publish` / `await_merge` reminder overlays prepend reopen guidance without
  hiding the ordinary publish, CI, sync, or merge-follow-up actions.
- `review-023-full` passed clean after the final reminder-overlay follow-up,
  with the focused status suite and the full Go test suite both still green.

## Review Summary

- `review-001-delta` passed clean for the controller/skill wording changes.
- `review-002-delta` passed clean for the state-model, CLI-contract, and
  plan-schema updates.
- `review-003-delta` passed clean for the initial status reminder
  implementation.
- `review-004-full` found one real blocking gap: the finalize warning test did
  not actually prove historical step-closeout evidence stayed satisfied.
- `review-005-full` found one real blocking gap: reminders disappeared after
  archive into `execution/finalize/publish` and `execution/finalize/await_merge`.
- Both finalize findings were fixed, validated, and cleared by
  `review-006-full`, which passed clean with `correctness`, `tests`, and
  `docs_consistency`.
- Reopened revision 2 then ran `review-007-delta` through `review-012-delta`
  to repair the PR-feedback slice: latest-round closeout semantics, in-flight
  duplicate-review suppression, unreadable-history conservatism, proactive
  reviewer-subagent approval guidance, the execute-time fallback wording, and
  helper-level regression coverage for unreadable active-round rescue.
- `review-012-delta` passed clean across correctness, tests, and
  docs-consistency, closing Step 4.
- `review-013-full` then found one remaining finalize-closeout issue: the
  tracked plan's archive-facing summaries still reflected pre-reopen metadata.
  Step 5 refreshed those sections, and `review-014-delta` passed clean for that
  docs-only closeout slice.
- `review-015-full` then found one last tracked-plan mismatch: the reopened
  acceptance criteria still read as pending after the revision 2 work had
  landed.
- Step 6 synchronized those acceptance criteria, `review-016-delta` passed
  clean for that docs-only slice, and `review-017-full` then passed clean
  across correctness, tests, and docs-consistency for the full archived
  candidate.
- Revision 3 added Step 7 for the new-step-consumption and archived-reminder
  follow-up. `review-018-delta` found one real tests gap in the active
  in-flight unreadable-manifest rescue path, and `review-019-delta` passed
  clean after the focused regression was added.
- `review-020-full` then found two finalize-closeout gaps in the tracked
  candidate: the revision 3 acceptance criteria still read as pending, and the
  archive-closeout reminder test did not yet assert the replacement repair
  guidance or warning path. Both findings are now fixed and the candidate is
  ready for a fresh finalize review.
- `review-021-full` then found one mixed-debt correctness edge and one matching
  docs-consistency gap: the missing-closeout overlay could replace ordinary
  finalize/archive guidance instead of prepending to it. The overlay merge
  logic now preserves blocker-specific and repair guidance while still putting
  the earliest closeout debt first, and the candidate is ready for another
  fresh full review.
- `review-022-full` then found one more archived-overlay consistency gap: the
  `publish` / `await_merge` reminder path still replaced those nodes' ordinary
  follow-up actions instead of prefixing the reopen-first reminder onto them.
  That merge behavior is now fixed, and the candidate is ready for another
  fresh full review.
- `review-023-full` then passed clean across correctness, tests, and
  docs-consistency for the full revision 3 finalize candidate.

## Archive Summary

- Archived At: 2026-03-22T22:28:13+08:00
- Revision: 3
- PR: [#25](https://github.com/yzhang1918/superharness/pull/25) on
  `codex/automatic-review-closeout-status-reminders`.
- Ready: the revision 3 candidate now includes the PR-review fixes for
  archived reminder guidance, summary rebuilding after missing-closeout debt,
  consumed `reopen --mode new-step` semantics after the first reopened step,
  explicit active in-flight unreadable-manifest recovery coverage, and the
  archive-closeout reminder assertions needed to keep the new repair guidance
  visible, with a clean finalize review in `review-023-full`.
- Merge Handoff: commit and push the archived plan move plus the tracked
  code/doc changes, reply to and resolve the open PR review threads, refresh
  publish/CI/sync evidence on the existing PR branch, and then return to
  await-merge for human approval.

## Outcome Summary

### Delivered

- Tightened `AGENTS.md` and the `harness-execute` skill pack so the controller
  owns routine review progression, runs `harness status` at named checkpoints,
  and does not pause to ask the human before ordinary step-closeout or finalize
  review.
- Updated `state-model.md`, `cli-contract.md`, and `plan-schema.md` so
  review-complete step closeout is explicit: a clean `step_closeout` review
  (`delta` by default, `full` allowed when needed) or
  `NO_STEP_REVIEW_NEEDED: <reason>` in `Review Notes`.
- Implemented reminder-only `harness status` warnings for earlier completed
  steps missing review-complete closeout, while keeping ordinary review prompts
  in `next_actions` and keeping the resolved node stable.
- Extended those reminders through the full finalize workflow, including
  `execution/finalize/publish` and `execution/finalize/await_merge`, so review
  debt does not disappear right before merge readiness.
- Reopened revision 2 fixed the follow-up PR comments by basing historical
  closeout satisfaction on the latest round, suppressing second-review prompts
  whenever any review round is already in flight, and making unreadable-history
  handling conservative both for step-bound and unknown-target cases.
- Added proactive plan-approval and execute-time fallback guidance for explicit
  reviewer-subagent authorization, and deleted the temporary postmortem now
  that the durable rules live in tracked docs, skills, tests, and GitHub
  issues.
- Added and repaired focused Go coverage for later-step warnings, finalize
  warnings, archived publish/await-merge reminders, clean historical full
  reviews, `NO_STEP_REVIEW_NEEDED` suppression, and the active-`reviewCtx`
  unreadable-history rescue path, while keeping the broader Go suite green.
- Revision 3 now also keeps archived `publish` / `await_merge` guidance
  reopen-aware, rebuilds finalize/archive/archived summaries when earlier
  closeout debt is still present, and consumes `reopen --mode new-step` once
  the first reopened step lands so later finalize-time findings can repair in
  place instead of proliferating extra steps.

### Not Delivered

- A hard execution/archive gate that rejects unresolved step-closeout review
  debt instead of warning.
- A dedicated retrospective step-closeout workflow or command beyond the
  reminder-only contract landed here.

### Follow-Up Issues

- `#24` Decide whether missed step-closeout review should stay reminder-only or
  grow a stronger deterministic gate and/or retrospective closeout workflow.
- `#26` Surface reopen placeholders before archive-time blockers so finalize
  closeout does not have to discover them late.
- `#27` Surface unchecked top-level acceptance criteria before archive-time
  blockers so finalize closeout gets earlier guidance.
