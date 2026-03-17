# Plan Schema

## Purpose

`superharness` keeps the durable execution contract in git-tracked plan files
and keeps raw execution trajectory in `.local`. The plan must remain readable
to humans, lintable by the CLI, and robust to agent session compaction.

This document defines the v0.1 tracked-plan contract.

## Directory Layout

Tracked plans live in:

- `docs/plans/active/`
- `docs/plans/archived/`

Local execution artifacts live in:

- `.local/harness/current-plan.json`
- `.local/harness/plans/<plan-stem>/state.json`
- `.local/harness/plans/<plan-stem>/events.jsonl`
- `.local/harness/plans/<plan-stem>/reviews/`
- `.local/harness/plans/<plan-stem>/ci/`
- `.local/harness/plans/<plan-stem>/sync/`
- `.local/harness/plans/<plan-stem>/publish/`

Tracked plans are the durable source of truth for scope and archived outcome.
Local artifacts are disposable execution support and must not be required to
understand the task's durable scope after archive.

### Local Artifact Ownership

The plan-local `.local` directory is intentionally structured so the CLI, not
the agent, owns most step-state and artifact bookkeeping:

- `state.json`
  - the latest local snapshot for the current plan
  - points to the latest relevant review round, CI snapshot, and publish
    attempt IDs
- `events.jsonl`
  - append-only local trajectory
  - useful for debugging and UI later, but not required for durable history
- `reviews/<round-id>/`
  - one review round per directory
  - recommended contents: `manifest.json`, `submissions/<slot>.json`,
    `aggregate.json`
- `ci/<snapshot-id>.json`
  - exported CI or required-check status for one candidate state
- `sync/<snapshot-id>.json`
  - remote freshness, divergence, rebase, merge-conflict, or branch-sync
    metadata for one candidate state
- `publish/<attempt-id>.json`
  - PR or publish metadata for one push/update attempt

Repeated work on the same plan creates more review rounds, CI snapshots, and
publish attempts under the same plan stem. `state.json` points at the latest
relevant artifacts so `harness status` does not need to guess by scanning raw
history.

v0.1 standardizes `reviews/`, `ci/`, `sync/`, and `publish/`. Future harness
versions may add other command-owned artifact lanes, but these are the core
ones the initial CLI contract depends on.

## Source of Truth

The source-of-truth split is:

- this schema document is the normative contract
- [the packaged plan template asset](../../assets/templates/plan-template.md)
  is the canonical authoring example shipped by the harness
- `harness plan template` is a convenience wrapper that renders the packaged
  asset with seeded metadata

Consumer repositories do not need to track the template file in git. The
template belongs to the harness package version, not the user's plan history.

Skills should not duplicate lifecycle enums or placeholder tokens. They should
refer agents to `harness --help`, `harness plan template`, and
`harness plan lint` rather than carrying a second copy of the plan contract.

## File Naming

Each plan file name is its durable identifier.

Required pattern:

- `YYYY-MM-DD-short-topic.md`

Where:

- `YYYY-MM-DD` is the creation date
- `short-topic` is a compact human-readable topic slug

Short-topic rules:

- keep it concrete and searchable
- name the affected area plus the actual change or problem
- prefer roughly 3-7 kebab-case words
- avoid generic endings such as `task`, `work`, `thing`, `update`, or
  `foundations` unless they are qualified by the actual scope
- if two same-day plans feel close, widen the topic with a concrete qualifier
  from the scope instead of appending an opaque counter

Examples:

- `2026-03-17-superharness-cli-and-plan-foundations.md`
- `2026-03-17-review-round-audit-contract.md`
- `2026-03-17-status-handoff-recovery.md`

The goal is not uniqueness through numbering. The goal is a filename that stays
clear and searchable when humans or agents scan plan history.

Lint should enforce the `YYYY-MM-DD-short-topic.md` shape, including a valid
calendar date and a kebab-case short topic.

The file stem is the durable plan identifier used by local
`.local/harness/plans/<plan-stem>/` state.

## Frontmatter

Every plan must start with YAML frontmatter containing these required fields:

```yaml
---
status: active
lifecycle: executing
revision: 1
template_version: 0.1.0
created_at: 2026-03-17T10:12:01+08:00
updated_at: 2026-03-17T10:12:01+08:00
source_type: direct_request
source_refs: []
---
```

### Required Fields

- `status`
  - Allowed values: `active`, `archived`
  - Must match the containing directory.
- `lifecycle`
  - Allowed values: `awaiting_plan_approval`, `executing`, `blocked`,
    `awaiting_merge_approval`
  - Only the coarse workflow phase belongs in the tracked plan.
