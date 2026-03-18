---
status: archived
lifecycle: awaiting_merge_approval
revision: 2
template_version: 0.1.0
created_at: "2026-03-17T10:12:01+08:00"
updated_at: "2026-03-18T21:27:33+08:00"
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

- `go test ./...` passes across the current module after the revision 2
  lifecycle-state and review-ID changes.
- Focused lifecycle coverage now exercises unresolved review/CI/sync archive
  rejection and stale-state cleanup during reopen.
- Real CLI smoke runs confirmed:
  - `harness reopen` moved the archived plan back to active revision 2
  - a delta review created `review-003-delta` and aggregated cleanly
  - a full review created `review-004-full` and aggregated cleanly
  - `harness status` now resumes from the reopened plan and points toward
    archive once closeout summaries are present

## Review Summary

- Addressed both open PR review findings against lifecycle handling:
  - `archive` now blocks unresolved local review, CI, and sync state
  - `reopen` now clears stale CI and sync state from the archived candidate
- Revision 2 delta review `review-003-delta` passed with no findings after the
  lifecycle fixes landed.
- Revision 2 full review `review-004-full` passed with no blocking or
  non-blocking findings on the archive candidate.

## Archive Summary

- Archived At: 2026-03-18T21:27:33+08:00
- Revision: 2
- PR: https://github.com/yzhang1918/superharness/pull/3
- Ready: The revision 2 candidate addresses the open lifecycle review
  findings, keeps review round IDs compact and deterministic, and satisfies
  the tracked acceptance criteria, automated tests, and clean full review gate.
- Merge Handoff: Run `harness archive`, commit and push the archived-plan
  move, then wait for human merge approval or merge manually from the PR once
  checks are green.

## Outcome Summary

### Delivered

- Shipped revision 2 fixes that prevent `harness archive` from freezing a
  candidate with unresolved local review, CI, or sync state.
- Shipped revision 2 fixes that clear stale CI and sync signals during
  `harness reopen`, so the next revision starts from fresh local-state
  evidence.
- Replaced long timestamp-based review round IDs with compact plan-local IDs
  such as `review-003-delta` and `review-004-full`.
- Created follow-up issues for deferred plan list/docs navigation work,
  skill-system design, and shared test infrastructure.

### Not Delivered

- `harness ui` remains deferred to #2.
- `harness plan list` and the longer-term docs navigation decision remain
  deferred to #4.
- The first reusable skill system, including the reviewer skill contract,
  remains deferred to #5.
- Shared test fixtures and broader integration infrastructure remain deferred
  to #6.

### Follow-Up Issues

- #2 Add harness ui for local status and trajectory visualization
- #4 Add harness plan list and revisit docs navigation
- #5 Design the first skill system around the harness CLI
- #6 Build shared test infrastructure for harness workflows
