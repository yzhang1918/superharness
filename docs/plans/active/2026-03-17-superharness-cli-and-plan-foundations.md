---
status: active
lifecycle: executing
revision: 1
template_version: 0.1.0
created_at: 2026-03-17T10:12:01+08:00
updated_at: 2026-03-18T00:15:27+08:00
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

### Out of Scope

- `harness ui` and any web UI implementation.
- Runtime-specific subagent spawning inside repository code.
- GitHub merge automation beyond the archive/merge handoff contract.

## Acceptance Criteria

- [x] The repository contains a durable v0.1 plan schema spec and matching plan
      template.
- [x] The repository contains a v0.1 CLI contract spec for agent-facing command
      behavior and output.
- [ ] Every behavior-changing command in the v0.1 slice is covered by
      automated tests.
- [x] `harness plan template` and `harness plan lint` are implemented against
      the documented contract.
- [x] `harness status` reports plan state plus step state from
      local artifacts without requiring manual state bookkeeping.
- [ ] `harness review start`, `harness review submit`, and
      `harness review aggregate` implement the review-round contract without
      binding to a specific subagent runtime.
- [ ] `harness archive` and `harness reopen` implement the freeze/reopen
      lifecycle with agent-friendly `next_actions`.

## Deferred Items

- `harness ui` is intentionally deferred until the CLI and local-state
  contracts are stable enough to serve as a clean backend.

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
robustness gaps, which were fixed before closing this step.

### Step 4: Implement the review-round contract

- Status: pending

#### Objective

Implement `harness review start`, `submit`, and `aggregate` so review rounds
have deterministic manifests, output paths, and aggregation.

#### Details

The CLI should validate and persist an agent-supplied review spec, not invent
review dimensions or a path list on the agent's behalf.

#### Expected Files

- `cmd/...`
- `internal/...`
- `*_test.go`
- `docs/specs/cli-contract.md`

#### Validation

- Review rounds emit stable manifest and ledger artifacts for both delta and
  full review kinds.
- Review submission and aggregation work without embedding runtime-specific
  subagent launch logic in the CLI.
- Automated tests cover review-round creation, valid and invalid reviewer
  submission, and delta versus full aggregation outcomes.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 5: Implement archive and reopen

- Status: pending

#### Objective

Implement the freeze-and-summarize archive flow plus mechanical reopen
behavior for archived plans.

#### Details

Archive should validate both top-level archive summaries and completed-step
closeout notes before moving the plan. Reopen should restore local editing
without forcing the next agent to rediscover context from scratch.

#### Expected Files

- `cmd/...`
- `internal/...`
- `*_test.go`
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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
