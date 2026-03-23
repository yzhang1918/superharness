# State Transitions

## Purpose

This document enumerates every allowed `current_node` transition in the v0.2
state model.

If a transition is not listed here, it is not a supported workflow move.
Command names and exact payload schemas live in the CLI contract; this file is
the normative transition matrix.

## Conventions

- `step-<n>` means the current unfinished step.
- `step-<m>` means the next unfinished step after `step-<n>`.
- "Durably complete" means the step `Done` marker is checked and the plan
  satisfies the required step-local closeout notes.

## Entering Work

| From | To | Driver | Required inputs | Notes |
| --- | --- | --- | --- | --- |
| `idle` | `plan` | Derived from tracked plan presence | Exactly one active plan exists and execution-start is absent | This is how newly approved or newly resumed work first appears in the runtime model. |
| `plan` | `execution/step-<n>/implement` | `harness execute start` | Current plan is approved for execution and has at least one unfinished step | The CLI records the execution-start milestone; the first unfinished step becomes current. |

## Step Execution Loop

| From | To | Driver | Required inputs | Notes |
| --- | --- | --- | --- | --- |
| `execution/step-<n>/implement` | `execution/step-<n>/review` | `harness review start` | Review round targets the current step | Review nodes require real review artifacts. |
| `execution/step-<n>/implement` | `execution/step-<m>/implement` | Derived from tracked plan edits | Step `<n>` becomes durably complete, any required step review is clean, and another unfinished step exists | A failed step review must be repaired and rerun before this transition is allowed. |
| `execution/step-<n>/implement` | `execution/finalize/review` | Derived from tracked plan edits | Step `<n>` becomes durably complete, any required step review is clean, and no unfinished steps remain | Finalize review stays distinct from step review. |
| `execution/step-<n>/review` | `execution/step-<n>/implement` | `harness review aggregate` | Latest aggregate is clean | Review is no longer in flight; the controller may continue implementation or mark the step done. |
| `execution/step-<n>/review` | `execution/step-<n>/implement` | `harness review aggregate` | Latest aggregate has actionable findings or an unrecoverable conservative outcome | The step remains current and must be repaired plus rerun through review before it may advance. |

## Finalize Loop

| From | To | Driver | Required inputs | Notes |
| --- | --- | --- | --- | --- |
| `execution/finalize/review` | `execution/finalize/fix` | `harness review aggregate` | Latest finalize review aggregate has actionable findings or an unrecoverable conservative outcome | Finalize review findings stay distinct from step-local findings. |
| `execution/finalize/review` | `execution/finalize/archive` | Derived from clean finalize review | Finalize review is satisfied and archive closeout work remains | Archive closeout includes summary refresh and placeholder replacement. |
| `execution/finalize/fix` | `execution/finalize/review` | `harness review start` | A new finalize review round is started after repair | Finalize repair must pass a later branch-level review before archive. |
| `execution/finalize/fix` | `execution/step-<m>/implement` | Derived from tracked plan edits | Reopen mode is `new-step`, the first new unfinished step has been added, and that new step is now current | Once the first reopened step exists, the special `new-step` requirement is consumed and ordinary step execution resumes. |

## Archive and Publish Handoff

| From | To | Driver | Required inputs | Notes |
| --- | --- | --- | --- | --- |
| `execution/finalize/archive` | `execution/finalize/publish` | `harness archive` | Finalize review is satisfied and archive closeout is ready | `archive` performs the tracked-file move and records archive metadata. |
| `execution/finalize/publish` | `execution/finalize/await_merge` | Derived from latest publish, CI, and sync evidence | Publish evidence identifies the candidate, CI is good enough or explicit `not_applied`, sync is acceptable or explicit `not_applied`, and no unresolved fix condition remains | `await_merge` is a merge-ready state, not merely an archived state. |
| `execution/finalize/publish` | `execution/finalize/fix` | `harness reopen --mode finalize-fix` | Archived candidate has been invalidated but does not justify a new step | Reopen is the command-owned reversal of archive-time assumptions. |
| `execution/finalize/publish` | `execution/finalize/fix` | `harness reopen --mode new-step` | Archived candidate has been invalidated and the change deserves a new unfinished step | Status stays in finalize-scope repair until the first new unfinished step is actually added. |

## Await-Merge and Land

| From | To | Driver | Required inputs | Notes |
| --- | --- | --- | --- | --- |
| `execution/finalize/await_merge` | `land` | `harness land --pr <url> [--commit <sha>]` | Human approval exists, merge happened outside harness, and land entry records the PR URL | Optional commit SHA enriches the record but is not required because merge strategies vary. |
| `execution/finalize/await_merge` | `execution/finalize/fix` | `harness reopen --mode finalize-fix` | Merge-ready archived candidate has been invalidated without justifying a new step | Reopen preserves audit history instead of blanking archive-time text. |
| `execution/finalize/await_merge` | `execution/finalize/fix` | `harness reopen --mode new-step` | Merge-ready archived candidate has been invalidated and the change deserves a new unfinished step | Status stays in finalize-scope repair until the first new unfinished step is actually added. |
| `land` | `idle` | `harness land complete` | Merge cleanup is done and land completion is intentionally recorded | `idle` is restored only by an explicit command-owned completion. |

## State-Preserving Updates

The following operations are allowed to preserve the current node instead of
advancing it:

- `plan -> plan`
  - plan edits, scope refinement, or approval discussion before
    `harness execute start`
- `execution/step-<n>/implement -> execution/step-<n>/implement`
  - continued implementation, validation, or note updates that do not start
    review or complete the step
- `execution/step-<n>/review -> execution/step-<n>/review`
  - reviewer submissions or aggregation are still pending
- `execution/finalize/review -> execution/finalize/review`
  - reviewer submissions continue arriving before finalize review is resolved
- `execution/finalize/fix -> execution/finalize/fix`
  - finalize-scope repair continues, or a `new-step` reopen is still waiting
    for the first new unfinished step to be added
- `execution/finalize/archive -> execution/finalize/archive`
  - archive summaries and placeholders are being refreshed before
    `harness archive`
- `execution/finalize/publish -> execution/finalize/publish`
  - new publish, CI, or sync evidence arrives but the archived candidate is
    still not merge-ready
- `land -> land`
  - post-merge cleanup continues before `harness land complete`

## Invalid Shortcuts

The following are intentionally invalid in v0.2:

- direct `plan -> execution/finalize/...` jumps
- direct step-implement jumps that skip the finalize review gate once all
  steps are complete
- step advancement after a failed step review without a later clean review for
  that same step
- direct `execution/finalize/fix -> execution/finalize/archive` jumps without a
  later clean finalize review
- direct `execution/finalize/archive -> execution/finalize/await_merge`
  jumps without publish, CI, and sync evidence
- direct `execution/finalize/await_merge -> idle` jumps without explicit land
  entry and completion
