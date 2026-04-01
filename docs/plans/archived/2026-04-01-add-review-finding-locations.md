---
template_version: 0.2.0
created_at: "2026-04-01T21:14:00+08:00"
source_type: direct_request
source_refs:
    - '#90'
---

# Add location strings to review findings

## Goal

Extend reviewer submissions so each finding can optionally point at one or more
repo-relative source locations using a lightweight GitHub-style string format.
That gives controller agents, humans, and future UI surfaces a precise anchor
without turning the review contract into a heavy structured span model.

The new contract should stay intentionally small: `locations` is optional,
older submissions without locations remain valid, and aggregate artifacts
preserve the submitted location strings verbatim.

## Scope

### In Scope

- Add optional `locations` support to review findings in the review submission
  input, persisted submission artifact, and aggregate artifact contracts.
- Document the supported location string formats:
  - `path/to/file.go`
  - `path/to/file.go#L123`
  - `path/to/file.go#L1-L3`
- Keep validation lightweight so reviewers are not blocked by strict parsing or
  normalization.
- Preserve backward compatibility for existing reviewer artifacts and existing
  reviewers that do not emit locations.
- Update reviewer-facing skill/docs guidance and sync bootstrap-generated skill
  output.

### Out of Scope

- Introducing structured line/column objects, diff-side metadata, or multiple
  location syntaxes beyond the three agreed string forms.
- Building a new UI surface for clickable review findings.
- Requiring strict repo-path or line-range linting beyond basic non-empty
  string handling.

## Acceptance Criteria

- [x] `harness review submit` accepts findings with optional `locations:
      []string` while continuing to accept findings that omit the field.
- [x] The review submission and aggregate artifacts both preserve `locations`
      verbatim when present.
- [x] The schema/contracts/docs describe the three supported string forms and
      continue to allow older artifacts that omit `locations`.
- [x] Reviewer guidance examples teach the new field through the bootstrap
      source assets, and the repo's materialized `.agents/skills` output is
      resynced from `assets/bootstrap/`.
- [x] Focused tests cover valid location-bearing submissions, aggregation
      preservation, and backward-compatible handling of findings without
      locations.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Define the lightweight locations contract

- Done: [x]

#### Objective

Update the normative review contract and reviewer guidance to describe optional
`locations` arrays on findings using the agreed lightweight string syntax.

#### Details

Keep the contract small and reviewer-friendly: each finding may include
multiple location strings, but each string is just a GitHub-style path anchor.
Document the supported forms, state that paths are repo-relative by convention,
and keep validation intentionally lightweight rather than turning the CLI into a
strict location parser.

#### Expected Files

- `docs/specs/cli-contract.md`
- `docs/specs/state-model.md`
- `assets/bootstrap/skills/harness-reviewer/SKILL.md`
- `assets/bootstrap/skills/harness-execute/references/review-orchestration.md`

#### Validation

- The docs and reviewer guidance make the optional `locations` field and its
  three supported string forms unambiguous to a cold reader.
- The contract description stays compatible with findings that omit
  `locations`.

#### Execution Notes

Updated the CLI/state-model docs plus reviewer-facing bootstrap guidance to
describe optional finding `locations` arrays with the three agreed lightweight
string forms. Synced the materialized `.agents/skills/` output from
`assets/bootstrap/` after the source guidance changed.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only changed documentation and reviewer
guidance, and its contract wording was reviewed together with the Step 2
behavioral implementation that enforced the same payload shape.

### Step 2: Persist locations through review submission and aggregation

- Done: [x]

#### Objective

Teach the review contracts, schemas, and service layer to accept and preserve
finding locations without adding heavy parsing or normalization.

#### Details

Add `locations` to the Go review finding contract and generated schemas, then
thread the field through submission persistence and aggregate generation.
Validation should remain intentionally light so a reviewer is not blocked by
format nitpicks; the aggregate artifact should preserve the submitted strings
verbatim.

#### Expected Files

- `internal/contracts/review.go`
- `internal/review/service.go`
- `schema/inputs/review.submission.schema.json`
- `schema/artifacts/review-submission.schema.json`
- `schema/artifacts/review-aggregate.schema.json`
- `schema/commands/review.aggregate.result.schema.json`
- `schema/commands/review.submit.result.schema.json`
- `schema/index.json`

#### Validation

- A submission with `locations` is accepted and written to disk unchanged.
- Aggregation retains those `locations` on the corresponding blocking and
  non-blocking findings.
- A submission without `locations` remains valid and behaves the same as
  before.

#### Execution Notes

Added optional `locations []string` to review finding contracts, preserved them
through review submission and aggregation, and kept validation lightweight by
only rejecting blank location strings. Regenerated the checked-in contract
schemas from the Go contract source afterward.

#### Review Notes

`review-001-delta` surfaced two important findings: explicit empty location
arrays were not round-tripping verbatim, and the backward-compatible omitted
`locations` case plus artifact/aggregate schema surfaces were not pinned by
tests. The repair introduced presence-aware location round-tripping plus the
missing regressions, but `review-002-delta` then found that `locations: null`
slipped through validation and persisted invalid `null` artifacts. Added an
explicit validator rejection plus a raw JSON regression for that boundary, and
`review-003-delta` then passed cleanly with no findings.

