---
template_version: 0.2.0
created_at: "2026-04-11T23:16:13+08:00"
source_type: direct_request
source_refs: []
size: XXS
---

# Stabilize cramped review explorer rows and header metadata wrapping in the harness UI

## Goal

Tighten the read-only `harness ui` review layout so the review explorer and the
selected-round header stay legible under realistic narrow explorer widths. The
current review explorer packs the title, status dot, and compact status text
into one horizontal row, which makes long titles fight for space and creates an
overlapped, visually unstable row when the explorer is narrow. The selected
round header also keeps `Artifacts`, the status badge, and the timestamp on one
line, which makes the timestamp dominate the header and feel too long.

This slice should keep the existing review semantics and read-only behavior but
reflow the information hierarchy so the explorer uses a clear two-line pattern
and the selected-round header demotes the timestamp onto its own line. This
plan is being added retroactively because implementation started before the
tracked plan existed; the plan now records the approved scope and validation in
repository-visible form.

## Scope

### In Scope

- Reflow review explorer rows so the first line contains only the title and the
  second line carries review metadata plus the compact status label.
- Remove the extra review explorer status dot now that the row already has the
  colored edge treatment and an inline text status.
- Rebalance the selected-round header metadata so `Artifacts` and the status
  badge stay grouped while the timestamp moves to its own line.
- Keep the shared `ExplorerItem` API flexible enough to render richer review
  subtitle and meta content without changing the underlying review data model.
- Refresh the embedded UI bundle under `internal/ui/static/*`.

### Out of Scope

- Changing review aggregation logic, review status semantics, or the review UI
  data contract.
- Reworking the broader workbench shell, explorer resizing behavior, or
  non-review pages.
- Converting review status to color-only signaling.
- Introducing new review actions, tabs, or persisted layout preferences.

## Acceptance Criteria

- [x] Review explorer rows render the title on the first line and compact
      `Step x · a/b · STATUS` style metadata on the second line.
- [x] The review explorer no longer renders the extra per-row status dot, and
      narrow explorer widths do not force the title and status to compete on
      the same line.
- [x] The selected-round header keeps `Artifacts` and the status badge on the
      first metadata line while the timestamp appears on its own second line.
- [x] The frontend sources and embedded static UI bundle build cleanly after
      the layout update.

## Deferred Items

- Any broader visual redesign of the review explorer beyond the targeted row
  reflow and header metadata hierarchy.
- Additional browser-level fixture coverage for populated review data in this
  exact layout state.

## Work Breakdown

### Step 1: Reflow review explorer row metadata

- Done: [x]

#### Objective

Make the review explorer resilient to narrow widths by moving status out of the
title row and into the existing second metadata line.

#### Details

Keep the explorer dense and scan-friendly by preserving the existing two-line
row height instead of introducing a third line or a separate right-hand status
column. The row should use the title as the sole first-line content, then show
the existing step/submission metadata plus compact status text on the second
line. The visual status dot becomes redundant once the row already has a
colored trailing edge and a textual status label, so remove it in this slice.

#### Expected Files

- `web/src/pages.tsx`
- `web/src/workbench.tsx`
- `web/src/styles.css`

#### Validation

- Review explorer rows no longer place title and status in the same horizontal
  competition zone.
- Compact status remains visible in the row metadata and stays readable across
  status tones.

#### Execution Notes

