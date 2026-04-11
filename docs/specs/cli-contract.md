# CLI Contract

## Purpose

`easyharness` is a CLI for agents first. The command surface should help an
agent decide what to do next, not dump long raw logs and force the model to
reconstruct workflow state from scratch. The public project name is
`easyharness`, while the executable name remains `harness`.

This document defines the normative v0.2 CLI contract. The command surface and
JSON envelopes described here assume the canonical-node runtime model from
[State Model](./state-model.md) and the exact transition matrix from
[State Transitions](./state-transitions.md).

The prose in this spec remains normative for command purpose, workflow intent,
and compatibility boundaries. Field-level JSON structure for the current
command outputs and inputs lives in the checked-in schema registry at
[`schema/index.json`](../../schema/index.json), sourced from the Go-owned
contract module under `internal/contracts` and explained at
[Contract Registry](./contract.md).

Bootstrap resource semantics, ownership rules, and support boundaries live in
[Bootstrap Install](./bootstrap-install.md). This CLI contract references that
spec instead of duplicating the bootstrap details here.

## Command Surface

The current command surface is:

- `harness plan template`
- `harness plan lint`
- `harness init`
- `harness skills install`
- `harness skills uninstall`
- `harness instructions install`
- `harness instructions uninstall`
- `harness execute start`
- `harness evidence submit`
- `harness status`
- `harness ui`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness archive`
- `harness reopen --mode <finalize-fix|new-step>`
- `harness land --pr <url> [--commit <sha>]`
- `harness land complete`

The root CLI also exposes one debug-oriented flag outside that stateful
workflow surface:

- `harness --version`

## Design Principles

### Agent-Friendly by Default

Stateful commands must return:

- a concise summary
- the current durable state
- the current step when it can be inferred
- key artifact paths or identifiers when they are useful
- recommended `next_actions`

They must not default to dumping long raw logs to stdout.

### JSON-First for Stateful Commands

Commands that inspect or mutate workflow state should default to a stable JSON
envelope.

Raw command output, subprocess logs, and verbose diagnostics belong behind an
explicit verbose or debug mode.

Commands that primarily render content, such as `harness plan template`, may
default to markdown or plain text instead of the JSON envelope.

`harness --version` is also a plain-text exception because it is a binary
identity/debug probe rather than a workflow-state command.

`harness ui` is another plain-text exception because it starts a local
read-only workbench server rather than returning a workflow-state JSON
envelope.

The bootstrap commands described in [Bootstrap Install](./bootstrap-install.md)
are JSON-first, but they may omit workflow `state` because they manage
bootstrap assets rather than the tracked plan lifecycle.

The lifecycle commands `harness execute start`, `harness archive`,
`harness reopen`, `harness land`, and `harness land complete` are not special
legacy exceptions. They should use the same v0.2 envelope vocabulary as
`harness status`, centered on `state.current_node` plus concise `facts`,
transition-relevant `artifacts`, and stable `next_actions`.

### Help Must Be Actionable

Every command must have complete `--help` text that explains:

- what the command is for
- required inputs
- key side effects
- common next steps

Skills may refer to `harness --help` or `harness <subcommand> --help`, but the
CLI should remain understandable without repository-specific prompt text.

### Crash-Safe Runstate Writes

Commands that rewrite CLI-owned JSON runstate must protect those files against
interrupted or overlapping writes.

- write `.local/harness/current-plan.json` and any plan-local `state.json`
  through atomic replacement in the destination directory
- acquire a shared per-plan state-mutation lock before loading and rewriting
  `.local/harness/plans/<plan-stem>/state.json`
- fail with a clear contention error when that state lock is already held
  instead of waiting silently or risking a stale overwrite

## Shared Output Envelope

Stateful commands share a common JSON envelope vocabulary, but not every
stateful command returns every field. Commands that report workflow position
should return an envelope shaped like:

```json
{
  "ok": true,
  "command": "status",
  "summary": "Plan is executing Step 3 and nothing is currently blocking continued work.",
  "state": {
    "current_node": "execution/step-3/implement"
  },
  "facts": {
    "current_step": "Step 3: Implement local state and harness status"
  },
  "artifacts": {
    "plan_path": "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md"
  },
  "next_actions": [
    {
      "command": null,
      "description": "Continue Step 3 implementation; no blocking review or CI artifact is currently active."
    },
    {
      "command": "harness review aggregate --round <round-id>",
      "description": "Run this once all reviewer submissions for the current round are present."
    }
  ],
  "warnings": []
}
```

### Required Fields

- `ok`
- `command`
- `summary`
- `next_actions`

`state` is required for commands that report workflow position, such as
`harness status`. Commands whose job is bootstrap, review-orchestration
artifacts, or append-only evidence recording may omit `state` when they do not
need to report a workflow-position payload.

### Common Optional Fields

- `artifacts`
- `blockers`
- `warnings`
- `errors`

When present, `state` should describe post-command state for mutating commands
and current state for read-only stateful commands.

`artifacts` is optional and command-specific. Omit it when there are no stable
artifact paths or IDs worth returning.

`plan_path` may point to a tracked active plan under `docs/plans/active/`, a
tracked standard archive under `docs/plans/archived/`, or a lightweight local
archive under `.local/harness/plans/archived/<plan-stem>.md`.

When a matching `supplements/<plan-stem>/` directory exists for that markdown
plan, commands may also surface it through command-specific `artifacts`
without changing the markdown path's role as the primary plan handle.

`next_actions` should be short, concrete, non-empty, and ordered from the most
likely next step to less common alternatives.

For `harness status` specifically:

- use `next_actions` for ordinary workflow guidance such as continuing the
  current step, starting routine review, or aggregating the active round
- use `warnings` for recoverable ambiguity or workflow-discipline reminders
  that should not by themselves change `state.current_node`
- avoid heuristic warnings for "the current slice may now be reviewable"; keep
  that kind of prompt in ordinary `next_actions`

## Status State Contract

The v0.2 CLI uses one canonical runtime state field:

- `state.current_node`
  - required for stateful commands that report workflow position
  - examples: `plan`, `execution/step-2/implement`,
    `execution/finalize/publish`, `land`, `idle`

`facts` is optional and should carry only selected, high-signal fields that
help explain the node:

- `current_step`
- `revision`
- `reopen_mode`
- `review_kind`
- `review_trigger`
  - optional derived label such as `step_closeout` or `pre_archive`
- `review_title`
  - optional derived human-readable review title
- `review_status`
- `archive_blocker_count`
- `publish_status`
- `pr_url`
- `ci_status`
- `sync_status`
- `land_pr_url`
- `land_commit`

`artifacts` may include stable pointers such as:

- `plan_path`
- `supplements_path`
- `local_state_path`
- `review_round_id`
- latest evidence record IDs
- last-landed context

Legacy v0.1 fields are not part of the `harness status` contract and must not
be emitted by `harness status`:

- `plan_status`
- `lifecycle`
- `step`
- `step_state`
- `handoff_state`
- `worktree_state`

When a current step exists, `harness status` should infer it from the first
unfinished plan step and return it as `facts.current_step`.
When the current plan is `lightweight`, status should also surface the
repo-visible breadcrumb requirement through the summary or `next_actions`
before the candidate is treated as ready to wait for merge approval.

## Command Contracts

### Bootstrap Commands

Purpose:

- bootstrap or refresh repo/user instructions and skill packages without
  mutating tracked plan lifecycle state

Contract:

- `harness init` is the quick-start repo bootstrap entrypoint
- `harness skills ...` and `harness instructions ...` provide the granular
  resource commands
- bootstrap resource semantics, ownership/version rules, and support boundaries
  are defined in [Bootstrap Install](./bootstrap-install.md)
- bootstrap commands share a JSON result envelope documented by the checked-in
  contract registry and may omit workflow `state`

Recommended next action:

- rerun without `--dry-run` to apply a previewed bootstrap change
- open the target instructions file or skills directory to review the installed
  contract

### `harness ui`

Purpose:

- start the local read-only harness UI workbench for the current repository

Contract:

- keep the UI read-only; it must not mutate harness state or trigger workflow
  commands directly in the first slice
- serve a self-contained local web application from the `harness` binary
  rather than requiring a separate frontend runtime at execution time
- support a small local-server flag surface:
  - `--host`
  - `--port`
  - `--no-open`
- default to binding a local interface and opening the browser automatically
  unless `--no-open` is set
- expose a resource-first read-only API surface for the embedded app instead
  of introducing a second hidden product-only state store
- render the current worktree's harness state through the UI more richly; it
  must not fork a competing interpretation of workflow state from the CLI
- expose `Status` and `Timeline` through real read-only resources while
  keeping `Review`, `Diff`, and `Files` as explicit WIP placeholders until
  their data surfaces are implemented
- keep timeline data grounded in command-owned runtime artifacts rather than
  reconstructing history from ad hoc client-side state

Recommended next action:

- open the local UI in a browser and inspect the current `Status` view
- use `--no-open` or a fixed `--port` when debugging or automating the local
  server

### `harness plan template`

Purpose:

- render the canonical plan template with seeded metadata

Contract:

- use [the packaged template asset](../../assets/templates/plan-template.md) as
  the canonical template source
- print the rendered template to stdout by default
- optionally support writing directly to a target path
- support a lightweight authoring mode such as `--lightweight`
- support enough parameters to seed title, date, source metadata, and the
  required `size` field
- when only a date is provided, preserve the current local time-of-day on that
  date instead of silently forcing `created_at` to local midnight
- seed `template_version` from the packaged asset so generated plans record the
  schema/template version they started from
- avoid introducing a second handwritten template source of truth inside code
- when the caller does not provide a size, keep the required `size` field
  explicit in the rendered template rather than silently defaulting it behind
  the author's back
- in lightweight mode, seed `workflow_profile: lightweight`, a shorter
  single-step low-risk authoring shape, and guidance that the active plan
  still lives under `docs/plans/active/` while the archive goes to the local
  lightweight archive path
- lightweight authoring must only be available for `size: XXS`; command UX may
  enforce that either by requiring an explicit `XXS` size or by rendering the
  lightweight template with explicit `size: XXS`
- in standard mode, preserve current behavior when `workflow_profile` is
  omitted

The template asset belongs to the harness version, not to the user's tracked
plan history. Upgrading the harness may upgrade the generated template for new
plans without rewriting historical plans already checked into the repository.

For a Go implementation, the template should be embedded into the binary rather
than loaded from the user's current working directory at runtime. The source
file may live under `assets/`, `internal/templates/`, or a similar package-local
path in the harness source tree, but the built CLI should not depend on that
source path existing in the consumer repository.

One straightforward Go layout would be:

- `internal/templates/`
  - holds the canonical template source file
- `internal/templates/embed.go`
  - exposes an embedded `fs.FS` or string via `//go:embed`
