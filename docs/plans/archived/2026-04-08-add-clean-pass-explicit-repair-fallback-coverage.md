---
template_version: 0.2.0
created_at: "2026-04-08T13:50:00Z"
source_type: issue
source_refs:
    - https://github.com/catu-ai/easyharness/issues/113
---

# Add clean-pass explicit repair fallback coverage

## Goal

Add focused regression coverage proving that a clean explicit earlier-step
repair returns status to the ordinary later frontier or finalize node for the
same candidate.

This slice is a narrow follow-up for issue `#113`. It should close the
remaining clean-pass coverage gap without changing runtime behavior or
reopening the explicit-step repair contract work that was already landed.

## Scope

### In Scope

- Add focused `internal/status` regression coverage for clean explicit
  earlier-step repair fallback from a later unfinished frontier.
- Add focused `internal/status` regression coverage for clean explicit
  earlier-step repair fallback from finalize context.
- Adjust nearby test helpers only if needed to express the already-supported
  behavior clearly.

### Out of Scope

- Changing runtime `status`, `review`, or lifecycle behavior.
- Reworking the broader explicit-step repair contract or spec prose.
- Expanding this slice into broader E2E work beyond the focused regression
  needed to close issue `#113`.

## Acceptance Criteria

- [x] Focused regression coverage proves that a clean explicit earlier-step
      repair started from a later unfinished frontier returns status to the
      ordinary later frontier rather than keeping the repaired step pinned.
- [x] Focused regression coverage proves that a clean explicit earlier-step
      repair started from finalize context returns status to the ordinary
      finalize node rather than keeping the repaired step pinned.
- [x] The relevant focused Go test targets pass without any production logic
      changes.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Add focused clean-pass fallback regression coverage

- Done: [x]

#### Objective

Extend the focused status regression suite so clean explicit earlier-step
repair fallback cannot regress silently.

#### Details

Use the existing explicit-step repair test scaffolding and add the smallest
new cases that prove clean aggregates return the candidate to its ordinary
later frontier or finalize node. Prefer `internal/status/service_test.go`
unless a second package becomes clearly necessary.

#### Expected Files

- `internal/status/service_test.go`

#### Validation

- The new clean-pass fallback assertions fail if status incorrectly keeps the
  repaired step pinned after a passing aggregate.
- Focused `go test` targets covering the touched regression suite pass.

#### Execution Notes

Added two focused regressions in `internal/status/service_test.go` covering
clean explicit earlier-step repair fallback from a later reopened frontier and
from finalize review. The later-frontier assertion now matches the current
implementation detail that the frontier returns to `step-3/implement` while
the latest repair review still surfaces as `ReviewStatus: "pass"` in facts.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only adds focused regression coverage for the
existing explicit-step repair behavior and does not change production logic.

### Step 2: Revalidate and archive the follow-up slice

- Done: [x]

#### Objective

Confirm the new coverage closes issue `#113` cleanly and leave an archive-ready
record of the narrow follow-up.

#### Details

Keep the closeout small: focused validation, routine review as needed, and a
clear archive summary that records this slice as the clean-pass fallback
follow-up to the earlier explicit-step repair clarification work.

#### Expected Files

- `docs/plans/active/2026-04-08-add-clean-pass-explicit-repair-fallback-coverage.md`

#### Validation

- The focused validation and review results are recorded in the plan.
- The plan is archive-ready with no placeholder text left behind.

#### Execution Notes

Focused validation completed with:
`go test ./internal/status`
and
`go test ./internal/review`

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is validation and archive-closeout preparation
for a focused regression-only slice. Routine finalize review still applies
before archive.

## Validation Strategy

- Run the focused Go tests for the touched regression suite.
- Re-read the new tests against the existing explicit-step repair semantics to
  confirm they lock current behavior rather than inventing a new one.

## Risks

- Risk: The new tests might accidentally encode a broader workflow redesign
  instead of the current clean-pass fallback behavior.
  - Mitigation: keep the assertions tightly scoped to status fallback after a
    passing explicit repair aggregate.
- Risk: The coverage could duplicate the existing E2E assertions without
  adding focused protection where the previous gap actually lived.
  - Mitigation: place the new checks in `internal/status/service_test.go`,
    which is the focused suite that issue `#113` called out.

## Validation Summary

- Added two focused `internal/status` regressions covering clean explicit
  earlier-step repair fallback from a later reopened frontier and from
  finalize review.
- Validated the follow-up slice with:
  `go test ./internal/status`
  and
  `go test ./internal/review`

## Review Summary

- Finalize review `review-001-full` passed cleanly with no blocking or
  non-blocking findings.

## Archive Summary

- Archived At: 2026-04-08T21:55:13+08:00
- Revision: 1
- PR: NONE
- Ready: The candidate has a clean finalize review, focused validation is
  green, and this slice closes the clean-pass fallback regression gap tracked
  in issue `#113`.
- Merge Handoff: Archive the plan, commit the focused regression update and
  tracked archive move, push the branch, open a PR, and record publish, CI,
  and sync evidence until the candidate reaches merge-ready handoff.

## Outcome Summary

### Delivered

- Added focused status regressions proving that clean explicit earlier-step
  repair returns to the ordinary later frontier and to the ordinary finalize
  review node instead of keeping the repaired step pinned.
- Closed the regression gap intentionally deferred from the earlier explicit
  step-repair clarification slice.

### Not Delivered

- No runtime behavior or spec wording changes were made in this follow-up.

### Follow-Up Issues

- NONE.
