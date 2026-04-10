---
template_version: 0.2.0
created_at: "2026-04-10T09:43:25+08:00"
source_type: direct_request
source_refs: []
---

# Add a current-plan browser page to harness UI

## Goal

Add a new read-only `Plan` page to `harness ui` so the current tracked plan
becomes a first-class browsing surface alongside `Status`, `Timeline`, and
`Review`. The page should help a human read the current plan package
without dropping into the filesystem: browse the main markdown plan by heading
hierarchy, inspect companion `supplements/<plan-stem>/` content, and keep the
experience aligned with the existing workbench shell.

This slice should follow the same product boundary as the existing UI pages:
the Go backend assembles a read-only view model from the current plan package,
and the frontend renders that model. Prefer the clean target design over
compatibility layers or fallback behavior that preserves older UI assumptions.

## Scope

### In Scope

- Add a read-only `Plan` resource for `harness ui` that loads the current
  tracked plan package for the worktree, including archived merge-handoff
  states while the worktree is not idle.
- Add a `Plan` page to the existing page rail and workbench shell.
- Model the left explorer as a hierarchical, collapsible tree with:
  - the main plan markdown represented by heading-based TOC nodes
  - a `supplements/` folder node when a matching package directory exists
  - recursive supplement child directories and files
- Make the right pane behave as a document reader:
  - selecting a plan heading keeps the full plan markdown visible and scrolls
    or jumps to the selected section
  - selecting a supplement file switches the pane to that file's rendered or
    plain-text content
- Support an initial extension allowlist for richer preview:
  - `md`
  - `txt`
  - `json`
  - `yaml`
  - `yml`
- Treat other text-readable files as plain-text fallback.
- Treat binary files, image files, unknown unsupported formats, and files above
  the chosen preview size threshold as `not supported`.
- Keep the page usable when no current plan exists by rendering a clear empty
  state for truly idle worktrees.
- Validate the slice with focused automated coverage and an interactive
  Playwright pass that includes real clicks, screenshots, and visual review.

### Out of Scope

- Browsing arbitrary archived plans or showing a recent archived-plan fallback
  when the worktree is idle.
- Any UI-triggered write, command execution, plan mutation, or supplement
  editing flow.
- Turning the page into a generic repository file browser outside the active
  plan package.
- Rich preview support for images, CSV datasets, PDFs, or other binary-heavy
  artifacts in this first slice.
- Adding compatibility shims that preserve the old three-page UI assumption
  when a cleaner four-page workbench is available.

## Acceptance Criteria

- [x] `harness ui` exposes a new read-only `Plan` page in the page rail.
- [x] The page reads the current tracked plan package for the worktree,
      including archived merge-handoff states, and does not invent a recent
      archived fallback when the worktree is idle.
- [x] When no current plan exists, the page renders a clear empty state that
      explains there is no current plan to browse.
- [x] The left explorer presents a hierarchical, collapsible navigation tree
      that includes the main plan heading structure and, when present, a
      `supplements/` folder subtree.
- [x] The main plan heading tree defaults to an expanded depth that surfaces
      headings through `H3` while still allowing deeper nodes to be expanded
      on demand.
- [x] Selecting a plan heading keeps the full markdown document in the reader
      and navigates to the chosen section instead of replacing the document
      with an isolated fragment.
- [x] Selecting a supplement file replaces the reader content with that file's
      preview while preserving the workbench shell and explorer selection.
- [x] `md`, `txt`, `json`, `yaml`, and `yml` render as supported previews.
- [x] Text-readable files outside the richer preview allowlist degrade to
      plain-text rendering without pretending to provide rich semantics.
- [x] Binary files, image files, unsupported formats, and files above the
      configured preview-size threshold render a clear `not supported` state.
- [x] The implementation introduces or updates automated tests that cover the
      read model, current-plan and idle-state handling, file support and
      size-threshold
      gating, and core page interactions.
- [x] Before closeout, the page is exercised interactively with Playwright:
      open the page, expand and collapse explorer nodes, click plan headings,
      open supplement files, capture screenshots, and confirm the visual
      hierarchy and reading experience match the accepted direction.

## Deferred Items

- Archived-plan browsing, history switching, or a plan-package picker.
- Persistent explorer expansion memory beyond what the browser already keeps in
  local runtime state during one session.