- `internal/plan/`
  - owns rendering and linting logic against that embedded asset

Recommended next action:

- edit the generated plan content
- run `harness plan lint`

### `harness plan lint`

Purpose:

- validate a plan against the plan schema

Contract:

- stop with targeted structural errors instead of guessing or silently fixing
  invalid plan data
- report issues in a compact machine-readable form
- distinguish active-plan errors from archived-plan errors
- validate path/profile compatibility for tracked active plans, tracked
  standard archives, and lightweight local archived plans
- validate supported `template_version` values without invalidating older
  historical plans created by earlier harness versions
- reject malformed plan filenames and malformed `### Step N: ...` headings

Recommended next action:

- fix the listed fields or sections
- rerun lint

### `harness status`

Purpose:

- summarize the current plan and local execution state in the current worktree

This is the primary resume and handoff command. Another agent, a compacted
session, or a human should be able to run `harness status` and quickly
understand what is happening now and what to do next.

Contract:

- detect the current plan artifact, whether it is a tracked active plan, a
  tracked standard archive, or a lightweight local archive
- resolve the canonical `state.current_node` from the current plan,
  execute-start milestones, review artifacts, append-only evidence, reopen
  milestones, archive state, and land milestones
- return pure v0.2 JSON centered on `state.current_node`, selected `facts`,
  `artifacts`, `summary`, and `next_actions`
