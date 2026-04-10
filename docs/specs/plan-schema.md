# Plan Schema

## Purpose

This document defines the normative v0.2 plan contract for `easyharness`.

In v0.2, standard and lightweight plans share one markdown schema, but the
durable planning unit is a markdown-led plan package rather than a lone file.
Active plans for both profiles live in tracked markdown under
`docs/plans/active/`, and may carry approval-scoped companion material under a
matching `supplements/<plan-stem>/` directory. Lightweight diverges only at
archive time, when the archived snapshot moves into command-owned local
storage. Runtime lifecycle, milestone timestamps, review rounds, evidence
history, and resolved node state live in `.local/harness/`.

## Directory Layout

Tracked plan locations live in:

- `docs/plans/active/`
- `docs/plans/archived/`

Tracked plan package companion locations live in:

- `docs/plans/active/supplements/<plan-stem>/`
- `docs/plans/archived/supplements/<plan-stem>/`

Lightweight local archived snapshots live in:

- `.local/harness/plans/archived/<plan-stem>.md`
- `.local/harness/plans/archived/supplements/<plan-stem>/`

There is no local active lightweight plan path in v0.2. Both `standard` and
`lightweight` active plans live under `docs/plans/active/`.

The markdown file remains the canonical plan path for status resolution,
current-plan pointers, and most command artifacts. When a matching
`supplements/<plan-stem>/` directory exists, it is part of the same approved
plan package and must move with the markdown plan during archive and reopen.

Command-owned local artifacts live under:

- `.local/harness/current-plan.json`
- `.local/harness/plans/<plan-stem>/state.json`
- `.local/harness/plans/<plan-stem>/reviews/`
- `.local/harness/plans/<plan-stem>/evidence/`

Tracked plans remain durable repository history for active work and standard
archives. Their optional `supplements/` directories are execution input during
active work and cold-backup context after archive, not a durable dependency
surface that later execution should keep relying on. Before archive, any
normative or reusable material that the repository must still depend on should
be absorbed into formal durable locations such as `docs/specs/`, code, tests,
or other tracked docs. The markdown plan remains the default reading
entrypoint. Lightweight archived snapshots are command-owned local execution
artifacts. `.local` is still disposable execution support and trajectory;
lightweight workflow use must therefore leave a small repo-visible breadcrumb
outside the local archive path, as defined by the CLI and agent guidance
contracts.

## Source of Truth

The source-of-truth split is:

- this schema document defines the shared plan contract for standard and
  lightweight profiles
- [the packaged plan template asset](../../assets/templates/plan-template.md)
  is the canonical authoring example shipped by harness
- `harness plan template` is a convenience wrapper around the packaged asset

Skills and agent prompts should point operators back to this schema, the state
model, and CLI help instead of duplicating the contract.

## Plan Package Semantics

Each tracked or archived markdown plan may own an optional companion directory
at `supplements/<plan-stem>/` under the same active or archived root.

Rules:

- approval covers the markdown plan and its matching supplements directory as
  one plan package
- bulky but durable execution detail that should survive context compaction may
  live in supplements, such as spec drafts, formulas, design notes, or other
  structured reasoning that is too large or awkward for the main markdown
- supplements are a staging area for approved execution detail, not the final
  durable home for repository-facing normative content; before archive, anything
  the repository should keep depending on must be absorbed into formal tracked
  locations
- supplements are not free-form scratch space; they share the same governance
  boundary as the markdown plan
- the markdown file stays concise and remains the main review and archive
  entrypoint even when supplements exist
- `harness plan lint` validates the markdown file directly and also rejects
  invalid companion placement, such as a plan markdown stored under a
  `supplements/` subtree or a supplements path that is not a directory

Execution-time update rules:

- after approval, agents may update execution-facing closeout, review notes,
  and supplement absorption tracking without reopening approval
- agents must not silently change approved intent, scope, acceptance criteria,
  or key design constraints in either the markdown plan or supplements
- if a package change would alter the approved intent, reuse the normal plan
  update or reopen approval boundary instead of drifting the package silently

## File Naming

Each plan file name is its durable identifier.

Required pattern:

- `YYYY-MM-DD-short-topic.md`

Where:

- `YYYY-MM-DD` is the creation date
- `short-topic` is a compact kebab-case topic slug

The file stem is the durable identifier used by command-owned local state:

