---
name: harness-plan
description: Create or update a tracked harness plan for medium/large work once the direction is clear enough to execute. Use this when work needs a self-contained plan that a future agent can complete from the repository alone, without relying on discovery chat or hidden session memory.
---

# Harness Plan

## Purpose

Use this skill to create or update the tracked plan that will drive execution.

## When to Use

- discovery has converged and the work needs a new tracked plan
- an active plan needs a scoped update before archive
- reopened work needs the tracked plan refreshed before execution resumes

## Workflow

1. Start from `harness plan template` when creating a new plan.
2. Name the file with the plan-schema convention:
   `YYYY-MM-DD-clear-topic.md`.
3. Make the topic meaningful and specific. It should tell a cold reader what is
   changing, not just name a vague area.
4. Write a plan that is clear to both humans and future agents:
   - concrete goal
   - explicit scope and out-of-scope
   - acceptance criteria
   - reviewable work breakdown
5. Make the plan self-contained. Fold in decisions from discovery or prior
   discussion so another agent can execute from the plan plus repository state
   alone.
6. Keep execution detail concise. Push runtime mechanics into skills and CLI
   contracts instead of bloating the plan.
7. Reread the plan as if the chat history were unavailable. Fix anything that
   still depends on hidden context.
8. Run `harness plan lint <plan-path>`.
9. Present the plan for approval before execution starts.
   If the approved execution loop is likely to require reviewer subagents,
   ask for explicit subagent authorization in the same approval exchange so
   execution does not stall later at review time.

## Commands

- `harness plan template --help`
- `harness plan lint --help`

## Exit Criteria

The plan is ready when:

- lint passes
- the resulting tracked plan would resolve to `plan` until
  `harness execute start` is recorded
- the human can approve or challenge it without hidden context
- when reviewer subagents are likely later, the approval handoff makes that
  expected authorization explicit instead of deferring it implicitly
- a future agent could continue the work from the plan alone

## Do Not

- Do not start `harness-execute` before the plan is approved.
- Do not duplicate full CLI enums or placeholder rules from the specs when a
  command or spec already defines them.
- Do not let deferred work float without being named clearly in the plan.
- Do not leave key decisions only in discovery chat or session memory.