Updated the review explorer row rendering to use the title on line one and a
wrapped metadata row on line two. The final compact form is
`Step x · a/b · STATUS`-style metadata so the narrowest explorer width still
keeps the status visible. Expanded
`ExplorerItem` to accept richer subtitle/meta children so the review page could
render the new inline structure without special-casing the base component.
Removed the extra review status dot and retuned the compact status styling to
read as metadata instead of a competing headline. TDD was not practical for
this step because the bug was a visual layout regression in an existing
read-only UI flow without a focused populated-review fixture that could first
express the failure as an automated red test; instead, the slice relied on
targeted static validation plus browser inspection of the live shell.
Finalize review `review-001-full` then found that the second metadata line was
still not width-constrained enough at the 220px minimum explorer width, so the
repair tightened the subtitle container into a real two-column row with an
ellipsis-prone metadata cell and a fixed visible status cell. Follow-up delta
review `review-002-delta` then exposed that the saved explorer evidence path
was still carrying an older screenshot and that the longer `submitted` copy was
using too much space; the repair compacted the metadata label to `a/b` and
refreshed the tracked supplement screenshot
`docs/plans/archived/supplements/2026-04-11-stabilize-review-ui-layout/review-explorer-row-220.png`
from the rebuilt UI.
Delta review `review-003-delta` then caught one more real containment gap: the
row's outer button still had `scrollWidth > clientWidth` because
`.explorer-item-main` lacked an explicit `minmax(0, 1fr)` grid track. The
repair added that track plus `max-width: 100%` constraints so a true 220px
measurement now shows the button, main column, row, and subtitle all fully
contained while the long title text alone truncates by ellipsis. The exact
candidate commit, screenshot paths, and width measurements are now durably
recorded in
`docs/plans/archived/supplements/2026-04-11-stabilize-review-ui-layout/containment-validation.md`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the explorer reflow and selected-round header changes
form one small integrated visual slice and are easier to review together.

### Step 2: Demote review header timestamps beneath the primary metadata row

- Done: [x]

#### Objective

Reduce selected-round header crowding by moving the timestamp under the primary
metadata row while keeping the status affordances easy to scan.

#### Details

Keep `Artifacts` and the selected round status badge on the first metadata row
so the header still exposes the actionable and semantic information first. Move
the timestamp to a second row with lighter styling so long formatted dates do
not visually outweigh the review title or status badge. This should be handled
with review-specific structure and styles so other workbench pages do not
change accidentally.

#### Expected Files

- `web/src/pages.tsx`
- `web/src/styles.css`
- `internal/ui/static/*`

#### Validation

- The review header shows `Artifacts` and the status badge together above the
  timestamp.
- The embedded frontend bundle refreshes successfully after the UI changes.

#### Execution Notes

Wrapped the review header metadata in a review-specific two-row container,
keeping `Artifacts` and the status badge on the first row and moving the
formatted timestamp to a second line with lighter, non-uppercase styling.
Rebuilt the embedded UI assets under `internal/ui/static/*` so the shipped Go
binary sees the updated frontend. This stayed in the same non-TDD validation
path as Step 1 because the change was a presentation-only hierarchy adjustment
inside the same UI slice rather than a contract or state transition that could
be proved first with a narrow failing unit test. After `review-001-full`
requested stronger evidence for populated review data, the repair loop used the
current worktree's real review round to capture browser evidence at the minimum
review explorer width in
`docs/plans/archived/supplements/2026-04-11-stabilize-review-ui-layout/review-explorer-row-220.png`
and
`docs/plans/archived/supplements/2026-04-11-stabilize-review-ui-layout/review-header-meta.png`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this is the same bounded layout polish as Step 1 and
does not benefit from a synthetic intermediate review boundary.

## Validation Strategy

- Run `pnpm --dir web check` after the component and style edits.
- Run `pnpm --dir web build` to refresh and validate the embedded static UI
  assets.
- Use `git diff --check` to catch formatting or whitespace mistakes.
- Open the local `harness ui` review route in a browser and capture saved
  screenshots against the populated review data now present in this worktree,
  specifically at the 220px explorer-width setting for the review page.
- Use browser-side DOM measurements at the 220px explorer width to confirm the
  review row container itself is contained (`clientWidth == scrollWidth`) even
  when the title text still needs ellipsis.
- Record the exact candidate commit, screenshot paths, and 220px DOM
  measurements in the tracked containment supplement so archive-time review does
  not depend on terminal scrollback or mutable local-memory claims.

## Risks

- Risk: Review-specific layout changes could accidentally affect other
  workbench pages if the shared explorer/header component surfaces are changed
  too broadly.
  - Mitigation: Keep the shared `ExplorerItem` change limited to accepting
    `ComponentChildren`, then scope the new structure and styling to review
    page classes.
- Risk: A code-only verification pass could miss a browser-only visual
  regression in the populated review state.
  - Mitigation: Keep the DOM structure narrowly aligned to the agreed layout,
    run the real UI shell locally, and keep the tracked screenshot evidence and
    220px DOM measurements in the containment supplement so archive-time review
    does not depend on ephemeral local output.

