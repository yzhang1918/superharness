# Step Inner Loop

The inner loop is how you finish one plan step cleanly.

## Inner Loop

1. Confirm the current step from `harness status` and the tracked plan.
2. For behavior changes, run Red/Green/Refactor:
   - Red: write or update a test that fails for the intended behavior.
   - Green: implement the smallest change that makes the test pass.
   - Refactor: improve structure without changing the behavior you just proved.
3. If TDD is genuinely impractical for this slice, record the reason in
   `Execution Notes` before continuing.
4. Run focused validation for the slice.
5. Update the step's `Execution Notes` with a concise summary.
6. If the slice is green and meaningfully reviewable, make a small commit.
7. Run `harness status` before step closeout so the next action reflects the
   current step, any active review, and any warning-driven follow-up.
8. If the slice is ready for review, run step-closeout review now. Use `delta`
   by default for a completed step, but use `full` when a narrower review would
   be misleading or the slice needs a broader pass.
9. If no step-closeout review is needed, record
   `NO_STEP_REVIEW_NEEDED: <reason>` in `Review Notes` before marking the step
   done.
10. Fix findings, rerun focused validation, and update `Review Notes`.
11. Make another small commit when a review-driven fix meaningfully changes the
   branch.
12. Mark the step complete only when the step objective, validation, and review
    closeout are genuinely satisfied.

## Step Notes

Keep step-local notes useful to the next agent:

- `Execution Notes`
  - what changed, what was validated, what remains
- `Review Notes`
  - latest delta/full review outcome, major findings, and what was fixed
  - or `NO_STEP_REVIEW_NEEDED: <reason>` when the step was too small or low
    risk to justify a separate closeout review

Keep these notes high-signal and brief. Summarize the core change and outcome;
do not turn them into transcripts.

Do not wait until archive to reconstruct step history from memory.