- Rich preview for images, CSV, PDF, or other heavier supplement formats.
- Search, filtering, or cross-link graphing within plan content.

## Work Breakdown

### Step 1: Define the plan-page read model and preview contract

- Done: [x]

#### Objective

Lock the backend read-only contract for current-plan package browsing,
including heading tree extraction, supplement enumeration, and preview gating.

#### Details

Follow the same read-only pattern as `status`, `timeline`, and `review`: the
backend should derive the current tracked plan, load the markdown file plus any
matching `supplements/<plan-stem>/` directory, and expose a UI-facing payload
without changing plan lifecycle or write-side contracts. Make the preview
policy explicit in code and tests so a future agent can grow the supported
extensions list intentionally rather than by accident.

This step should define the decision rules for:

- current-plan loading while non-idle
- idle empty state
- heading extraction and stable node identifiers for in-page navigation
- recursive supplement tree shape
- supported rich preview extensions
- plain-text fallback detection
- unsupported/binary/image handling
- maximum previewable file size and the payload shape returned when the limit
  is exceeded

If the resource becomes part of the documented UI contract, update the
relevant schema/spec surfaces rather than leaving the API shape implicit in Go
tests alone.

#### Expected Files

- `internal/ui/server.go`
- new read-only plan resource file(s) under `internal/`
- `internal/ui/server_test.go`
- relevant contract/schema docs if the new resource is documented there

#### Validation

- The resource loads the current tracked plan package, including archived
  merge-handoff states, and returns a stable empty-state payload only when the
  worktree is idle.
- Tests cover heading extraction, supplement enumeration, supported preview
  files, plain-text fallback, unsupported binary/image files, and oversize
  files.
- A cold reader can tell from the contract that the page is read-only and tied
  to the current tracked plan package rather than generic repo browsing.

#### Execution Notes

Added a dedicated `internal/planui` read-only service plus `/api/plan` server
wiring and a public `PlanResult` contract/schema. The backend now loads the
current tracked plan package, emits a heading tree for the main markdown
document, walks matching `supplements/<plan-stem>/` directories recursively,
and applies explicit preview gating for supported rich preview, plain-text
fallback, image/binary rejection, and oversize files. Focused validation:
`go test ./internal/planui ./internal/ui ./internal/contractsync`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 backend work was intentionally reviewed together
with the UI wiring and browser validation because the read model is only
meaningful when the explorer/reader behavior lands on top of it.

### Step 2: Build the Plan workbench page and reader interactions

- Done: [x]

#### Objective

Ship the `Plan` page as a first-class workbench page with a VS Code-like
explorer and a document-oriented reader pane.

#### Details

Add `Plan` to the page rail and keep the page aligned with the existing shell
language established by `Status`, `Timeline`, and `Review`. The explorer
should feel like a compact IDE tree rather than a flat list: hierarchical
nodes, clear folder/file affordances for supplements, and collapsible heading
branches for the main plan. The main plan remains one readable document in the
right pane, so selecting headings should navigate within that document instead
of fragmenting it into separate cards.

For supplements, prefer one coherent preview model over many special cases.
Supported extensions can get richer rendering where it is cheap and readable,
while plain-text fallback should still look intentional. Unsupported and
oversize content should not crash or silently omit nodes; show an explicit
reader state so humans understand why preview is unavailable.

#### Expected Files

- `web/src/main.tsx`
- `web/src/pages.tsx`
- `web/src/types.ts`
- `web/src/helpers.ts`
- `web/src/workbench.tsx`
- `web/src/styles.css`
- `internal/ui/static/*`

#### Validation

- `Plan` appears in the rail and routes cleanly inside the existing SPA shell.
- The explorer renders heading nodes and supplement folder/file nodes with
  collapsible hierarchy and stable selection behavior.
- Selecting plan headings moves the reader to the intended section while
  keeping the full markdown document visible.
- Selecting supplement files swaps the reader content appropriately and makes
  unsupported or oversize states explicit rather than ambiguous.
- Embedded UI assets are rebuilt after the frontend changes.

#### Execution Notes

Added `Plan` to the page rail, SPA routing, frontend types, and shared shell.
The new workspace renders a VS Code-like hierarchical explorer for plan
headings and supplements, keeps the main document as one markdown reader, and
switches the inspector to file previews for supplements. Added `markdown-it`
for document rendering, introduced current-plan package supplements for
dogfooding, and rebuilt the embedded UI assets after the frontend changes.
Validation: `pnpm --dir web check`, `pnpm --dir web build`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The Step 2 UI slice shares one review boundary with the
backend contract and the browser-validation work, so a step-local review would
have been narrower than the real user-visible change.

