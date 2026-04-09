---
template_version: 0.2.0
created_at: "2026-04-09T21:13:00+08:00"
source_type: direct_request
source_refs: []
---

# Fix harness UI workbench scroll boundaries so panes scroll without moving the whole page

## Goal

Fix the `harness ui` workbench layout so desktop browsing behaves like an IDE
workspace instead of a long document. The accepted direction is a shared
workbench scroll contract: the outer app shell stays fixed, and each page's
Explorer and Inspector panes own their own overflow when content exceeds the
available height.

This slice is intentionally scoped as a shared layout correction rather than a
Timeline-only patch. The clean end-state is one workbench-level scrolling model
that applies consistently across `Status`, `Timeline`, and `Review`, without
compatibility shims or page-specific exceptions.

## Scope

### In Scope

- Lock the desktop `harness ui` shell so document/body scrolling does not occur
  during normal workbench browsing.
- Ensure the shared workbench layout gives Explorer and Inspector panes proper
  bounded heights and independent overflow behavior.
- Verify the accepted scroll contract on `Timeline`, and confirm that shared
  `Status` and `Review` pages inherit the same layout behavior.
- Update frontend assets shipped by the Go UI server if the built bundle
  changes.

### Out of Scope

- Redesigning the overall workbench layout, density, or navigation structure.
- Adding new pages, tabs, or read/write controls.
- Timeline-specific visual polish unrelated to scroll containment.
- Mobile-specific redesign beyond any minimal responsive adjustment required to
  preserve a sane scroll model.

## Acceptance Criteria

- [ ] On desktop, `harness ui` does not rely on document/body scrolling for
      normal browsing inside the workbench.
- [ ] When Timeline history is long, the Explorer column scrolls independently
      without dragging the whole page with it.
- [ ] When Inspector content is long, the Inspector pane scrolls independently
      without dragging the whole page with it.
- [ ] The same shared scroll contract holds for `Status` and `Review`, not just
      `Timeline`.
- [ ] Frontend validation and any required rebuilt embedded assets reflect the
      final layout contract.

## Deferred Items

- Fine-grained scroll polish such as sticky subheaders, custom scrollbar
  styling, or page-specific overscroll tuning.
- Broader mobile/tablet interaction redesign unless the fix exposes a concrete
  regression that must be addressed in the same slice.

## Work Breakdown

### Step 1: Correct the shared workbench shell and pane overflow model

- Done: [x]

#### Objective

Adjust the shared app/workbench layout so the outer shell stays height-bounded
and overflow is owned by the Explorer and Inspector panes.

#### Details

This step should trace the full height chain from `html/body/#app` through the
top-level app shell and into the shared workbench frame. The implementation
should prefer a single coherent scroll contract over local Timeline hacks:
desktop pages should render inside a fixed-height shell, while pane bodies use
their own bounded overflow regions. If current wrappers or flex/grid defaults
allow content to escape and enlarge the document, tighten those boundaries at
the shared layer.

Keep the behavior readable in code. A future agent should be able to point at
the shared shell/workbench styles and understand why whole-page scroll no
longer occurs.

#### Expected Files

- `web/src/styles.css`
- `web/src/workbench.tsx`
- `web/src/main.tsx`

#### Validation

- A cold reader can identify one shared scroll boundary model in the workbench
  styles instead of page-specific overflow patches.
- Long Explorer content and long Inspector content both remain inside pane
  scroll containers on desktop.
- The updated frontend still passes typecheck/build.

#### Execution Notes

Locked the desktop shell onto a fixed-height grid and tightened the shared
height chain through `.layout`, `.main-stage`, `.workbench-page`,
`.workbench-explorer`, and `.workbench-inspector` so document/body scrolling no
longer owns the workbench. The key left-pane fix was making
`.workbench-explorer-body` a real flex child with `flex: 1`; before that, the
explorer overflow container collapsed to content height and could not own its
own scrolling.

Desktop now uses `overflow: hidden` on the outer shell while Explorer and
Inspector panes keep `overflow: auto` plus `overscroll-behavior: contain`.
Mobile/narrow layouts explicitly relax back to `height: auto` and visible outer
overflow so the desktop fix does not trap small-screen browsing.

Red/Green TDD was not practical for the initial shell diagnosis because the bug
was layout- and browser-behavior-specific, not a unit-contract gap. The
implementation was validated immediately with frontend typecheck/build, focused
Go UI tests, and direct Playwright browser probes against a seeded Timeline
fixture.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This shared layout correction and the validation closeout
were intentionally kept as one integrated slice, so the candidate will receive
one finalize full review after both tracked steps are complete.

### Step 2: Validate the contract across Timeline, Status, and Review

- Done: [x]

#### Objective

Prove that the fix holds for the shared workbench experience rather than only
for the original Timeline repro.

#### Details

This step should validate the final behavior with the repo's existing frontend
and UI-server checks, plus a direct browser-level verification of the scrolling
behavior on representative pages. If the frontend build output changes, rebuild
the embedded UI assets so `harness ui` serves the corrected layout.

Where practical, keep the evidence focused on the accepted contract: outer page
stays fixed, panes scroll independently, and shared pages inherit the same
behavior.

#### Expected Files

- `web/src/styles.css`
- `internal/ui/static/index.html`
- `internal/ui/static/assets/`
- any updated UI validation code under `internal/ui/` if needed

#### Validation

- `pnpm --dir web check`
- `pnpm --dir web build`
- Focused Go/UI validation that still passes after the rebuilt assets land.
- Manual or scripted browser verification confirms independent pane scrolling
  on `Timeline`, with shared behavior intact on `Status` and `Review`.

#### Execution Notes

Rebuilt the frontend bundle so `internal/ui/static/` reflects the new shell and
pane overflow contract, then rebuilt the repo-local `harness` binary so the
embedded UI served the updated assets. Added a browser-level Timeline scroll
regression to `scripts/ui-playwright-smoke` that asserts the document stays
fixed while Explorer and Inspector panes consume wheel scrolling.

Validation completed with `pnpm --dir web check`, `pnpm --dir web build`, and
`go test ./internal/ui`. The repo smoke helper hit an environment-specific
failure while rerunning `scripts/install-dev-harness`, so the scroll contract
was additionally verified directly through Playwright against a seeded local
Timeline fixture: `body` and `.app-shell` stayed `overflow: hidden`, document
height matched the viewport on `Timeline`, `Status`, and `Review`, and wheel
scrolling moved the Timeline Explorer and Inspector panes without moving the
document.

Finalize review then flagged that the new smoke assertion still only exercised
Timeline even though the layout change is shared. The repair generalized the
browser-level probe so `scripts/ui-playwright-smoke` now applies the same
document-vs-pane wheel assertions to `Status`, `Timeline`, and `Review`,
injecting temporary spacer content when a fixture route is otherwise too short
to scroll on its own.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This validation step exists to close the same UI slice as
Step 1 and is better reviewed as part of one finalize full-candidate pass than
as a separate step-only review.

## Validation Strategy

- Use frontend typecheck/build to catch layout regressions and to regenerate any
  embedded assets consumed by the Go UI server.
- Run focused Go/UI tests after rebuilding assets so the served workbench stays
  in sync with the checked-in frontend output.
- Verify the original repro in a browser and sanity-check the shared pages so
  the fix remains workbench-wide.

## Risks

- Risk: Tightening shell height and overflow boundaries could create clipped
  content or double-scroll behavior on narrower viewports.
  - Mitigation: Keep the scroll contract centralized in shared shell/workbench
    styles, then validate both the original Timeline case and the other shared
    workbench pages before execution closes.

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
