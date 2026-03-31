---
name: harness-execute
description: Use when a tracked harness plan has been approved and the controller agent should drive implementation, review, archive closeout, publish/CI/sync evidence work, and merge-readiness follow-up until the archived candidate is genuinely ready to wait for merge approval. This is the main controller skill for day-to-day execution after approval.
---

# Harness Execute

## Purpose

Use this skill after plan approval to drive the repository from active work to
an archived, merge-ready candidate.

ALWAYS use `harness-execute` whenever `harness status` resolves a current
approved tracked plan and `state.current_node` is still in `plan` or
`execution/...`, or the archived candidate still needs publish follow-up
before human merge approval. That includes fresh sessions, resumed sessions
after compaction, and pick-up work where the safest next move is to follow the
current plan instead of improvising a new workflow.

The controller agent stays in `harness-execute` for the whole execution loop,
including review orchestration. Do not switch the controller into
`harness-reviewer`; that skill is only for spawned reviewer subagents assigned
to specific review slots.

Run `harness status` at controller checkpoints, not just once per session:

- at start or resume
- before marking a step done
- after each review aggregate
- before trusting later-step or finalize progression after warnings, repair, or
  review follow-up

Routine review progression is controller-owned. Once the approved plan reaches
an ordinary step-closeout or finalize-review boundary, the controller should
start that review flow without asking the human to micromanage it.

If the approved plan is likely to require reviewer subagents and explicit
authorization has not been obtained yet, ask for that authorization as soon as
the need becomes foreseeable. Do not wait until reviewer spawning is the only
remaining next action before surfacing the request.
If execution still reaches a reviewer-subagent boundary without that explicit
approval, pause only long enough to request it, then continue the review flow
once the human answers.

Keep exactly one active review round at a time. The detailed review rules live
in [review-orchestration.md](references/review-orchestration.md).

For behavior-changing work, default to Red/Green/Refactor TDD. Only skip TDD
when it is genuinely impractical, and record the reason in the step's
`Execution Notes`.

## Start Here

1. Run `harness status`.
2. If `harness status` points to a current tracked plan that is already
   approved for execution, stay in `harness-execute` and open that plan from
   `plan_path`.
   Active work uses a tracked plan even when the profile is lightweight; only
   archived lightweight snapshots move into `.local/`.
3. Identify the active or next plan step.
4. Use the status output to answer four questions:
   - which tracked plan is current
   - which `current_node` the worktree is currently in
   - which step or finalize phase is active or next
   - whether local state already shows review, evidence, or land work in flight
5. If `harness` is unavailable or resolves to the wrong binary, first follow
   the repository's documented setup path. If no setup path is documented, ask
   the human to install or expose the correct `harness` command.
6. Read only the references needed for the current part of the loop.

## Node Hints

- `plan`
  - wait for approval or update the plan if scope changed before
    `harness execute start`
- `execution/step-<n>/implement`
  - continue the current step, fix review findings, or mark the step done once
    the slice is genuinely complete
  - rerun `harness status` before marking the step done so the next action
    reflects whether review, repair, or a warning-driven follow-up is due
- `execution/step-<n>/review`
  - review is in flight; aggregate or wait for reviewer submissions rather
    than continuing implementation blindly
- `execution/finalize/review|fix|archive`
  - the step list is done and the branch is in closeout, review, or
    archive-prep work
  - treat `execution/finalize/review` as a continuation state: start or
    aggregate finalize review as the status guidance indicates instead of
    stopping to ask the human whether routine review closeout should happen
- `execution/finalize/publish`
  - the plan is archived, but publish, CI, or sync evidence still needs work
  - for lightweight work, keep the repo-visible breadcrumb requirement in view
    while driving the candidate toward `await_merge`
- `execution/finalize/await_merge`
  - the archived candidate is merge-ready; stay in execute until explicit
    human merge approval switches the controller into `harness-land`

## Reference Guide

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
- `harness status` resolves `state.current_node` to
  `execution/finalize/await_merge`
- durable closeout summaries are written into the tracked plan

## Do Not

- Do not ask the human to micromanage routine execution once the plan is
  approved.
- Do not ask the human whether routine step-closeout or finalize review should
  start once `harness status` and the tracked plan make the next review action
  clear.
- Do not silently stall at review orchestration because reviewer subagent
  authorization is missing; request it explicitly as soon as you know it will
  be required, and if you still reach the reviewer boundary without approval,
  pause only long enough to ask and then resume once the answer arrives.
- Do not bypass node or review gates just because the next action feels obvious.
- Do not skip TDD for behavior changes without documenting why the usual
  Red/Green/Refactor loop was not practical.
- Do not rely on chat memory when `harness status`, the tracked plan, or local
  artifacts can tell you the truth more directly.
- Do not archive based on memory alone; use the current plan plus `.local`
  artifacts.