### Step 3: Lock behavior and visual quality with automated and interactive browser validation

- Done: [x]

#### Objective

Prove both the functional behavior and the visual reading experience before
the slice is considered ready for review.

#### Details

Add or extend automation for the core behaviors that are likely to regress:
idle empty state, supported preview rendering, unsupported/oversize handling,
explorer interaction, and heading-driven navigation. Then run an interactive
Playwright session against a real `harness ui` instance and treat that pass as
part of the acceptance bar, not as optional polish.

The interactive pass should include real clicks and visual inspection:

- open the `Plan` page
- expand and collapse the heading tree
- expand and collapse the `supplements/` folder when present
- click multiple heading levels and confirm the reader scroll target feels
  correct
- open supported supplement files and confirm the content presentation matches
  the intended format
- open unsupported or oversize supplement files and confirm the `not
  supported` state is legible
- capture screenshots of the main states and review spacing, hierarchy,
  density, and overall coherence with the rest of the workbench

Use the [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md)
skill for browser work whenever it is needed during execution or closeout.

#### Expected Files

- `internal/ui/server_test.go`
- existing or new UI/browser validation scripts under `scripts/`
- `output/playwright/` artifacts produced during validation
- any updated frontend/backend files needed to address issues found during the
  validation pass

#### Validation

- Automated coverage exercises the accepted page behaviors and degraded states.
- The interactive Playwright pass produces screenshots that demonstrate the
  accepted explorer hierarchy and reader behavior.
- Any visual or interaction issue found during the manual browser pass is
  either fixed before closeout or captured explicitly as deferred follow-up.

#### Execution Notes

Extended automated validation with new Go coverage, a dedicated
`scripts/ui-playwright-plan-smoke` browser flow for plan browsing and idle
empty-state behavior, and targeted browser assertions that prove the full
markdown reader stays mounted during heading navigation. Earlier finalize
repairs already tightened the preview gate ordering so binary content is
rejected before richer-preview allowlisting and added a corrupt-`.json`
fixture so renamed binary supplements do not render as supported previews.

After the archive candidate reached `await_merge`, two additional issues
surfaced during human interactive use: the current archived plan package was
hidden even while the worktree was still non-idle, and explorer heading clicks
targeted a non-scrolling inner reader node so the visible reading area did not
jump. Repaired `/api/plan` to keep the current archived plan package readable
until the worktree becomes idle, updated the reader navigation to scroll the
actual nearest scrollable reading container, and expanded the Playwright smoke
fixture to cover archived-current-plan browsing plus visible heading alignment.

Focused rerun after the repair:

- `pnpm --dir web check`
- `go test ./internal/planui ./internal/ui`
- `scripts/ui-playwright-plan-smoke`

The Playwright rerun produced fresh screenshots under
`output/playwright/harness-ui-plan-smoke-20493-1775829253619805000-3623/`,
including `plan-scope.png` for heading navigation and
`plan-archived-notes.png` for archived-current-plan browsing.

After another round of human visual feedback, removed the trailing package
metadata block from the reader, changed explorer branch labels so title clicks
fold/unfold the branch instead of requiring precise chevron hits, replaced the
ASCII expand markers with proper chevrons, and added task-list rendering on
top of `markdown-it` so plan checklists display as checkboxes. Refreshed the
visual checklist wording to say "current plan package" and extended the smoke
flow to assert the metadata panel stays gone, the task-list checkboxes render,
and the `Scope` explorer label click actually collapses its child headings.
That first task-list styling pass used a flex layout for each task item, which
made inline content wrap awkwardly during real reading. Repaired the layout by
keeping normal text flow inside each list item and absolutely positioning the
checkbox instead.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is itself the validation and closeout slice;
the controller will still run a full-candidate review before archive rather
than treating the validation step as a substitute for candidate review.

## Validation Strategy

- Run focused Go tests for the new read-only plan resource and `/api/plan`
  server wiring.
- Run frontend checks and build steps for the updated `web/` app before
  rebuilding embedded assets.
- Add or update browser automation for page routing, empty state, explorer
  interaction, supported preview rendering, and unsupported or oversize
  handling.