- `.local/harness/plans/<plan-stem>/...`
- matching `supplements/<plan-stem>/` package directories

## Frontmatter

Every plan must start with YAML frontmatter containing these durable fields:

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
  - for ordinary new plans, reflect the real creation time rather than a
    synthetic midnight placeholder
  - when `harness plan template` is seeded with a date but not an explicit
    timestamp, keep the current local time-of-day on that date
  - historical plans that already carry midnight timestamps remain valid; this
    field is durable history, not a backfilled runtime value
- `source_type`
  - short lower-snake-case or kebab-case intake label
  - examples: `direct_request`, `issue`, `backlog`, `incident`, `other`
- `source_refs`
  - array of external references such as issue IDs or URLs
  - use `[]` when there are none

### Optional Fields

- `workflow_profile`
  - optional explicit workflow selector
  - supported value is `lightweight`
  - omitted means `standard`
  - `standard` must not be written explicitly; omitting the field preserves
    the current standard behavior
  - tracked plans under `docs/plans/active/` may set this field only to
    `lightweight`
  - tracked plans under `docs/plans/archived/` must omit this field
  - local archived plans under `.local/harness/plans/archived/` must set it to
    `lightweight`

## Lightweight Eligibility

`workflow_profile: lightweight` is allowed only when every rule below is true:

- the whole slice is one intentionally narrow low-risk change that can be
  planned, implemented, and validated as a single bounded step
- the expected edits are limited to non-behavioral repository maintenance such
  as README wording, documentation copy, comments, or similarly narrow
  explanatory cleanup
- no `harness` CLI behavior, state resolution rule, review/archive contract,
  persistence behavior, release or CI automation, security-sensitive logic, or
  other user-visible product behavior changes
- if the boundary is unclear, the risk is disputed, or the slice stops looking
  obviously lightweight, it must use `standard`

Examples that may use `lightweight`:

- fixing a broken README link
- correcting narrow documentation wording
- cleaning up comments without changing behavior

Examples that must stay `standard`:

- any change under `docs/specs/` that changes the product contract
- any change to Go code, tests that prove new behavior, or release/CI workflow
  logic
- any change that needs more than one meaningful implementation step or a
  broader review posture than bounded low-risk maintenance

### Removed v0.1 Runtime Fields

v0.2 plans must not carry these top-level runtime fields:

- `status`
- `lifecycle`
- `revision`
- `updated_at`

Path placement plus optional `workflow_profile: lightweight` answer whether a
plan is standard or lightweight and whether it is active or archived. Runtime
milestones and revision history belong in command-owned artifacts, not plan
frontmatter.

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

The markdown plan does not need a dedicated `Supplements` top-level section.
When supplements exist, keep the package discoverable by naming the directory
with the plan stem and by mentioning important supplement absorption in the
existing archive-facing summaries.

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
- `Review Notes` may record `NO_STEP_REVIEW_NEEDED: <reason>` when a completed
  step is too small or low risk to justify a separate step-closeout review
- if `Step Acceptance Criteria` exists, every entry must be a markdown checkbox
- archived plans require all step-local acceptance checkboxes to be checked
- when discovery detail is too bulky for `Details`, move that material into the
  matching `supplements/<plan-stem>/` package directory rather than burying it
  only in chat history

## Placeholder Policy

These exact active-plan placeholders are allowed in active tracked plans for
both profiles:

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
- `## Archive Summary` / `## Outcome Summary`
  - the place to summarize supplement absorption, such as which draft files
    became formal specs, code, tests, or other durable repository artifacts

Archive readiness rules:

- if `Deferred Items` still contains real items at archive time, `Follow-Up
  Issues` must not remain `NONE`
- if there are no deferred items and no follow-up, `Follow-Up Issues` may stay
  `NONE`

## Lightweight Eligibility

`workflow_profile: lightweight` is only for narrow low-risk work that keeps
human steering but does not justify a tracked archived plan artifact.

The lightweight profile is eligible only when all of these are true:

- the human explicitly approves using the lightweight profile
- the plan still describes a small bounded slice that one short plan can steer
- the expected change is limited to low-risk repository surfaces such as:
  - README or docs wording
  - comments or other non-behavioral text cleanup
  - similarly narrow metadata or wording fixes that do not change product
    behavior, state transitions, or command semantics
- the controller can explain the lightweight choice in one small repo-visible
  breadcrumb such as a PR body note
