# Controller Truth Surfaces

Use this checklist at high-risk controller transitions. It is a short
self-check for the controller, not a second reviewer protocol.

## Pre-Review

- scope truth
  - decide whether this round is really `delta` or `full`, and say why a
    narrower or broader pass would be less trustworthy
- anchor and diff truth
  - for `delta`, verify the review anchor is a real git commit and the current
    change boundary still matches the intended review slice
- contract scan
  - scan the active plan, touched contracts, docs wording, and focused
    validation so the controller does not outsource all completeness checking
    to reviewers
- dispatch sanity
  - make sure the review spec and reviewer prompt carry the actual round
    context, anchor, and bounded change summary instead of forcing reviewers to
    guess

## Pre-Aggregate

- submission truth
  - verify every expected slot submitted a real result rather than a missing,
    invalid, or still-skeleton artifact
- round-state truth
  - verify you are aggregating the current active round and not mixing older
    findings, newer repairs, or the wrong revision
- synthesis sanity
  - read the submitted findings once before aggregation so obvious duplicates,
    missing severities, or malformed claims do not slide through by inertia

## Pre-Archive

- placeholder debt
  - replace placeholder summaries, unchecked acceptance criteria, and step
    markers before archive instead of letting `archive` discover them late
- narrative debt
  - make sure the tracked plan tells a durable story of what changed, how it
    was validated, what review concluded, and what follow-up remains
- publish-readiness sanity
  - confirm the branch is truly in archive closeout rather than still needing
    review, repair, or unresolved handoff work

## Pre-Land

- PR truth
  - refresh the actual PR state instead of trusting a stale local impression of
    readiness
- CI truth
  - verify the latest relevant runs and distinguish `success` from cancelled,
    stale, superseded, or still-running checks
- sync truth
  - refresh branch freshness against the remote base before merge-sensitive
    handoff or merge work
- merge and bookkeeping truth
  - confirm the remaining PR comment, issue follow-up, evidence, and land
    bookkeeping work rather than assuming merge-ready means done
