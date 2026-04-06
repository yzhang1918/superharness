---
template_version: 0.2.0
created_at: "2026-04-06T22:24:18+08:00"
source_type: issue
source_refs:
    - '#94'
---

# Unify the harness UI workbench model and topbar workflow summary

## Goal

Finish the remaining UI-polish slice for `harness ui` by making `Status`,
`Timeline`, and `Review` behave like one coherent workbench instead of three
similar-but-divergent pages. The accepted product shape is a shared
`Page -> Explorer -> Inspector` model across all three pages, plus a compact
global workflow summary integrated into the existing topbar.

This slice is intentionally allowed to make breaking frontend changes. It
should prefer the cleanest long-term product and code shape over compatibility
preservation, migration scaffolding, or incremental exceptions. The UI must
remain read-only and stay within the existing steering-surface boundary.

## Scope

### In Scope

- Replace the current topbar right-side `READ-ONLY` / `LOCAL` chrome with a
  compact global workflow summary that surfaces:
  - current node
  - blocker count when present
  - warning count
  - next-action count
- Make those topbar summary items navigable so they can jump into the
  corresponding `Status` section instead of duplicating detail in the topbar.
- Rework `Status` into the same `Explorer -> Inspector` browsing model used by
  `Timeline` and `Review`, treating `Summary`, `Next actions`, `Warnings`,
  `Facts`, and `Artifacts` as the browsable objects for that page.
- Introduce shared frontend abstractions for the repeated workbench patterns so
  Explorer and Inspector layout, headers, list rows, empty/loading/error
  states, and selection behavior no longer live in three mostly-separate page
  implementations.
- Unify shell, density, spacing, typography, and navigation language across
  `Status`, `Timeline`, and `Review` without widening the product scope.
- Rebuild embedded UI assets after the frontend refactor.
- Validate the slice with focused automated coverage plus interactive
  Playwright-driven visual inspection that includes captured screenshots.

### Out of Scope

- Any write action, command execution, or mutation initiated from the UI.
- Any new page beyond `Status`, `Timeline`, and `Review`.
- Reintroducing `Diff` / `Files` browsing or any general file/diff surface.
- Compatibility shims or migration layers that preserve the current frontend
  component split when a cleaner shared abstraction is available.
- Backend contract expansion unless the frontend refactor discovers a small,
  necessary read-only gap that cannot be solved with the existing resources.

## Acceptance Criteria

- [ ] The topbar keeps the existing brand/workspace-path structure but replaces
      `READ-ONLY` / `LOCAL` with a compact global workflow summary.
- [ ] The topbar summary exposes current node plus warning/action counts, and
      blocker count when applicable, using dense VS Code-like compact items
      rather than verbose text.
- [ ] Clicking a topbar summary item navigates to the corresponding `Status`
      section instead of trying to render detailed state inline.
- [ ] `Status`, `Timeline`, and `Review` all use the same page-level mental
      model: page switcher, Explorer column, Inspector pane.
- [ ] `Status` no longer behaves like the last dashboard-style exception and
      instead uses Explorer selection plus a single focused Inspector surface.
- [ ] Explorer typography, row density, selection treatment, metadata styling,
      and pane headers are visibly consistent across all three pages.
- [ ] Inspector headers, tabs, empty states, loading states, and error states
      use shared abstractions or patterns rather than page-specific re-creation
      where the role is the same.
- [ ] The frontend code clearly centralizes the shared workbench structure so a
      future agent does not need to modify three separate page skeletons to
      evolve Explorer/Inspector behavior.
- [ ] The UI remains read-only and does not widen the current steering-surface
      boundary.
- [ ] Embedded UI assets are rebuilt and shipped from the refactored frontend.
- [ ] Validation includes both automated browser checks and an interactive
      Playwright session with screenshots that are reviewed for visual density,
      hierarchy, and coherence before closeout.

## Deferred Items

- New backend resources unless a narrow read-only contract gap is discovered
  during execution and judged necessary to preserve the accepted UI shape.
- Additional topbar affordances beyond global workflow summary navigation.
- New actions, inline edits, or command-triggering controls.
- Broader theme experimentation unrelated to the accepted workbench direction.

## Work Breakdown

### Step 1: Define the shared workbench framework and page contracts

- Done: [x]

#### Objective

Lock the frontend architecture for a shared `Page -> Explorer -> Inspector`
framework before reshaping individual pages.

#### Details

