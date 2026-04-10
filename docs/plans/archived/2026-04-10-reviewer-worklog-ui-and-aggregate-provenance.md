---
template_version: 0.2.0
created_at: "2026-04-10T09:02:00+08:00"
source_type: issue
source_refs:
    - '#125'
---

# Expose reviewer worklog detail and clearer aggregate provenance in the harness UI

## Goal

Extend the read-only `harness ui` review surface so humans can inspect each
reviewer slot's progressive worklog directly in the reviewer detail pane
instead of falling back to raw artifact spelunking. The UI should surface the
high-signal reviewer-progress fields already preserved in `submission.json`
while keeping the main reading path concise, structured, and clearly separate
from raw JSON.

This slice should also make aggregate findings on the review summary pane
easier to attribute to the originating reviewer and dimension. The current
summary shows aggregate comments, but the provenance does not stand out enough
when multiple reviewers contribute findings. The updated UI should make that
source context easier to scan without turning the overview into a second
reviewer-detail page.

## Scope

### In Scope

- Extend the review UI read model so reviewer detail pages receive normalized
  progressive-worklog fields from reviewer submissions.
- Surface reviewer context in the detail pane, including review kind and anchor
  information when available.
- Render reviewer progressive-worklog content in collapsible sections stacked
  vertically above or alongside the existing submitted summary and findings
  content, without changing the overall review-page shell.
- Keep raw submission payload access available through a secondary `Show raw
  JSON` entry point that opens outside the main content flow.
- Improve aggregate finding presentation on the round summary page so a human
  can more easily distinguish which reviewer slot or dimension produced each
  comment.
- Add or update backend/frontend validation for normalized reviewer worklog
  rendering, provenance labeling, and raw JSON access.

### Out of Scope

- Changing `harness review start`, `harness review submit`, or `harness review
  aggregate` write-side behavior or payload contracts.
- Adding progressive-worklog summaries to the round overview metrics or summary
  page beyond clearer reviewer/dimension provenance on aggregate findings.
- Introducing heuristic matching between candidate findings and final findings.
- Treating unknown extra submission fields as first-class main-UI content.
- Reworking the overall `harness ui` information architecture or adding new
  review actions.

## Acceptance Criteria

- [x] The read-only review UI contract exposes normalized reviewer-progress
      fields derived from the known `submission.json` worklog payload:
      `full_plan_read`, `checked_areas`, `open_questions`,
      `candidate_findings`, and relevant review context such as `kind` and
      `anchor_sha` when available.
- [x] The reviewer detail pane shows those normalized progressive-review
      fields in vertical, collapsible sections while preserving the existing
      assigned task, submitted summary, and final findings content.
- [x] The reviewer detail pane includes a secondary `Show raw JSON` control
      that reveals the raw reviewer submission payload outside the main reading
      flow.
- [x] The round summary page continues to stay overview-first, but aggregate
      findings are easier to distinguish by reviewer slot and/or dimension at a
      glance.
- [x] Unknown extra top-level submission fields do not become primary UI
      sections in this slice.
- [x] The page remains read-only and continues to degrade conservatively when
      review artifacts are partial, missing, or malformed.
- [x] Focused backend and frontend tests cover normalized worklog loading,
      reviewer-detail rendering, aggregate provenance display, and raw JSON
      access behavior.

## Deferred Items

- Heuristic or explicit linkage between progressive candidate findings and
  final findings beyond simple co-display.
- Elevating progressive reviewer-worklog summaries into the round overview.
- Rendering arbitrary unknown extra submission fields in the main reviewer UI.

## Work Breakdown

### Step 1: Extend the review read model with normalized reviewer worklog fields

- Done: [x]

#### Objective

Teach the read-only review resource how to extract the known progressive-review
fields from reviewer submissions and expose them to the frontend as stable UI
data.

#### Details

Keep this slice read-only and narrowly scoped to the already-agreed worklog
shape. The backend should normalize the known reviewer-owned fields already
preserved in submission artifacts rather than exposing arbitrary extra payloads
as first-class UI structure. In the same step, thread through the round/review
context a reviewer detail pane needs, especially review kind and delta anchor
context when the manifest records it.

