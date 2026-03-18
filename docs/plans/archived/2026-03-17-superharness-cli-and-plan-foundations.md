---
status: archived
lifecycle: awaiting_merge_approval
revision: 3
template_version: 0.1.0
created_at: "2026-03-17T10:12:01+08:00"
updated_at: "2026-03-18T22:13:16+08:00"
source_type: direct_request
source_refs: []
---

# Superharness v0.1 Foundations

## Goal

Define the first durable planning and CLI contracts for `superharness`, then
use those contracts to guide the initial implementation of the agent-facing
core workflow.

## Scope

### In Scope

- Lock the v0.1 plan schema and authoring/archive rules.
- Lock the v0.1 CLI contract for plan, review, status, archive, and reopen
  workflows.
- Add a reusable plan template that matches the schema.
- Implement the initial CLI around those contracts in later steps.
- Address PR review feedback on archive/reopen state handling.
- Shorten review round identifiers while keeping them deterministic per plan.
- Make archive review gating revision-aware and persist review decisions in
  plan-local state.

### Out of Scope

- `harness ui` and any web UI implementation.
- Runtime-specific subagent spawning inside repository code.
- GitHub merge automation beyond the archive/merge handoff contract.

## Acceptance Criteria

- [x] The repository contains a durable v0.1 plan schema spec and matching plan
      template.
- [x] The repository contains a v0.1 CLI contract spec for agent-facing command
      behavior and output.
- [x] Every behavior-changing command in the v0.1 slice is covered by
      automated tests.
- [x] `harness plan template` and `harness plan lint` are implemented against
      the documented contract.
- [x] `harness status` reports plan state plus step state from
      local artifacts without requiring manual state bookkeeping.
- [x] `harness review start`, `harness review submit`, and
      `harness review aggregate` implement the review-round contract without
      binding to a specific subagent runtime.
- [x] `harness archive` and `harness reopen` implement the freeze/reopen
      lifecycle with agent-friendly `next_actions`.
- [x] `harness archive` refuses to archive while local review, CI, or sync
      state shows unresolved work.
- [x] `harness reopen` clears stale CI and sync state from the archived
      candidate before the next revision resumes.
- [x] Review round identifiers are compact, deterministic, and monotonic
      within a plan without relying on long timestamp suffixes.
- [x] Archive requires a passing `full` review for revision 1 and a passing
      review result for later reopened revisions.
- [x] Review aggregation persists the review decision in local state so
      archive can distinguish a failed aggregated review from a passing one.
- [x] Legacy aggregated review state without a stored `decision` field still
      resolves review outcome from the round aggregate artifact.

## Deferred Items

- `harness ui` is intentionally deferred until the CLI and local-state
  contracts are stable enough to serve as a clean backend. Follow-up: #2.
- `harness plan list` and the longer-term docs navigation shape are deferred
  until the core lifecycle flow is stable enough to expose history through the
  CLI. Follow-up: #4.
- The first reusable skill system, including the reviewer skill contract, is
  deferred until the CLI boundaries are stable enough to anchor the skills.
  Follow-up: #5.
- Shared test fixtures and broader integration coverage are deferred to a
  dedicated test-infrastructure pass after this first foundations PR lands.
  Follow-up: #6.

## Work Breakdown

### Step 1: Lock the planning and CLI contracts

- Status: completed

#### Objective

Write the first active plan plus the core plan-schema, CLI, and template specs
that the future CLI implementation will follow.

#### Details

Keep the tracked lifecycle coarse and keep step state derived from local
evidence instead of hand-maintained by agents.

#### Expected Files

- `docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md`
- `docs/specs/index.md`
- `docs/specs/plan-schema.md`
- `docs/specs/cli-contract.md`
- `assets/templates/plan-template.md`

#### Validation

- The plan, plan template, and specs agree on frontmatter fields, lifecycle
  states, required sections, and archive behavior.
- The CLI contract treats the CLI as agent-facing and requires concise
  JSON-first output with `next_actions` for stateful commands.

#### Execution Notes

Established the v0.1 plan schema, CLI contract, packaged template-asset
direction, and the current active plan as the dogfood target.

#### Review Notes

Incorporated feedback on naming, state vocabulary, archive/reopen semantics,
review-round scope ownership, and step-local note placement.

### Step 2: Implement `harness plan template` and `harness plan lint`

- Status: completed

#### Objective

Implement template rendering and structural validation for active and archived
plans.

#### Details

This step should lock the first real parser and validator for the tracked plan
format, including step-local note placeholders and archive-only summary rules.

#### Step Acceptance Criteria

- [x] `harness plan template` renders the packaged asset with seeded metadata.
- [x] `harness plan lint` distinguishes active-plan errors from archived-plan
      errors.