- `revision`
  - Positive integer.
  - Starts at `1`.
  - Increments only when `harness reopen` moves an archived plan back to
    active work.
- `template_version`
  - Harness template/schema version used when the plan was generated or last
    intentionally migrated.
  - v0.1 examples use `0.1.0`.
  - Lint should accept the bundled template version and older historical
    versions, while rejecting versions newer than the current harness knows
    how to validate.
- `created_at`
  - RFC3339 timestamp with offset.
- `updated_at`
  - RFC3339 timestamp with offset.
  - Must be updated whenever the tracked plan content changes.
- `source_type`
  - Short lower-snake-case or kebab-case string describing intake source.
  - Recommended values include `direct_request`, `issue`, `backlog`,
    `incident`, or `other`.
- `source_refs`
  - Array of issue IDs, URLs, or other external references.
  - Use `[]` when there are none.

Read `status` and `lifecycle` together like this:

- `status` answers whether the plan is still active or already archived
- `lifecycle` answers what stage of the workflow the plan is in

### Intentionally Omitted Fields

The v0.1 plan contract does not require:

- `plan_slug`
- `kind`
- `branch`
- `pr_url`
- step state in tracked frontmatter

Those either come from the file path/title, belong in archive-time summaries,
or are better inferred from local artifacts.

## Lifecycle Model

The tracked plan stores only the coarse lifecycle:

- `awaiting_plan_approval`
  - Discovery is done and the plan is waiting for explicit approval.
- `executing`
  - The agent is implementing, reviewing, publishing, or preparing to archive.
- `blocked`
  - Work cannot proceed without human input or an external dependency.
- `awaiting_merge_approval`
  - The plan is archived and frozen; merge approval happens outside the plan.

Step state such as "testing", "waiting for CI", or "resolving conflicts" must
be inferred from step context plus `.local` review/CI/publish/sync artifacts
and shown in CLI output rather than hand-maintained in the tracked markdown.

## Required Sections

Every plan must contain these sections in order:

1. `# <Title>`
2. `## Goal`
3. `## Scope`
4. `## Acceptance Criteria`
5. `## Deferred Items`
6. `## Work Breakdown`
7. `## Validation Strategy`
8. `## Risks`
9. `## Validation Summary`
10. `## Review Summary`
11. `## Archive Summary`
12. `## Outcome Summary`

`## Deferred Items` is the single place for consciously deferred slice items,
whether they look more like postponed tasks or accepted-but-tracked risks.
It is different from `Outcome Summary > Follow-Up Issues`, which records the
actual tracker entries created to own those deferred items after archive.

### Scope Structure

`## Scope` must include these subsections:

- `### In Scope`
- `### Out of Scope`

### Acceptance Criteria

- Every criterion must be a markdown checkbox.
- Active plans may mix checked and unchecked criteria.
- Archived plans must have every criterion checked.

### Work Breakdown

The work breakdown must use step headings:

```md
### Step 1: Define the contract
```

Every step must include:

- a `Status:` line directly under the step heading
  - Allowed values: `pending`, `in_progress`, `completed`, `blocked`
- `#### Objective`
- `#### Details`
- `#### Expected Files`
- `#### Validation`
- `#### Execution Notes`
- `#### Review Notes`

Optional step section:

- `#### Step Acceptance Criteria`

`Objective` should stay concise. Longer planning detail belongs in
`#### Details`. If there is no extra detail beyond the objective, write
`NONE`.

`#### Execution Notes` and `#### Review Notes` are the preferred place for
step-local closeout as work advances. They make it easier for another agent or
a compacted session to see what already happened before archive-time summaries
exist.

If `#### Step Acceptance Criteria` is present, every criterion in that section
must be a markdown checkbox. Active steps may mix checked and unchecked boxes.
Completed archived steps must have every step-local criterion checked.

Example:

```md
### Step 1: Define the contract

- Status: completed

#### Objective

Write the v0.1 plan and CLI specs.

#### Details

Keep lifecycle coarse and let the CLI infer smaller execution hints.

#### Expected Files

  - `docs/specs/plan-schema.md`
  - `docs/specs/cli-contract.md`

#### Validation

  - The docs agree on lifecycle, archive, and output-envelope rules.

#### Execution Notes

Captured the initial durable schema and CLI contract.

#### Review Notes

No step-local review needed yet because this was a docs-only planning step.
```

Behavior-changing implementation steps should mention the automated tests they
expect to add or update in `Expected files` or `Validation`. v0.1 does not add
a separate mandatory `Tests` field.

