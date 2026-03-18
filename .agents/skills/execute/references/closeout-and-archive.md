# Closeout and Archive

Archive is a freeze-and-summarize step, not just a file move.

## Before Archive

1. Make sure acceptance criteria are checked and steps are completed.
2. Read the latest review, CI, sync, and publish artifacts under `.local`.
3. Update the tracked plan's durable summaries from those artifacts:
   - `Validation Summary`
   - `Review Summary`
   - `Archive Summary`
   - `Outcome Summary`
4. Make sure deferred items that still matter have follow-up GitHub issues.
5. Run:

   ```bash
   harness plan lint <plan-path>
   ```

6. Archive the plan:

   ```bash
   harness archive
   ```

## After Archive

Archive changes tracked files, so it still needs the normal git flow:

1. commit the archive move and summary updates
2. push the branch
3. let CI re-run if the repository requires it
4. wait for human merge approval or switch to `land` only when asked

If new feedback or remote changes invalidate the archived candidate, use:

```bash
harness reopen
```