- never emit legacy v0.1 fields such as `lifecycle`, `step_state`, or
  `handoff_state`
- surface aggregated review failures as a concrete repair signal rather than
  falling back to a generic step summary
- if review metadata cannot be recovered safely, degrade conservatively rather
  than writing a fallback cache answer into local state
- once all steps and acceptance criteria are complete, surface archive blockers
  early through a structured `blockers` list plus repair-first next actions
  instead of making the controller learn them only from `harness archive`
- surface stale or unknown remote freshness as warnings and next actions rather
  than as a derived state layer
- if no current plan is active but `.local/harness/current-plan.json` records a
  landed candidate, return `state.current_node: idle` with landed context in
  `artifacts`
- when the current plan uses the lightweight profile, remind the controller to
  leave the agreed repo-visible breadcrumb, such as a PR body note explaining
  why the lightweight path was used
- return recommended next actions for both "continue work" and "wait/observe"
  situations
- if an already completed earlier step is missing review-complete closeout,
  keep the current node stable, surface a warning, and put the earliest repair
  guidance first in `next_actions`
- if unreadable historical review metadata cannot be mapped back to a tracked
  step, keep the current node stable, preserve a conservative warning, and
  steer the controller toward repairing artifacts or rerunning the relevant
  step-closeout review instead of silently trusting older clean passes
