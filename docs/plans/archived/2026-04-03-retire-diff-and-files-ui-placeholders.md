---
template_version: 0.2.0
created_at: "2026-04-03T23:22:00+08:00"
source_type: issue
source_refs:
    - '#91'
---

# Retire the Diff and Files UI placeholders

## Goal

Realign `harness ui` with the product boundary we want to keep. The current UI
shell still advertises `Diff` and `Files` pages as explicit WIP placeholders,
but we no longer want `harness` to grow into a general-purpose file or diff
browser now that users already have mature IDEs such as VS Code for that work.

This slice should remove those placeholder pages and their navigation entry
points while preserving the existing IDE-like workbench aesthetic for the
remaining steering surfaces. The repository documentation and issue trail
should also reflect that this is a deliberate product decision, not an
unfinished implementation gap.

## Scope

### In Scope

- Remove the `Diff` and `Files` routes or navigation entries from `harness ui`.
- Remove the generic placeholder experience that currently keeps those pages
  visible as deferred work.
- Keep the current overall UI visual direction and information density for the
  remaining pages.
- Update repository documentation so it no longer promises `Diff` or `Files`
  as future read-only workbench pages.
- Update the UI proposal language so it keeps the IDE-like steering-surface
  aesthetic without implying that `harness` should become a general-purpose IDE
  file/diff browser.
- Leave a GitHub comment on issue `#91` explaining the direction change and
  close the issue.

### Out of Scope

- Redesigning the `Status`, `Timeline`, or `Review` pages.
- Reworking the top-level visual style away from the current IDE-like shell.
- Adding replacement diff/file features elsewhere in the product.
- Expanding this slice into a broader UI information-architecture rewrite.

## Acceptance Criteria

- [x] `harness ui` no longer exposes `Diff` or `Files` as visible pages or WIP
      placeholders.
- [x] The UI code no longer frames those two pages as deferred implementation
      work.
- [x] [README.md](/Users/yaozhang/.codex/worktrees/f58b/superharness/README.md)
      no longer says `Diff` and `Files` are placeholders waiting for future
      hookup.
- [x] [docs/specs/proposals/harness-ui-steering-surface.md](/Users/yaozhang/.codex/worktrees/f58b/superharness/docs/specs/proposals/harness-ui-steering-surface.md)
      preserves the IDE-like workbench direction but clearly treats general
      file/diff browsing as out of scope for `harness ui`.
- [x] Issue `#91` has a closing comment that records the product decision and
      the issue is closed.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Remove the Diff and Files placeholder surfaces from the UI shell

- Done: [x]

#### Objective

Update the frontend shell so only the pages we intend to support remain in the
navigation and route model.

#### Details

The implementation should preserve the current workbench feel for the remaining
pages and avoid a larger shell redesign. If the generic placeholder component
becomes unused after this change, remove or simplify it instead of leaving dead
UI scaffolding behind.

#### Expected Files

- `web/src/main.tsx`
- `web/src/styles.css`
- `internal/ui/static/*`

#### Validation

- `harness ui` only exposes the supported steering pages after rebuilding the
  embedded frontend assets.
- No empty or broken navigation state appears after removing the two pages.
- Update or add tests only if existing coverage depends on the removed routes.

#### Execution Notes

Removed `Diff` and `Files` from the frontend page enum, route validation, rail
icons, and placeholder rendering path so the shipped workbench now exposes only
`Status`, `Timeline`, and `Review`. Removed the unused placeholder CSS and
rebuilt `internal/ui/static/*`, then reinstalled the repo-local `harness`
binary so the embedded assets matched the updated UI shell. Focused validation:
`pnpm --dir web check`, `pnpm --dir web build`, `go test ./...`, `rg -n
"Diff|Files|placeholder|/diff|/files|pane-placeholder|placeholder-copy|diff,
and files" web/src web/index.html README.md
docs/specs/proposals/harness-ui-steering-surface.md internal/ui/static -S`,
and `harness ui --no-open --host 127.0.0.1 --port 4310` plus `curl` checks for
the root HTML and `/api/status`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this UI-shell reduction is a narrow, self-validating
slice and will receive a finalize review once the paired doc/issue closeout
step is included.

### Step 2: Align repository docs and issue history with the narrowed product boundary

- Done: [x]

#### Objective

Make the repository's durable documentation and GitHub record match the new
decision so future contributors do not keep treating issue `#91` as deferred UI
implementation work.

#### Details

The docs should keep the current IDE-like visual framing where it is still
useful, but they should no longer imply that `harness` plans to become a
general repository browser. The issue closeout comment should briefly explain
that we are intentionally relying on external IDEs for file/diff exploration
and keeping `harness ui` focused on steering surfaces.

