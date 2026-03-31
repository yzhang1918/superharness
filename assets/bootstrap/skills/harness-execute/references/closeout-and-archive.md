# Closeout and Archive

Archive is a freeze-and-summarize step, not just a file move.

## Before Archive

1. Run `harness status` and confirm the plan is actually archive-ready.
   - If `status` returns `blockers`, fix those first instead of learning them
     from a failing `harness archive`.
2. Make sure acceptance criteria are checked and steps are completed.
3. Read the latest finalize review artifacts under `.local` and confirm the
   branch really is in `execution/finalize/archive` rather than still needing
   review or repair.
4. Update the tracked plan's durable summaries from those artifacts:
   - `Validation Summary`
   - `Review Summary`
   - `Archive Summary`
   - `Outcome Summary`
   - for lightweight work, the active plan is still tracked before archive,
     while the archived snapshot later moves to `.local/harness/plans/archived/`
5. If `## Deferred Items` still contains real items, replace `Follow-Up Issues`
   with durable handoff details before archive. Issue links are fine, but the
   main rule is that it must not stay `NONE`.
6. Run:

   ```bash
   harness plan lint <plan-path>
   ```

7. Archive the plan:

   ```bash
   harness archive
   ```

## After Archive

Archive still needs an explicit handoff flow:

1. Commit the archive move and summary updates.
2. Push the branch.
3. Open or update the PR.
4. If the profile is `lightweight`, update the agreed repo-visible breadcrumb
   such as the PR body note before treating the candidate as ready to wait for
   merge approval.
5. Run `harness status` again to confirm the archived candidate now reports the
   expected `execution/finalize/publish` or
   `execution/finalize/await_merge` node for this worktree.
6. Record publish, CI, and sync facts through `harness evidence submit`.
7. Wait for human merge approval or switch to `harness-land` only when asked
   once status reaches `execution/finalize/await_merge`.

If new feedback or remote changes invalidate the archived candidate, use:

```bash
harness reopen --mode <finalize-fix|new-step>
```
