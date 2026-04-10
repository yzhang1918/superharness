---
template_version: 0.2.0
created_at: "2026-04-10T23:05:00+08:00"
source_type: direct_request
source_refs: []
---

# Deduplicate PR CI triggers

## Goal

Remove the redundant double execution of the `CI / Go Test` check on pull
requests while preserving an explicit CI signal for direct updates to `main`.

This slice is intentionally narrow and limited to GitHub Actions trigger
configuration. The expected end state is that pull requests run the existing
Go test workflow once via the `pull_request` event, while `main` still runs
the same workflow on direct pushes.

## Scope

### In Scope

- Update `.github/workflows/ci.yml` trigger configuration to avoid duplicate
  PR executions caused by firing on both `push` and `pull_request`.
- Preserve the current `go test ./...` job behavior and check name.
- Validate the workflow syntax after the trigger change.

### Out of Scope

- Adding new test jobs, matrices, caches, or release automation behavior.
- Changing branch protection settings or GitHub repository-level policy.
- Modifying release workflows or the VERSION-driven tag automation.

## Acceptance Criteria

- [ ] Pull request updates trigger the `CI / Go Test` workflow once rather
      than once for `push` and once for `pull_request`.
- [ ] Direct pushes to `main` still trigger the existing CI workflow.
- [ ] The workflow continues to run `go test ./...` with no additional job
      behavior changes.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Narrow the CI trigger surface

- Done: [x]

#### Objective

Update the CI workflow triggers so PRs no longer run duplicate Go test checks.

#### Details

Prefer the smallest trigger-only change that preserves current behavior for
PR validation and `main` branch validation. The intended direction from
discovery is to keep `pull_request` and restrict `push` to `main`.

#### Expected Files

- `.github/workflows/ci.yml`

#### Validation

- The workflow definition clearly expresses `pull_request` plus `push` on
  `main` only.
- The Go test job definition remains unchanged apart from any formatting
  needed around the trigger block.

#### Execution Notes

Restricted the `push` trigger in `.github/workflows/ci.yml` to the `main`
branch while keeping `pull_request` unchanged. This preserves CI on direct
updates to `main` and removes the duplicate PR execution path created by
running the same workflow for both `push` and `pull_request`.

Local validation for this trigger-only change used:
`ruby -e 'require "yaml"; YAML.load_file(".github/workflows/ci.yml"); puts "yaml-ok"'`
and a manual diff review confirming only the `on:` block changed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step is a one-line trigger-scope adjustment in a
single workflow file, and finalize review will still cover the complete slice
before archive.

### Step 2: Revalidate the workflow change

- Done: [ ]

#### Objective

Confirm the workflow remains valid and leave the plan ready for execution
closeout.

#### Details

Use the smallest practical validation for a trigger-only Actions change.
Validation may be local if it is sufficient to catch YAML or workflow-shape
regressions before the branch is pushed.

#### Expected Files

- `docs/plans/active/2026-04-10-deduplicate-pr-ci-triggers.md`

#### Validation

- The workflow file passes the chosen validation command or lint check.
- The plan records what was validated and any limits of local validation.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Validate the workflow file locally after editing it.
- Re-read the trigger block to confirm only the event surface changed and the
  Go test job still runs the same command.

## Risks

- Risk: Narrowing `push` incorrectly could suppress CI on branches where the
  repository still expects it.
  - Mitigation: keep `pull_request` coverage for all PRs and preserve `push`
    coverage on `main`, which matches the agreed goal from discovery.
- Risk: A trigger-only edit might accidentally change job naming or behavior
  and break expected branch protection wiring.
  - Mitigation: keep the job body unchanged and limit edits to the `on:` block.

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