## Validation Summary

- Revision 2 reopens the archived candidate only for a clean `origin/main`
  sync repair after publish evidence showed revision 1 was stale against the
  merge base.
- `pnpm --dir web check`, `pnpm --dir web build`,
  `scripts/ui-playwright-review-smoke`, and `git diff --check` now pass for
  the reopened revision-2 candidate.
- Browser validation at the true 220px review explorer width confirmed the
  selected row container, title row, and subtitle row all stayed contained
  while the long title text alone truncated by ellipsis.
- Tracked browser evidence now lives under
  `docs/plans/archived/supplements/2026-04-11-stabilize-review-ui-layout/`,
  including `review-explorer-row-220.png`, `review-header-meta.png`,
  `review-smoke-rev2-initial.png`, `review-smoke-rev2-row-active.png`, and the
  exact measurements recorded in `containment-validation.md`.
- The latest passing review smoke reran on code head `4f1757a`, the commit that
  contains the review-smoke selector repair. The sync merge from revision 1's
  archived commit through `4f1757a` did not change `web/src` or `internal/ui`,
  so the 220px containment captures remain the current UI-specific evidence
  while the fresh review smoke covers the merged revision-2 browser path. The
  remaining follow-up after `4f1757a` is documentation-only evidence
  bookkeeping.

## Review Summary

- Revision 2 reopens the previously archived candidate only because publish
  handoff recorded `sync_status=stale`; the repair itself is a clean merge of
  `origin/main` plus fresh validation before a new finalize review.
- Finalize review started with `review-001-full`, which found the initial
  narrow-width containment and validation gaps in the explorer metadata layout.
- Follow-up delta rounds `review-002-delta` through `review-005-delta`
  tightened the subtitle containment, compacted the metadata copy, and moved
  the validation story from mutable local screenshots into tracked supplement
  evidence.
- `review-006-delta` passed cleanly after the tracked evidence repair, and the
  required archive-gate `review-007-full` then passed with one non-blocking
  note about stale plan wording that this archive closeout resolves.
- Reopened revision-2 review `review-008-full` then caught the stale
  review-smoke selector and the need for a fresh current-head validation
  record after the clean `origin/main` merge.
- Delta follow-up `review-009-delta` narrowed the remaining blocker to the
  validation anchor itself, and `review-010-delta` passed cleanly once the
  docs-only closeout tied the smoke result to validated code head `4f1757a`.

## Archive Summary

- Archived At: 2026-04-12T00:41:16+08:00
- Revision: 2
- Revision 1 archived at `2026-04-12T00:16:38+08:00`, then reopened as
  revision 2 solely because publish handoff recorded `sync_status=stale`
  against `origin/main`.
- PR: Existing PR [#147](https://github.com/catu-ai/easyharness/pull/147)
  remains open for the revision-2 repair candidate.
- Ready: Revision 2 has now merged `origin/main` cleanly, refreshed local and
  browser validation, passed follow-up finalize review in `review-010-delta`,
  and is ready to rearchive for merge handoff.
- Merge Handoff: Push the refreshed branch to PR `#147`, and record updated
  publish, CI, and sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Reflowed review explorer items so the title owns the first line while the
  second line carries compact `Step x · a/b · STATUS` metadata.
- Removed the redundant review status dot from explorer rows and hardened the
  row layout so the narrow 220px explorer width no longer lets the row content
  push past the container.
- Demoted the selected-round header timestamp onto its own second metadata line
  beneath `Artifacts` and the status badge.
- Rebuilt the embedded UI bundle and recorded durable tracked evidence for the
  final narrow-width browser validation.
- Reopened the archived candidate only for a clean `origin/main` sync repair,
  updated `scripts/ui-playwright-review-smoke` to follow the surviving status
  text affordance, and recorded revision-2 browser evidence in the tracked
  supplement.

### Not Delivered

- No broader review explorer redesign was attempted beyond the agreed layout
  polish.
- No additional populated-review browser fixture coverage was added in this
  lightweight slice.

### Follow-Up Issues

- [#146](https://github.com/catu-ai/easyharness/issues/146) tracks the deferred
  broader review UI polish and stronger populated-review browser fixture
  coverage.