- Run an interactive Playwright session against a live `harness ui` instance,
  capture screenshots of the major states, and use that pass to verify visual
  hierarchy and reading quality rather than relying on DOM assertions alone.

## Risks

- Risk: Parsing markdown headings into a stable explorer tree while keeping one
  full-document reader could create awkward selection or anchor behavior.
  - Mitigation: Define stable heading node IDs in the backend contract and
    validate navigation with both automated checks and interactive clicking.
- Risk: Supplement preview rules could sprawl into ad hoc per-extension logic.
  - Mitigation: Centralize a supported-extension list, explicit plain-text
    fallback rules, and one size threshold so capability growth stays
    intentional.
- Risk: The page could become visually noisy if the explorer tries to behave
  like a generic file browser instead of a plan reader.
  - Mitigation: Keep the product centered on one current plan package, use the
    established workbench language, and require screenshot-based visual review
    before closeout.

## Validation Summary

- Focused backend and server validation passed with `go test ./internal/planui
  ./internal/ui` after the current-archived-plan and reader-scroll repairs,
  on top of the earlier preview-contract and binary-gate fixes.
- Frontend checks passed with `pnpm --dir web check` and `pnpm --dir web
  build`, and the embedded UI assets were rebuilt before browser validation.
- Browser validation passed with `scripts/ui-playwright-plan-smoke`, including
  heading navigation against the actual reading-area scroll container,
  recursive supplements tree checks, archived-current-plan browsing,
  unsupported preview states, idle empty state, hidden package metadata, and
  rendered task-list checkboxes in the reader.
- Fresh browser artifacts were captured under
  `output/playwright/harness-ui-plan-smoke-22415-1775832353429795000-24157/`,
  including `plan-initial.png` and `plan-scope.png` for visual review.
- After CI later failed on stale generated contract artifacts, reran the exact
  failing path locally with `scripts/sync-contract-artifacts --check`,
  `go test ./tests/smoke -run TestSyncContractArtifactsCheckPassesForCurrentRepo`,
  and `go test ./internal/contractsync ./internal/planui ./internal/ui`.
- Revision `5` focused on the markdown reader regression exposed by human
  screenshots: reran `pnpm --dir web check` plus
  `scripts/ui-playwright-plan-smoke` after restoring inline-code flow,
  preserving fenced code blocks, and adding browser assertions that prove both
  inline code and fenced code keep the intended whitespace semantics.

## Review Summary

- `review-001-full` requested changes for three real issues: archived
  current-plan pointers still surfaced the wrong `/api/plan` behavior, the
  dedicated Plan smoke script treated nested supplement traversal as optional,
  and the heading-navigation assertion did not prove the full markdown reader
  stayed mounted.
- `review-002-full` requested one additional change after the first repair:
  allowlisted extensions could still bypass binary rejection and render renamed
  binary payloads as supported previews.
- `review-003-full` passed clean after the second repair. The final candidate
  was then reopened from `await_merge` for one more finalize-fix slice after
  human interactive testing exposed archived-current-plan visibility and
  heading-scroll issues.
- `review-004-full` passed for revision `2` after the reopen repair. It
  reported one non-blocking docs-consistency finding: the visual checklist in
  `docs/plans/active/supplements/2026-04-10-add-current-plan-browser-page/visual-checklist.yaml`
  still says "active plan package," which is narrower than the repaired
  current-plan behavior during merge handoff.
- Revision `3` addresses the remaining UX polish feedback from human
  interactive use: remove the noisy package-metadata panel, let explorer title
  clicks fold/unfold branch rows, replace the ASCII expand glyphs with real
  chevrons, and render markdown task lists as checkboxes.
- An initial `review-005-full` pass began for revision `3`, but before
  aggregation a fresh human screenshot surfaced one more real issue: the new
  task-list styling broke inline wrapping inside checklist items.
- After that layout repair landed, `review-005-full` aggregated with one
  blocking docs-consistency finding: the public `PlanResult.Artifacts` comment
  in `internal/contracts/plan_ui.go` still said "active-plan package paths"
  instead of "current-plan package paths".
- `review-006-delta` passed clean after the narrow follow-up fixed that public
  contract wording.
- When post-archive CI later failed because `schema/ui-resources/plan.schema.json`
  had not been regenerated after that contract wording change, the candidate
  was reopened again for a narrow finalize-fix slice.
