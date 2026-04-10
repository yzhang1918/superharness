# Publish, CI, and Sync

Once implementation is materially complete, the execute loop expands beyond the
current step and eventually into archived-candidate handoff.

## Publish and CI

1. commit reviewable progress
2. push the branch
3. open or update the PR
4. wait for required CI
5. fix failures
6. decide whether the repair needs delta review or full review

For archived candidates, use the same sequence as post-archive handoff work,
but record the observed external facts through `harness evidence submit`:

1. commit the archive move
2. push the branch
3. open or update the PR
4. run `harness evidence submit --kind publish` once the PR or handoff target
   exists
5. wait for post-archive CI and record updates with
   `harness evidence submit --kind ci`
6. refresh remote readiness and record it with
   `harness evidence submit --kind sync`
7. once those remote facts exist, run the `Pre-Land` scan from
   [controller-truth-surfaces.md](controller-truth-surfaces.md) before treating
   the archived candidate as genuinely merge-ready
8. only then treat the candidate as ready to enter
   `execution/finalize/await_merge`

## Remote Freshness

Refresh remote state before merge-sensitive handoff work.

If remote changes introduce real conflict work:

- resolve the conflicts
- rerun focused validation
- run delta or full review depending on how broad the repair was

Do not create a new review round while an earlier one is still active.