The read model should also preserve access to the raw submission artifact
payload so the frontend can show it from a secondary control without requiring
artifact-tab spelunking. Unknown extra fields remain part of the raw payload
view, not the normalized UI contract.

#### Expected Files

- `internal/contracts/review_ui.go`
- `schema/ui-resources/review.schema.json`
- `internal/reviewui/service.go`
- `internal/reviewui/service_test.go`

#### Validation

- The review resource returns normalized reviewer-progress fields for clean
  reviewer submissions that include the known worklog structure.
- Delta rounds expose anchor context to the reviewer detail read model when
  the manifest records it.
- Missing or malformed progressive-worklog fields degrade conservatively
  without failing the rest of the round.
- Focused backend tests pin the normalized-field mapping and raw submission
  payload access path.

#### Execution Notes

Extended the read-only review UI contract with round-level `anchor_sha` plus
reviewer-level normalized `worklog` and `raw_submission` fields. Updated
`internal/reviewui` to extract known reviewer-progress fields from
`submission.json` (`full_plan_read`, `checked_areas`, `open_questions`,
`candidate_findings`, `review_kind`, and `anchor_sha`) while preserving the
raw submission payload for secondary UI inspection. Added focused backend and
handler coverage in `internal/reviewui/service_test.go` and
`internal/ui/server_test.go`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the read-model contract and reviewer-detail UI are one
tightly coupled slice, so a step-local review here would create an artificial
boundary before the integrated finalize review.

### Step 2: Add reviewer worklog sections and clearer aggregate provenance to the review UI

- Done: [x]

#### Objective

Update the review page so reviewer detail panes show progressive-review context
cleanly, while the summary pane makes aggregate comment provenance easier to
scan.

#### Details

Keep the round summary page overview-first. Do not add new summary metrics or
reviewer-progress digests there. Instead, improve the presentation of aggregate
findings so slot and/or dimension provenance reads more clearly when multiple
reviewers contribute comments.

In the reviewer detail pane, introduce vertical collapsible sections for the
normalized worklog fields. The arrangement should make it easy to inspect:
review context, covered areas/checkpoints, open questions, candidate findings,
and then the final returned findings. Raw JSON should be reachable through a
secondary `Show raw JSON` affordance that opens outside the main content area
instead of occupying a permanent tab or inline block.

#### Expected Files

- `web/src/types.ts`
- `web/src/helpers.ts`
- `web/src/pages.tsx`
- `web/src/workbench.tsx`
- `web/src/styles.css`
- `internal/ui/static/*`

#### Validation

- Reviewer detail panes render normalized worklog fields in readable
  collapsible sections without overwhelming the main review layout.
- The raw JSON control is clearly secondary and does not permanently consume
  main-pane space.
- Aggregate findings on the summary view are easier to distinguish by reviewer
  source and dimension in multi-reviewer rounds.
- The updated frontend builds successfully into the embedded UI bundle.

#### Execution Notes

Updated the review workspace so reviewer detail panes render progressive-review
content in vertical collapsible sections, keep `Assigned task` and `Returned
result` visible, and expose a secondary `Show raw JSON` overlay for the raw
submission payload. The summary view stayed overview-first, but aggregate
finding provenance is now surfaced as clearer source pills instead of a faint
single-line suffix. Refreshed the embedded frontend bundle under
`internal/ui/static/*`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the UI work depends directly on the new read-model
fields and is better reviewed as part of the integrated candidate than as an
isolated frontend checkpoint.

### Step 3: Lock behavior with focused validation and visual checks

- Done: [x]

#### Objective

Prove the new reviewer-detail and aggregate-provenance behavior with backend
tests, frontend checks, and browser-level validation.

#### Details

Add or update coverage at the right layers for this slice: backend tests for
the read model, frontend/static tests where they already exist, and browser
automation or manual UI checks to confirm the new sections, provenance labels,
and raw JSON affordance remain readable. Keep the validation aligned with the
existing `harness ui` workflow used in this repository rather than inventing a
new test path.

#### Expected Files

- `internal/reviewui/service_test.go`
- `internal/ui/server_test.go`
- `scripts/ui-playwright-review-smoke`
- browser-focused test files or snapshots under `.local/` as needed

