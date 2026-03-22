# Plan Schema

## Purpose

This document defines the normative v0.2 tracked-plan contract for
`superharness`.

In v0.2, tracked plans keep durable scope, step closeout, and archive-time
summaries. Runtime lifecycle, milestone timestamps, review rounds, evidence
history, and resolved node state live in `.local/harness/`.

## Directory Layout

Tracked plans live in:

- `docs/plans/active/`
- `docs/plans/archived/`

Command-owned local artifacts live under:

- `.local/harness/current-plan.json`
- `.local/harness/plans/<plan-stem>/state.json`
- `.local/harness/plans/<plan-stem>/reviews/`
- `.local/harness/plans/<plan-stem>/evidence/`

The tracked plan is the durable contract. `.local` is disposable execution
support and trajectory.

## Source of Truth

The source-of-truth split is:

- this schema document defines the tracked-plan contract
- [the packaged plan template asset](../../assets/templates/plan-template.md)
  is the canonical authoring example shipped by harness
- `harness plan template` is a convenience wrapper around the packaged asset

Skills and agent prompts should point operators back to this schema, the state
model, and CLI help instead of duplicating the contract.

## File Naming

Each plan file name is its durable identifier.

Required pattern:

- `YYYY-MM-DD-short-topic.md`

Where:

- `YYYY-MM-DD` is the creation date
- `short-topic` is a compact kebab-case topic slug

The file stem is the durable identifier used by command-owned local state:

- `.local/harness/plans/<plan-stem>/...`

## Frontmatter

Every plan must start with YAML frontmatter containing exactly these durable
fields:

```yaml
---
template_version: 0.2.0
created_at: 2026-03-17T10:12:01+08:00
source_type: direct_request
source_refs: []
---
```

### Required Fields

- `template_version`
  - semver-like version for the tracked-plan schema/template
  - older historical versions remain lint-valid if the current harness still
    knows how to validate them
  - newer versions than the current harness understands must be rejected
- `created_at`
  - RFC3339 timestamp with offset
- `source_type`
  - short lower-snake-case or kebab-case intake label
  - examples: `direct_request`, `issue`, `backlog`, `incident`, `other`
- `source_refs`
  - array of external references such as issue IDs or URLs
  - use `[]` when there are none

### Removed v0.1 Runtime Fields

v0.2 tracked plans must not carry these top-level runtime fields:

- `status`
- `lifecycle`
- `revision`
- `updated_at`

Path placement already answers active vs archived. Runtime milestones and
revision history belong in command-owned artifacts, not tracked frontmatter.

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

`## Scope` must include:

- `### In Scope`
- `### Out of Scope`

`## Outcome Summary` must include, in order:

- `### Delivered`
- `### Not Delivered`
- `### Follow-Up Issues`

## Acceptance Criteria

- every acceptance criterion must be a markdown checkbox
- active plans may mix checked and unchecked criteria
- archived plans must have every acceptance criterion checked

## Work Breakdown

The work breakdown uses step headings:

```md
### Step 1: Define the contract
```

Every step must begin with a durable completion marker directly under the step
heading:

```md
- Done: [ ]
```

or

```md
- Done: [x]
```

Legacy `- Status: ...` lines are no longer part of the v0.2 tracked-plan
contract and must be rejected by lint.

Every step must also contain:

- `#### Objective`
- `#### Details`
- `#### Expected Files`
- `#### Validation`
- `#### Execution Notes`
- `#### Review Notes`

Optional step section:

- `#### Step Acceptance Criteria`

Rules:

- `Objective` should stay concise
- longer planning detail belongs in `Details`
- if no extra detail is needed, write `NONE` in `Details`
- `Execution Notes` and `Review Notes` carry durable closeout history as work
  progresses
- if `Step Acceptance Criteria` exists, every entry must be a markdown checkbox
- archived plans require all step-local acceptance checkboxes to be checked

## Placeholder Policy

These exact active-plan placeholders are allowed:

- `Execution Notes`: `PENDING_STEP_EXECUTION`
- `Review Notes`: `PENDING_STEP_REVIEW`
- `Validation Summary`: `PENDING_UNTIL_ARCHIVE`
- `Review Summary`: `PENDING_UNTIL_ARCHIVE`
- `Archive Summary`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Delivered`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Not Delivered`: `PENDING_UNTIL_ARCHIVE`
- `Outcome Summary > Follow-Up Issues`: `NONE`

