# Review Orchestration

The controller agent stays in `harness-execute` during review orchestration.
Only the spawned reviewer subagents should switch to `harness-reviewer`.

Starting routine review is controller-owned. Once `harness status`, the tracked
plan, and the current step state make ordinary step-closeout or finalize review
the next action, start that review without asking the human for permission.

## When to Use Delta vs Full

- use `delta` after a completed step or a narrow follow-up change
- use `full` for step closeout when a narrower pass would be misleading or the
  slice needs a broader risk scan
- use `full` when the branch looks like an archive candidate
- after reopen:
  - narrow follow-up work may use `delta`
  - broad follow-up work should rerun `full`

Treat those as controller heuristics, not rigid gates. Strong agents may
promote a would-be `delta` review to `full` when a narrow pass would likely
miss the real risk surface. Common signals include:

- the repair touches multiple contracts or spreads across loosely related files
- the change summary is no longer a trustworthy boundary for the risk
- reopen, remote-sync churn, or wider branch drift means a clean reread is safer
- you want an unbiased broad reread more than reviewer continuity

## Delta Anchors

`delta` review must anchor to a real git commit.

Use the latest passed-review anchor commit as the default starting point for a
later `delta` review. In practice:

- a completed step normally creates a commit that becomes the next step-closeout
  `delta` anchor
- after a later narrow repair passes and more follow-up is still likely, make a
  fresh anchor commit before the next `delta` review

The anchor is a starting lens, not a hard inspection boundary. Reviewers should
begin from the anchored change and then deepen inspection when related logic,
plan intent, or contract meaning warrants it.

## Routine Start Rules

- Before starting a review round, run the `Pre-Review` scan from
  [controller-truth-surfaces.md](controller-truth-surfaces.md).
- After a completed step becomes reviewable, start a step-bound review before
  treating the step as durably done, unless the step will instead record
  `NO_STEP_REVIEW_NEEDED: <reason>` in `Review Notes`.
- Once all tracked steps are complete and no warning-driven repair remains,
  start finalize review for the full candidate before archive closeout.
- If `harness status` surfaces an earlier completed step that still lacks
  review closeout, resolve that warning before trusting later-step or finalize
  progression.

## Review Spec

Create a review spec and pass it to:

```bash
harness review start --spec <path>
```

Use a compact JSON shape like:

```json
{
  "kind": "delta",
  "anchor_sha": "<base-commit-sha>",
  "review_title": "Step 2: Refactor review metadata",
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
- `anchor_sha`
  - optional for `full`
  - required for `delta`
  - for `delta`, persist the controller-chosen git commit anchor here so the
    manifest records the durable review starting point
- `review_title`
  - optional human-readable review title for the controller and reviewers
- `step`
  - optional 1-based tracked step number
  - usually omit it and let `harness` bind the round automatically
  - only include it when you need to point review at a specific tracked step explicitly
- `dimensions`
  - one reviewer slot per dimension after normalization
  - each dimension should have a short name and a concrete instruction

The controller agent should not invent workflow metadata like `trigger` or
`target`. `harness review start` infers whether the round is step-bound or
finalize-bound from the current node and persists that structure itself.

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

2. Spawn or resume reviewer subagents: one reviewer per returned slot or review
   dimension.
   For a slot's first pass in a tracked step or for one finalize review scope
   in one revision, or whenever reuse is not clearly safe, use clean reviewer
   subagents. Moving to a different tracked step, moving from step review into
   finalize review, or moving to a different revision always starts with fresh
   reviewers. Do not inherit the controller's long chat context into reviewer
   threads. Use only the fixed reviewer prompt template for reviewer spawning.
   For a narrow same-slot `delta` follow-up after a verified earlier
   submission, prefer `resume_agent` over a fresh reviewer when the review
   scope is still materially the same.
3. Use a fixed reviewer prompt template so model or runtime changes do not
   silently change the reviewer contract.
4. Keep track of every spawned reviewer agent ID.
5. Wait for all reviewer subagents to finish before aggregation.
6. Verify each reviewer actually submitted a valid result for its slot.
7. Close the finished reviewer agent after verification, even when its
   submission was missing or invalid. Closed is the default steady state
   between rounds; do not leave reviewer agents hanging open just in case they
   might be useful later.
8. If a reviewer finished without a valid submission, respawn a clean reviewer
   for that slot immediately.
9. Before aggregation, run the `Pre-Aggregate` scan from
   [controller-truth-surfaces.md](controller-truth-surfaces.md).
10. Only after every expected reviewer slot has a valid submission and every
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
Review kind: <delta-or-full>
Active plan context: <Step N: title | Finalize: title>
Review title: <review-title>
Revision: <candidate-revision-or-none>
Slot: <slot>
Assigned dimension: <dimension-name>
Instructions: <dimension-instructions>
Anchor SHA: <commit-sha-or-none>
Change summary: <bounded-change-summary>
```

