# CLI Contract

## Purpose

`superharness` is a CLI for agents first. The command surface should help an
agent decide what to do next, not dump long raw logs and force the model to
reconstruct workflow state from scratch.

This document defines the normative v0.2 CLI contract. The command surface and
JSON envelopes described here assume the canonical-node runtime model from
[State Model](./state-model.md) and the exact transition matrix from
[State Transitions](./state-transitions.md).

## Command Surface

The current command surface is:

- `harness plan template`
- `harness plan lint`
- `harness execute start`
- `harness evidence submit`
- `harness status`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness archive`
- `harness reopen --mode <finalize-fix|new-step>`
- `harness land --pr <url> [--commit <sha>]`
- `harness land complete`

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

### Help Must Be Actionable

Every command must have complete `--help` text that explains:

- what the command is for
- required inputs
- key side effects
- common next steps

Skills may refer to `harness --help` or `harness <subcommand> --help`, but the
CLI should remain understandable without repository-specific prompt text.

## Shared Output Envelope

Stateful commands should return an envelope shaped like:

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
    "plan_path": "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md"
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
- `state`
- `next_actions`

### Common Optional Fields

- `artifacts`
- `blockers`
- `warnings`
- `errors`

`state` should describe post-command state for mutating commands and current
state for read-only stateful commands.

`artifacts` is optional and command-specific. Omit it when there are no stable
artifact paths or IDs worth returning.

`next_actions` should be short, concrete, non-empty, and ordered from the most
likely next step to less common alternatives.

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
- `review_target`
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
- `local_state_path`
- `review_round_id`
- latest evidence record IDs
- last-landed context

Legacy v0.1 fields are not part of the contract and must not be emitted:

- `plan_status`
- `lifecycle`
- `step`
- `step_state`
- `handoff_state`
- `worktree_state`

When a current step exists, `harness status` should infer it from the first
unfinished tracked step and return it as `facts.current_step`.

## Command Contracts

### `harness plan template`

Purpose:

- render the canonical tracked plan template with seeded metadata

Contract:

- use [the packaged template asset](../../assets/templates/plan-template.md) as
  the canonical template source
- print the rendered template to stdout by default
- optionally support writing directly to a target path
- support enough parameters to seed title, date, and source metadata
- seed `template_version` from the packaged asset so generated plans record the
  schema/template version they started from
- avoid introducing a second handwritten template source of truth inside code

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

- validate a plan against the tracked-plan schema

Contract:

- stop with targeted structural errors instead of guessing or silently fixing
  invalid plan data
- report issues in a compact machine-readable form
- distinguish active-plan errors from archived-plan errors
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

- detect the current tracked plan
- resolve the canonical `state.current_node` from the tracked plan,
  execute-start milestones, review artifacts, append-only evidence, reopen
  milestones, archive state, and land milestones
- return pure v0.2 JSON centered on `state.current_node`, selected `facts`,
  `artifacts`, `summary`, and `next_actions`
- never emit legacy v0.1 fields such as `lifecycle`, `step_state`, or
  `handoff_state`
- surface aggregated review failures as a concrete repair signal rather than
  falling back to a generic step summary
- if review metadata cannot be recovered safely, degrade conservatively and do
  not refresh the cached `current_node`
- once all steps and acceptance criteria are complete, surface archive blockers
  early through a structured `blockers` list plus repair-first next actions
  instead of making the controller learn them only from `harness archive`
- surface stale or unknown remote freshness as warnings and next actions rather
  than as a derived state layer
- if no current plan is active but `.local/harness/current-plan.json` records a
  landed candidate, return `state.current_node: idle` with landed context in
  `artifacts`
- return recommended next actions for both "continue work" and "wait/observe"
  situations

Recommended next action examples:

- continue the current step
- update step-local `Execution Notes` or `Review Notes` after a delta review or
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
- initialize a dispatch or audit ledger
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
  "target": "Step 3: Implement local state and harness status",
  "trigger": "step_closeout",
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

Round identifiers should be short and plan-local:

- use `review-<NNN>-<kind>`
- examples: `review-001-delta`, `review-002-full`
- keep precise timestamps in the manifest and aggregate artifacts rather than
  embedding them in the round ID

`target` should be free-form and human-readable. Examples:

- delta after a step: `Step 3: Implement local state and harness status`
- full pre-archive review: `Full branch candidate before archive`
- follow-up after human feedback: `Changes addressing human comments on archive summary`

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
- store the structured reviewer artifact in the round's owned location
- update the dispatch or audit ledger
- return a submission receipt plus clear next actions

Recommended next action:

- on success, report the receipt to the controller agent and end the reviewer
  thread unless asked to keep working
- on validation failure, fix the reviewer artifact and resubmit

### `harness review aggregate`

Purpose:

- aggregate a review round into a concise decision surface for the controller
  agent

Contract:

- require `--round <round-id>` to select the round
- collect reviewer artifacts
- compute blocking and non-blocking findings
- stop with an error when expected reviewer slots are missing or invalid
- write an aggregate artifact that captures the review decision surface
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

### `harness archive`

Purpose:

- freeze the tracked plan locally for merge handoff

Contract:

- validate that the plan is active and archive-ready
- run the shared archive-readiness evaluation before any tracked-file or local
  state write happens so a failing archive attempt leaves the current candidate
  untouched
- assume the plan's durable summary sections have already been written from the
  current plan plus local artifacts, not reconstructed from agent memory
- require finalize review to be satisfied before archive succeeds
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
- move the plan from `docs/plans/active/` to `docs/plans/archived/`
- update `.local/harness/current-plan.json` and any existing plan-local
  `state.json` pointers to the archived path
- keep publish, CI, and sync follow-up out of the archive gate; those belong to
  `execution/finalize/publish`
- return next actions that explicitly include committing and pushing the archive
  change

Important note:

- `harness archive` changes tracked files locally
- the controller agent should commit and push the archive move before treating
  the plan as truly waiting for merge approval
- after archive, record publish, CI, and sync observations through
  `harness evidence submit` instead of treating missing evidence as success
- PR checks may rerun on that archive commit; if new feedback or check failures
  appear, use `harness reopen --mode <finalize-fix|new-step>`
- merge actor, merge timestamp, and other land-only notes should go to PR
  comments or remote history rather than back into the archived plan
- if deferred items exist, the controller agent should replace `NONE` in
  `Follow-Up Issues` with durable handoff details before archive completes

Recommended next action:

- create or verify durable follow-up notes for deferred work
- commit and push the archived plan
- update the PR if needed
- wait for post-archive CI or human merge approval once publish, CI, and sync
  evidence move the candidate into `execution/finalize/await_merge`
- reopen with `harness reopen --mode finalize-fix` for narrow repair or
  `harness reopen --mode new-step` when the invalidation deserves a new step

### `harness reopen`

Purpose:

- restore an archived plan to active execution

Contract:

- move the plan from `docs/plans/archived/` back to `docs/plans/active/`
- increment command-owned revision state
- require an explicit mode such as `finalize-fix` or `new-step`
- preserve archive audit history via explicit update-required placeholders
- clear stale review, evidence, and handoff cache signals from the prior
  archived candidate
- update `.local/harness/current-plan.json` and any existing plan-local
  `state.json` pointers back to the active path
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

- require the current tracked plan to be archived before accepting evidence
- support `--kind <ci|publish|sync>` with JSON payloads documented in
  `--help`
- write a timestamped evidence artifact under
  `.local/harness/plans/<plan-stem>/evidence/<kind>/`
- update the thin `state.json` cache with the latest pointer for that kind
- preserve trajectory by never editing older evidence artifacts in place
- accept explicit `not_applied` payloads when a domain truly does not apply

### `harness land --pr <url> [--commit <sha>]`

Purpose:

- record merge confirmation for the current archived candidate and enter land
  cleanup

Contract:

- require the current tracked plan to still be the archived candidate
- require `--pr <url>` and optionally accept `--commit <sha>`
- validate that publish, CI, and sync evidence make the candidate merge-ready
- record merge confirmation in plan-local runtime state
- leave tracked plans untouched; this is a local-state milestone only
- return next actions that guide post-merge cleanup

### `harness land complete`

Purpose:

- record post-merge cleanup completion and restore idle worktree state

Contract:

- require prior `harness land --pr <url>` for the same archived candidate
- persist local completion metadata in plan-local runtime state
- rewrite `.local/harness/current-plan.json` so `plan_path` is cleared
- record `last_landed_plan_path` and `last_landed_at` for worktree handoff
- leave tracked plans untouched; this is local-state cleanup only
- return next actions that guide the worktree back to idle or on to the next
  slice

## Review Runtime Boundary

The CLI does not own subagent spawning.

The controller agent decides how to launch reviewer subagents. In Codex, that
means using `spawn_agent` rather than trying to do reviewer work in the main
agent thread. The reviewer skill or reviewer prompt should own the details of
calling `harness review submit`.

The CLI only owns deterministic local contracts:

- manifest persistence
- output paths
- submission validation
- aggregation
- audit trail

## Deferred Commands

No additional user-facing command is committed in this spec yet beyond the
surface listed above.
