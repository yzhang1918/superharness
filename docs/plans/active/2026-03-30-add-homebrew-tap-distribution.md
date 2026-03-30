---
template_version: 0.2.0
created_at: "2026-03-30T14:03:45+08:00"
source_type: direct_request
source_refs:
    - '#42'
---

# Add Homebrew tap distribution for tagged releases

## Goal

Add a Homebrew installation path for public `easyharness` releases without
replacing the existing binary-first GitHub Release contract. The result should
let an external user install the current public release from a dedicated
`catu-ai` tap with `brew install catu-ai/tap/easyharness`, while the packaged
binary name remains `harness`.

This slice should stay focused on one default Homebrew channel backed by tagged
GitHub Releases. For now, the default formula should track the current alpha
line; when stable releases exist later, the same formula can move to the stable
line. Nightly or separate prerelease channels are explicitly deferred.

## Scope

### In Scope

- Define the public Homebrew contract for `easyharness`: formula name
  `easyharness`, installed executable `harness`, and install commands through a
  dedicated `catu-ai/homebrew-tap` repository.
- Add repo-owned automation that updates the tap formula from tagged GitHub
  Releases after release assets and checksums are published.
- Keep the formula pointed at the default release channel, which is alpha today
  and may become stable later without renaming the formula.
- Document the tap setup, install flow, upgrade flow, and maintainer
  prerequisites in the README and release guide.
- Add focused validation that the formula content matches the published release
  asset naming and checksum contract.

### Out of Scope

- Publishing to `homebrew/core`.
- A separate `easyharness-alpha` or `easyharness-nightly` formula.
- Nightly binary publishing or `--HEAD` Homebrew support.
- Replacing the existing GitHub Release archives as the primary public
  distribution surface.
- Broad release-pipeline redesign beyond the tap update needed for tagged
  releases.

## Acceptance Criteria

- [x] The tracked docs define one default Homebrew install path through a
      dedicated `catu-ai/homebrew-tap` tap, and they state that the formula is
      named `easyharness` while the installed executable remains `harness`.
- [x] Tagged releases in `catu-ai/easyharness` can update the tap formula
      automatically on GitHub alone, using an explicit cross-repo credential
      rather than hidden local steps.
- [x] The generated formula points at the published release asset and checksum
      contract for the supported Homebrew target archive and remains consistent
      with the release asset naming scheme.
- [x] The release and maintainer docs explain the prerequisites for tap
      publishing, including the required secret or app token and the expected
      repair path if a tap update fails.
- [x] The implementation includes deterministic validation for the formula
      rendering or update path and does not regress the current release
      workflow for non-Homebrew users.

## Deferred Items

- Stable-vs-prerelease channel splitting beyond the single default formula.
- Nightly install flows, including a separate nightly formula or `--HEAD`
  guidance.
- Submission to `homebrew/core`.
- Broader package-manager work beyond Homebrew.

## Work Breakdown

### Step 1: Lock the Homebrew contract and remote prerequisites

- Done: [x]

#### Objective

Capture the accepted Homebrew direction in tracked form so execution can
proceed without relying on discovery chat or hidden release assumptions.

#### Details

Discovery converged on a dedicated tap repo rather than a manual in-repo
formula or `homebrew/core` push. The public package name should be
`easyharness`, the installed binary should remain `harness`, and the default
formula should track the current release channel: alpha now, stable later if
the project begins shipping stable tags. This step should also record the
remote prerequisites that execution must satisfy: a `catu-ai/homebrew-tap`
repository, a cross-repo credential available to GitHub Actions in
`catu-ai/easyharness`, the tap branch fixed at `main` for this first slice,
and permission to publish commits into the tap.

#### Expected Files

- `docs/plans/active/2026-03-30-add-homebrew-tap-distribution.md`

#### Validation

- The tracked plan records the accepted package naming, channel policy, and
  remote prerequisites clearly enough for another agent to execute from the
  repository alone.

#### Execution Notes

