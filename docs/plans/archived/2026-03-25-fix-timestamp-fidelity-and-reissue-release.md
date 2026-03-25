---
template_version: 0.2.0
created_at: "2026-03-25T00:05:25+08:00"
source_type: direct_request
source_refs: []
---

# Fix timestamp fidelity for release artifacts and plan templates

## Goal

Restore believable timestamp semantics for the two public-facing places where
`superharness` currently looks wrong: release archives unpack with `Jan 1 2000
00:00`, and many tracked plans record `created_at` at local midnight even
though they were created later in the day.

This slice should fix the forward behavior without rewriting history. Future
release artifacts and future plans should carry timestamps that match user
expectations, and the repository should publish a follow-up alpha release so
external testing no longer starts from the misleading `v0.1.0-alpha.1`
packaging metadata.

## Scope

### In Scope

- Diagnose and document the intended timestamp semantics for release artifacts
  versus tracked-plan metadata.
- Update release packaging so published archives no longer force a fake
  `2000-01-01 00:00` file timestamp while preserving deterministic release
  behavior where it still matters.
- Update plan-template generation so new tracked plans default to a believable
  creation timestamp instead of local midnight when callers seed only a date.
- Refresh tests, docs, and any agent-facing instructions that currently assume
  midnight plan timestamps or fixed year-2000 release mtimes.
- Cut and verify a replacement alpha release after the timestamp fixes land.

### Out of Scope

- Bulk-editing historical archived plans that already carry midnight
  `created_at` values.
- Rewriting or deleting `v0.1.0-alpha.1`; this slice should publish a newer
  alpha instead.
- Broader release-channel changes such as Homebrew, Apple notarization, or the
  pending project rename.
- General metadata cleanup unrelated to timestamps.

## Acceptance Criteria

- [x] The repository has an explicit forward-looking timestamp policy for both
      tracked-plan creation and release artifacts, including why historical
      plans are left unchanged.
- [x] Building release archives no longer makes unpacked files appear as `Jan 1
      2000 00:00`, and tests cover the intended replacement behavior.
- [x] `harness plan template` and the documented planning workflow no longer
      produce misleading midnight `created_at` values for ordinary new plans.
- [x] A new alpha release is published from the fixed codepath and verified as
      the recommended build for external testing.

## Deferred Items

- Historical cleanup of already-archived plan frontmatter.
- Any migration or compatibility work tied to a future repo rename or org move.
- Optional provenance enhancements beyond the timestamp semantics needed for
  honest public alpha artifacts.

## Work Breakdown

### Step 1: Align timestamp semantics and document the intended policy

- Done: [x]

#### Objective

Turn the observed timestamp complaints into a durable contract so execution
does not fix one surface while leaving the other ambiguous.

#### Details

The current behavior comes from two different intentional choices that no
longer match user expectations: `scripts/build-release` fixes archive entry
mtimes to `200001010000` for deterministic zips, and `harness plan template
--date` explicitly seeds `created_at` at local midnight. This step should
decide the new forward policy for both behaviors, explain why historical plan
files are not mass-edited, and capture the expected follow-up release version
shape for the replacement alpha.

#### Expected Files

- `docs/plans/active/2026-03-25-fix-timestamp-fidelity-and-reissue-release.md`
- `docs/releasing.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`
- `README.md`

#### Validation

- The tracked plan, release docs, and spec-adjacent docs all describe the same
  intended timestamp behavior.
- The plan records the decision to move forward without historical backfill and
  to publish a newer alpha release instead of mutating `v0.1.0-alpha.1`.

#### Execution Notes

Captured the forward policy directly in the tracked plan and the adjacent
release/spec docs: future release archives now use the tagged commit timestamp
instead of a fake year-2000 placeholder, future date-seeded plans should keep
the current local time-of-day instead of snapping to midnight, and historical
archived plans are intentionally left unchanged because `created_at` is durable
history rather than runtime state.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this policy step was implemented and validated together
with the Step 2 code changes, so a separate docs-only delta review would be
redundant.

