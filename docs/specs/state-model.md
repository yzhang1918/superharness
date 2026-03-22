# State Model

## Purpose

This document defines the normative v0.2 state model for `superharness`.

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

Agents do not set `current_node` directly. The CLI resolves it from tracked
plan content plus command-owned artifacts and may cache the latest answer in a
thin CLI-owned `state.json`.

### Durable Plan, Disposable Runtime

Tracked plans remain the durable source of scope, step closeout, and archive
summaries. Runtime trajectory, milestone timestamps, and external-fact capture
belong in `.local/harness/`.

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

### Tracked Plan Owns

- durable scope and non-goals
- acceptance criteria
- step list and step `Done` markers
- step-local `Execution Notes`
- step-local `Review Notes`
- archive-time summaries and outcome notes

### Command-Owned Runtime Artifacts Own

- worktree-level current-plan and last-landed context
- execute-start milestones
- review manifests, ledgers, submissions, and aggregates
- append-only `ci`, `publish`, and `sync` evidence records
- archive milestones
- reopen milestones, including the explicit reopen mode
- land entry and land completion milestones
- the thin `state.json` cache containing the latest resolved `current_node`
  plus latest artifact pointers

### Agents Must Not Directly Edit

- `current_node`
- CLI-owned pointer files
- CLI-owned `state.json`
- review manifests or aggregates
- command-owned evidence records that should have been created through
  `harness evidence submit`

## Current Plan Selection

v0.2 assumes one active tracked plan per repository.

Resolution rules:

- if exactly one active plan exists under `docs/plans/active/`, that plan is
  current for `plan` and `execution/...` nodes
- if no active plan exists, CLI-owned archived or landed context may still
  identify the current archived candidate or the most recent landed candidate
- if multiple active plans exist, state resolution is invalid and should fail
  rather than guess

## Runtime Inputs

`harness status` resolves `current_node` from:

- the tracked plan content
- whether execution-start has been recorded
- the first unfinished step from the tracked plan
- review artifacts for the current step or the finalize gate
- append-only `ci`, `publish`, and `sync` evidence
- archive, reopen, and land milestones
- worktree-level last-landed context when no current work remains

## High-Level Resolution Order

1. If merge has been confirmed and land cleanup is still incomplete, resolve
   `land`.
2. Otherwise, if no current work exists, resolve `idle`.
3. Otherwise, if the current active plan exists but execution-start has not
   been recorded, resolve `plan`.
4. Otherwise, if an unfinished step exists, resolve the appropriate
   `execution/step-<n>/...` node.
5. Otherwise, resolve the appropriate `execution/finalize/...` node.

The exact transition matrix is normative in
[State Transitions](./state-transitions.md).

## Node Semantics

### `idle`

No current work is in flight. This is the normal post-land resting state.

### `plan`

A current active plan exists, but execution has not started. Plan edits,
approval, and step refinement happen here.

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
reopened work that did not justify a new step, or archived candidate
invalidation that must be repaired before archive or merge readiness can be
claimed again.

### `execution/finalize/archive`

Finalize review is satisfied and the remaining work is archive-closeout:
refreshing required summaries, resolving placeholders, and preparing for the
tracked-file move.

### `execution/finalize/publish`

The plan is already archived, but merge readiness still depends on external
handoff facts recorded through `publish`, `ci`, and `sync` evidence.

### `execution/finalize/await_merge`

The archived candidate is ready for human merge approval. PR existence, CI,
and sync freshness or conflict checks are already satisfied or explicitly
marked `not_applied`.

### `land`

Merge is confirmed and post-merge cleanup is in progress. Cleanup remains in
`land` until `harness land complete` intentionally restores `idle`.

## Step and Review Rules

- The first unfinished step determines the current execution step.
- A step should not be marked done until its implementation, execution notes,
  review notes, and relevant review loop are complete, or the step records why
  no review was needed.
- A completed step is review-complete when either:
  - a clean `step_closeout` review exists for that step
  - or `Review Notes` records `NO_STEP_REVIEW_NEEDED: <reason>`
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
by default.

## Commits and Nodes

Git commits are workflow guidance, not state transitions.

- A small reviewable commit usually happens before step review starts.
- A meaningful review-driven repair usually gets another small commit before a
  fresh step review round starts.
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
  - records cleanup completion and restores `idle`

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