Discovery concluded that `#42` should use a dedicated
`catu-ai/homebrew-tap` repo with GitHub-only automation from tagged releases.
The default formula remains `easyharness`; users install the `harness`
executable from that formula. The first slice should support the current alpha
line through the default formula, with a future stable release allowed to take
over the same formula. The tracked prerequisite contract also assumes the tap
branch is `main`, because the release workflow publishes to that branch
explicitly. Nightly and split prerelease channels stay deferred.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step records approved discovery decisions and
execution prerequisites in the tracked plan.

### Step 2: Add formula generation and release-workflow automation

- Done: [x]

#### Objective

Teach the repository-owned release flow how to render the Homebrew formula and
publish it into the tap after tagged release assets are available.

#### Details

Prefer thin repo-owned automation over hand-maintained shell history. The
release workflow should continue publishing GitHub Release assets first, then
derive the Homebrew formula update from the published release metadata and
checksums. Keep the formula implementation narrow: target the current supported
Homebrew archive, make the release URL and checksum explicit, and use one
cross-repo secret or app token to push into the tap repository. Add focused
tests or validation around any formula-rendering helper so refactors do not
silently break the tap output.

#### Expected Files

- `.github/workflows/release.yml`
- `scripts/` helper(s) for formula rendering or tap updates
- `tests/` or focused validation fixtures for Homebrew formula generation
- optional metadata files if the workflow needs a tracked formula template

#### Validation

- Add or update deterministic tests for formula rendering or release-metadata
  translation.
- Run the relevant script locally against a known release version or fixture to
  confirm the produced formula points at the expected asset URL and checksum.
- Verify the release workflow still passes static validation and preserves the
  existing archive publishing steps.

#### Execution Notes

Implemented the Homebrew automation path with Red/Green/Refactor TDD. Added a
small renderer at `scripts/homebrewformula/main.go` plus the
`scripts/render-homebrew-formula` wrapper so the release workflow can generate
`Formula/easyharness.rb` directly from the staged `dist/release/SHA256SUMS`
file and the tagged release version. Updated `.github/workflows/release.yml`
to render the formula after release publication, warn when
`EASYHARNESS_HOMEBREW_TAP_TOKEN` is missing, and push the updated formula into
`catu-ai/homebrew-tap` when the secret is configured. Added deterministic smoke
coverage in `tests/smoke/homebrew_formula_test.go` for both successful formula
rendering and the missing-checksum failure path. Validation passed with
`go test ./tests/smoke -run 'TestRenderHomebrewFormula|TestVerifyReleaseNamespace' -count=1`,
`go test ./... -count=1`, and a live render against the published
`v0.1.0-alpha.5` `SHA256SUMS`, producing
`.local/homebrew-formula-check/easyharness.rb` with the current release URLs
and checksums.

#### Review Notes

`review-001-delta` requested changes in the `correctness` and `tests` slots.
The blocking bug was the tap push path in `.github/workflows/release.yml`,
which used `git push origin HEAD` from a detached `actions/checkout` worktree.
The review also identified two real coverage gaps: the renderer smoke only
asserted two of the four platform asset branches, and the token-gated tap
update flow was not exercised deterministically.

The repair batch introduced `scripts/update-homebrew-tap`, which skips cleanly
when `EASYHARNESS_HOMEBREW_TAP_TOKEN` is absent, pushes with an explicit
`HEAD:refs/heads/<branch>` refspec, and takes the tap branch as an explicit
input so detached checkouts do not depend on `origin/HEAD`. The release
workflow now delegates the tap update to that script. The smoke suite now
asserts all four rendered asset URL/checksum pairs and covers both the
missing-token skip path and the detached-checkout commit/push path with local
git remotes. Follow-up validation passed with
`go test ./tests/smoke -run 'TestRenderHomebrewFormula|TestUpdateHomebrewTap|TestVerifyReleaseNamespace' -count=1`
before the full-suite rerun and fresh delta review. `review-002-delta`
requested one additional visibility fix because the refactor had removed the
documented GitHub Actions warning for missing tap-token runs. Restored the
`::warning title=Homebrew tap update skipped::...` annotation in
`scripts/update-homebrew-tap`, updated the skip test accordingly, and closed
the step with `review-003-delta`, which passed with no remaining findings.