### Step 2: Fix future plan and release timestamp generation

- Done: [x]

#### Objective

Change the packaging and planning codepaths so newly generated artifacts use
believable timestamps without losing the safety and determinism the current
implementation was trying to preserve.

#### Details

This step covers the actual implementation. For plans, prefer making normal
plan creation reflect the real creation time while keeping an explicit path for
tests or backfills that truly need a fixed timestamp. For release artifacts,
replace the year-2000 archive timestamps with the chosen forward policy and
update release smoke coverage so the new behavior is intentional rather than an
accidental side effect of a packaging-tool default. Any CLI help text, docs, or
skills that encode the old semantics should move in the same step so future
plans stop inheriting midnight metadata.

#### Expected Files

- `scripts/build-release`
- `tests/smoke/release_build_test.go`
- `internal/plan/template.go`
- `internal/plan/*_test.go`
- `internal/cli/app.go`
- `internal/cli/*_test.go`
- optional skill or doc files that currently instruct `--date`-driven plan
  creation

#### Validation

- Add or update focused tests for release-archive timestamp expectations.
- Add or update focused tests for `harness plan template` timestamp seeding and
  any changed CLI help/contract text.
- Run `go test ./internal/cli ./internal/plan ./tests/smoke -count=1`.
- Run `scripts/build-release --version <candidate> --output-dir .local/...`
  and inspect the produced archive metadata on the host platform.

#### Execution Notes

Added focused regression tests first, then changed `harness plan template
--date` to preserve the current local time-of-day on the requested date rather
than forcing `created_at` to midnight. Updated `scripts/build-release` so
staged release files and zip entry timestamps derive from the source commit
time in UTC, keeping deterministic packaging while replacing the misleading
`2000-01-01 00:00` metadata. Updated the release and CLI/spec docs to describe
the new policy. Focused validation passed with `go test ./internal/cli -run
'TestPlanTemplateDateSeedsCurrentLocalTimeOfDay' -count=1`, `go test
./tests/smoke -run 'TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary'
-count=1`, `go test ./internal/cli ./internal/plan ./tests/smoke -count=1`,
`scripts/build-release --version v0.1.0-alpha.2 --output-dir
.local/release-timestamp-check --platform $(go env GOOS)/$(go env GOARCH)`,
and `zipinfo -l` on the generated host archive showing `26-Mar-24 03:21`
instead of the old year-2000 timestamp.

#### Review Notes

`review-001-delta` requested changes after both reviewers caught that Info-ZIP
stores entry mtimes with 2-second precision, so the first exact-second
contract/test was too strict for odd-second commits. Follow-up commit
`8405f2c` narrowed the release docs and smoke assertions to
commit-derived timestamps within ZIP precision, and `review-002-delta` passed
cleanly with no remaining findings.

### Step 3: Publish and verify the replacement alpha release

- Done: [x]

#### Objective

Ship a new prerelease from the corrected codepath so external testing can move
to artifacts whose timestamps look sane.

#### Details

The expected outcome is a follow-up alpha release, likely `v0.1.0-alpha.2`
unless a stronger versioning reason appears during execution. This step should
rerun the documented release flow, verify the GitHub Release assets and
checksums, and confirm the unpacked macOS artifact no longer shows the
misleading fixed year-2000 timestamp behavior that prompted this slice.

#### Expected Files

- `docs/releasing.md`
- GitHub release/tag metadata for the replacement alpha
- optional README wording if the release recommendation text changes

#### Validation

- `go test ./... -count=1`
- Local release smoke for the chosen tag succeeds before publishing.
- The GitHub Release workflow succeeds for the replacement tag.
- Downloaded release assets and `harness --version` from the host binary match
  the new tag and no longer present the misleading timestamp issue.

#### Execution Notes