- treat `state.json` as a control-plane artifact for cross-command runtime
  state, not as a cache of the latest resolved node or evidence pointers

Recommended next action examples:

- continue the current step
- start step-closeout review before marking a completed step done
- update step-local `Execution Notes` or `Review Notes` after a review or
  step closeout
- update the plan if scope changed
- run review aggregation
- refresh remote state if the latest sync evidence is stale or missing
- wait for CI
- archive the plan
- commit, push, and update the PR after archive before waiting for merge
  approval

### `harness review start`

Purpose:

- begin a deterministic review round without embedding runtime-specific agent
  spawning in the CLI

Contract:

- accept an agent-supplied review spec instead of inventing one inside the CLI
  itself
- create a `round_id`
- require a `kind` of either `delta` or `full`
- accept the review spec via a structured input such as `--spec <path>` or
  stdin
- validate and persist the supplied review spec as the round manifest
- normalize each review dimension into a deterministic reviewer slot
- reserve reviewer output paths
- precreate one reviewer-owned folder per slot with a `submission.json`
  skeleton that the reviewer can progressively update during the round
- initialize a dispatch or audit ledger
- when `step` is omitted and the inferred binding would be finalize review,
  reject the request if earlier completed steps still lack review-complete
  closeout and direct the controller toward explicit `step=<i>` repair instead
- when mutating both review artifacts and local state, acquire the review
  mutation lock before the state mutation lock instead of inventing a separate
  acquisition order for this command
- update local `state.json` so `harness status` can surface the active round
- return round metadata plus next actions for the controller agent

The controller agent should only need to know the round ID, review kind,
dimension definitions, and how to invoke the reviewer skill. It should not
need to remember the reviewer-submission storage contract from memory.

`harness review start` is still useful even when the agent provides the review
spec because the CLI owns:

- round ID allocation
- manifest validation and persistence
- deterministic artifact locations
- dispatch and audit bookkeeping
- local-state updates for `harness status`

