# Publish, CI, and Sync

Once implementation is materially complete, the execute loop expands beyond the
current step.

## Publish and CI

1. commit reviewable progress
2. push the branch
3. open or update the PR
4. wait for required CI
5. fix failures
6. decide whether the repair needs delta review or full review

## Remote Freshness

Refresh remote state before archive-sensitive or merge-sensitive work.

If remote changes introduce real conflict work:

- resolve the conflicts
- rerun focused validation
- run delta or full review depending on how broad the repair was

Do not create a new review round while an earlier one is still active.

