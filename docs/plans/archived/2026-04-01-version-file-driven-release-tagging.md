---
template_version: 0.2.0
created_at: "2026-04-01T09:37:52+08:00"
source_type: direct_request
source_refs:
    - https://github.com/catu-ai/easyharness/issues/87
---

# Drive releases from a tracked VERSION file

## Goal

Make `easyharness` releases feel like a conventional project release flow
instead of a tag-only maintainer ritual. The repository should carry a visible
`VERSION` file that represents the version to publish, and a dedicated release
PR should become the normal way to bump that value before a release.

When a release PR updates `VERSION` and merges to `main`, automation should
create the matching `v*` tag and reuse the existing tag-driven release
workflow to build GitHub Release assets and update the Homebrew tap. This
slice should document the release-PR convention clearly, but rely on team
agreement rather than adding new CI guardrails that block mixed `VERSION` and
code changes.

## Scope

### In Scope

- Add a repo-tracked `VERSION` file as the release entry point.
- Define how `VERSION` maps to the git tag and release artifact version.
- Add automation that observes `VERSION` changes on `main` and creates the
  matching tag when it does not already exist.
- Keep the existing tag-driven release workflow as the publishing path after
  the tag exists.
- Update maintainer-facing release docs to describe the dedicated release-PR
  convention and the automatic tag creation path.
- Add or update tests and workflow coverage needed to keep the new release
  entry path trustworthy.

### Out of Scope

- Defining a broader release cadence, support policy, or stable/beta promotion
  policy beyond this workflow change.
- Enforcing release-PR purity with CI file-path allowlists or protected branch
  rules.
- Replacing the existing release workflow with a fully version-file-driven
  publish job that skips git tags.
- Adding changelog generation, release-note templating, or semantic commit
  inference.
- Expanding distribution beyond the current GitHub Release plus Homebrew tap
  path.

## Acceptance Criteria

- [x] The repository includes a root `VERSION` file that serves as the human
      and automation entry point for release version bumps.
- [x] A merge to `main` that changes `VERSION` causes automation to create the
      matching `v*` git tag when that tag does not already exist.
- [x] The existing release publication flow continues to run from the created
      tag without regressing current GitHub Release or Homebrew behavior.
- [x] Release tooling and tests agree on one source of truth for the release
      version instead of scattering manual tag strings.
- [x] Release documentation explains the dedicated release-PR convention: bump
      `VERSION`, include any release docs updates, merge, then let automation
      create the tag and publish.

## Deferred Items

- CI enforcement that a release PR may only touch `VERSION` plus docs or other
  release materials.
- Automatic changelog synthesis or release-note curation from merged PRs.
- A future policy decision on when to cut stable versus prerelease tags.
- Any post-release workflow that automatically bumps `VERSION` again for the
  next planned iteration.

## Work Breakdown

### Step 1: Capture the VERSION-driven release contract in tracked form

- Done: [x]

#### Objective

Record the accepted release-entry model so future execution does not depend on
today's discovery chat.

#### Details

Discovery converged on a simple convention: `VERSION` becomes the visible
release bump knob, release PRs are expected to be separate from ordinary
feature work, and merging that release PR should automatically create the
matching tag. The plan must make clear that this slice is intentionally
workflow-focused rather than a broader release-policy rewrite. It should also
capture the explicit non-goal that team agreement, not CI enforcement, is the
only guardrail for keeping release PRs separate.

#### Expected Files

- `docs/plans/active/2026-04-01-version-file-driven-release-tagging.md`

#### Validation

- The tracked plan describes the release-PR convention, the intended
  automation chain, and the non-goals around policy and enforcement.

#### Execution Notes

Recorded the accepted discovery outcome in this tracked plan before starting
implementation. The durable contract is: add a root `VERSION` file, treat a
dedicated release PR as the normal version-bump entrypoint, auto-create the
matching release tag after that PR merges to `main`, and keep release-PR
separation as a team convention rather than a repository-enforced rule.

#### Review Notes

NO_STEP_REVIEW_NEEDED: planning-only step with no repository behavior or
implementation change beyond the tracked plan itself.

### Step 2: Add VERSION as the release source of truth and auto-tag entrypoint

- Done: [x]

#### Objective

Introduce the minimum repository and workflow changes needed for `VERSION`
bumps on `main` to create the matching release tag safely.

#### Details