### Step 3: Publish the tap contract and user-facing docs

- Done: [x]

#### Objective

Make the Homebrew path discoverable and operable for both users and release
maintainers.

#### Details

Update the live docs so they no longer say Homebrew is deferred. The README and
release guide should explain the install and upgrade commands, clarify that the
formula name is `easyharness` while the binary remains `harness`, and record
the maintainer prerequisites for cross-repo publishing. If execution creates
the tap repository during this slice, document its expected layout and formula
path so later maintainers can repair or rerun a failed update without reverse
engineering the workflow.

#### Expected Files

- `README.md`
- `docs/releasing.md`
- optional additional maintainer docs if the tap repo needs explicit operator
  guidance

#### Validation

- The live docs match the implemented Homebrew contract and no longer describe
  Homebrew as merely deferred.
- The install and upgrade commands are accurate for a cold reader.
- Any maintainer-only prerequisites are explicit rather than hidden in chat or
  shell history.

#### Execution Notes

Updated `README.md` and `docs/releasing.md` so the public install contract no
longer says Homebrew is merely deferred. The live docs now explain that users
install `easyharness` from `catu-ai/tap` while the installed executable
remains `harness`, that the default formula tracks the current public release
line (alpha today, stable later), and that maintainers need a public
`catu-ai/homebrew-tap` repo plus the `EASYHARNESS_HOMEBREW_TAP_TOKEN` secret
to let tagged releases publish `Formula/easyharness.rb`. The release guide
also records the repair path: fix the token or tap repo state, then rerun the
Release workflow for the same tag.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the README and release-guide changes were reviewed as
part of Step 2's broader delta review because the user-facing contract and the
release-workflow behavior changed together in one bounded slice.

## Validation Strategy

- Keep the existing `go test ./...` release baseline intact.
- Add narrow deterministic coverage for formula rendering or tap-update
  metadata rather than broad network-dependent tests.
- Validate the Homebrew docs against the actual formula name, binary name, and
  release asset naming scheme.

## Risks

- Risk: Cross-repo authentication for tap updates is the main operational
  dependency, and a missing or mis-scoped token can leave releases published
  without a matching formula update.
  - Mitigation: make the credential requirement explicit in tracked docs,
    design the workflow so release asset publication happens before the tap
    update, and document a manual repair path for re-running the formula sync.
- Risk: Locking the default formula to alpha tags today could create confusion
  when stable releases arrive later if the contract is not explicit.
  - Mitigation: document now that `easyharness` is the default channel, with
    alpha releases using that channel until a stable release replaces them.
- Risk: Homebrew support can sprawl into extra channels or release-policy work
  if the first slice does not stay narrow.
  - Mitigation: keep nightly, separate alpha formulas, and `homebrew/core`
    explicitly out of scope in the plan and docs.

## Validation Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Added deterministic smoke coverage in
  `tests/smoke/homebrew_formula_test.go` for formula rendering, missing
  checksum failure, token-gated tap updates, detached-checkout push behavior,
  release-workflow wiring, and live staged-tap Homebrew verification.
- `go test ./tests/smoke -run 'TestReleaseWorkflowWiresHomebrewTapPublishing|TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled' -count=1`
  passed after the finalize repair that removed the dead verify-job tap
  checkout and extended live Homebrew coverage to exercise install plus
  upgrade when a compatible earlier release exists.
- `EASYHARNESS_RUN_LIVE_BREW_SMOKE=1 EASYHARNESS_LIVE_GH_REPO=catu-ai/easyharness EASYHARNESS_LIVE_GH_TAG=v0.1.0-alpha.5 go test ./tests/smoke -run TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled -count=1`
  passed against the current public release after the smoke test began
  resolving only earlier releases that match the four-archive Homebrew
  contract.
