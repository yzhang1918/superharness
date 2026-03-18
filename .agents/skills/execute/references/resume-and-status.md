# Resume and Status

Always begin or resume execute work with:

```bash
harness status
```

Use the status output to answer four questions:

1. Which tracked plan is current?
2. What lifecycle is it in?
3. Which step is active or next?
4. Is local state telling you that review, CI, or conflict work is already in
   flight?

## Resume Sequence

1. Run `harness status`.
2. Open the `plan_path` named by the command.
3. If `review_round_id` or `ci_snapshot_id` is present, inspect the referenced
   local artifacts before assuming the step is clear.
4. Follow the highest-priority `next_actions`.
5. Update the tracked plan rather than carrying hidden progress in chat.

## Lifecycle Hints

- `awaiting_plan_approval`
  - wait for approval or update the plan if scope changed
- `executing`
  - continue the current step and use `step_state` as a local hint
- `blocked`
  - resolve the blocker or get human input
- `awaiting_merge_approval`
  - wait for merge approval or switch to `land` only when asked

If `harness` is unavailable, bootstrap it before doing more work:

```bash
scripts/install-dev-harness
```

