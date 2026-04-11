# State Model

## Purpose

This document defines the normative v0.2 state model for `easyharness`.

The v0.2 model replaces the v0.1 layered vocabulary of tracked lifecycle,
derived step state, derived handoff state, and worktree hints with one
canonical runtime node:

```json
{
  "current_node": "execution/step-2/review"
}
```

Exact transition enumeration lives in
[State Transitions](./state-transitions.md). Exact tracked-plan and CLI schema
details live in [Plan Schema](./plan-schema.md) and
[CLI Contract](./cli-contract.md).

Field-level JSON structure for the CLI-owned local artifacts described here
lives in the checked-in schema registry at
[`schema/index.json`](../../schema/index.json). See
[Contract Registry](./contract.md) for the ownership model and discovery
entrypoints.

## Non-Goals

The v0.2 state model does not:

- preserve v0.1 compatibility layers such as `lifecycle`, `step_state`,
  `handoff_state`, or `worktree_state`
- keep top-level execution state in tracked plan frontmatter
- support multiple simultaneous active plans in one repository
- make harness a wrapper around routine `git` or `gh` operations

## Core Principles

### One Canonical Node

Every workflow question should reduce to one node string. Summary text,
selected facts, and recommended next actions are all derived from that node
plus the latest relevant artifacts.

### CLI-Owned Resolution

Agents do not set `current_node` directly. The CLI resolves it from the
current plan artifact plus command-owned artifacts. The plan-local
`state.json` remains a CLI-owned control artifact for runtime facts that must
persist across commands, but it is not the storage location for a cached
latest `current_node`.

### Safe Local Persistence

CLI-owned runstate files must stay parseable even when commands run close
together or a process exits during persistence.

- `.local/harness/current-plan.json` must be written with atomic replacement
  rather than in-place overwrite writes
- `.local/harness/plans/<plan-stem>/state.json` must also use atomic
  replacement
- any command that mutates a plan-local `state.json` must acquire a shared
  per-plan state-mutation lock before it loads and rewrites that file
- if the per-plan state lock is already held, the command should fail with a
  clear error rather than waiting silently or risking a stale overwrite

### Durable Plan, Disposable Runtime

Tracked active plans remain the durable source of scope, step closeout, and
archive summaries for both profiles. Lightweight work uses the same schema and
the same tracked active-plan location, but its archived snapshot moves into
`.local/harness/plans/archived/` so the workflow can stay lightweight for
narrow low-risk changes. Runtime trajectory, milestone timestamps, and
external-fact capture also belong in `.local/harness/`. There is no separate
local active lightweight plan path in this model.

### Explicit Command Boundaries

Commands own milestones and append-only trajectory where consistency and
timestamps matter. Agents still own plan edits, reviewable code and docs, and
all direct `git` or GitHub actions.

## Canonical Node Tree

```text
root
├── idle
├── plan
├── execution
│   ├── step-<n>
│   │   ├── implement
│   │   └── review
│   └── finalize
│       ├── review
│       ├── fix
│       ├── archive
│       ├── publish
│       └── await_merge
└── land
```

## Ownership Split

### Plan Artifact Owns

- durable scope and non-goals
- acceptance criteria
- step list and step `Done` markers
- step-local `Execution Notes`
- step-local `Review Notes`
- archive-time summaries and outcome notes

For active work in both profiles, this plan artifact is a tracked file under
`docs/plans/active/`. Standard archives stay tracked under
`docs/plans/archived/`. Lightweight archived snapshots move to
`.local/harness/plans/archived/`.

### Command-Owned Runtime Artifacts Own

- worktree-level current-plan and last-landed context
- execute-start milestones
- review manifests, ledgers, submissions, and aggregates, including optional
  reviewer-provided finding locations preserved in submission and aggregate
  artifacts
- append-only timeline event indexes under
  `.local/harness/plans/<plan-stem>/events.jsonl`
- append-only `ci`, `publish`, and `sync` evidence records
- archive milestones
- reopen milestones, including the explicit reopen mode
- land entry and land completion milestones
- the plan-local `state.json` control artifact containing only cross-command
  runtime facts that are not otherwise reconstructed directly from plans or
  append-only artifacts

These CLI-owned JSON artifacts are disposable runtime state, but they still
need crash-safe persistence so the controller can trust `harness status` after
any interrupted or overlapping command.

### Agents Must Not Directly Edit

- `current_node`
- CLI-owned pointer files
- CLI-owned `state.json`
- review manifests or aggregates
- command-owned evidence records that should have been created through
  `harness evidence submit`

## Current Plan Selection

v0.2 assumes one active plan artifact per repository.

Resolution rules:

- if more than one active tracked plan exists under `docs/plans/active/`,
  state resolution is invalid and should fail rather than guess