Add a root `VERSION` file and wire the relevant tooling to treat it as the
canonical release identifier for the version-bump path. Introduce a workflow
that reacts to `VERSION` changes on `main`, reads the file content, validates
that it is a release-compatible `v*` value or can be mapped deterministically
to one, checks whether the tag already exists, and creates the tag only when
needed. Preserve the current tag-driven release workflow rather than merging
tag creation and publication into one monolithic job.

#### Expected Files

- `VERSION`
- `.github/workflows/*`
- `scripts/*`
- any small shared helpers needed by the release automation

#### Validation

- Add or update automated coverage for parsing and validating the `VERSION`
  file and for the auto-tag workflow behavior where practical.
- Verify that the new automation is idempotent when the tag already exists.
- Confirm the created tag matches the release version consumed by the existing
  publish workflow.

#### Execution Notes

Added a root `VERSION` file seeded to the latest published alpha without the
leading `v`, plus a `scripts/read-release-version` helper that validates the
file format and emits either the raw version or the matching tag. Added a
dedicated GitHub Actions workflow that watches `VERSION` changes on `main`,
resolves the matching `v*` tag, skips cleanly when the tag already exists, and
pushes the tag otherwise. Added smoke coverage for the helper, the repository
`VERSION` contract, and the workflow wiring. Because the branch also includes
release-doc alignment before step closeout, Step 2 should use a `full`
step-closeout review rather than a narrower delta pass.

#### Review Notes

`review-001-full` requested changes in `correctness` and `tests`. The round
identified four concrete gaps: loose VERSION validation that did not prove the
generated tag was a valid git ref, silent success when an existing tag pointed
at a different commit, missing negative-path coverage for malformed or missing
`VERSION` input, and no executable proof of the idempotent tag-exists path.
The repair batch tightened VERSION parsing with `git check-ref-format`,
factored tag creation into a reusable `scripts/create-release-tag-from-version`
helper that fails on commit mismatches and skips only when the tag already
matches the target commit, and added smoke coverage for those paths.
`review-002-delta` then passed with no remaining findings.

### Step 3: Align release docs and verification with the new entry path

- Done: [x]

#### Objective

Make the maintainer workflow legible and prove the new release entry path does
not drift from the existing publish pipeline.

#### Details

Update the README and release-maintainer docs so they describe the new normal
flow: merge release PR -> auto-create tag -> publish release. Adjust any
release tests or workflow fixtures that currently assume maintainers always
choose and push tags manually. Keep the documentation explicit that release PR
separation is a team convention, not a mechanically enforced repository rule.

#### Expected Files

- `README.md`
- `docs/releasing.md`
- `tests/smoke/*`
- optional workflow-test fixtures or helper tests

#### Validation

- `go test ./...` passes with the updated release path.
- Release-related tests cover the new `VERSION`-driven behavior where the
  repository currently has durable release workflow coverage.
- The docs are internally consistent about whether humans bump `VERSION`,
  create tags manually, or rely on automation.

#### Execution Notes