- [x] Completed archived steps cannot retain `PENDING_STEP_EXECUTION` or
      `PENDING_STEP_REVIEW`.

#### Expected Files

- `cmd/harness/main.go`
- `go.mod`
- `go.sum`
- `assets/templates/embed.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/plan/template.go`
- `internal/plan/template_test.go`
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`
- `*_test.go`
- `docs/specs/plan-schema.md`

#### Validation

- The CLI can render the packaged template asset with seeded metadata without
  introducing a second template source of truth.
- Lint rejects invalid frontmatter, missing sections, bad step status,
  malformed step subsections, and archive-time placeholder tokens with compact
  targeted errors.
- Automated tests cover valid and invalid active plans plus valid and invalid
  archived plans.

#### Execution Notes

Added the first Go module and CLI skeleton, embedded the packaged plan template
as a build asset, implemented `harness plan template`, and implemented
`harness plan lint` with frontmatter, section-order, step-structure, active,
and archived-plan validation rules.

#### Review Notes

`go test ./...` passes. Smoke checks confirmed `harness plan template` renders
seeded metadata, both `--help` surfaces exit cleanly, and `harness plan lint`
accepts the current active plan plus a generated plan under
`docs/plans/active/...`. Reviewer feedback also surfaced two panic paths plus
missing filename, step-heading, and historical-template-version validation;
follow-up negative tests and lint guards now cover those cases.

### Step 3: Implement local state and `harness status`

- Status: completed

#### Objective

Add the local `.local/harness/...` state layout and a status command that
computes useful next steps from plan state plus local artifacts.

#### Details

`harness status` should become the first command another agent runs after a
handoff or compacted session. The output needs to explain the current plan,
current step, current step state, and next likely actions.

#### Expected Files

- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `internal/plan/document.go`
- `internal/plan/document_test.go`
- `internal/runstate/state.go`
- `internal/status/service.go`
- `internal/status/service_test.go`
- `docs/specs/cli-contract.md`

#### Validation

- `harness status` can identify the current plan, infer the current step, and
  offer resume or handoff next actions after agent compaction or agent
  changes.
- The implementation derives step state from local evidence instead of
  requiring agents to manually keep local runtime state in sync.
- Automated tests cover minimal no-artifact status, review-in-progress status,
  waiting-for-CI status, resolving-conflicts status, ready-for-archive status,
  and archived status.

#### Execution Notes

Added a thin read-only `.local/harness/...` state model, a parsed plan-document
helper, and `harness status` with plan detection, current-step inference,
step-state inference, warnings, and agent-friendly next actions.

#### Review Notes

`go test ./...` passes. Real CLI smoke checks now cover `go run ./cmd/harness
--help`, `plan template --help`, `plan lint --help`, `status --help`, a
generated-plan template-plus-lint roundtrip, and `go run ./cmd/harness status`
against the current worktree. A reviewer subagent also confirmed adjacent lint
robustness gaps, which were fixed before closing this step. During archive
prep dogfooding, `harness status` also surfaced a `ready_for_archive`
handoff mismatch; the summary and next actions now correctly recommend
archiving instead of pointing back to already-written closeout summaries.

### Step 4: Implement the review-round contract

- Status: completed

#### Objective

Implement `harness review start`, `submit`, and `aggregate` so review rounds
have deterministic manifests, output paths, and aggregation.

#### Details

The CLI should validate and persist an agent-supplied review spec, not invent
review dimensions or a path list on the agent's behalf.

#### Expected Files

- `internal/cli/app.go`
- `internal/plan/current.go`
- `internal/review/service.go`
- `internal/review/service_test.go`
- `internal/runstate/state.go`
- `internal/status/service.go`
- `docs/specs/cli-contract.md`

#### Validation

- Review rounds emit stable manifest and ledger artifacts for both delta and
  full review kinds.
- Review submission and aggregation work without embedding runtime-specific
  subagent launch logic in the CLI.
- Automated tests cover review-round creation, valid and invalid reviewer
  submission, and delta versus full aggregation outcomes.

#### Execution Notes

Implemented a thin CLI-owned review service with deterministic round IDs,
manifest and ledger persistence, slot-normalized submissions, aggregate
artifacts, and local-state updates that keep `harness status` in sync without
owning subagent spawning.

#### Review Notes

`go test ./...` passes. Real CLI smoke checks now cover `review start --help`,
`review submit --help`, `review aggregate --help`, and a full dogfood
`start -> status -> submit -> aggregate -> status` cycle on the current repo's
active plan. During that smoke run, `harness status` moved to `reviewing`
after `start` and back to `implementing` after `aggregate`, which matches the
intended local-state contract.

### Step 5: Implement archive and reopen

- Status: completed

#### Objective

Implement the freeze-and-summarize archive flow plus mechanical reopen
behavior for archived plans.

#### Details

Archive should validate both top-level archive summaries and completed-step
closeout notes before moving the plan. Reopen should restore local editing
without forcing the next agent to rediscover context from scratch.

#### Expected Files

- `internal/cli/app.go`
- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/plan/document.go`
- `internal/plan/lint.go`
- `internal/plan/lint_test.go`
- `internal/runstate/state.go`
- `docs/specs/plan-schema.md`
- `docs/specs/cli-contract.md`