### Step 3: Lock the behavior in with regression coverage and synced assets

- Done: [x]

#### Objective

Add focused tests and regenerate bootstrap materialized assets so the new
review contract stays stable.

#### Details

Cover the new field in review service tests and any schema/contract-sensitive
tests that assert finding payload shapes. After updating bootstrap source
assets, run `scripts/sync-bootstrap-assets` so the materialized `.agents/skills`
tree stays in sync with the source of truth.

#### Expected Files

- `internal/review/service_test.go`
- `.agents/skills/harness-reviewer/SKILL.md`
- `.agents/skills/harness-execute/references/review-orchestration.md`

#### Validation

- Focused tests fail before the implementation and pass after it.
- The synced `.agents/skills` output matches the updated bootstrap source.

#### Execution Notes

Extended review/contract sync tests for location-bearing findings, verified the
new validation boundary with focused package tests, resynced bootstrap assets,
and ran `go test ./...` successfully.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This step only added regression coverage and refreshed
generated/bootstrap outputs, and those changes were already exercised during
the Step 2 review loop plus the final `go test ./...` pass.

## Validation Strategy

- Lint the tracked plan with `harness plan lint`.
- Run focused Go tests for review submission and aggregation behavior.
- Inspect representative reviewer guidance/examples to confirm the new field is
  described consistently in bootstrap source and synced output.

## Risks

- Risk: A too-clever location model could add contract complexity without
  improving real reviewer ergonomics.
  - Mitigation: Keep the field optional, string-based, and lightly validated.
- Risk: Updating bootstrap source guidance without syncing generated assets
  could leave repo-local skills drifting from the intended contract.
  - Mitigation: Treat `assets/bootstrap/` as the source of truth and run
    `scripts/sync-bootstrap-assets` before closeout.

## Validation Summary

Validated the new review-finding locations contract in layers:

- `harness plan lint docs/plans/active/2026-04-01-add-review-finding-locations.md`
- `scripts/sync-contract-artifacts`
- `scripts/sync-bootstrap-assets`
- focused package validation with `go test ./internal/review` and
  `go test ./internal/contractsync`
- full repository validation with `go test ./...`

The resulting coverage now proves the intended optional-field behavior across
runtime and generated-contract surfaces: omitted `locations` remains valid,
non-empty location arrays persist through submission and aggregation, explicit
empty arrays round-trip when present, blank location strings are rejected, and
`locations: null` is rejected rather than being persisted as an invalid schema
shape.

Revision 2 reopened only for remote-sync repair after the archived candidate
was found to be behind `origin/main`. That repair merged the latest `README.md`
and `VERSION` changes from `origin/main` cleanly and revalidated the refreshed
candidate with another `go test ./...` pass before rerunning finalize review.

## Review Summary

Review history for this candidate:

- `review-001-delta` on Step 2 found two important issues: explicit empty
  location arrays were not preserved verbatim, and the backward-compatible
  omitted-field/runtime-schema boundary was not pinned by tests.
- `review-002-delta` on Step 2 found one blocker after the first repair:
  presence tracking had accidentally allowed `locations: null` to pass through
  validation and persist as invalid `null` artifacts.
- `review-003-delta` passed clean after the repair tightened validation around
  `locations: null` and added the missing raw-JSON regression.
- `review-004-full` passed clean as the finalize gate with no correctness,
  tests, or docs-consistency findings.
- `review-005-delta` on revision 2 found two reopen follow-up issues after the
  remote-sync merge: the reopened active plan still stamped `Revision: 1` in
  the archive summary, and the durable plan artifact was only present as an
  untracked active file while the archived copy was deleted.
- `review-006-delta` passed clean after the reopen closeout repair restamped
  the active archive summary as revision 2 and restored the active plan to a
  tracked git path before re-archiving.

## Archive Summary

- Archived At: 2026-04-01T22:09:52+08:00
- Revision: 2
- PR: [#100](https://github.com/catu-ai/easyharness/pull/100)
- Ready: Revision 2 cleanly merges the latest `origin/main` release metadata
  changes, keeps the review-finding `locations` contract intact, passed
  `review-006-delta` as the current finalize gate, and revalidated the updated
  branch with `go test ./...`.
- Merge Handoff: Re-archive revision 2, commit the tracked plan move plus the
  remote-sync merge, push the refreshed `codex/review-finding-locations`
  branch, refresh PR #100, and record publish/CI/sync evidence until
  `harness status` reaches `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Optional `locations []string` support on review findings using lightweight
  repo-relative string anchors such as `path/to/file.go`, `#L123`, and
  `#L1-L3`.
- Presence-aware submission and aggregation behavior that preserves explicit
  empty arrays while still omitting absent `locations`.
- Validation that rejects blank location strings and `locations: null`.
- Generated schema updates for the input, submission artifact, aggregate
  artifact, and related command-result surfaces.
- Updated CLI/spec/bootstrap guidance plus synced `.agents/skills` output.
- Regression coverage for omitted, non-empty, empty, blank, and `null`
  location cases, plus a passing `go test ./...` validation sweep.

### Not Delivered

- Richer structured span metadata such as columns, diff-side information, or
  multi-shape location objects.
- UI-specific clickable rendering or normalization beyond the agreed
  lightweight string contract.

### Follow-Up Issues

NONE