This step should identify the shared primitives needed across `Status`,
`Timeline`, and `Review`: shell sections, page headers, Explorer list
containers, Explorer rows, Inspector headers, tab bars, and common
loading/error/empty treatments. The plan should favor shared composition over
configuration-heavy genericism: create one coherent page framework that pages
can fill with their own content, without forcing every detail into opaque
schema-driven rendering.

The resulting code shape should remove the current "three similar pages with
slightly different custom markup" pattern. `Status` should be explicitly
reframed as a browsable page of status objects rather than a dashboard
exception. If any existing helper names or file structure fight this model,
prefer renaming or moving them now instead of preserving awkward seams.

#### Expected Files

- `web/src/main.tsx`
- `web/src/styles.css`
- new or updated shared frontend files under `web/src/`

#### Validation

- A cold reader can point to the shared Explorer/Inspector abstractions without
  hunting across three page-specific render branches.
- The chosen framework is simple enough that future UI slices can extend one
  shared pattern instead of copying structure.
- Any renamed or moved frontend files still fit the repo's current build.

#### Execution Notes

Split the previous single-file frontend into shared `types`, `helpers`,
`workbench`, and `pages` modules so the app now has a clear shared
Explorer/Inspector framework instead of three loosely related page branches.
The structural refactor intentionally treated `Status` as a first-class
workbench page rather than preserving the old dashboard exception.

Validated the new structure with `pnpm --dir web check` and
`pnpm --dir web build`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This architectural step only becomes meaningful when the
topbar/status reshape and cross-page visual alignment land on top of it, so
the review boundary is the integrated UI-polish slice rather than this
refactor alone.

### Step 2: Rebuild the topbar summary and Status page on the shared model

- Done: [x]

#### Objective

Land the product-level reshape by turning the topbar into the accepted global
workflow summary and rebuilding `Status` as a true Explorer/Inspector page.

#### Details

Replace the current low-value topbar labels with compact workflow items that
communicate the current node and high-signal counts at a glance. Keep the
topbar concise: it is a navigation and orientation surface, not a second
status page. Summary items should be clickable and route to the corresponding
`Status` section.

For `Status`, use the same structural language as the other workbench pages.
The Explorer should list `Summary`, `Next actions`, `Warnings`, `Facts`, and
`Artifacts` with stable counts and compact metadata. The Inspector should show
one focused section at a time, with hierarchy and evidence treatment that make
summary/actions/warnings primary and raw facts/artifacts secondary.

#### Expected Files

- `web/src/main.tsx`
- `web/src/styles.css`
- new or updated shared frontend files under `web/src/`
- `internal/ui/static/*`

#### Validation

- The topbar right side shows the accepted workflow summary and no longer shows
  `READ-ONLY` / `LOCAL`.
- Clicking topbar summary items lands on the intended `Status` section.
- `Status` now reads and behaves like the same kind of page as `Timeline` and
  `Review`, not a special-case dashboard.
- Embedded assets are rebuilt after the frontend changes.

#### Execution Notes

Replaced the old `READ-ONLY` / `LOCAL` topbar chrome with the accepted compact
workflow summary that surfaces current node plus warnings/actions, and wires
those items back into `Status` navigation. Rebuilt `Status` as a real
Explorer/Inspector page with focused section selection and a summary-first
inspector instead of the previous mixed dashboard layout.

Interactive Playwright screenshots now show the accepted topbar treatment in
the shared shell, including `output/playwright/manual-status-polish/` for the
current-worktree `Status` view.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The topbar/status reshape is coupled to the shared
workbench framework and the later Timeline/Review alignment, so reviewing it in
isolation would miss the main product goal of cross-page consistency.

### Step 3: Align Timeline and Review with the shared Explorer/Inspector language

- Done: [x]

#### Objective

Bring `Timeline` and `Review` onto the same visual and structural system as
the rebuilt `Status` page without losing their page-specific strengths.

#### Details

This step should use the shared framework to normalize Explorer density,
typography, row metadata treatment, Inspector headers, tab styling, and
section spacing. The goal is not to flatten all three pages into identical
content, but to make their common roles unmistakable: the second column is the
Explorer, the third column is the Inspector, and both should feel like one
product family.

If existing page-specific markup or styling blocks resist the shared model,
prefer deleting or replacing them rather than layering overrides forever. The
end state should reduce divergence in both DOM shape and styling vocabulary.

#### Expected Files