In this contract, the review spec is the command input. The persisted round
manifest is the CLI-owned output artifact derived from that input plus CLI-owned
fields such as `round_id`, timestamps, and artifact paths.

Canonical input shape:

```json
{
  "kind": "delta",
  "anchor_sha": "<base-commit-sha>",
  "review_title": "Check the completed step for state-machine mistakes and handoff clarity.",
  "dimensions": [
    {
      "name": "correctness",
      "instructions": "Look for state-machine mistakes, stale-state bugs, and missing negative-path tests."
    },
    {
      "name": "agent_ux",
      "instructions": "Check whether command output is concise and helpful for another agent resuming work."
    },
    {
      "name": "docs_consistency",
      "instructions": "Verify the implementation still matches the tracked schema and CLI docs for this slice."
    }
  ]
}
```

Example invocation:

```bash
harness review start --spec /tmp/review-spec.json
```

The command returns JSON describing the created round, persisted manifest path,
owned artifact paths, and next actions for the controller agent.

Review-spec semantics:

- `kind`
  - required
  - enum: `delta` or `full`
- `dimensions`
  - required
  - one reviewer slot per normalized dimension
- `anchor_sha`
  - optional for `full`
  - required for `delta`
  - for `delta`, carries the controller-chosen git commit anchor so the
    persisted manifest records the review starting point durably
- `review_title`
  - optional
  - human-readable review title shown back to the controller and reviewers
- `step`
  - optional 1-based tracked step number
  - usually omitted
  - when present, explicitly binds the round to that tracked step's
    step-closeout review, even if the current execution frontier is already on
    a later step or in finalize repair
  - use this path for earlier-step closeout repair when missing or failed
    closeout evidence needs to be repaired intentionally rather than only
    surfaced as passive warning debt
  - when omitted, `harness review start` infers the binding from workflow state:
    - during `execution/step-<n>/implement`, the round binds to the current step
    - during `execution/finalize/review` or `execution/finalize/fix`, the round binds to finalize review

Agents should not supply structural workflow tags such as `step_closeout` or
`pre_archive`. The CLI owns that inference and persists the bound step or
finalize scope in the round manifest and local state.

Explicit `step` binding intentionally re-enters the targeted step's review
loop. If the controller is already executing `step-k` or finalize work and
starts a repair review for earlier `step-i`, status may report
`execution/step-<i>/review` while the round is in flight and
`execution/step-<i>/implement` after a non-pass aggregate. This is distinct
from passive warning debt, where status may keep the later frontier stable
until the controller explicitly starts a repair review for the earlier step.

Round identifiers should be short and plan-local:

- use `review-<NNN>-<kind>`
- examples: `review-001-delta`, `review-002-full`
- keep precise timestamps in the manifest and aggregate artifacts rather than
  embedding them in the round ID

If `review_title` is omitted, the CLI fills one in:

- step-bound review defaults to the tracked step title
- finalize `full` review defaults to `Full branch candidate before archive`
- finalize `delta` review defaults to `Branch candidate before archive`

If an explicit earlier-step repair review aggregates with findings, the
follow-up work still belongs to the current candidate, but status should pin
the repaired step as current until that step-closeout debt is resolved by a
later clean repair review or explicit `NO_STEP_REVIEW_NEEDED` closeout.

Dimension-specific reviewer instructions belong in the input review spec.
Generic reviewer behavior, such as "inspect the current diff and submit results
through the harness contract," belongs in the reviewer skill or in command
output helpers, not duplicated in every review spec.

Recommended next action:

- launch reviewer subagents using the runtime's native delegation mechanism
  and have each subagent use the reviewer skill or reviewer prompt that owns
  submission details

### `harness review submit`

Purpose:

- record one reviewer result for a specific review round and reviewer slot

This command is primarily for reviewer subagents rather than the main
controller agent.

Contract:

