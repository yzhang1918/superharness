---
template_version: 0.2.0
created_at: "2026-04-02T23:12:05+08:00"
source_type: issue
source_refs:
    - '#95'
---

# Hook review round data into the harness UI

## Goal

Replace the `Review` page placeholder with a real read-only workbench for the
current plan's review rounds. The page should help a human understand current
review state, compare rounds, inspect each reviewer slot's assigned task and
submitted result, and judge what needs attention next without falling back to
raw artifact spelunking.

This slice should follow the same product boundary as the live `Status` page:
the Go backend builds a read-only view model from existing harness-owned local
artifacts, and the frontend renders that model. It must not change CLI command
contracts, mutate review artifacts, or introduce new write-side indexing just
to support the UI.

## Scope

### In Scope

- Add a read-only review resource for `harness ui` that reads review rounds
  for the current tracked plan, including archived-but-not-landed candidates.
- Build a `Review` round browser with:
  - a round list in the navigation pane
  - an overview-first detail pane for the selected round
  - reviewer-focused content that combines each slot's assigned instructions
    with its submitted summary and findings
- Treat `manifest`, `ledger`, `aggregate`, and raw submission artifacts as
  supporting evidence rather than the page's primary organizing principle.
- Handle incomplete, in-progress, missing, or malformed review artifacts
  conservatively so the UI stays usable without pretending the data is clean.