- `review-007-delta` passed clean after regenerating the contract artifacts and
  verifying the exact `sync-contract-artifacts --check` failure path locally.
- Revision `5` reopened after human interactive testing surfaced one more real
  reader regression: global code styling was still breaking inline markdown
  code flow inside checklist prose.
- `review-008-delta` correctly requested changes because the first CSS repair
  restored inline code flow but still risked collapsing fenced code blocks in
  markdown readers, and the smoke did not yet prove the block-code half of the
  contract.
- `review-009-delta` passed clean after the follow-up restored preformatted
  fenced code styling and extended the Playwright smoke to assert both inline
  code and fenced code behavior explicitly.

## Archive Summary

- Archived At: 2026-04-10T22:50:05+08:00
- Revision: 5
- The earlier revision `2` archive candidate was reopened with
  `harness reopen --mode finalize-fix` because additional Plan-page UX polish
  still needed repair.
- The earlier revision `3` archive candidate was reopened with
  `harness reopen --mode finalize-fix` because post-archive CI found stale
  generated contract artifacts.
- The earlier revision `1` archive candidate was reopened with
  `harness reopen --mode finalize-fix` because merge-handoff behavior still
  needed repair.
- The earlier revision `4` archive candidate was reopened with
  `harness reopen --mode finalize-fix` because human screenshot review exposed
  a markdown-reader regression where inline code still rendered with block-like
  wrapping behavior.
- PR: [#134](https://github.com/catu-ai/easyharness/pull/134)
- Ready: Revision `5` has a clean finalize review in `review-009-delta`, and
  focused validation is green for the reader-style regression repair.
- Merge Handoff: Archive the repaired candidate, refresh publish/CI/sync
  evidence on PR
  [#134](https://github.com/catu-ai/easyharness/pull/134), and return to
  `execution/finalize/await_merge` to wait for explicit human merge approval.

## Outcome Summary

### Delivered

- Added a new `Plan` page to the read-only UI rail and workbench shell so the
  current tracked plan is browsable without leaving `harness ui`.
- Added a dedicated `/api/plan` read model and schema that expose the current
  plan markdown document, heading-based TOC tree, supplements directory tree,
  and explicit preview states for supported, fallback, and unsupported files.
- Implemented a VS Code-like explorer for plan headings and supplements while
  keeping the right pane as one document reader for the main plan and a file
  previewer for supplements.
- Corrected the plan-loading boundary so the current archived plan package
  stays visible during merge handoff while truly idle worktrees still show a
  clear empty state.
- Corrected heading navigation so explorer clicks scroll the visible reading
  area to the selected section instead of targeting a non-scrolling inner
  element.
- Removed the trailing package-metadata panel from the Plan reader so the page
  stays focused on document and supplement content.
- Updated explorer rows so clicking a branch title toggles fold/unfold without
  requiring precise chevron hits, and replaced the ASCII expand markers with
  proper chevron icons.
- Added task-list rendering on top of `markdown-it` so markdown checkboxes
  display as disabled reader checkboxes instead of raw `[ ]` and `[x]` text.
- Corrected the public Plan contract wording so the `PlanResult.Artifacts`
  comment now matches current-plan behavior during archived merge handoff.
- Regenerated the checked-in plan schema so contract-sync checks stay aligned
  with the updated current-plan wording in CI.
- Corrected the markdown reader styling so inline code stays readable inside
  flowing prose and checklist items without breaking fenced code blocks.
- Added focused validation for the new resource and browser flow, including
  archived-current-plan coverage, stronger heading-navigation assertions, fresh
  Playwright screenshots, binary-content rejection ahead of preview
  allowlisting, and explicit browser assertions for both inline-code and
  fenced-code rendering.

### Not Delivered

- Archived-plan browsing, history switching, or a plan-package picker are still
  deferred.
- Explorer expansion memory beyond one browser session is still deferred.
- Rich preview support for images, CSV, PDF, or other heavier supplement
  formats is still deferred.
- Search, filtering, or graphing inside plan content is still deferred.

### Follow-Up Issues

- [#133](https://github.com/catu-ai/easyharness/issues/133) tracks the
  intentionally deferred plan-browser enhancements, including archived-plan
  browsing, richer previews, expansion-state persistence, and in-page
  search/filtering work.
