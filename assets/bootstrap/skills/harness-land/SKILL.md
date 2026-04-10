---
name: harness-land
description: Use when a tracked harness plan is already archived and the human has explicitly approved merge, so the agent should merge the PR and complete the required post-merge bookkeeping.
---

# Harness Land

## Purpose

Use land only after the plan is archived and a human has said it is time to
merge.

## Workflow

1. Run `harness status`.
2. If `state.current_node` is not `execution/finalize/await_merge`, stop.
   - If the candidate is no longer valid, run
     `harness reopen --mode finalize-fix` for narrow repair or
     `harness reopen --mode new-step` when the change deserves a new step, and
     then return to `harness-execute`.
   - If the plan is still executing, stay in `harness-execute`.
3. Re-read the `Pre-Land` scan in
   [../harness-execute/references/controller-truth-surfaces.md](../harness-execute/references/controller-truth-surfaces.md)
   so merge readiness, CI truth, and required bookkeeping are freshly checked.
4. Verify the PR still looks merge-ready.
5. Merge the PR.
6. Prefer `Merge commit` unless the human explicitly asks for a different
   strategy.
7. Run:

   ```bash
   harness land --pr <url> [--commit <sha>]
   ```

   This records merge confirmation and enters required post-merge bookkeeping.
8. Finish the required post-merge bookkeeping before leaving `land`.
   - Add a final PR comment when the permanent PR record still needs a durable
     merge closeout note or follow-up handoff after the merge.
   - The minimum PR comment content is:
     - confirmation that the PR merged
     - the merged commit or merge reference when it is available
     - whether linked issues were closed or intentionally left open
     - any durable follow-up pointer that a later reader needs
   - Close a linked issue when the merged change fully resolves it.
   - If a linked issue is only partially addressed, intentionally deferred, or
     still needs later work, do not close it silently; add or update the issue
     with a follow-up reference to the merged PR and the remaining work.
9. Finish local cleanup, branch sync, and any final remote follow-up.
10. Run:

   ```bash
   harness land complete
   ```

   Run this only after the required PR and issue bookkeeping is complete. This
   records required post-merge bookkeeping completion, clears the current
   candidate pointer, and
   records the last landed archived plan so `harness status` returns to `idle`
   with the landed context preserved in artifacts.
11. Sync local `main`, delete the feature branch if appropriate, and leave the
    worktree clean.

## Do Not

- Do not merge without explicit human approval.
- Do not edit the archived plan after merge just to record merge metadata.
- Do not skip the explicit land milestones; run `harness land --pr <url>`
  after merge and `harness land complete` after the required bookkeeping so
  status stops reporting the archived candidate as current work.
- Do not treat PR comments, linked-issue closure, or linked-issue follow-up as
  optional best-effort cleanup when they are needed for the permanent record.
- Do not keep using `land` if the candidate is no longer valid; switch back to
  `harness-execute` via `harness reopen --mode <finalize-fix|new-step>`.