Ran `go test ./... -count=1`, committed the latest tracked plan closeout notes,
and published `v0.1.0-alpha.2` from commit `aec8d255afb9090ab45e347bc12d3144ff9dd137`.
The Release workflow succeeded at run `23500329107`, producing a prerelease at
`https://github.com/yzhang1918/superharness/releases/tag/v0.1.0-alpha.2` with
the expected four platform archives plus `SHA256SUMS`. Downloaded the published
`darwin/arm64` archive, verified its `zipinfo -l` timestamps now show
`26-Mar-24 16:23` instead of the old year-2000 placeholder, and ran the
packaged binary's `--version` successfully:
`version: v0.1.0-alpha.2`, `mode: release`,
`commit: aec8d255afb9090ab45e347bc12d3144ff9dd137`. To keep the release proof
durable for archive review, the authoritative downloaded artifact and derived
evidence now live under
`.local/harness/plans/2026-03-25-fix-timestamp-fidelity-and-reissue-release/release-verification/v0.1.0-alpha.2-aec8d25/`,
including the downloaded archive, `SHA256SUMS`, `release-view.json`,
`zipinfo.txt`, `version.txt`, and the locally recomputed SHA256 for the
downloaded `darwin/arm64` asset. That proof directory supersedes the earlier
scratch `alpha.2` checks under `.local/`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step exercised the already-reviewed release path,
verified the externally published assets directly, and the branch will still
receive a full pre-archive review before archive.

## Validation Strategy

- Use a mix of focused unit/smoke tests and end-to-end release verification.
- Validate timestamp semantics at the source of generation rather than only by
  eyeballing checked-in files.
- Re-check the published GitHub Release after tagging so the external user path
  is covered, not just local packaging.

## Risks

- Risk: Preserving deterministic packaging while changing archive timestamps may
  be trickier than expected.
  - Mitigation: Make the intended determinism boundary explicit in tests before
    shipping the replacement release.
- Risk: Changing `harness plan template --date` semantics could surprise any
  callers that intentionally relied on midnight.
  - Mitigation: Keep `--timestamp` as the explicit fixed-time path and update
    CLI/help text so the new distinction is obvious.
