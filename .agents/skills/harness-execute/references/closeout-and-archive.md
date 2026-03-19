# Closeout and Archive

Archive is a freeze-and-summarize step, not just a file move.

## Before Archive

1. Run `harness status` and confirm the plan is actually archive-ready.
2. Make sure acceptance criteria are checked and steps are completed.
3. Read the latest review, CI, sync, and publish artifacts under `.local`.
4. Update the tracked plan's durable summaries from those artifacts:
   - `Validation Summary`
   - `Review Summary`
   - `Archive Summary`
   - `Outcome Summary`
5. Make sure deferred items that still matter have follow-up GitHub issues.
6. Run:

   ```bash
   harness plan lint <plan-path>
   ```

7. Archive the plan:

   ```bash
   harness archive
   ```

## After Archive

Archive changes tracked files, so it still needs the normal git flow:

1. Commit the archive move and summary updates.
2. Push the branch.
3. Run `harness status` again so the next agent sees `awaiting_merge_approval`.
4. Let CI re-run if the repository requires it.
5. Wait for human merge approval or switch to `harness-land` only when asked.

If new feedback or remote changes invalidate the archived candidate, use:

```bash
harness reopen
```