- `web/src/main.tsx`
- `web/src/styles.css`
- new or updated shared frontend files under `web/src/`
- `internal/ui/static/*`

#### Validation

- Explorer rows across all three pages share the same density and selection
  language.
- Inspector headers and tab treatments feel clearly related across pages.
- Timeline and Review still preserve their page-specific content strengths
  while using the unified workbench system.

#### Execution Notes

Moved `Timeline` and `Review` onto the same workbench frame, Explorer-row
density, inspector header, and shared tab system as `Status`, while keeping
their page-specific content models intact. The visual system now uses one
consistent shell language for left rail, Explorer headers, inspector tabs,
metrics, warnings, and supporting evidence treatment.

Interactive Playwright screenshots under `output/playwright/` were used to
check the resulting `Timeline` and `Review` layouts directly instead of only
relying on DOM assertions.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The consistency pass spans all three pages and shares
the same validation evidence as the branch-level UI review, so a separate step
review would be artificially narrow.

### Step 4: Prove the visual result with automated and interactive Playwright validation

- Done: [x]

#### Objective

Validate both behavior and visual quality with browser automation, including a
human-reviewed interactive pass with screenshots.

#### Details

Update or add Playwright coverage so automated checks verify the new topbar
summary, cross-page navigation shape, `Status` Explorer/Inspector behavior,
and the unified shell language across `Status`, `Timeline`, and `Review`.

This step must also include an interactive Playwright run that exercises the
live UI in a browser, captures screenshots of the key pages and states, and
uses those screenshots for a deliberate visual review. The screenshots are not
optional garnish; they are required evidence that spacing, typography,
hierarchy, and density were judged visually after implementation rather than
only inferred from DOM assertions.

Use the repo's Playwright skill/workflow where appropriate. Keep the
interactive validation concrete: open the UI, navigate across all three pages,
capture screenshots for topbar + Explorer + Inspector views, and confirm the
workbench now reads as one coherent product.

#### Expected Files

- `scripts/ui-playwright-smoke`
- `scripts/ui-playwright-review-smoke`
- new or updated browser test files under `web/` or `tests/`
- screenshot artifacts under `.local/` or another disposable runtime path used
  during validation

#### Validation

- Automated browser coverage asserts the topbar summary and the shared page
  model across all three routes.
- An interactive Playwright session captures screenshots for at least:
  - `Status`
  - `Timeline`
  - `Review`
  - one view that clearly shows the topbar summary treatment
- Closeout notes can point to those screenshots as evidence for the visual
  review.

#### Execution Notes

Ran `pnpm --dir web check` and `pnpm --dir web build`, then updated the
Playwright smoke scripts to validate the new topbar summary and shared
Explorer/Inspector structure. Interactive Playwright runs produced fresh
snapshots and screenshots for `Status`, `Timeline`, and `Review`, including:

- `output/playwright/manual-status-polish/status-current.png`
- `output/playwright/harness-ui-smoke-*/timeline-inspector-initial.png`
- `output/playwright/ruis-*/review-initial.png`
- `output/playwright/ruis-*/review-ux.png`

Reviewed those screenshots manually for topbar density, Explorer consistency,
inspector hierarchy, and cross-page visual coherence. The shell wrappers in
this environment produced the expected artifacts and assertions but were
slower to return than ordinary unit/build commands, so the screenshot artifacts
themselves were used as the durable evidence source.

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run focused frontend/build validation after the refactor so the shared page
  framework still compiles into embedded UI assets.
- Run targeted automated browser coverage for topbar summary navigation, page
  switching, and Explorer/Inspector rendering across `Status`, `Timeline`, and
  `Review`.
- Perform an interactive Playwright-driven manual pass against the live UI and
  capture screenshots for visual inspection before calling the slice complete.
- Treat the screenshot-backed visual pass as required validation, not optional
  polish.

## Risks

- Risk: A shared abstraction could become over-generic and harder to evolve
  than the current duplication.
  - Mitigation: Prefer a small number of explicit shared building blocks over
    schema-heavy configuration; keep page-specific content local.
- Risk: Rebuilding `Status` on the shared model could accidentally reduce
  readability of high-signal state details.
  - Mitigation: Keep `Summary`, `Next actions`, and `Warnings` visually primary
    in the Inspector and verify the result with screenshot-based review.
- Risk: Topbar summary density could become cramped or noisy.
  - Mitigation: Keep the topbar to compact counts and current-node context
    only, with detail routed back into `Status`.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