Reviewer submissions may include optional finding `locations` arrays using
lightweight repo-relative anchors such as `path/to/file.go`,
`path/to/file.go#L123`, or `path/to/file.go#L1-L3`.

## Fixed Reviewer Resume Prompt Template

Use resume only for a narrow same-slot follow-up after the controller has
already verified and closed that reviewer's earlier successful submission.
Do not resume across tracked steps, or from a step review into finalize
review. Those boundaries always start with fresh reviewers.
Resume is only valid while the review scope itself is still the same:

- for step review, the same tracked step title
- for finalize review, the same candidate review title for the same revision

If reopen, a new tracked step, a new revision, or a new finalize candidate
changes that scope, start with fresh reviewers.

Use this controller prompt shape when resuming a previously closed reviewer:

```text
You are resuming the same harness review slot you handled earlier.

Use the harness-reviewer skill and follow it exactly.

New round ID: <new-round-id>
Review kind: <delta-or-full>
Active plan context: <Step N: title | Finalize: title>
Review title: <review-title>
Revision: <candidate-revision-or-none>
Slot: <slot>
Assigned dimension: <dimension-name>
Instructions: <dimension-instructions>
Anchor SHA: <commit-sha-or-none>
Change summary since your last submission: <bounded-change-summary>
```

## Codex-Specific Subagent Rules

Codex reviewer subagents inherit the shared lifecycle rules from `Harness
Subagent Use` in the managed `AGENTS.md` contract. This section adds the
reviewer-specific orchestration constraints on top of that shared baseline.

Codex reviewer subagents are asynchronous.

- `spawn_agent` returns immediately.
- `wait_agent(ids=[...])` waits for whichever agent finishes first, not for all
  of them automatically.
- A completed reviewer agent may still remain open in the background until you
  close it.
- Use `spawn_agent(..., fork_context=false)` for reviewer slots so the reviewer
  starts from a clean context and sees only the fixed reviewer prompt.
- After a reviewer is cleanly closed, `resume_agent` may reopen that same
  reviewer later. For a narrow same-slot `delta` follow-up with a verified
  earlier submission, continuity is the default.
- Do not append extra controller reasoning, artifact tours, or side
  instructions to the fixed reviewer prompt when spawning Codex reviewer
  subagents.

Prefer `resume_agent` by default when all of these are true:

- the earlier reviewer submission for that agent was valid and already verified
  by the controller
- the new round is still narrow enough for `delta` review
- the new round stays within the same tracked step review boundary, or within
  the same finalize review scope for the same revision
- the new round keeps the same review scope as the earlier submission
- the reviewer keeps the same slot and materially the same dimension
  instructions
- the controller can give a bounded change summary that is directly tied to the
  earlier findings

Treat the closed reviewer as retired for this follow-up and spawn a fresh clean
reviewer instead when any of these are true:

- the earlier submission was missing, invalid, or never verified
- the controller is moving to a different tracked step, or from a step review
  into finalize review
- the review scope changed because of reopen, a new tracked step, a new
  revision, or a later finalize pass against a different candidate
- the follow-up broadened into `full` review or otherwise changed scope
  materially
- the slot or instructions changed enough that the old reviewer context would
  mislead more than help
- the repair batch includes unrelated changes, remote-sync churn, or other
  broad context that deserves a clean reread
- you want an unbiased second look more than continuity

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

After a later repair round starts, you may either spawn fresh reviewers again
or reopen an eligible closed reviewer with `resume_agent` only for the same
tracked step, or for the same finalize review scope in the same revision,
then deliver only the fixed resume prompt for the new round. Even when you
reuse a reviewer this way, close it again immediately after the new submission
is verified.

This is required to avoid premature aggregation, dangling background agents,
and stale-context leakage.
