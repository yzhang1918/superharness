---
name: execute
description: Use when a superharness plan has been approved and the agent should drive implementation, review, CI, sync, closeout, and archive work until the plan reaches awaiting_merge_approval.
---

# Execute

## Purpose

Use execute after plan approval to drive the repository from active work to an
archived, merge-ready candidate.

The controller agent remains in `execute` even while a review round is active.
The `reviewer` skill is only for the spawned reviewer subagents.

## Start Here

1. Make sure `harness` resolves as a direct command.
   - If it does not, run `scripts/install-dev-harness`.
2. Run `harness status`.
3. Read the current tracked plan and then load the references you need.

## Reference Guide

- Read [resume-and-status.md](references/resume-and-status.md) at the start of
  every execute session or handoff.
- Read [step-inner-loop.md](references/step-inner-loop.md) when implementing or
  validating the current step.
- Read [review-orchestration.md](references/review-orchestration.md) whenever a
  review round is active or about to start.
- Read [publish-ci-sync.md](references/publish-ci-sync.md) when publish, CI, or
  remote-sync work becomes relevant.
- Read [closeout-and-archive.md](references/closeout-and-archive.md) before any
  archive attempt.

## Exit Criteria

Execute is done when:

- the plan is archived
- lifecycle is `awaiting_merge_approval`
- durable closeout summaries are written into the tracked plan

## Do Not

- Do not start a new review round while another review round is still active.
- Do not aggregate review until every expected reviewer subagent has finished.
- Do not leave reviewer subagents open after their results have been consumed.
- Do not archive based on memory alone; use the current plan plus `.local`
  artifacts.