- accept the reviewer payload via `--input <path>` or stdin
- validate that the submission matches an expected slot
- treat top-level `summary` and `findings` as the canonical aggregate fields
- allow extra top-level reviewer worklog fields in the submission payload and
  preserve them in the stored submission artifact
- allow each finding to omit `locations` or provide `locations: []string`
  using lightweight repo-relative anchors in one of these forms:
  - `path/to/file.go`
  - `path/to/file.go#L123`
  - `path/to/file.go#L1-L3`
- store the structured reviewer artifact in the round's owned location
- update the dispatch or audit ledger
- stay on the review-artifact mutation path only; reviewer submission should
  not acquire the plan-local state mutation lock
- return a submission receipt plus clear next actions

Recommended next action:

- on success, report the receipt to the controller agent and end the reviewer
  thread; a runtime may later reopen the same reviewer for a narrow same-slot
  follow-up for the same tracked step or the same finalize review scope in
  the same revision, but only after the earlier submission was verified and
  only when the slot instructions still materially match; immediate closeout
  is the safe default
- on validation failure, fix the reviewer artifact and resubmit

### `harness review aggregate`

Purpose:

- aggregate a review round into a concise decision surface for the controller
  agent

Contract:

- require `--round <round-id>` to select the round
- reject the request unless `--round` matches the current active review round
  for the executing plan; in the v0.1 single-active-round model, `review
  aggregate` is not a historical backfill or repair command for older rounds
- collect reviewer artifacts
- compute blocking and non-blocking findings
- stop with an error when expected reviewer slots are missing or invalid
- ignore preserved extra top-level reviewer worklog fields when computing the
  decision surface
- write an aggregate artifact that captures the review decision surface and
  preserves any finding `locations` verbatim
- when mutating both review artifacts and local state, acquire the review
  mutation lock before the state mutation lock instead of inventing a separate
  acquisition order for this command
- update local `state.json` with the aggregate result, including whether the
  round passed or requested changes
- allow later commands to recover that decision from the round aggregate
  artifact when older local state predates the stored `decision` field
- return next actions that depend on the review kind

Recommended next action:

- for a passing `delta` review, continue the current step or mark the step
  complete, then update the step's `Execution Notes` and `Review Notes`
- for a failing `delta` review, fix the current slice and rerun a delta review
- for a passing `full` review, move toward final CI and archive readiness
- for a failing `full` review, fix findings before archive

## Review Sequencing

The CLI contract should assume this review cadence:

- use `delta` review after a completed plan step or after a narrow follow-up fix
- anchor `delta` review to a real git commit chosen by the controller; the
  latest passed-review anchor should be the default start point for a later
  `delta` review
- allow a `full` review to satisfy step closeout when a narrower review would
  be misleading for that completed step
- use `full` review once all planned work appears complete and the branch looks
  like an archive candidate
- if CI failure or conflict resolution creates a narrow, well-bounded change,
  run a `delta` review on that change
- if CI or conflict repair is broad or invalidates the prior full-review scope,
  rerun `full` review before archive

Archive readiness requires:

- a clean `full` review for the initial archive candidate (`revision: 1`)
- a clean review result for later reopened revisions, where a narrow fix may
  use `delta` review instead of forcing another `full` review
- no unresolved active review round
- no unresolved finalize repair work

Post-archive merge readiness additionally requires:

- publish evidence with a PR URL
- CI good enough or explicit `not_applied`
- sync freshness or explicit `not_applied`

### `harness execute start`

Purpose:

- record the execution-start milestone for the current active plan

Contract:

- require the current plan to be active before recording execution start
- persist the execution-start milestone in plan-local runtime state
- update `.local/harness/current-plan.json` to point at the active tracked plan
- return the shared v0.2 envelope with the post-command
  `state.current_node`, `facts.revision`, transition-relevant `artifacts`, and
  actionable `next_actions`
- avoid emitting a lifecycle-specific state sublanguage separate from
  `current_node`