- `go test ./... -count=1` passed after the final workflow, docs, and smoke
  repairs.

## Review Summary

UPDATE_REQUIRED_AFTER_REOPEN

- `review-001-delta` and `review-002-delta` requested changes in Step 2 for
  detached-checkout tap pushes, incomplete archive-matrix assertions, missing
  token-path coverage, and lost skip-warning visibility; those were fixed
  before `review-003-delta` passed clean.
- Finalize reviews `review-004-full` through `review-010-full` progressively
  tightened the GitHub Actions contract around secret gating, explicit tap
  branch handling, checkout-step coverage, published asset-matrix validation,
  and real staged-tap `brew install` execution.
- `review-011-full` requested changes because the macOS verify job still
  carried a dead tap-checkout step and the live Homebrew smoke did not cover
  the documented upgrade flow. The repair removed the dead workflow step and
  extended the live smoke to install an earlier compatible release before
  upgrading to the current tagged formula when possible.
- `review-012-full` requested one last closeout fix because the tracked plan
  still ended with archive placeholders. This revision writes the durable
  validation, review, archive, and outcome summaries plus the deferred issue
  handoff before rerunning finalize review.
- `review-013-full` requested two final smoke-hardening fixes: paginating the
  GitHub release lookup used to find prior upgrade candidates and asserting
  the live Homebrew job's required env wiring in the workflow smoke test.
- `review-014-full` passed clean across `correctness`, `tests`, and
  `docs_consistency`, so the candidate is now ready to archive.

## Archive Summary

UPDATE_REQUIRED_AFTER_REOPEN

- Archived At: 2026-03-30T15:23:35+08:00
- Revision: 1
- PR: not created yet; post-archive publish evidence should record the PR URL.
- Ready: `review-014-full` passed clean, acceptance criteria are satisfied,
  the release workflow now owns the Homebrew tap update path on GitHub alone,
  and the latest validation evidence covers formula render, tap update, live
  staged-tap install, and upgrade/test behavior against the current public
  release.
- Merge Handoff: archive the plan, commit the tracked move plus summary
  updates, push the branch, open or update the PR, and record publish/CI/sync
  evidence before treating the candidate as merge-ready.

## Outcome Summary

### Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Added repo-owned Homebrew formula rendering via
  `scripts/homebrewformula/main.go` and `scripts/render-homebrew-formula`,
  with `Formula/easyharness.rb` generated directly from tagged release
  metadata plus `SHA256SUMS`.
- Added `scripts/update-homebrew-tap` and release-workflow wiring so tagged
  releases can publish `Formula/easyharness.rb` into
  `catu-ai/homebrew-tap` on branch `main` using the explicit
  `EASYHARNESS_HOMEBREW_TAP_TOKEN` credential.
- Added deterministic and live validation for the Homebrew path, including
  release-workflow wiring checks, release-namespace asset verification,
  detached-checkout push coverage, and staged-tap `brew install` plus
  `brew upgrade` plus `brew test` smoke coverage when a compatible prior
  release exists.
- Updated `README.md` and `docs/releasing.md` so the public contract now
  documents `brew install catu-ai/tap/easyharness`, the `harness` executable
  name, maintainer prerequisites for tap publishing, and the repair path when
  the tap update secret or repo state is wrong.

### Not Delivered

UPDATE_REQUIRED_AFTER_REOPEN

- Separate stable-versus-prerelease Homebrew channels.
- Nightly Homebrew distribution, including `--HEAD` guidance or a dedicated
  nightly formula.
- Submission to `homebrew/core`.
- Package-manager distribution beyond Homebrew.

### Follow-Up Issues

UPDATE_REQUIRED_AFTER_REOPEN

- `#61` Decide whether Homebrew should split stable and prerelease channels.
- `#62` Evaluate nightly Homebrew distribution options.
- `#64` Assess readiness for eventual Homebrew/core submission.
- `#63` Evaluate package-manager distribution beyond Homebrew.

