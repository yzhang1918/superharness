# Review Orchestration

Keep exactly one active review round at a time.

Do not start a new review round until the current round has been aggregated and
its outcome has been addressed, or until the plan has explicitly changed enough
that the current round should be abandoned.

The controller agent stays in `harness-execute` during review orchestration.
Only the spawned reviewer subagents should switch to `harness-reviewer`.

## When to Use Delta vs Full

- use `delta` after a completed step or a narrow follow-up change
- use `full` when the branch looks like an archive candidate
- after reopen:
  - narrow follow-up work may use `delta`
  - broad follow-up work should rerun `full`

## Review Spec

Create a review spec and pass it to:

```bash
harness review start --spec <path>
```

Use a compact JSON shape like:

```json
{
  "kind": "delta",
  "target": "Step 3: Make skill contracts more distributable",
  "trigger": "step_closeout",
  "dimensions": [
    {
      "name": "correctness",
      "instructions": "Look for contract mistakes, stale assumptions, or missing negative-path handling."
    },
    {
      "name": "agent_ux",
      "instructions": "Check whether another agent could resume the task cleanly from the updated docs and skills."
    }
  ]
}
```

Field rules:

- `kind`
  - enum: `delta` or `full`
  - `delta` is for a completed step or narrow follow-up change
  - `full` is for an archive candidate or another broad branch-level pass
- `target`
  - free-form description of what this round is reviewing
- `trigger`
  - free-form tag describing why the round exists
  - it is not a CLI enum today
  - useful common values include:
    - `step_closeout`
    - `review_feedback`
    - `review_fix`
    - `pre_archive`
    - `human_feedback`
    - `ci_repair`
    - `conflict_repair`
- `dimensions`
  - one reviewer slot per dimension after normalization
  - each dimension should have a short name and a concrete instruction

Suggested dimensions:

- `correctness`
  - logic, node-resolution, and contract mistakes
- `tests`
  - missing coverage, weak validation, or misleading smoke claims
- `docs_consistency`
  - README, AGENTS, skills, and plan drift
- `agent_ux`
  - handoff quality, clarity, and next-action guidance
- `risk_scan`
  - unresolved blockers, deferred risks that leaked back in, or unsafe defaults

Choose only the dimensions that fit the current change. Do not force every
round to use the same set.

## Controller Flow

1. Create the round:

   ```bash
   harness review start --spec <path>
   ```

2. Spawn multiple reviewer subagents in parallel: one reviewer per returned
   slot or review dimension.
   Use clean reviewer subagents for these slots. Do not inherit the
   controller's long chat context into reviewer threads. Use only the fixed
   reviewer prompt template for reviewer spawning.
3. Use a fixed reviewer prompt template so model or runtime changes do not
   silently change the reviewer contract.
4. Keep track of every spawned reviewer agent ID.
5. Wait for all reviewer subagents to finish before aggregation.
6. Verify each reviewer actually submitted a valid result for its slot.
7. Close the finished reviewer agent after verification, even when its
   submission was missing or invalid.
8. If a reviewer finished without a valid submission, respawn a reviewer for
   that slot immediately.
9. Only after every expected reviewer slot has a valid submission and every
   reviewer agent is closed, run:

   ```bash
   harness review aggregate --round <round-id>
   ```

## Fixed Reviewer Prompt Template

The returned `manifest_path` is for the controller, not the reviewer. Use it
when you need to inspect the CLI-normalized slots, expected artifact paths, or
ledger-owned review metadata. Reviewer subagents do not need it unless your
runtime prefers passing a single manifest pointer.

Use this controller prompt shape when spawning a reviewer subagent:

```text
You are the reviewer for one harness review slot.

Use the harness-reviewer skill and follow it exactly.

Round ID: <round-id>
Slot: <slot>
Assigned dimension: <dimension-name>
Instructions: <dimension-instructions>
```

## Codex-Specific Subagent Rules

Codex reviewer subagents are asynchronous.

- `spawn_agent` returns immediately.
- `wait_agent(ids=[...])` waits for whichever agent finishes first, not for all
  of them automatically.
- A completed reviewer agent may still remain open in the background until you
  close it.
- Use `spawn_agent(..., fork_context=false)` for reviewer slots so the reviewer
  starts from a clean context and sees only the fixed reviewer prompt.
- Do not append extra controller reasoning, artifact tours, or side
  instructions to the fixed reviewer prompt when spawning Codex reviewer
  subagents.

Use this pattern:

1. keep a pending set of reviewer agent IDs
2. call `wait_agent` on the pending set
3. remove whichever reviewer finished
4. verify it submitted a valid result for its assigned slot
5. call `close_agent` for that reviewer immediately after verification
6. if the submission was missing or invalid, respawn that reviewer slot
   immediately
7. repeat until the pending set is empty and every expected slot is filled
8. aggregate only after all reviewer agents are both finished and closed

This is required to avoid premature aggregation and dangling background agents.