Recommended next action:

- continue the current tracked step and keep step-local `Execution Notes` and
  `Review Notes` current as implementation proceeds

### `harness archive`

Purpose:

- freeze the current plan locally for merge handoff

Contract:

- validate that the plan is active and archive-ready
- run the shared archive-readiness evaluation before any tracked-file or local
  state write happens so a failing archive attempt leaves the current candidate
  untouched
- assume the plan's durable summary sections have already been written from the
  current plan plus local artifacts, not reconstructed from agent memory
- require finalize review to be satisfied before archive succeeds
- reject archive while earlier completed steps still lack review-complete
  closeout, even if the latest finalize review is clean
- if the plan still contains `## Deferred Items`, require
  `## Outcome Summary > Follow-Up Issues` to be something other than `NONE`
  before allowing archive to succeed
- reject archive when plan-local state still shows unresolved finalize review
  or archive-closeout blockers for the current candidate
- require plan-local review state to retain the latest review decision, or
  recover it from the latest review round's aggregate artifact for older local
  state, so archive can distinguish a failed aggregated review from a passing
  one
- require the pre-archive `Archive Summary` to include structured `PR`,
  `Ready`, and `Merge Handoff` lines
- move the plan from its active path to its archived path:
  - `docs/plans/active/` -> `docs/plans/archived/` for `standard`
  - `docs/plans/active/` ->
    `.local/harness/plans/archived/<plan-stem>.md` for `lightweight`
- when a matching `supplements/<plan-stem>/` directory exists, move it with
  the markdown plan to the corresponding archived root
- for `lightweight`, that archived root is the local snapshot path under
  `.local/harness/plans/archived/supplements/<plan-stem>/`, not tracked git
- update `.local/harness/current-plan.json` to the archived plan path
- keep publish, CI, and sync follow-up out of the archive gate; those belong to
  `execution/finalize/publish`
- return the shared v0.2 envelope with `state.current_node` set to the
  post-archive handoff node, concise `facts`, transition artifacts, and
  actionable `next_actions`
- return next actions that explicitly include the profile-appropriate handoff:
  commit and push the archive move for `standard`, or update the repo-visible
  breadcrumb for `lightweight`

Important note:

- `harness archive` changes tracked files locally for both profiles because the
  active tracked plan is removed from `docs/plans/active/`
- the controller agent should commit and push the archive change before
  treating the candidate as truly waiting for merge approval
- the controller agent should also update the agreed repo-visible breadcrumb
  for `lightweight` before treating the candidate as truly waiting for merge
  approval
- after archive, record publish, CI, and sync observations through
  `harness evidence submit` instead of treating missing evidence as success
- after archive, correctness should not depend on archived supplements still
  being present; anything the repository must keep relying on should already be
  absorbed into formal tracked locations
- PR checks may rerun on that archive commit; if new feedback or check failures
  appear, use `harness reopen --mode <finalize-fix|new-step>`
- merge actor, merge timestamp, and other land-only notes should go to PR
  comments or remote history rather than back into the archived plan
- if deferred items exist, the controller agent should replace `NONE` in
  `Follow-Up Issues` with durable handoff details before archive completes

Recommended next action:

- create or verify durable follow-up notes for deferred work
- commit and push the archived plan for `standard`, or update the repo-visible
  breadcrumb for `lightweight`
- wait for post-archive CI or human merge approval once publish, CI, and sync
  evidence move the candidate into `execution/finalize/await_merge`
- reopen with `harness reopen --mode finalize-fix` for narrow repair or
  `harness reopen --mode new-step` when the invalidation deserves a new step

### `harness reopen`

Purpose:

- restore an archived plan to active execution

Contract:

- move the plan from its archived path back to the matching active path
- when a matching `supplements/<plan-stem>/` directory exists, move it with
  the markdown plan back to the active root
