---
name: plan
description: Use when superharness work is ready to become or update a tracked plan, including creating a new plan file, refining scope, and preparing for human approval before execution.
---

# Plan

## Purpose

Use the plan skill to create or update the tracked plan that will drive
execution.

## When to Use

- discovery has converged and the work needs a new tracked plan
- an active plan needs a scoped update before archive
- reopened work needs its tracked plan refreshed before execution resumes

## Workflow

1. Start from `harness plan template` when creating a new plan.
2. Name the file with the plan-schema convention:
   `YYYY-MM-DD-short-topic.md`.
3. Write a plan that is clear to both humans and future agents:
   - concrete goal
   - explicit scope and out-of-scope
   - acceptance criteria
   - reviewable work breakdown
4. Keep execution detail concise. Push runtime mechanics into skills and CLI
   contracts instead of bloating the plan.
5. Run `harness plan lint <plan-path>`.
6. Present the plan for approval before execution starts.

## Commands

- `harness plan template --help`
- `harness plan lint --help`

## Exit Criteria

The plan is ready when:

- lint passes
- lifecycle is `awaiting_plan_approval`
- the human can approve or challenge it without hidden context

## Do Not

- Do not start `execute` before the plan is approved.
- Do not duplicate full CLI enums or placeholder rules from the specs when a
  command or spec already defines them.
- Do not let deferred work float without being named clearly in the plan.

