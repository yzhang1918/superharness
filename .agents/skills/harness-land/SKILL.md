---
name: harness-land
description: Use when a tracked harness plan is already archived and the human has explicitly approved merge, so the agent should merge the PR and perform post-merge cleanup.
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
3. Verify the PR still looks merge-ready.
4. Merge the PR.
5. Prefer `Merge commit` unless the human explicitly asks for a different
   strategy.
6. Add any final PR comment or remote update that belongs on the permanent
   record.
7. Close or update linked issues when the merge resolves them.
8. Run:

   ```bash
   harness land --pr <url> [--commit <sha>]
   ```

   This records merge confirmation and enters post-merge cleanup.
9. Finish local cleanup, branch sync, and any final remote follow-up.
10. Run:

   ```bash
   harness land complete
   ```

   This records cleanup completion, clears the current candidate pointer, and
   records the last landed archived plan so `harness status` returns to
   `idle` with the landed context preserved in artifacts.
11. Sync local `main`, delete the feature branch if appropriate, and leave the
    worktree clean.

## Do Not

- Do not merge without explicit human approval.
- Do not edit the archived plan after merge just to record merge metadata.
- Do not skip the explicit land milestones; run `harness land --pr <url>`
  after merge and `harness land complete` after cleanup so status stops
  reporting the archived candidate as current work.
- Do not keep using `land` if the candidate is no longer valid; switch back to
  `harness-execute` via `harness reopen --mode <finalize-fix|new-step>`.