#### Expected Files

- `README.md`
- `docs/specs/proposals/harness-ui-steering-surface.md`

#### Validation

- Documentation no longer advertises `Diff` and `Files` as pending pages.
- The proposal remains internally consistent about the intended product
  boundary.
- Issue `#91` is commented on and closed after the code/doc changes are ready.

#### Execution Notes

Updated `README.md` so the delivered UI surface no longer promises deferred
diff/file pages, and tightened the proposal to keep the IDE-like steering
surface language while dropping `Changes`/`Files` as planned product sections.
Left a GitHub comment on issue `#91` documenting the product decision to rely
on external IDEs for general file/diff browsing, then closed the issue.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this documentation and issue-history closeout is tightly
coupled to the Step 1 decision and is better reviewed together during finalize
review than as an isolated step.

## Validation Strategy

- Rebuild the embedded UI assets as part of the normal frontend flow used by
  the repository so the shipped binary reflects the updated navigation.
- Run the focused frontend/backend validation needed to confirm `harness ui`
  still starts cleanly after the route reduction.
- Manually verify the remaining navigation labels and empty states stay
  coherent.

## Risks

- Risk: Removing two top-level pages could leave stale references in docs,
  compiled assets, or route-selection logic.
  - Mitigation: Update the shell, rebuild embedded assets, and grep for
    remaining `Diff`/`Files` placeholder references before closeout.
- Risk: The proposal may still overstate IDE-style artifact browsing even after
  the code changes.
  - Mitigation: Tighten the proposal language in the same slice instead of
    leaving product intent split across code and docs.

## Validation Summary

- `pnpm --dir web check` passed after removing the `Diff` / `Files` page enum,
  route acceptance, rail icons, placeholder component path, and stale
  description text from the frontend.
- `pnpm --dir web build` passed and refreshed `internal/ui/static/*` with the
  reduced three-page workbench bundle.
- `go test ./...` passed for the full repository after rebuilding the embedded
  UI assets.
- `harness ui --no-open --host 127.0.0.1 --port 4310` started successfully,
  `curl /api/status` returned the live Step 1 execution state, and `curl /`
  confirmed the embedded HTML description no longer mentions diff/files.
- `rg -n "Diff|Files|placeholder|/diff|/files|pane-placeholder|placeholder-copy|diff, and files" web/src web/index.html README.md docs/specs/proposals/harness-ui-steering-surface.md internal/ui/static -S`
  returned no matches after the final rebuild.

## Review Summary

- `review-001-full` covered `correctness` and `docs_consistency` for the final
  archive candidate.
- The `correctness` reviewer verified that the source changes, embedded bundle,
  and runtime behavior all agree on removing `Diff` / `Files` while keeping the
  remaining steering pages coherent.
- The `docs_consistency` reviewer verified that `README.md`, the UI proposal,
  the tracked plan, and the issue-91 closeout all describe the same narrowed
  product boundary.
- `harness review aggregate --round review-001-full` passed cleanly with
  decision `pass`, zero blocking findings, and zero non-blocking findings.

## Archive Summary

- Archived At: 2026-04-03T23:29:17+08:00
- Revision: 1
- Candidate: retire the unused `Diff` / `Files` placeholder surfaces from
  `harness ui` and realign the repo docs plus issue history around the decision
  to leave general file/diff browsing to external IDEs.
- PR: Not opened yet. Create a PR for this archived candidate after the archive
  move is committed and pushed.
- Ready: Acceptance criteria are satisfied, `review-001-full` passed cleanly,
  the embedded UI assets were rebuilt into the repo-local harness binary, and
  the issue-91 closeout is already recorded on GitHub.
- Merge Handoff: Run `harness archive`, commit the tracked archive move plus
  closeout summaries, push the branch, open the PR, and then move the archived
  candidate to `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Removed the `Diff` and `Files` pages from the frontend page model, route
  validation, rail icons, and placeholder rendering path so the shipped UI now
  exposes only `Status`, `Timeline`, and `Review`.
- Removed the last placeholder-specific CSS and updated the embedded UI bundle
  and HTML metadata to match the narrower workbench surface.
- Updated `README.md` and the UI steering-surface proposal so they preserve the
  current IDE-like workbench direction without promising a general-purpose diff
  or file browser inside `harness ui`.
- Left a closing comment on GitHub issue `#91` documenting the product
  decision, then closed the issue.

### Not Delivered

- No replacement diff or file-browsing feature was added elsewhere in the UI.
- The overall IDE-like shell and the existing `Status`, `Timeline`, and
  `Review` page designs were intentionally left unchanged.

### Follow-Up Issues

NONE