- the plan can stay clear and reviewable without depending on supplements as a
  default authoring pattern

The lightweight profile is not eligible when any of these are true:

- the change affects CLI, runtime, release, review, archive, evidence, or
  state-machine behavior
- the change modifies normative product contracts, schema meaning, or command
  semantics
- the change spans multiple risk areas or would make a reviewer reasonably ask
  for a tracked archive record
- the controller is unsure whether the slice is truly low-risk

Lightweight plans should normally avoid `supplements/`. If a lightweight plan
temporarily needs one, keep it minimal, treat it with the same approval
governance as the markdown plan, and ensure archive writes it only to
`.local/harness/plans/archived/supplements/<plan-stem>/` rather than treating
it as durable tracked history.

When there is any doubt, escalate to the standard tracked-plan workflow.

## Active Plan Rules

An active plan must satisfy all of these:

- the file lives under `docs/plans/active/`
- the file does not live inside `docs/plans/active/supplements/`
- lightweight does not create a separate local active-plan location
- the required frontmatter fields are present
- any optional `workflow_profile` field is compatible with the path:
  - omitted means `standard`
  - explicit `workflow_profile` is allowed only for `lightweight`
- every step uses a `Done` marker
- archive-only summary sections may still contain the documented active-plan
  placeholders
- reopen update markers are allowed only while the plan is active again after
  a reopen
- when a matching `docs/plans/active/supplements/<plan-stem>/` directory
  exists, it is part of the same approved package
- when the active plan uses `workflow_profile: lightweight`, supplements are
  supported but should be exceptional rather than the default way to carry plan
  detail

## Archived Plan Rules

An archived plan must satisfy all of these:

- the file lives under either:
  - `docs/plans/archived/`
  - `.local/harness/plans/archived/`
- any optional `workflow_profile` field is compatible with the path:
  - tracked archived plans must omit `workflow_profile`
  - local archived plans require `workflow_profile: lightweight`
- every acceptance criterion is checked
- the markdown plan does not live inside an archived `supplements/` subtree
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
- when a matching archived `supplements/<plan-stem>/` directory exists, it is
  retained only as cold-backup context rather than becoming the primary reading
  entrypoint or a durable correctness dependency
- archive-time correctness must not depend on archived supplements continuing to
  exist; content that still matters after archive must already be absorbed into
  formal tracked locations outside the supplements tree
- when the archived plan is `lightweight`, any supplements snapshot lives only
  under `.local/harness/plans/archived/supplements/<plan-stem>/` and must not
  be treated as tracked repository history

### Required Archive Summary Contents

The `Archive Summary` section must include:

- `- Archived At: <RFC3339 timestamp>`
- `- Revision: <current revision>`
- `- PR: <URL or NONE>`
- `- Ready: <why this candidate is ready to wait for merge approval>`
- `- Merge Handoff: <handoff note for the archived candidate>`

When supplements existed during execution, the archive-facing summaries should
also note the important absorption result at a human-readable level, such as:

- which supplement drafts became formal specs or code
- which supplement files remain only as archived backup context

Those summaries should make it clear that archive-time correctness no longer
depends on the supplements remaining available verbatim.

`Revision` is command-owned runtime history that must be stamped into the
tracked archive summary. It is no longer tracked as frontmatter.

## Reopen Behavior

`harness reopen --mode <...>` is a mechanical transition from archived back to
active:

- move the file from the archived path back to the corresponding active path:
  - `docs/plans/archived/` -> `docs/plans/active/` for `standard`
  - `.local/harness/plans/archived/` -> `docs/plans/active/` for
    `lightweight`
- move the matching `supplements/<plan-stem>/` directory with the markdown
  plan when it exists
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
  - once that first reopened step exists, later finalize-time repair may update
    the reopened work in place instead of forcing yet another new unfinished
    step by default

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
- unsupported `workflow_profile` values
- plans stored outside:
  - `docs/plans/active/`
  - `docs/plans/archived/`
  - `.local/harness/plans/archived/`
- plan markdown stored under a `supplements/` subtree
- plans whose path and `workflow_profile` disagree
- supplements paths that are present but are not directories
- archived plans with unchecked acceptance criteria, incomplete steps, or
  unchecked step-local acceptance criteria
- archived plans that still contain active-plan or reopen update placeholders
- archived plans with deferred items but `Follow-Up Issues: NONE`
