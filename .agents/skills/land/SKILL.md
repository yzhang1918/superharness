---
name: land
description: Use when a superharness plan is already archived and the human has explicitly approved merge, so the agent should merge the PR and perform post-merge cleanup.
---

# Land

## Purpose

Use land only after the plan is archived and a human has said it is time to
merge.

## Workflow

1. Run `harness status` and confirm the plan is `awaiting_merge_approval`.
2. Verify the PR still looks merge-ready.
3. Merge the PR.
4. Prefer `Merge commit` unless the human explicitly asks for a different
   strategy.
5. Add any final PR comment or issue update that belongs on the remote record.
6. Sync local `main`, delete the feature branch if appropriate, and leave the
   worktree clean.

## Do Not

- Do not merge without explicit human approval.
- Do not edit the archived plan after merge just to record merge metadata.
- Do not keep using `land` if the candidate is no longer valid; switch back to
  `execute` via `harness reopen`.