Archive must reject any plan that still contains `PENDING_UNTIL_ARCHIVE`.

Archived plans must not leave `PENDING_STEP_EXECUTION` or
`PENDING_STEP_REVIEW` in any completed step.

## Reopen Update Markers

`harness reopen` must preserve prior archive-time wording instead of blanking
it out.

The reopen marker token is:

- `UPDATE_REQUIRED_AFTER_REOPEN`

Usage rules:

- reopen-sensitive sections keep their prior archived content
- the controller or CLI prepends `UPDATE_REQUIRED_AFTER_REOPEN` to:
  - `Validation Summary`
  - `Review Summary`
  - `Archive Summary`
  - every `Outcome Summary` subsection
- active reopened plans may temporarily contain this marker
- archive must reject any plan that still contains this marker

This makes it obvious that the plan was once archived and that the archived
summary now needs refresh before the next archive.

## Deferred Items and Follow-Up Tracking

Use these two surfaces deliberately:

- `## Deferred Items`
  - work or risk deliberately left out of the current slice
- `## Outcome Summary > Follow-Up Issues`
  - the durable handoff note recorded at archive time for deferred items that
    remain intentionally out of scope

Archive readiness rules:

- if `Deferred Items` still contains real items at archive time, `Follow-Up
  Issues` must not remain `NONE`
- if there are no deferred items and no follow-up, `Follow-Up Issues` may stay
  `NONE`

## Active Plan Rules

An active plan must satisfy all of these:

- the file lives under `docs/plans/active/`
- the required frontmatter fields are present
- every step uses a `Done` marker
- archive-only summary sections may still contain the documented active-plan
  placeholders
- reopen update markers are allowed only while the plan is active again after
  a reopen

## Archived Plan Rules

An archived plan must satisfy all of these:

- the file lives under `docs/plans/archived/`
- every acceptance criterion is checked
- every step is `Done: [x]`
- every step-local acceptance checkbox is checked when present
- no completed step still contains:
  - `PENDING_STEP_EXECUTION`
  - `PENDING_STEP_REVIEW`
- no archive-time placeholder token remains:
  - `PENDING_UNTIL_ARCHIVE`
  - `UPDATE_REQUIRED_AFTER_REOPEN`
- if `Deferred Items` contains real items, `Follow-Up Issues` must not be
  `NONE`

### Required Archive Summary Contents

The `Archive Summary` section must include:

- `- Archived At: <RFC3339 timestamp>`
- `- Revision: <current revision>`
- `- PR: <URL or NONE>`
- `- Ready: <why this candidate is ready to wait for merge approval>`
- `- Merge Handoff: <handoff note for the archived candidate>`

`Revision` is command-owned runtime history that must be stamped into the
tracked archive summary. It is no longer tracked as frontmatter.

## Reopen Behavior

`harness reopen --mode <...>` is a mechanical transition from archived back to
active:

- move the file from `docs/plans/archived/` to `docs/plans/active/`
- preserve prior archive-time wording
- prepend `UPDATE_REQUIRED_AFTER_REOPEN` markers to reopen-sensitive summaries
- clear stale review and evidence facts that belonged to the invalidated
  archived candidate
- increment the command-owned revision
- update current-plan and plan-local state pointers back to the active path

Mode-specific rules:

- `finalize-fix`
  - reopened work remains finalize-scope repair
- `new-step`
  - reopened work must be represented by a new unfinished step
  - the controller adds that new step after reopen rather than editing old
    completed steps

## Lint Expectations

`harness plan lint` must stop with compact targeted errors on:

- missing required frontmatter fields
- legacy frontmatter runtime fields that are no longer allowed
- invalid or unsupported `template_version`
- non-RFC3339 `created_at`
- missing required sections or wrong section order
- missing `### In Scope` / `### Out of Scope`
- acceptance criteria or step-local acceptance criteria that are not checkboxes
- missing step `Done` markers or legacy `Status:` lines
- missing required step subsections
- plans stored outside `docs/plans/active/` or `docs/plans/archived/`
- archived plans with unchecked acceptance criteria, incomplete steps, or
  unchecked step-local acceptance criteria
- archived plans that still contain active-plan or reopen update placeholders
- archived plans with deferred items but `Follow-Up Issues: NONE`
