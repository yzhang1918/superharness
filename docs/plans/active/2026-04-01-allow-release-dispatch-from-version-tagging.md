---
template_version: 0.2.0
created_at: 2026-04-01T21:24:05+08:00
source_type: direct_request
source_refs: []
---

# Allow VERSION Tag Workflow To Dispatch Release

## Goal

Repair the VERSION-driven release path so a merge to `main` that updates
`VERSION` can both create the new release tag and successfully dispatch the
existing `Release` workflow. The current workflow creates the tag but fails the
dispatch step with `HTTP 403: Resource not accessible by integration`, which
leaves the tag in place without a GitHub Release. While validating a manual
dispatch for `v0.1.0-alpha.6`, a second issue surfaced: the release workflow's
test environment leaks `EASYHARNESS_HOMEBREW_TAP_TOKEN` into a smoke test that
expects the token to be absent, causing the release run itself to fail.

This plan is still intentionally narrow. It should add the permission needed
for the workflow-owned `gh workflow run release.yml ...` call, stop the
release workflow's test step from leaking the Homebrew tap token into smoke
tests, isolate the one smoke test that incorrectly inherits the tap token,
validate those fixes, and leave the rest of the release chain unchanged.

## Scope

### In Scope

- Update `.github/workflows/tag-release-from-version.yml` so its
  `GITHUB_TOKEN` can dispatch `release.yml`.
- Update `.github/workflows/release.yml` so its test step does not inherit
  `EASYHARNESS_HOMEBREW_TAP_TOKEN`.
- Make the smoke test for the no-token Homebrew path explicitly clear
  `EASYHARNESS_HOMEBREW_TAP_TOKEN` so it stays valid inside the release job.
- Confirm the workflow continues to create tags and dispatch the `Release`
  workflow for new `VERSION` bumps.
- Record the current manual dispatch of `v0.1.0-alpha.6` as operational
  context in the plan closeout.

### Out of Scope

- Redesigning the release workflow chain.
- Reworking the one-time bootstrap failure for `v0.1.0-alpha.5`.
- Any product or CLI behavior changes unrelated to release automation
  permissions.

## Acceptance Criteria

- [ ] `.github/workflows/tag-release-from-version.yml` grants the minimal
      additional permission needed to dispatch `release.yml`.
- [ ] `.github/workflows/release.yml` keeps the Homebrew tap token available
      for publishing steps without leaking it into the workflow test step.
- [ ] The release smoke coverage no longer fails when the job environment
      provides `EASYHARNESS_HOMEBREW_TAP_TOKEN`.
- [ ] The tracked workflow definition clearly still creates tags before
      dispatching the release workflow.
- [ ] Focused validation and review show no unintended release-flow drift.
- [ ] The candidate reaches archived, published, merge-ready state for a small
      workflow-permission fix PR.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Grant workflow dispatch permission

- Done: [x]

#### Objective

Add the missing GitHub Actions permission required for
`gh workflow run release.yml ...` inside the VERSION-tag workflow.

#### Details

The change should be as small as possible: preserve the current workflow shape,
keep `contents: write`, and add the specific permission needed for workflow
dispatch instead of broader unrelated changes.

#### Expected Files

- `.github/workflows/tag-release-from-version.yml`
- `.github/workflows/release.yml`
- `tests/smoke/homebrew_formula_test.go`

#### Validation

- The workflow file clearly requests the permission needed for workflow
  dispatch.
- The release workflow test step explicitly clears the tap token while leaving
  later publish steps unchanged.
- The no-token Homebrew smoke test explicitly controls its environment so it
  still passes under release-workflow job env.

#### Execution Notes

Updated `.github/workflows/tag-release-from-version.yml` to add
`actions: write` alongside `contents: write`, which is the missing permission
for the workflow-owned `gh workflow run release.yml ...` dispatch call. While
preparing the durable fix, manually dispatched `release.yml` for
`v0.1.0-alpha.6` via run `23850865499` so the already-created tag can still
publish release assets. That manual run exposed a second issue: release-job env
injects `EASYHARNESS_HOMEBREW_TAP_TOKEN` into
`TestUpdateHomebrewTapWarnsWithoutToken`, so the test needs to explicitly clear
that variable when verifying the no-token path, and the release workflow's test
step also needs to clear the token so tagged source tests do not inherit the
publishing secret. Implemented both fixes by adding
`EASYHARNESS_HOMEBREW_TAP_TOKEN: ""` to the workflow test step and by clearing
that env var inside `TestUpdateHomebrewTapWarnsWithoutToken`. Validated with
`go test ./tests/smoke -run 'TestUpdateHomebrewTapWarnsWithoutToken|TestReleaseWorkflowWiresHomebrewTapPublishing' -count=1`
and with
`EASYHARNESS_HOMEBREW_TAP_TOKEN=dummy-token go test ./tests/smoke -run TestUpdateHomebrewTapWarnsWithoutToken -count=1`.

#### Review Notes

`review-001-delta` passed clean for `correctness` and `tests`. The permission
repair stayed narrow to `actions: write`, and the release-smoke isolation fix
now covers both the workflow test step and the targeted no-token smoke test.

### Step 2: Validate and hand off the workflow fix

- Done: [x]

#### Objective

Run focused validation and move the permission fix through review, archive, and
publish handoff.

#### Details

Validation can be narrow because the change is confined to a workflow
permission block. The plan closeout should also note that `v0.1.0-alpha.6` was
manually dispatched as an operational repair before this durable fix lands.

#### Expected Files

- `docs/plans/active/2026-04-01-allow-release-dispatch-from-version-tagging.md`

#### Validation

- Relevant local validation for workflow-file changes passes.
- Durable summaries capture both the root cause and the repaired permission.

#### Execution Notes

Validated the workflow repair locally with the targeted smoke coverage and then
used the patched branch workflow to manually dispatch
`release.yml` for `v0.1.0-alpha.6` via run `23851570624`. That run completed
successfully, published the prerelease at
`https://github.com/catu-ai/easyharness/releases/tag/v0.1.0-alpha.6`, and
passed the Homebrew verification job. This confirms both the dispatch
permission repair and the release-test environment isolation against the real
failure path.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 only recorded external validation and handoff for
the already-reviewed workflow/test fix. A full finalize review will still cover
the final candidate before archive.

## Validation Strategy

- Re-read the workflow file after the permission update to ensure the dispatch
  step still targets `release.yml` with the resolved tag input.
- Run the narrowest relevant local validation and rely on PR CI for broader
  confirmation.

## Risks

- Risk: Adding the wrong permission could still leave dispatch broken or make
  the workflow broader than intended.
  - Mitigation: Limit the change to the documented workflow-dispatch
    permission and keep the rest of the workflow untouched.

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
