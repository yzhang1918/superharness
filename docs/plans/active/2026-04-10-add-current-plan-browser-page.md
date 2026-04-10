---
template_version: 0.2.0
created_at: "2026-04-10T09:43:25+08:00"
source_type: direct_request
source_refs: []
---

# Add a current-plan browser page to harness UI

## Goal

Add a new read-only `Plan` page to `harness ui` so the current active tracked
plan becomes a first-class browsing surface alongside `Status`, `Timeline`,
and `Review`. The page should help a human read the active plan package
without dropping into the filesystem: browse the main markdown plan by heading
hierarchy, inspect companion `supplements/<plan-stem>/` content, and keep the
experience aligned with the existing workbench shell.

This slice should follow the same product boundary as the existing UI pages:
the Go backend assembles a read-only view model from the active plan package,
and the frontend renders that model. Prefer the clean target design over
compatibility layers or fallback behavior that preserves older UI assumptions.

## Scope

### In Scope

- Add a read-only `Plan` resource for `harness ui` that loads only the current
  active tracked plan package.
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
- Keep the page usable when no active plan exists by rendering a clear empty
  state instead of falling back to archived plans.
- Validate the slice with focused automated coverage and an interactive
  Playwright pass that includes real clicks, screenshots, and visual review.

### Out of Scope

- Browsing archived plans or showing a recent archived-plan fallback when the
  worktree is idle.
- Any UI-triggered write, command execution, plan mutation, or supplement
  editing flow.
- Turning the page into a generic repository file browser outside the active
  plan package.
- Rich preview support for images, CSV datasets, PDFs, or other binary-heavy
  artifacts in this first slice.
- Adding compatibility shims that preserve the old three-page UI assumption
  when a cleaner four-page workbench is available.

## Acceptance Criteria

- [ ] `harness ui` exposes a new read-only `Plan` page in the page rail.
- [ ] The page reads only the current active tracked plan package and never
      falls back to archived plans.
- [ ] When no active plan exists, the page renders a clear empty state that
      explains there is no current plan to browse.
- [ ] The left explorer presents a hierarchical, collapsible navigation tree
      that includes the main plan heading structure and, when present, a
      `supplements/` folder subtree.
- [ ] The main plan heading tree defaults to an expanded depth that surfaces
      headings through `H3` while still allowing deeper nodes to be expanded
      on demand.
- [ ] Selecting a plan heading keeps the full markdown document in the reader
      and navigates to the chosen section instead of replacing the document
      with an isolated fragment.
- [ ] Selecting a supplement file replaces the reader content with that file's
      preview while preserving the workbench shell and explorer selection.
- [ ] `md`, `txt`, `json`, `yaml`, and `yml` render as supported previews.
- [ ] Text-readable files outside the richer preview allowlist degrade to
      plain-text rendering without pretending to provide rich semantics.
- [ ] Binary files, image files, unsupported formats, and files above the
      configured preview-size threshold render a clear `not supported` state.
- [ ] The implementation introduces or updates automated tests that cover the
      read model, active-plan empty state, file support and size-threshold
      gating, and core page interactions.
- [ ] Before closeout, the page is exercised interactively with Playwright:
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
backend should derive the current active plan, load the markdown file plus any
matching `supplements/<plan-stem>/` directory, and expose a UI-facing payload
without changing plan lifecycle or write-side contracts. Make the preview
policy explicit in code and tests so a future agent can grow the supported
extensions list intentionally rather than by accident.

This step should define the decision rules for:

- active-plan-only loading
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

- The resource loads only the active tracked plan package and returns a stable
  empty-state payload when the worktree is idle.
- Tests cover heading extraction, supplement enumeration, supported preview
  files, plain-text fallback, unsupported binary/image files, and oversize
  files.
- A cold reader can tell from the contract that the page is read-only and tied
  to the current active plan package rather than generic repo browsing.

#### Execution Notes

Added a dedicated `internal/planui` read-only service plus `/api/plan` server
wiring and a public `PlanResult` contract/schema. The backend now loads only
the active tracked plan package, emits a heading tree for the main markdown
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
`scripts/ui-playwright-plan-smoke` browser flow for active-plan browsing and
empty-state behavior, and an updated shared `scripts/ui-playwright-smoke`
assertion so the repo's existing review-hidden wording matches the live UI.
Ran `scripts/ui-playwright-plan-smoke` and `scripts/ui-playwright-smoke`, then
performed a headed interactive Playwright pass against the current worktree's
`Plan` page, capturing screenshots under `output/playwright/manual-plan-review/`
for the main document view plus markdown and YAML supplement previews.

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
  - Mitigation: Keep the product centered on one active plan package, use the
    established workbench language, and require screenshot-based visual review
    before closeout.

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