Updated `README.md` and `docs/releasing.md` so maintainers now have one
consistent story: a dedicated release PR bumps the bare `VERSION` file,
merging that PR to `main` triggers automatic `v*` tag creation, and the
existing `Release` workflow remains the publish path for GitHub Releases and
Homebrew updates. Validation passed with `go test ./tests/smoke -count=1` and
`go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the Step 3 doc changes were already covered by the
`docs_consistency` slot in `review-001-full`, and the remaining closeout work
for this step was validation-only.

## Validation Strategy

- Run `harness plan lint docs/plans/active/2026-04-01-version-file-driven-release-tagging.md`
  before execution starts and after material plan edits.
- During implementation, keep release workflow and smoke coverage green while
  introducing the `VERSION` file and auto-tag path.
- Before archive, verify the end-to-end release entry path on a safe candidate
  version: update `VERSION`, confirm automation creates the expected tag once,
  and confirm the existing release workflow still consumes that tag correctly.

## Risks

- Risk: The repository ends up with two competing sources of truth, one in
  `VERSION` and one in hand-chosen tags.
  - Mitigation: Update docs and automation so the version-bump path clearly
    flows from `VERSION` to tag creation, and keep the publishing workflow
    consuming that same tag.
- Risk: Automatic tag creation becomes noisy or unsafe when rerun.
  - Mitigation: Make the workflow idempotent by checking for the existing tag
    before creating anything.
- Risk: Contributors misread the release-PR convention as a hard guarantee.
  - Mitigation: Document plainly that release PR separation is a team
    agreement, not a repository-enforced rule in this slice.
- Risk: Tests only prove local helpers while the real GitHub workflow drifts.
  - Mitigation: Reuse existing release workflow tests and extend them around
    the new entrypoint rather than relying only on ad hoc scripts.

## Validation Summary

Validated the release-entry change in layers:

- `harness plan lint docs/plans/active/2026-04-01-version-file-driven-release-tagging.md`
- focused smoke coverage for `read-release-version`, executable tag creation,
  and workflow wiring
- `go test ./tests/smoke -count=1`
- `go test ./... -count=1`

The smoke suite now covers unprefixed repository `VERSION` content, malformed
and missing `VERSION` failures, valid-tag normalization, idempotent tag
creation when the tag already matches the target commit, mismatched-tag
failure, and the GitHub Actions workflow triggers and step ordering around the
VERSION-driven tag path.

This local validation proves the repo-owned helper semantics and the checked-in
workflow wiring. It does not simulate a real GitHub Actions push-to-`main`
execution end to end; that merged-release path still depends on remote Actions
running against the created tag after branch publication.

## Review Summary

Review history for this candidate:

- `review-001-full` requested changes in `correctness` and `tests`, catching
  loose git-ref validation, unsafe existing-tag handling, incomplete negative
  path coverage, and missing executable proof of the idempotent skip path.
- `review-002-delta` passed after the repair batch introduced stricter
  VERSION validation, the reusable `create-release-tag-from-version` helper,
  and stronger smoke coverage.
- `review-003-full` surfaced two finalize blockers: archive-facing plan
  placeholders that still needed durable summaries, and workflow wiring
  assertions that were present but not explicit enough about trigger scope and
  checkout/order guarantees.
- `review-004-delta` passed after the finalize repair batch filled the
  archive-facing summaries and strengthened the workflow smoke assertions to
  pin trigger scope, checkout, and step ordering.

## Archive Summary

- Archived At: 2026-04-01T13:05:47+08:00
- Revision: 1
The archive candidate converts release entry from a maintainer-pushed tag to a
repo-tracked `VERSION` bump on `main`, while preserving the existing
tag-driven `Release` workflow as the publish mechanism for GitHub Releases and
Homebrew updates. The candidate adds the root `VERSION` file, two repo-owned
tagging helpers, a dedicated tag-creation workflow, release docs for the new
release-PR convention, and smoke coverage for the helper and workflow wiring.
Finalize review is clean, the tracked plan lints, and the candidate is ready
to archive and move into publish/merge handoff.

- PR: NONE. The candidate has not been pushed or opened as a PR yet.
- Ready: Acceptance criteria are satisfied, the VERSION-driven release entry
  path is implemented and validated, and the remaining work is archive/publish
  handoff rather than further feature development.
- Merge Handoff: After archive, commit the tracked plan move, push branch
  `codex/version-file-release-tagging`, open or refresh the PR, and record
  publish/CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Root `VERSION` file with unprefixed release-version semantics.
- `scripts/read-release-version` for validated version/tag resolution.
- `scripts/create-release-tag-from-version` for idempotent tag creation with
  mismatched-tag failure.
- `.github/workflows/tag-release-from-version.yml` to create the matching
  `v*` tag when `VERSION` changes on `main`.
- README and release-maintainer docs describing the dedicated release-PR flow.
- Smoke coverage and full-suite validation for the new release entry path.

### Not Delivered

- Repository-enforced CI guardrails that block mixed `VERSION` and code
  changes in a release PR.
- A broader release cadence or beta/stable promotion policy.
- Automatic changelog generation or post-release version bump automation.

### Follow-Up Issues

- Existing issue [#87](https://github.com/catu-ai/easyharness/issues/87)
  remains the durable follow-up for the broader release cadence and
  alpha/beta/stable promotion policy that this slice intentionally did not
  resolve.
- Additional deferred scope still needs issue coverage before merge:
  repository-enforced release-PR guardrails, changelog generation, and
  post-release next-version automation. Creating those GitHub issues from this
  worktree is currently blocked because local `gh` authentication is missing
  (`gh issue list --repo catu-ai/easyharness` returned HTTP 401 on
  2026-04-01), so the merge handoff must either authenticate and file them or
  capture equivalent durable issue references before merge.