## Placeholder Policy

These exact placeholder tokens are allowed only while a plan is still active:

- `Step > Execution Notes`: `PENDING_STEP_EXECUTION`
- `Step > Review Notes`: `PENDING_STEP_REVIEW`
- `Validation Summary`: `PENDING_UNTIL_ARCHIVE`
- `Review Summary`: `PENDING_UNTIL_ARCHIVE`
- `Archive Summary`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Delivered`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Not Delivered`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Follow-Up Issues`: `NONE`

`harness archive` must reject any plan that still contains
`PENDING_UNTIL_ARCHIVE`.

Archived plans must not leave `PENDING_STEP_EXECUTION` or `PENDING_STEP_REVIEW`
in any completed step.

`NONE` is not itself an error. It is allowed when there are no follow-up issues.

## Deferred Items and Follow-Up Tracking

Use these two surfaces deliberately:

- `## Deferred Items`
  - work or risk deliberately deferred from the current slice
- `## Outcome Summary > Follow-Up Issues`
  - actual tracker items created to own deferred items or post-archive follow-up

Archive readiness should enforce this distinction:

- if `## Deferred Items` contains real items at archive time, the archived plan
  must not leave `### Follow-Up Issues` as `NONE`
- archived follow-up entries should include concrete tracker references such as
  issue numbers or URLs
- if there are no deferred items and no post-archive follow-up, `### Follow-Up
  Issues` may remain `NONE`

## Active Plan Rules

An active plan must satisfy all of these:

- File path is under `docs/plans/active/`
- `status: active`
- `lifecycle` is one of:
  - `awaiting_plan_approval`
  - `executing`
  - `blocked`
- Archive-only summary sections may still use the documented placeholder
  tokens.

## Archived Plan Rules

An archived plan must satisfy all of these:

- File path is under `docs/plans/archived/`
- `status: archived`
- `lifecycle: awaiting_merge_approval`
- Every acceptance criterion is checked
- Every work-breakdown step is `completed`
- If `## Deferred Items` contains real items, `### Follow-Up Issues` is not
  `NONE`
- Every `#### Step Acceptance Criteria` section that is present is fully checked
- No completed step still contains:
  - `PENDING_STEP_EXECUTION`
  - `PENDING_STEP_REVIEW`
- No `PENDING_UNTIL_ARCHIVE` token remains in:
  - `Validation Summary`
  - `Review Summary`
  - `Archive Summary`
  - `Outcome Summary`

### Required Archive Summary Contents

The `## Archive Summary` section must include:

- archived timestamp
- current revision
- PR URL, or an explicit `PR: NONE` marker if the branch has not been
  published yet
- a concise statement of why the plan is ready to wait for merge approval
- a concise merge handoff note stating that the archived plan must be committed
  and pushed before merge approval is final

Do not require a tracked `HEAD` SHA in the archived plan. Archive itself
changes tracked files, so the definitive merge candidate exists only after the
archive commit is created and pushed.

Branch names and commit SHAs may still appear in local artifacts or CLI output
when they help the current agent, but they are not required durable plan data
in v0.1.

Merge actor, merge timestamp, and post-merge notes belong in the PR history or
PR comments after land, not in the archived plan.

## Reopen Behavior

`harness reopen` is a mechanical transition from archived back to active:

- move the file from `docs/plans/archived/` to `docs/plans/active/`
- set `status: active`
- set `lifecycle: executing`
- increment `revision`
- update `updated_at`
- reset archive-only summary sections to their active placeholder tokens

After reopen, the agent may edit plan content directly. If reopened work
materially changes scope or acceptance criteria, the agent should update the
tracked plan and may set `lifecycle: awaiting_plan_approval` before resuming
implementation.

## Lint Expectations

`harness plan lint` should stop with compact targeted errors on:

- missing required frontmatter fields
- invalid `status`, `lifecycle`, or step `Status` values
- non-RFC3339 timestamps
- missing required sections
- missing `### In Scope` / `### Out of Scope`
- acceptance criteria that are not checkboxes
- step-acceptance criteria that are present but not checkboxes
- step headings without the required status line or required subsections
- active plans stored under `archived/` or archived plans stored under
  `active/`
- archived plans with unchecked step-local acceptance criteria
- archived plans with deferred items but no follow-up issue references
- archived plans with completed steps that still contain
  `PENDING_STEP_EXECUTION` or `PENDING_STEP_REVIEW`
- archived plans that still contain `PENDING_UNTIL_ARCHIVE`
- archived plans with unchecked acceptance criteria or incomplete step status
