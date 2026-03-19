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

Suggested dimensions:

- `correctness`
  - logic, lifecycle, and contract mistakes
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

Use this controller prompt shape when spawning a reviewer subagent:

```text
You are the reviewer for one harness review slot.

Use the harness-reviewer skill and follow it exactly.

Round ID: <round-id>
Slot: <slot>
Manifest path: <manifest-path>
Assigned dimension: <dimension-name>
Instructions: <dimension-instructions>

Review the current change for this slot only. Inspect the relevant diff, plan,
and local artifacts. Submit your result with:

harness review submit --round <round-id> --slot <slot> --input <path>

After a successful submit, report the receipt back to the controller and stop.
Do not call any other harness commands.
```

## Codex-Specific Subagent Rules

Codex reviewer subagents are asynchronous.

- `spawn_agent` returns immediately.
- `wait_agent(ids=[...])` waits for whichever agent finishes first, not for all
  of them automatically.
- A completed reviewer agent may still remain open in the background until you
  close it.

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
