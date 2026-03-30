---
template_version: 0.2.0
created_at: "2026-03-30T14:03:45+08:00"
source_type: direct_request
source_refs:
  - "#42"
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

- [ ] The tracked docs define one default Homebrew install path through a
      dedicated `catu-ai/homebrew-tap` tap, and they state that the formula is
      named `easyharness` while the installed executable remains `harness`.
- [ ] Tagged releases in `catu-ai/easyharness` can update the tap formula
      automatically on GitHub alone, using an explicit cross-repo credential
      rather than hidden local steps.
- [ ] The generated formula points at the published release asset and checksum
      contract for the supported Homebrew target archive and remains consistent
      with the release asset naming scheme.
- [ ] The release and maintainer docs explain the prerequisites for tap
      publishing, including the required secret or app token and the expected
      repair path if a tap update fails.
- [ ] The implementation includes deterministic validation for the formula
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
`catu-ai/easyharness`, and permission to publish commits into the tap.

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
over the same formula. Nightly and split prerelease channels stay deferred.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step records approved discovery decisions and
execution prerequisites in the tracked plan.

### Step 2: Add formula generation and release-workflow automation

- Done: [ ]

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

PENDING_STEP_REVIEW

### Step 3: Publish the tap contract and user-facing docs

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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