- Risk: A replacement alpha release could create confusion if docs do not steer
  users away from `v0.1.0-alpha.1`.
  - Mitigation: Document the follow-up release clearly and verify the new tag
    is the recommended test target.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-25-fix-timestamp-fidelity-and-reissue-release.md`
  passed when the plan was created and again during archive closeout after the
  durable summaries were refreshed.
- Focused regression coverage passed with
  `go test ./internal/cli -run 'TestPlanTemplateDateSeedsCurrentLocalTimeOfDay' -count=1`
  and
  `go test ./tests/smoke -run 'TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary' -count=1`,
  proving the new date-seeded plan timestamps and commit-derived ZIP mtimes.
- Combined targeted validation also passed with
  `go test ./internal/cli ./internal/plan ./tests/smoke -count=1` and
  `scripts/build-release --version v0.1.0-alpha.2 --output-dir .local/release-timestamp-check --platform $(go env GOOS)/$(go env GOARCH)`,
  where the generated host archive showed `26-Mar-24 03:21` rather than the
  old year-2000 placeholder.
- Full regression coverage passed with `go test ./... -count=1` before the
  replacement prerelease was cut.
- External release verification passed for GitHub Actions run `23500329107`
  after publishing `v0.1.0-alpha.2`: the downloaded `darwin/arm64` archive in
  `.local/harness/plans/2026-03-25-fix-timestamp-fidelity-and-reissue-release/release-verification/v0.1.0-alpha.2-aec8d25/`
  matches `SHA256SUMS`, `zipinfo -l` shows `26-Mar-24 16:23`, and the packaged
  binary reports `version: v0.1.0-alpha.2`, `mode: release`, and
  `commit: aec8d255afb9090ab45e347bc12d3144ff9dd137`.

## Review Summary

- `review-001-delta` requested changes after both reviewer slots caught that
  Info-ZIP stores mtimes with 2-second precision, so the first exact-second
  release-timestamp contract was too strict for odd-second commits.
- Follow-up commit `8405f2c` narrowed the ZIP timestamp contract to
  commit-derived UTC mtimes within ZIP precision, and `review-002-delta`
  passed cleanly with no remaining findings.
- `review-003-delta` then passed as the fresh `step_closeout` rerun for Step 2,
  confirming the timestamp semantics, docs, and focused coverage were strong
  enough to continue into release publication.
- `review-004-full` requested one finalize repair after the tests and
  docs-consistency slots showed that the active plan still pointed at older
  scratch `.local` alpha.2 checks instead of the actually published
  `aec8d255afb9090ab45e347bc12d3144ff9dd137` release artifacts.
- Commit `0f1b73a` refreshed Step 3 closeout around the authoritative downloaded
  alpha.2 proof directory, and `review-005-full` passed cleanly as the
  `review_fix` rerun.
- `review-006-full` then passed with zero blocking or non-blocking findings as
  the fresh `pre_archive` gate, leaving this revision in archive closeout with
  a clean finalize review.

## Archive Summary

- Archived At: 2026-03-25T00:37:37+08:00
- Revision: 1
- PR: https://github.com/yzhang1918/superharness/pull/47
- Ready: Revision 1 documents the forward timestamp contract, updates
  `harness plan template --date` to keep the current local time-of-day on the
  requested date, switches release packaging from the fake year-2000 mtime to
  commit-derived UTC timestamps within ZIP precision, and carries focused plus
  full regression coverage for both paths. The replacement prerelease
  `v0.1.0-alpha.2` is already published from commit
  `aec8d255afb9090ab45e347bc12d3144ff9dd137`, the downloaded release proof is
  stored under the plan-local `release-verification/` directory, and
  `review-006-full` has already passed as the fresh full `pre_archive` review.
- Merge Handoff: Refresh publish, CI, and sync evidence for PR #47 until
  `harness status` returns `execution/finalize/await_merge`, then wait for
  human merge approval before switching into `harness-land`.

## Outcome Summary

### Delivered

- Defined and documented a forward-looking timestamp policy that keeps future
  release artifacts and future tracked-plan `created_at` values believable
  while intentionally leaving historical archived plans untouched.
- Updated `harness plan template --date` so ordinary date-seeded plans preserve
  the current local time-of-day instead of snapping to local midnight, and
  added focused CLI regression coverage for that behavior.
- Updated `scripts/build-release` so staged files and ZIP entry mtimes derive
  from the source commit timestamp in UTC rather than the old
  `2000-01-01 00:00` placeholder, with smoke coverage that allows for ZIP's
  2-second timestamp precision.
- Published and externally verified `v0.1.0-alpha.2` as the replacement alpha
  release, including downloaded-asset checksum, archive timestamp, and packaged
  binary version checks stored under the plan-local release-verification proof
  directory.

### Not Delivered

- Historical archived plans were not bulk-edited or rewritten in this slice.
- `v0.1.0-alpha.1` was not modified or deleted; the fix-forward path is to use
  `v0.1.0-alpha.2` for external testing.
- Homebrew distribution, macOS notarization, repository rename, and org
  migration remain outside this timestamp-fidelity slice.
- Broader provenance enhancements beyond the current release tag, checksum, and
  build metadata proofs remain deferred.

### Follow-Up Issues

- `#46` tracks the deferred decision on whether historical tracked plan
  `created_at` timestamps should ever be backfilled or annotated.
- `#45` tracks the pending project and repository rename from `superharness`
  to `microharness`.
- `#44` tracks the evaluation of moving the project into a dedicated GitHub
  organization.
- `#32` tracks richer optional version/build metadata and related provenance
  improvements beyond the timestamp semantics delivered here.
