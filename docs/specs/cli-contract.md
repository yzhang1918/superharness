# CLI Contract

## Purpose

`superharness` is a CLI for agents first. The command surface should help an
agent decide what to do next, not dump long raw logs and force the model to
reconstruct workflow state from scratch.

This document defines the v0.1 CLI contract.

## Command Surface

The initial v0.1 command surface is:

- `harness plan template`
- `harness plan lint`
- `harness status`
- `harness review start`
- `harness review submit`
- `harness review aggregate`
- `harness archive`
- `harness reopen`

Deferred from v0.1:

- `harness ui`

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
    "plan_status": "active",
    "lifecycle": "executing",
    "step": "Step 3: Implement local state and harness status",
    "step_state": "implementing"
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
- `warnings`
- `errors`

`state` should describe post-command state for mutating commands and current
state for read-only stateful commands.

`artifacts` is optional and command-specific. Omit it when there are no stable
artifact paths or IDs worth returning.

`next_actions` should be short, concrete, non-empty, and ordered from the most
likely next step to less common alternatives.

## State Vocabulary

The CLI should use a small, consistent state vocabulary:

- `plan_status`
  - mirrors tracked plan placement and mutability
  - expected values: `active`, `archived`
- `lifecycle`
  - the coarse human-steer workflow stage from the tracked plan
- `step`
  - the current work-breakdown step when one can be inferred
- `step_state`
  - the current operating mode for the current step inside `executing`
  - derived from current plan content plus local artifacts rather than tracked
    directly in markdown

Most humans and agents should read these together as one sentence, for example:

- `plan_status: active`
- `lifecycle: executing`
- `step: Step 3: Implement local state and harness status`
- `step_state: implementing`

In plain language, that means: "the active plan is in the executing lifecycle,
currently focused on Step 3, and the agent is implementing rather than waiting
on review or CI."

### Step Inference

`harness status` should infer `step` from the tracked plan:

- prefer the first `in_progress` step
- otherwise use the first `pending` step
- otherwise omit `step`

### Step-State Rules

`step_state` is primarily useful when `lifecycle: executing`.

If `lifecycle` is not `executing`, omit `step_state` unless a command has a
very strong reason to surface an extra local hint.

`step_state` is not a full state machine in v0.1. It is a best-effort local
hint that answers one question: "What is the agent mainly doing around the
current step right now?"

Do not treat `step_state` values as a strict progression. For example,
conflict-resolution work may happen mid-execution before a full review has ever
run.

v0.1 should keep `step_state` deliberately small:

- `reviewing`
  - an active review round exists and is not yet aggregated
- `waiting_ci`
  - the latest CI snapshot is still pending
- `resolving_conflicts`
  - local state explicitly says the branch is mid-conflict-resolution
- `ready_for_archive`
  - all steps are completed, all acceptance criteria are checked, and archive
    summary sections are no longer placeholders
- `implementing`
  - the default executing mode when nothing stronger applies

v0.1 does not need to infer a separate `testing` step state unless a CLI-owned
command writes a reliable signal into local state.

Remote freshness checks are usually too short-lived to deserve their own
`step_state`. If remote sync is needed, surface it as a warning or next action;
if it turns into real conflict work, use `resolving_conflicts`.

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
- report durable `plan_status` and `lifecycle`
- infer `step` from plan step status
- infer a minimal `step_state` from local review/CI/publish/sync state when
  `lifecycle: executing`
- surface stale or unknown remote freshness as warnings and next actions rather
  than as a long-lived `step_state`
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

- collect reviewer artifacts
- compute blocking and non-blocking findings
- stop with an error when expected reviewer slots are missing or invalid
- update local `state.json` with the aggregate result
- return next actions that depend on the review kind

Recommended next action:

- for a passing `delta` review, continue the current step or mark the step
  complete, then update the step's `Execution Notes` and `Review Notes`
- for a failing `delta` review, fix the current slice and rerun a delta review
- for a passing `full` review, move toward final CI and archive readiness
- for a failing `full` review, fix findings before archive

## Review Sequencing in v0.1

The CLI contract should assume this review cadence:

- use `delta` review after a completed plan step or after a narrow follow-up fix
- use `full` review once all planned work appears complete and the branch looks
  like an archive candidate
- if CI failure or conflict resolution creates a narrow, well-bounded change,
  run a `delta` review on that change
- if CI or conflict repair is broad or invalidates the prior full-review scope,
  rerun `full` review before archive

Archive readiness requires:

- a clean `full` review for the current candidate
- required CI green for the pushed archived candidate
- no unresolved conflict-repair work

### `harness archive`

Purpose:

- freeze the tracked plan locally for merge handoff

Contract:

- validate that the plan is active and archive-ready
- assume the plan's durable summary sections have already been written from the
  current plan plus local artifacts, not reconstructed from agent memory
- if the plan still contains `## Deferred Work`, require concrete follow-up
  issue references before allowing archive to succeed
- move the plan from `docs/plans/active/` to `docs/plans/archived/`
- update tracked fields such as `status`, `lifecycle`, and `updated_at`
- return next actions that explicitly include committing and pushing the archive
  change

Important note:

- `harness archive` changes tracked files locally
- the controller agent should commit and push the archive move before treating
  the plan as truly waiting for merge approval
- PR checks may rerun on that archive commit; if new feedback or check failures
  appear, use `harness reopen`
- merge actor, merge timestamp, and other land-only notes should go to PR
  comments or remote history rather than back into the archived plan
- if deferred work exists, the controller agent should ensure its corresponding
  follow-up issues are created and referenced in the archived plan before
  archive completes

Recommended next action:

- create or verify follow-up issues for deferred work
- commit and push the archived plan
- update the PR if needed
- wait for human merge approval
- reopen if the archived candidate no longer looks merge-ready

### `harness reopen`

Purpose:

- restore an archived plan to active execution

Contract:

- move the plan from `docs/plans/archived/` back to `docs/plans/active/`
- set `status: active`
- set `lifecycle: executing`
- increment `revision`
- update `updated_at`
- reset archive-only summary placeholders
- return next actions that help the next agent resume work

Recommended next action:

- review the feedback or remote change that caused reopen
- update plan content if scope or acceptance criteria changed
- continue the inferred current step, or set `awaiting_plan_approval` if the
  plan contract itself needs fresh approval

## Review Runtime Boundary

The CLI does not own subagent spawning in v0.1.

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

`harness ui` is intentionally deferred until the CLI and local-state contracts
are stable enough to support a useful read-only interface.