#### Validation

- Automated validation covers at least one round with progressive worklog
  fields and one round with multiple reviewer findings where provenance must be
  visually distinguishable.
- The controller performs an interactive or browser-driven review of the new
  UI sections before closeout.
- Rebuilding the embedded frontend assets and the relevant repo tests pass
  before archive.

#### Execution Notes

Validation completed with `pnpm --dir web check`, `pnpm --dir web build`,
`scripts/sync-contract-artifacts`, `scripts/sync-contract-artifacts --check`,
focused `go test ./internal/reviewui`, focused `go test ./internal/ui -run
'TestNewHandlerServesReviewJSON'`, and repo-level `go test ./...` package runs
through the updated review/UI surfaces. Updated
`scripts/ui-playwright-review-smoke` for the new reviewer-detail sections, raw
JSON affordance, and current aggregate/degraded review expectations, then used
the produced browser snapshots under `output/playwright/ruis-73344-13642/` to
confirm the new reviewer worklog and provenance presentation. The smoke wrapper
appeared to hang in cleanup after reaching the final snapshot set, so the
snapshot outputs were used as the browser-validation evidence.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step is validation and smoke-maintenance for the
same integrated UI slice, so the meaningful review boundary is the branch-level
finalize review rather than a separate step-local pass.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Add focused Go coverage for the normalized review read model and any server
  surfaces touched by the new UI resource shape.
- Run the repository's existing frontend checks and rebuild the embedded UI
  assets.
- Run review-page browser validation and do a quick manual readability pass on
  collapsible sections, provenance labeling, and the raw JSON affordance.

## Risks

- Risk: The normalized worklog contract may drift from the actual submission
  payload shape and become brittle.
  - Mitigation: Normalize only the known agreed fields and keep raw submission
    access as a secondary escape hatch.
- Risk: Adding more content to reviewer detail panes could make the review page
  feel heavier or repetitive.
  - Mitigation: Keep round summary unchanged, use collapsible sections, and
    keep raw JSON outside the main flow.
- Risk: Provenance labels on aggregate findings may still be too subtle when
  multiple reviewers share similar themes.
  - Mitigation: Make slot/dimension attribution an explicit design target in
    both the read model and browser validation.

## Validation Summary

- Ran the focused contract, backend, and frontend validation path for the
  landed slice:
  `scripts/sync-contract-artifacts`,
  `scripts/sync-contract-artifacts --check`,
  `pnpm --dir web install --frozen-lockfile`,
  `pnpm --dir web check`,
  `pnpm --dir web build`,
  `go test ./internal/reviewui`,
  `go test ./internal/ui -run 'TestNewHandlerServesReviewJSON|TestNewHandlerServesReviewJSONWithMalformedWorklogWarnings'`.
- Finalize review also revalidated the focused touched surfaces through clean
  reviewer worklogs in `review-005-full`, including
  `go test ./internal/reviewui ./internal/ui`,
  `pnpm --dir web check`,
  `pnpm --dir web build`,
  and
  `scripts/sync-contract-artifacts --check`.
- Browser validation came from the updated
  `scripts/ui-playwright-review-smoke` flow and the generated evidence under
  `output/playwright/ruis-85410-32636/`, including the reviewer-detail and raw
  JSON snapshots (`review-ux.png`, `review-ux.yml`, `review-ux-raw.yml`) plus
  the aggregate provenance round snapshot (`review-positive-round.yml`).
- The Playwright smoke wrapper still appeared to hang during cleanup after the
  final snapshot set, so the produced browser artifacts were treated as the
  durable UI evidence for this slice.
- Revision 2 finalize-fix validation rechecked the bounded UI hierarchy repair
  and degraded-state artifact access path with:
  `pnpm --dir web check`,
  `pnpm --dir web build`,
  `go test ./internal/ui`,
  and repeated successful
  `scripts/ui-playwright-review-smoke`
  runs.
- The final repaired smoke evidence for revision 2 lives under
  `output/playwright/ruis-2149-8927/`
  and
  `output/playwright/ruis-97786-7201/`,
  including the restored round-level artifact overlay checks for
  waiting-for-aggregation and degraded rounds.

## Review Summary