- Add focused Go/unit coverage plus browser automation using the
  [$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill.
- Include manual interactive Playwright verification with screenshots so the
  final UI slice is checked for readability, density, and overall aesthetics,
  not just data correctness.

### Out of Scope

- Any change to `harness review start`, `harness review submit`,
  `harness review aggregate`, or their CLI JSON contracts.
- Any new write-side artifact, background indexing, or timeline/event mutation
  added solely for the `Review` page.
- Reading review rounds outside the current tracked plan.
- UI-triggered review actions, command execution, or local-state mutation.
- Turning the page into a raw artifact browser where manifest/ledger/aggregate
  tabs dominate the default experience.

## Acceptance Criteria

- [x] `Review` no longer renders as a WIP placeholder and instead loads review
      rounds for the current tracked plan, including archived candidates.
- [x] The page presents a round browser shape where the second column lists
      available rounds with high-signal metadata such as round id, kind,
      title/target, timestamp, and current decision or waiting state.
- [x] The selected round's detail pane defaults to an overview-first view.
- [x] The overview surface makes review kind, title/target, revision, and
      timing easy to read.
- [x] The overview surface shows aggregate decision when present, or a
      conservative in-progress / incomplete status when not.
- [x] The overview surface shows reviewer submission progress.
- [x] The overview surface highlights high-signal blocking and non-blocking
      findings when aggregate data exists.
- [x] The detail pane supports reviewer-focused inspection where each reviewer
      slot combines the task/instructions that reviewer received with the
      submission summary, findings, and locations that reviewer returned.
- [x] Reviewer slots with no submission yet render a clear empty or pending
      state.
- [x] Supporting artifact views remain available for manifest, ledger,
      aggregate, and raw submissions, but they are secondary to the overview
      and reviewer content.
- [x] In-progress rounds with only partial artifacts still appear with a clear
      waiting status.
- [x] Missing or malformed artifacts produce warnings or degraded sections
      rather than crashing the page or showing false-clean status.
- [x] The page remains read-only and never rewrites local state to "repair"
      damaged data.
- [x] The implementation does not change review CLI contracts or write-side
      logic beyond read-only UI wiring.
- [x] Focused Go coverage and Playwright automation validate review data
      loading, round selection, degraded-state rendering, and reviewer detail
      presentation.
- [x] Before closeout, the implementation is also exercised interactively via
      Playwright with captured screenshots and a quick aesthetic pass on
      spacing, hierarchy, and legibility.

## Deferred Items

- Reading review history across plans other than the current tracked plan.
- Deep file-anchor navigation from review finding locations into `Diff` or
  `Files`.
- Editing or triggering review actions from the UI.
- Rich side-by-side raw artifact diffing beyond the supporting evidence tabs.

## Work Breakdown

### Step 1: Define the read-only review resource and degraded-state rules

- Done: [x]

#### Objective

Lock the backend read-model boundary for current-plan review rounds and document
how incomplete or damaged artifacts should degrade in the UI.

#### Details

Follow the `status` pattern instead of the `timeline` write-side pattern. The
backend should detect the current tracked plan, enumerate only that plan's
review rounds from existing local artifacts, and assemble a UI-facing read
model without changing any review command contract or mutation behavior.

This step should make the degraded-state rules explicit: rounds may be waiting
for submissions, waiting for aggregation, missing aggregate data, missing
submission files, or partially malformed because a human or external tool
damaged local state. The resource should surface those conditions as warnings
and conservative status labels rather than failing the whole page whenever one
artifact is imperfect.

#### Expected Files

- `internal/ui/server.go`
- new read-only review resource file(s) under `internal/`
- `internal/ui/server_test.go`

#### Validation

- The review resource loads rounds only from the current tracked plan.
- A cold reader can tell from the plan and resource shape that no review CLI
  contracts or write-side logic need to change.
- Tests cover at least one clean round, one in-progress round, and one damaged
  or partially missing round.

#### Execution Notes

Added a new read-only `internal/reviewui` service plus `/api/review` wiring in
the UI server. The read model only inspects review rounds under the current
tracked plan and stays on the `status` pattern: no command-contract changes and
no new write-side runtime artifacts. The service now degrades conservatively
for missing review directories, malformed JSON artifacts, missing submissions,
and aggregate gaps while still returning the rest of the round browser data.

Focused backend coverage now locks the core states requested during discovery:
clean rounds, in-progress rounds, degraded rounds, archived-plan empty state,
and `/api/review` endpoint integration.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Backend read-model work was intentionally landed as part
of one integrated review-UI slice with the frontend round browser and browser
validation, so a step-local review would be artificially narrower than the
real user-visible change.

### Step 2: Replace the Review placeholder with the round browser UI

- Done: [x]

#### Objective

Ship the `Review` page as an overview-first round browser that centers review
content rather than raw artifact management.

#### Details

The second column should list rounds for the current plan, with enough metadata
to quickly distinguish step/finalize rounds, newer versus older rounds, and
clean versus waiting/problem states. The selected round's detail pane should
lead with a compact overview of review state, then let the user inspect each
reviewer slot in a cleaner vertical flow that keeps assigned instructions and
returned results in one reviewer tab without wasting horizontal space.

Raw artifact access should still exist, but as supporting evidence. The
frontend should avoid making `manifest`, `ledger`, and `aggregate` tabs feel
like the main event. The visual result should stay aligned with the existing
workbench shell: dense, calm, technical, and readable next to the already-live
`Status` and `Timeline` pages.

#### Expected Files

- `web/src/main.tsx`
- `web/src/styles.css`
- `internal/ui/static/*`

#### Validation

- The page defaults to the most relevant available round and renders a stable
  empty state when the current plan has no review rounds.
- Reviewer panes clearly show task plus result in one place and still make
  pending reviewers understandable.
- Supporting artifact views are present but visually secondary.

#### Execution Notes

Replaced the `Review` placeholder with a dedicated review workspace instead of
forcing it through the generic sidebar layout. The page now uses the accepted
product shape: round browser in the second column, overview-first detail pane
in the third column, tabbed reviewers, and secondary raw artifact tabs for
manifest/ledger/aggregate/submissions. A later finalize-fix repair then kept
archived current-plan rounds visible after `harness archive`, collapsed the
reviewer detail into a vertical flow closer to `Timeline`, and stripped back
the heavier card styling so the page reads more like the rest of the workbench.

#### Review Notes

NO_STEP_REVIEW_NEEDED: The round-browser UI depends directly on the Step 1
read model and Step 3 browser validation, so the meaningful review boundary is
the integrated slice rather than this frontend-only checkpoint.

### Step 3: Lock behavior and polish with automated and interactive browser validation

- Done: [x]

#### Objective

Prove the review UI with focused automation and a final interactive visual pass
instead of relying on static reasoning alone.

#### Details

Add browser coverage for round-list loading, round switching, reviewer detail
inspection, and degraded states such as missing aggregate or pending
submissions. Use the
[$playwright](/Users/yaozhang/.codex/skills/playwright/SKILL.md) skill for
browser automation, and keep the plan explicit that the controller should also
run an interactive local UI session, click through the page, capture
screenshots, and make any necessary aesthetic refinements before declaring the
slice complete.

This step is not just about "does it render"; it is also about whether the UI
looks intentional. The final pass should check layout balance, hierarchy,
density, and whether the review content feels more important than the support
artifacts.

#### Expected Files

- Playwright validation artifacts under `.local/`
- `scripts/ui-playwright-smoke`
- browser-focused test files under `web/` or existing UI test locations

#### Validation

- Automated Playwright coverage exercises clean and degraded review states.
- The controller runs an interactive Playwright session against the local UI.
- Screenshots exist for final inspection, and any obvious visual issues found
  during that pass are fixed before closeout.

#### Execution Notes

Added `scripts/ui-playwright-review-smoke` for review-specific browser
coverage and updated the existing `scripts/ui-playwright-smoke` expectations
now that `/review` is no longer a WIP placeholder. Validation includes:

- `pnpm --dir web check`
- `pnpm --dir web build`
- `scripts/ui-playwright-review-smoke`
- `scripts/ui-playwright-smoke`
- interactive Playwright inspection of the live review page with headed
  browsing plus captured screenshots under `output/playwright/manual-review-visual/`
- `go test ./...`

#### Review Notes

NO_STEP_REVIEW_NEEDED: Browser automation and visual polish only make sense
after the integrated review workspace exists, so this closeout is folded into
the later finalize review of the whole slice.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Add focused Go tests for the review read model and UI server endpoint.
- Add or extend Playwright automation for review round browsing, reviewer
  detail inspection, and degraded artifact states.
- Run an interactive Playwright session against the implemented UI, capture
  screenshots, and use that pass to tune aesthetics as needed.

## Risks

- Risk: Reading raw local review artifacts directly could make the page brittle
  when artifacts are incomplete or damaged.
  - Mitigation: Define explicit conservative degradation rules in the read
    model and test them directly.
- Risk: A raw-artifact-first UI could technically work while still failing the
  product goal of helping humans steer review work.
  - Mitigation: Keep the default experience overview-first and reviewer-first,
    with supporting artifacts clearly secondary.
- Risk: Browser automation could verify data rendering but miss visual
  awkwardness or hierarchy problems.
  - Mitigation: Require an interactive Playwright pass with screenshots before
    closeout, not just automated checks.

## Validation Summary

- `pnpm --dir web check` passed after the final explorer/status readability
  cleanup, the component-level screenshot baseline wiring, and the review
  smoke refactor away from silent `run-code` failures.
- `go test ./...` passed for the full repository after the reopen follow-up
  fixes, the new screenshot comparison helper, and the latest embedded UI
  assets were rebuilt into the repo-local harness binary.
- `scripts/ui-playwright-review-smoke` passed with populated review rounds,
  empty current-plan state, damaged artifacts, malformed submissions, real
  Playwright failure propagation, and component-level screenshot diffs for the
  active explorer row, a peer explorer row, and the reviewer panel.
- `scripts/ui-playwright-smoke` passed with the archived-plan review browser
  hidden during land cleanup, visible again for archived-but-not-landed state,
  and the broader status/timeline/review workbench paths still intact.
- Interactive headed Playwright inspection confirmed the review page now reads
  closer to `Status` / `Timeline`: the explorer column distinguishes rounds by
  round id plus revision, the detail pane keeps a vertical summary-first flow,
  reviewer tabs feel scannable, and task/result folds remain readable without
  wasting horizontal space. Captured screenshots live under
  `output/playwright/manual-review-visual-r15/`, and the manual scroll probe
  confirmed the third pane scrolls (`scrollTop: 237`, `scrollHeight: 1036`,
  `clientHeight: 799`).

## Review Summary

- Earlier finalize rounds established the review browser, conservative
  degraded-state handling, archived current-plan visibility, and the first
  review-specific Playwright smoke, but the finalize-fix reopen exposed a
  second wave of polish issues around archived-plan visibility, explorer row
  readability, and the visual weight of the review layout.
- `review-023-full`, `review-024-full`, and `review-025-full` progressively
  narrowed the remaining blockers from “status is still effectively
  color-first” and “visual smoke is not actually trustworthy” down to one
  concrete concern: the screenshot smoke needed to guard the actual polished
  components rather than sampled brightness heuristics.
- Those findings were fixed by making `run-code` failures real test failures,
  expanding the smoke to capture component screenshots, and checking the
  active explorer row, peer explorer row, and reviewer panel against committed
  fixture baselines under `scripts/testdata/review-visual/`.
- `review-026-full` then passed cleanly across `correctness`, `tests`, and
  `agent_ux` with zero blocking and zero non-blocking findings.

## Archive Summary

- Archived At: 2026-04-03T13:12:21+08:00
- Revision: 3
written back.
- Candidate: finalize-fix reopen of the review UI round-browser slice after
  post-archive product feedback on archived-plan visibility, explorer layout,
  and Playwright visual validation strength.
- PR: Existing branch/PR follow-up remains on `codex/review-ui-round-browser`
  and [#104](https://github.com/catu-ai/easyharness/pull/104) once this
  repaired candidate is re-archived and republished.
- Ready: Acceptance criteria are satisfied, `review-026-full` passed cleanly,
  the archived-plan/current-plan boundary now matches the approved scope, and
  automated plus manual Playwright validation both cover the repaired UI.
- Merge Handoff: Archive this repaired candidate, commit the tracked plan move
  plus the reopen closeout updates, refresh publish/CI/sync evidence on the
  existing PR, and drive the candidate back to
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Added a read-only `/api/review` resource and contract wiring that only reads
  review rounds for the current tracked plan without changing review CLI
  contracts or write-side behavior.
- Replaced the `Review` placeholder with an overview-first round browser that
  lists rounds, summarizes round status, and centers reviewer task/result
  content instead of raw artifact management.
- Added conservative degraded-state handling for missing, incomplete, and
  malformed review artifacts, including semantic validation for required JSON
  fields and reviewer warnings when ledger/submission state disagrees.
- Kept manifest, ledger, aggregate, and submission payloads available as
  supporting evidence while making their summaries visible directly in the UI
  so damaged-artifact diagnosis does not require raw JSON spelunking.
- Added focused backend coverage, `/api/review` integration coverage, review
  smoke automation, main UI smoke updates, and manual headed Playwright
  screenshots for final aesthetic validation.
- Repaired the reopen follow-up by keeping archived-but-not-landed current
  plans visible, hiding landed candidates conservatively during land cleanup,
  and aligning the review browser layout more closely with the calmer
  `Status` / `Timeline` shell.
- Cleaned up the explorer round list so repeated finalize titles stay
  navigable through round id, revision, and timestamp instead of reading like
  duplicate rows.
- Upgraded the reopened Playwright review smoke from brittle text-only checks
  to real failure-propagating browser assertions plus component-level visual
  baseline diffs for the explorer rows and reviewer panel.

### Not Delivered

- Review history browsing beyond the current tracked plan.
- Deep finding navigation into the future `Diff` and `Files` data surfaces.
- UI-triggered review actions or command handoff affordances.
- Rich side-by-side raw artifact diffing beyond the current supporting
  evidence tabs.

### Follow-Up Issues

- Issue [#103](https://github.com/catu-ai/easyharness/issues/103) tracks the
  deferred review-browser follow-ups from this slice: cross-plan history,
  deeper finding navigation, potential review actions, and richer artifact
  inspection.
- Issue [#91](https://github.com/catu-ai/easyharness/issues/91) remains the
  related dependency for wiring finding locations into future `Diff` / `Files`
  views.