- lightweight archived snapshots under `.local/harness/plans/archived/` do not
  count as active-plan candidates
- if `.local/harness/current-plan.json` points to the sole active plan path and
  that path still exists, that plan is current
- otherwise, if exactly one active tracked plan exists under
  `docs/plans/active/`, that plan is current for `plan` and `execution/...`
  nodes
- if no active plan exists, CLI-owned archived or landed context may still
  identify the current archived candidate or the most recent landed candidate

## Runtime Inputs

`harness status` resolves `current_node` from:

- the current plan content
- the plan path and optional `workflow_profile: lightweight`
- whether execution-start has been recorded
- the first unfinished step from the current plan
- review artifacts for the current step or the finalize gate
- append-only `ci`, `publish`, and `sync` evidence
- archive, reopen, and land milestones
- worktree-level last-landed context when no current work remains

The plan-local `state.json` carries only the control-plane subset of those
inputs that do not already live in a more specific artifact:

- `execution_started_at`
- `revision`
- `active_review_round`
- `reopen`
- `land`

The mutation surfaces around those runtime artifacts stay split on purpose:

- `.state-mutation.lock` serializes rewrites of plan-local `state.json`
- `.review-mutation.lock` serializes review-artifact mutation such as round
  creation, submission, ledger updates, and aggregation
- `.timeline-mutation.lock` serializes appends to the plan-local
  `events.jsonl` index

When a review command mutates both review artifacts and `state.json`, it should
acquire the review mutation lock before the state mutation lock. Commands that
only submit reviewer output should stay on the review-artifact path and should
not acquire the state mutation lock just because the round also has local
state.

## High-Level Resolution Order

1. If merge has been confirmed and the required post-merge bookkeeping is still
   incomplete, resolve `land`.
2. Otherwise, if no current work exists, resolve `idle`.
3. Otherwise, if the current active plan exists but execution-start has not
   been recorded, resolve `plan`.
4. Otherwise, if an unfinished step exists, resolve the appropriate
   `execution/step-<n>/...` node.
5. Otherwise, resolve the appropriate `execution/finalize/...` node.

The exact transition matrix is normative in
[State Transitions](./state-transitions.md).

The lightweight profile does not add a second node tree. It reuses the same
canonical nodes while changing where the archived snapshot lives and what
closeout guidance `harness status` should emphasize.

## Node Semantics

### `idle`

No current work is in flight. This is the normal post-land resting state.

### `plan`

A current tracked active plan exists, but execution has not started. Plan
edits, approval, and step refinement happen here for both profiles.

### `execution/step-<n>/implement`

Execution has started, step `<n>` is the first unfinished step, and no active
review round is currently in flight for that step. This node covers both
ordinary implementation work and post-review repair work at step scope.

### `execution/step-<n>/review`

Step `<n>` is in a real review loop backed by review artifacts. This node
means review is actively in flight: reviewer submissions or aggregation are
still pending. Once the review outcome is known, resolution returns to
`execution/step-<n>/implement`.

### `execution/finalize/review`

All intended steps are durably complete, and the whole-branch candidate still
needs a final review gate. This is distinct from the last step's review.

### `execution/finalize/fix`

The whole-branch candidate needs repair because of finalize review findings,
reopened work that did not justify a new step, a `new-step` reopen that is
still waiting for the first new unfinished step to be added, or archived
candidate invalidation that must be repaired before archive or merge readiness
can be claimed again.

### `execution/finalize/archive`

Finalize review is satisfied and the remaining work is archive-closeout:
refreshing required summaries, resolving placeholders, and preparing for the
appropriate archive move or snapshot update.

### `execution/finalize/publish`

The plan is already archived, but merge readiness still depends on external
handoff facts recorded through `publish`, `ci`, and `sync` evidence. For
lightweight work, this phase is also where status should remind the controller
to leave the agreed repo-visible breadcrumb, such as a PR body note explaining
why the lightweight path was used.

### `execution/finalize/await_merge`

The archived candidate is ready for human merge approval. PR existence, CI,
and sync freshness or conflict checks are already satisfied or explicitly
marked `not_applied`.

### `land`

Merge is confirmed and required post-merge bookkeeping is in progress. This
work remains in `land` until `harness land complete` intentionally restores
`idle`.

## Step and Review Rules

- The first unfinished step determines the current execution step.
- In the ordinary loop, `execution/step-<k>/...` names the current execution
  frontier. However, an explicit earlier-step closeout repair may intentionally
  re-enter `execution/step-<i>/review` or `execution/step-<i>/implement` for a
  completed earlier step that is being repaired.
- A step should not be marked done until its implementation, execution notes,
  review notes, and relevant review loop are complete, or the step records why
  no review was needed.