- `review-001-full`
  - `changes_requested` after the `tests` reviewer found that the Playwright
    smoke was not yet pinning normalized reviewer worklog values from the main
    reviewer panel and that the aggregate provenance assertion was still too
    generic.
- `review-002-delta`
  - `pass` after tightening the smoke to assert the normalized reviewer-detail
    values and the rendered provenance affordance more directly.
- `review-003-full`
  - `changes_requested` after the `tests` reviewer found that slot-level
    provenance was still not forced to differ cleanly from the dimension label
    and that malformed worklog degradation still lacked explicit coverage.
- `review-004-delta`
  - `pass` after the positive smoke fixture moved to a distinct slot label
    (`risk_slot`) and focused malformed-worklog degradation coverage was added
    in both `internal/reviewui/service_test.go` and `internal/ui/server_test.go`.
- `review-005-full`
  - `pass` with no blocking or non-blocking findings across both
    `correctness` and `tests`.
- `review-006-full`
  - `changes_requested` after the `correctness` reviewer found that removing
    the review-page supporting-artifact surface made degraded round artifacts
    unreachable from the read-only review UI.
- `review-007-delta`
  - `changes_requested` after the `tests` reviewer found that the restored
    `Artifacts` entry existed in the UI but the Playwright smoke never opened
    or verified the new overlay path.
- `review-008-delta`
  - `pass` after the smoke was extended to open the round-level `Artifacts`
    overlay on waiting-for-aggregation and degraded rounds and verify the
    representative artifact payloads.

## Archive Summary

- Archived At: 2026-04-10T22:42:42+08:00
- Revision: 2
- PR: [#131](https://github.com/catu-ai/easyharness/pull/131)
- Ready: Revision 2 keeps the simplified reviewer-detail hierarchy, restores a
  secondary artifact-inspection path for degraded rounds through the new
  header-level `Artifacts` overlay, and closes the missing smoke coverage for
  that path. The acceptance criteria remain satisfied and the latest delta
  follow-up review passed cleanly.
- Merge Handoff: Archive the repaired active plan, commit the tracked rearchive
  move plus any updated embedded UI assets, push the branch, refresh PR #131,
  and record updated publish, CI, and sync evidence until the candidate
  returns to merge-ready handoff.

## Outcome Summary

### Delivered

- Extended the read-only review UI contract and service so reviewer detail
  pages receive normalized progressive-worklog fields, round/review context,
  and raw submission payload access from existing submission artifacts.
- Updated the review UI so reviewer detail panes show vertical collapsible
  sections for review context, covered areas, open questions, and candidate
  findings, while keeping the existing returned-summary and findings content in
  place.
- Added a secondary `Show raw JSON` overlay for raw submission inspection
  without promoting raw or unknown payload fields into the main reviewer UI.
- Improved aggregate finding attribution on the round summary page with clearer
  reviewer/dimension provenance pills so multiple reviewer comments are easier
  to distinguish at a glance.
- Locked the behavior with focused backend/service tests, `/api/review`
  handler coverage, schema sync, rebuilt embedded frontend assets, and the
  updated review Playwright smoke fixtures and assertions.
- Refined the reviewer-detail reading order so `Assigned task` and
  `Returned result` lead the page, while `Review process` remains a lighter
  secondary section with left-anchored collapsible rows.
- Replaced the removed always-visible supporting pane with a secondary
  round-level `Artifacts` overlay so degraded rounds still expose manifest,
  ledger, aggregate, and malformed submission payloads without bloating the
  main review surface.
- Extended the Playwright smoke to open and verify the restored `Artifacts`
  overlay on waiting-for-aggregation and degraded rounds before continuing the
  rest of the review flow.

### Not Delivered

- No heuristic or explicit linking between progressive candidate findings and
  final findings was added beyond simple co-display in the reviewer detail pane.
- The round summary page did not gain progressive reviewer-worklog digests or
  additional overview metrics.
- Unknown extra submission fields remain available only through the raw JSON
  overlay instead of becoming supported first-class main-pane sections.

### Follow-Up Issues

- #130 Track follow-up review UI work after reviewer worklog detail landing
  (`https://github.com/catu-ai/easyharness/issues/130`)