#### Validation

- Archive rejects plans whose durable summaries still contain placeholder
  tokens, whose completed steps still contain step placeholders, or whose
  active work is incomplete.
- Reopen restores an archived plan to active execution with `revision + 1` and
  a clear `next_actions` handoff.
- Automated tests cover archive move/update behavior plus reopen revision and
  placeholder reset behavior.

#### Execution Notes

Implemented lifecycle commands for `harness archive` and `harness reopen`,
including mechanical frontmatter updates, tracked-file moves, structured
archive-summary stamping, summary resets on reopen, and `.local` pointer
synchronization for both `current-plan.json` and plan-local `state.json`.

#### Review Notes

`go test ./...` passes. Real CLI smoke checks now cover `archive --help`,
`reopen --help`, and a built-binary roundtrip in a temporary worktree:
`plan template -> archive -> status -> reopen -> status -> plan lint`. That
roundtrip confirmed archived status handoff, `revision + 1` on reopen, current
plan pointer updates, and a clean active-plan lint result after reopen.

### Step 6: Address archive and reopen review feedback

- Status: completed

#### Objective

Tighten lifecycle behavior so archive/reopen respects local review, CI, and
sync state instead of relying only on tracked plan completeness.

#### Details

The current PR review surfaced two lifecycle gaps: `archive` can ignore
unresolved `.local` state, and `reopen` can preserve stale CI/sync evidence
from the archived candidate.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/status/service.go`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- `harness archive` rejects unresolved active review, pending/failed CI, and
  stale/conflicting sync state with agent-friendly errors.
- `harness reopen` clears stale lifecycle signals that should not survive into
  the reopened revision.
- Automated tests cover both the new rejection paths and the cleaned reopen
  state.

#### Execution Notes

`harness archive` now rejects unresolved plan-local review, CI, and sync state
before moving a plan to `docs/plans/archived/`. `harness reopen` now clears
stale `latest_ci` and `sync` signals, alongside `active_review_round`, so a
reopened revision does not inherit misleading archive-candidate state.

#### Review Notes

`go test ./internal/lifecycle ./internal/review ./internal/status` passes, and
the new lifecycle tests cover unresolved review/CI/sync archive rejection plus
stale-state cleanup on reopen.

### Step 7: Simplify review round identifiers

- Status: completed

#### Objective

Replace the current long timestamp-based review round IDs with a shorter,
plan-local sequence that is easier for humans and agents to read.

#### Details

The identifier should stay deterministic within a plan, preserve ordering, and
leave precise timestamps to the manifest instead of the round ID itself.

#### Expected Files

- `internal/review/service.go`
- `internal/review/service_test.go`
- `internal/status/service.go`
- `docs/specs/cli-contract.md`

#### Validation

- New review rounds use a compact `review-<NNN>-<kind>` shape.
- Sequence numbers increase monotonically within a plan-local review history.
- Automated tests cover the first round, later rounds, and status/reporting
  paths that expose the round ID.

#### Execution Notes

Review round IDs now use compact plan-local sequence numbers in the form
`review-<NNN>-<kind>`. New rounds derive the next sequence from existing
plan-local review directories, which lets this repo continue after the earlier
timestamp-based rounds without migration.

#### Review Notes

`go test ./...` passes, and a real dogfood review on this plan created
`review-003-delta`, then submitted and aggregated successfully with the new
compact ID format.

### Step 8: Make archive review gating revision-aware

- Status: completed

#### Objective

Require a passing `full` review for the initial archive candidate while still
allowing reopened, narrow follow-up work to archive after a passing `delta`
review.

#### Details

The remaining PR feedback is really about carrying durable review outcome into
archive gating. Revision 1 should require a passing `full` review. Later
revisions may archive after a passing `delta` review when the reopened change
is intentionally narrow.

#### Expected Files

- `internal/lifecycle/service.go`
- `internal/lifecycle/service_test.go`
- `internal/review/service.go`
- `internal/review/service_test.go`
- `internal/runstate/state.go`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- Revision 1 archive fails without a passing `full` review in local state.
- Reopened revisions can archive after a passing `delta` review.
- Aggregated failed review outcomes stay visible in local state and still block
  archive.
- Commands can recover the latest review decision from `aggregate.json` when
  older local state predates the stored `decision` field.
- A real reviewer subagent runs against the current branch and submits results
  through the harness contract rather than relying on main-agent-only smoke.

#### Execution Notes

Persisted aggregated review decisions into plan-local state, made revision-1
archive require a passing `full` review while later revisions can use a
passing `delta`, and taught `harness status` to surface aggregated review
failures as `fix_required` instead of falling back to generic implementation
guidance. A second real reviewer round then exposed a legacy-upgrade gap:
older aggregated review state may lack the stored `decision` field, so the
current slice now falls back to the round `aggregate.json` artifact when that
field is missing. A third real reviewer round then found a priority bug in
`harness status`: once all steps were complete, failed review states could
still surface closeout guidance instead of repair-and-rereview guidance. The
current slice now fixes that ordering as well. A fourth reviewer round then
found that legacy aggregated reviews whose outcome cannot be recovered at all
still need conservative repair guidance instead of falling back to generic
implementing or closeout/archive advice; the current slice now treats that
case as review follow-up too.

#### Review Notes

`review-005-delta`, `review-006-delta`, `review-007-delta`, and
`review-008-delta` each found one real state/handoff bug, and each finding is
now fixed in the current delta. `review-009-delta` passed clean through a real
subagent reviewer submission plus aggregate cycle.

## Validation Strategy

- Validate contracts by keeping the active plan, plan template, and spec docs
  aligned before code implementation begins.
- Prefer machine-checkable structure over prose-heavy workflow narration.
- Treat the CLI as agent-facing: stateful commands must produce concise,
  parseable output with recommended next steps instead of raw logs.
- Every behavior-changing CLI command in this plan must ship with automated
  tests for both success and representative failure paths.
- Use step-local `Execution Notes` and `Review Notes` to preserve context as
  work advances instead of relying on archive-time memory alone.
- Keep local execution history in `.local`; only durable summaries should be
  required in tracked plan files.

## Risks

- Risk: The plan schema becomes too verbose and starts duplicating trajectory
  that should live in `.local`.
  - Mitigation: Keep only durable conclusions in tracked plans and leave raw
    event history to local artifacts.
- Risk: The CLI overfits to one agent runtime or one repository layout.
  - Mitigation: Keep review and archive contracts runtime-agnostic and avoid
    repository-specific fields unless they are strictly necessary.
- Risk: Agents leak too much effort into manually maintaining status.
  - Mitigation: Store only the coarse lifecycle in plans and infer the smaller
    execution phase from local review/CI/publish artifacts.

## Validation Summary

Validated the lifecycle and status slice with `go test ./...`, plus targeted
package runs for `./internal/status`, `./internal/lifecycle`, and
`./internal/review` while iterating on the review-gating edge cases.
Additional dogfood validation covered repeated real `harness review start ->
submit -> aggregate` cycles, `harness status` after each aggregate outcome,
and legacy-compatibility tests that recover review decisions from
`aggregate.json` when older local state lacks the stored `decision` field.

## Review Summary

Real reviewer subagents drove five delta rounds on this revision. The first
four rounds surfaced substantive issues:

- `review-005-delta`: failed aggregated review decisions were not visible in
  `harness status`
- `review-006-delta`: older aggregated review state without
  `active_review_round.decision` broke archive/status semantics
- `review-007-delta`: completed plans with failed aggregated reviews still got
  closeout guidance instead of repair guidance
- `review-008-delta`: unrecoverable legacy review outcome still fell back to
  misleading archive/closeout guidance

Each of those findings is fixed in this slice. `review-009-delta` passed with
no findings.

## Archive Summary

- Archived At: 2026-03-18T22:13:16+08:00
- Revision: 3
- PR: https://github.com/yzhang1918/superharness/pull/3
- Ready: Revision-aware archive gating, review-decision persistence, legacy
  aggregate fallback, and conservative status handoff guidance are all in
  place, tested, and reviewed for this reopened revision.
- Merge Handoff: Run `harness archive`, commit and push the tracked plan move,
  then let the PR checks rerun before asking for merge approval.

## Outcome Summary

### Delivered

Documented and implemented the v0.1 foundation slice for plans, review rounds,
status, archive, and reopen. This revision tightened archive semantics so the
initial candidate requires a passing `full` review, reopened narrow fixes can
archive after a passing review result, aggregate decisions persist in local
state, legacy review decisions can be recovered from `aggregate.json`, and
`harness status` now gives repair-first guidance for failed or unrecoverable
review outcomes.

### Not Delivered

`harness ui`, `harness plan list`, the first reusable skill system, and shared
test fixtures remain intentionally deferred from this PR.

### Follow-Up Issues

- #2 `harness ui`
- #4 `harness plan list` and docs navigation
- #5 first skill system, including reviewer-skill contract
- #6 shared test infrastructure
