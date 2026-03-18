# Review Orchestration

v0.1 supports exactly one active review round at a time.

Do not start a new review round until the current round has either been
aggregated and addressed or intentionally abandoned by changing the plan.

The controller agent stays in `execute` during review orchestration. Only the
spawned reviewer subagents should switch to the `reviewer` skill.

## When to Use Delta vs Full

- use `delta` after a completed step or a narrow follow-up change
- use `full` when the branch looks like an archive candidate
- after reopen:
  - narrow follow-up work may use `delta`
  - broad follow-up work should rerun `full`

## Controller Flow

1. Create the round:

   ```bash
   harness review start --spec <path>
   ```

2. Spawn one reviewer subagent per returned slot or dimension.
3. Pass each reviewer:
   - the round ID
   - the assigned slot
   - the relevant instructions or manifest path
   - the `reviewer` skill
4. Keep track of every spawned reviewer agent ID.
5. Wait for all reviewer subagents to finish before aggregation.
6. After each reviewer result is received and consumed, close that reviewer
   agent explicitly.
7. Only after every expected reviewer has finished and been closed, run:

   ```bash
   harness review aggregate --round <round-id>
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
4. verify it submitted its result
5. call `close_agent` for that reviewer after consuming the result
6. repeat until the pending set is empty
7. aggregate only after all reviewer agents are both finished and closed

This is required to avoid premature aggregation and dangling background agents.
