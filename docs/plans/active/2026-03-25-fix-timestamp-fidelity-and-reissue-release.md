---
template_version: 0.2.0
created_at: 2026-03-25T00:05:25+08:00
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

- [ ] The repository has an explicit forward-looking timestamp policy for both
      tracked-plan creation and release artifacts, including why historical
      plans are left unchanged.
- [ ] Building release archives no longer makes unpacked files appear as `Jan 1
      2000 00:00`, and tests cover the intended replacement behavior.
- [ ] `harness plan template` and the documented planning workflow no longer
      produce misleading midnight `created_at` values for ordinary new plans.
- [ ] A new alpha release is published from the fixed codepath and verified as
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

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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
