---
name: harness-execute
description: Use when a tracked harness plan has been approved and the controller agent should drive implementation, review, CI, sync, closeout, and archive work until the plan reaches awaiting_merge_approval. This is the main controller skill for day-to-day execution after approval.
---

# Harness Execute

## Purpose

Use this skill after plan approval to drive the repository from active work to
an archived, merge-ready candidate.

The controller agent stays in `harness-execute` for the whole execution loop,
including review orchestration. Do not switch the controller into
`harness-reviewer`; that skill is only for spawned reviewer subagents assigned
to specific review slots.

Keep exactly one active review round at a time. The detailed review rules live
in [review-orchestration.md](references/review-orchestration.md).

## Start Here

1. Run `harness status`.
2. Open the current tracked plan from `plan_path`.
3. Identify the active or next plan step.
4. Read only the references needed for the current part of the loop.

## Reference Guide

- Read [resume-and-status.md](references/resume-and-status.md) at the start of
  every execute session or handoff.
- Read [step-inner-loop.md](references/step-inner-loop.md) when implementing or
  validating the current plan step.
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

- Do not ask the human to micromanage routine execution once the plan is
  approved.
- Do not bypass lifecycle gates just because the next action feels obvious.
- Do not rely on chat memory when `harness status`, the tracked plan, or local
  artifacts can tell you the truth more directly.
- Do not archive based on memory alone; use the current plan plus `.local`
  artifacts.