- A completed step is review-complete when either:
  - the latest known step-bound review for that step is clean
  - or `Review Notes` records `NO_STEP_REVIEW_NEEDED: <reason>` and no later
    in-flight or non-clean step-bound review exists for that step
- Step-closeout review should default to `delta`, but a `full` review may
  satisfy step closeout when a narrower pass would be misleading or the slice
  needs a broader risk scan.
- Review nodes require real review artifacts created by `harness review`.
- `execution/step-<n>/review` means review is still in progress.
- Once a step review aggregate exists, the state returns to
  `execution/step-<n>/implement`.
- If the latest step review aggregate is not clean, the step stays current and
  must not advance to the next step or to finalize review until a later clean
  review aggregate exists for that step.
- A clean step review does not automatically mark the step done; it only
  clears the review gate so the controller can either continue the step or mark
  it durably complete.
- Status facts and next actions must make unresolved failed step reviews
  explicit when `execution/step-<n>/implement` is being used for repair work.
- If `harness status` later discovers that an already completed earlier step is
  still missing review-complete closeout, it should keep the current step or
  finalize node stable, add warning-driven repair guidance, and avoid pretending
  that the earlier closeout is complete.
- If an explicit earlier-step closeout repair review is started, status should
  treat the targeted earlier step as current for that repair loop while the
  review is in flight or after a non-clean aggregate.
- If an explicit earlier-step closeout repair review later fails, the fix work
  still belongs to the same overall candidate, but the repaired step remains
  current until a later clean closeout review or explicit no-review-needed
  note resolves that earlier-step debt.
- Default finalize review start and archive must reject unresolved earlier-step
  review-complete debt even though status may continue surfacing the current
  later-step or finalize node as the stable workflow position.
- Finalize review remains a distinct whole-branch gate even if an earlier step
  review used a full-review recipe.
- After `execution/finalize/fix`, the candidate must pass a later finalize
  review before archive; finalize repair does not jump straight to archive.

## Publish, CI, and Sync Evidence Rules

The repository standardizes three command-owned evidence domains:

- `publish`
  - records PR or handoff facts for the archived candidate
- `ci`
  - records the latest relevant CI or required-check result
- `sync`
  - records remote freshness and conflict facts relevant to merge readiness

Rules:

- all three domains are recorded through `harness evidence submit`
- missing evidence never means success or not-applicable
- `not_applied` must be recorded explicitly when a domain truly does not apply
- freshness belongs to `execution/finalize/publish`, not to pre-archive
  readiness

## Reopen Rules

`harness reopen` is the mechanical reversal of archive-time assumptions and
requires an explicit mode:

- `finalize-fix`
  - reopened work stays in finalize-scope repair
- `new-step`
  - reopened work must be represented by a new unfinished step rather than
    being smuggled into prior completed steps

Reopen must preserve audit history:

- do not blank archive-time wording back to empty
- replace reopen-sensitive summaries with explicit update-required placeholders
- keep it obvious that the plan was once archived and is no longer current

When reopen mode is `new-step`, the controller should add the new step after
reopen and continue execution at that new step's `implement` node. Once that
first reopened step has been added, the `new-step` requirement is considered
consumed: later finalize-time findings should repair the latest reopened work
or resume finalize-scope repair instead of forcing another new unfinished step
by default. Until that first new unfinished step exists, status remains in
`execution/finalize/fix` and should keep prompting for the new step rather than
pretending implementation has already resumed.

## Commits and Nodes

Git commits are workflow guidance, not state transitions.

- `delta` review must anchor to a real git commit.
- A small reviewable commit should exist before a step-closeout `delta` review
  starts so that review has a durable starting point.
- A meaningful review-driven repair that may need later `delta` review should
  establish another small anchor commit before the fresh review round starts.
- Archive readiness still requires the archived tracked move to be committed
  and pushed before publish, CI, and merge-approval work can be treated as
  complete.

Commit boundaries can support reviewability and handoff, but they do not
change `current_node` by themselves.

## Land Rules

Land is explicit and two-stage:

- `harness land --pr <url> [--commit <sha>]`
  - records merge confirmation and enters `land`
- `harness land complete`
  - records required post-merge bookkeeping completion and restores `idle`

The PR URL is required for land entry in v0.2. Commit SHA is optional because
merge strategy may produce a different landed commit shape across merge-commit,
squash, or rebase flows.

## Status Rendering

`harness status` should render:

- the resolved `current_node`
- one concise summary
- selected supporting facts
- concrete next actions ordered by likely workflow need

Status output should explain where the controller is now and what kind of work
that node implies. It should not recreate v0.1 by reintroducing parallel
top-level lifecycle fields.