- increment command-owned revision state
- require an explicit mode such as `finalize-fix` or `new-step`
- preserve archive audit history via explicit update-required placeholders
- clear stale review and land control-plane signals from the prior archived
  candidate
- update `.local/harness/current-plan.json` back to the active path
- return the shared v0.2 envelope with the reopened post-command node,
  `facts.revision`, `facts.reopen_mode`, transition artifacts, and actionable
  `next_actions`
- return next actions that help the next agent resume work

Recommended next action:

- review the feedback or remote change that caused reopen
- update plan content if scope or acceptance criteria changed
- continue finalize repair for `finalize-fix`, or add a new unfinished step
  before resuming implementation for `new-step`

### `harness evidence submit`

Purpose:

- record append-only publish, CI, or sync evidence for the current archived
  candidate

Contract:

- require the current plan to already be archived before accepting evidence
- support `--kind <ci|publish|sync>` with JSON payloads documented in
  `--help`
- write a timestamped evidence artifact under
  `.local/harness/plans/<plan-stem>/evidence/<kind>/`
- let later status and land-readiness checks discover the latest evidence
  directly from append-only evidence artifacts instead of storing a pointer in
  `state.json`
- preserve trajectory by never editing older evidence artifacts in place
- accept explicit `not_applied` payloads when a domain truly does not apply

### `harness land --pr <url> [--commit <sha>]`

Purpose:

- record merge confirmation for the current archived candidate and enter land
  for the required post-merge bookkeeping

Contract:

- require the current plan artifact to still be the archived candidate
- require `--pr <url>` and optionally accept `--commit <sha>`
- validate that publish, CI, and sync evidence make the candidate merge-ready
- record merge confirmation in plan-local runtime state
- leave archived plan content untouched; this is a local-state milestone only
- return the shared v0.2 envelope with the `land` post-command node and any
  relevant `land_pr_url` / `land_commit` facts
- return next actions that guide the required post-merge bookkeeping

### `harness land complete`

Purpose:

- record required post-merge bookkeeping completion and restore idle worktree
  state

Contract:

- require prior `harness land --pr <url>` for the same archived candidate
- persist local completion metadata in plan-local runtime state
- rewrite `.local/harness/current-plan.json` so `plan_path` is cleared
- record `last_landed_plan_path` and `last_landed_at` for worktree handoff
- leave archived plan content untouched; this is local-state cleanup only
- return the shared v0.2 envelope with `state.current_node: idle`,
  `facts.revision`, and the artifact pointers needed for handoff confirmation
- return next actions that guide the worktree back to idle or on to the next
  slice

### `harness --version`

Purpose:

- print concise debug information for the running harness binary

Contract:

- remain a root-level flag rather than a workflow subcommand
- print plain text rather than the shared JSON envelope
- report the running binary's build commit
- report whether the binary is running in `dev` or `release` mode
- print the resolved binary path only in `dev` mode

## Review Runtime Boundary

The CLI does not own subagent spawning.

The controller agent decides how to launch reviewer subagents. In Codex, that
means using `spawn_agent` rather than trying to do reviewer work in the main
agent thread. The reviewer skill or reviewer prompt should own the details of
calling `harness review submit`.

Codex should still default to closing reviewer agents after each verified
submission. If a later narrow follow-up round keeps the same slot and
materially the same instructions for the same tracked step or the same
finalize review scope in the same revision, the controller may reopen that
previously closed reviewer with `resume_agent` instead of spawning fresh.
Moving to a different tracked step, moving from step review to finalize
review, changing the review scope because of reopen or a new revision, broad
follow-up, changed slot ownership, invalid earlier submissions, or any
situation where a clean reread is safer should stay on fresh `spawn_agent`
reviewer threads.

The CLI only owns deterministic local contracts:

- manifest persistence
- output paths
- submission validation
- aggregation
- audit trail

## Deferred Commands

No additional user-facing command is committed in this spec yet beyond the
surface listed above.
