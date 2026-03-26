---
template_version: 0.2.0
created_at: "2026-03-26T11:27:38+08:00"
source_type: direct_request
source_refs:
    - 'issue #44'
---

# Move the repository into the catu-ai organization

## Goal

Move `microharness` from the personal `yzhang1918` namespace into the existing
`catu-ai` GitHub organization before Homebrew or other new distribution paths
solidify around the personal namespace. This slice should leave the project in
its intended long-term GitHub home so future release, install, and branding
work can build on a stable namespace.

The outcome should be a coherent org-owned identity: the repository URL and Go
module path move to `github.com/catu-ai/microharness`, live docs and release
guidance point at the org namespace, and a fresh prerelease from the
transferred repository proves the public test path works after the move. The
CLI command intentionally remains `harness`.

## Scope

### In Scope

- Transfer the GitHub repository from `yzhang1918/microharness` to
  `catu-ai/microharness`.
- Update the Go module path and live in-repo imports from
  `github.com/yzhang1918/microharness` to `github.com/catu-ai/microharness`.
- Update README, release docs, specs, workflow metadata, and other live
  references so public guidance uses the org namespace.
- Verify that existing public artifacts that matter for the near term still
  resolve or are replaced by a fresh org-hosted prerelease.
- Record the migration impact on deferred distribution work such as Homebrew so
  later execution can assume the org namespace from the start.

### Out of Scope

- Implementing Homebrew distribution or creating a tap.
- Renaming the project again or changing the `harness` CLI command.
- Creating a new organization; `catu-ai` already exists.
- Rewriting historical archived plans, historical releases, or old issue text
  solely to erase personal-namespace references from past context.
- Adding macOS notarization, Windows artifacts, or other new release channels
  unrelated to the org move itself.

## Acceptance Criteria

- [ ] The GitHub repository is owned by `catu-ai`, and live documentation points
      at `https://github.com/catu-ai/microharness` rather than the personal
      namespace except where historical context is intentional.
- [ ] `go.mod` and in-repo imports use `github.com/catu-ai/microharness`, and
      the repository still builds and tests successfully.
- [ ] Release and install guidance are updated for the org namespace, with the
      CLI command still documented as `harness`.
- [ ] A fresh prerelease from `catu-ai/microharness` is published and verified
      so future external testing and Homebrew work can build on the org-owned
      repository.

## Deferred Items

- `#42` Homebrew distribution and tap design.
- Any broader org governance work such as teams, permissions policy, or
  repository templates beyond what this transfer needs.
- Website/domain work, if any, beyond keeping repo and release links correct.
- Any revisit of the CLI executable name; this slice intentionally keeps
  `harness`.

## Work Breakdown

### Step 1: Lock the org-move contract and migration prerequisites

- Done: [x]

#### Objective

Turn the namespace decision into a durable execution contract and confirm the
transfer prerequisites before code or repo metadata drift starts.

#### Details

This step should make the migration boundary explicit: the repository and Go
module path move into `catu-ai`, the executable remains `harness`, and
Homebrew stays deferred until the org namespace is stable. It should also
confirm the concrete prerequisites for transfer in this environment, such as
whether the acting account can transfer the repo into `catu-ai`, whether any
org restrictions or naming conflicts exist, and which live URLs or assets need
to move together.

#### Expected Files

- `docs/plans/active/2026-03-26-move-repository-to-catu-ai-org.md`
- `README.md`
- `docs/releasing.md`

#### Validation

- The tracked plan clearly states what moves now versus later.
- The plan captures the transfer prerequisites and public surfaces that must
  stay coherent.
- Deferred work for Homebrew and broader org setup remains explicit instead of
  implied.

#### Execution Notes

Confirmed the migration prerequisites before touching the remote transfer:
`catu-ai` already existed as an organization, `yzhang1918` had active `admin`
membership there, `catu-ai/microharness` was still unclaimed, and the current
GitHub token had the `repo` and `read:org` scopes needed for repo transfer and
follow-up release work. The plan and live docs were also tightened around the
real boundary for this slice: move the repo and module path into `catu-ai`,
keep the executable name as `harness`, and leave Homebrew deferred until the
org namespace is stable.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this prerequisite/contract step was implemented as part
of the broader namespace and transfer slice, so a separate step-only review
would be redundant and less accurate than the later branch-level review.

### Step 2: Update the codebase and live docs for the catu-ai namespace

- Done: [x]

#### Objective

Make the repository internally consistent with the target org namespace before
the remote transfer happens.

#### Details

This step covers the repository-owned changes needed for a stable org move:
update `go.mod`, imports, release packaging references, repo URLs, and live
docs/spec text so the codebase already expects `github.com/catu-ai/microharness`.
Validation should focus on current codepaths and live operator docs, not on
historical archived content. Any installer or release smoke that currently
asserts the personal namespace should move to the org namespace while
preserving the `harness` binary name.

#### Expected Files

- `go.mod`
- `cmd/harness/main.go`
- `internal/**/*.go`
- `scripts/build-release`
- `scripts/install-dev-harness`
- `README.md`
- `docs/releasing.md`
- `docs/specs/**/*.md`
- `tests/**/*.go`

#### Validation

- Update or add targeted checks that enforce the new org module path and
  release expectations.
- Run `go test ./... -count=1`.
- Run a repo-scoped search to confirm live references moved to `catu-ai` while
  only intentional historical references remain under archived/history content.
- Run `scripts/build-release --version <candidate> --output-dir .local/... --platform $(go env GOOS)/$(go env GOARCH)`
  and verify the produced archive still packages `harness` correctly under the
  updated org-owned repo metadata.

#### Execution Notes

Moved the live repository-owned namespace from
`github.com/yzhang1918/microharness` to
`github.com/catu-ai/microharness` across `go.mod`, imports, tests, build/release
helpers, and live docs. The installer kept a compatibility bridge for both the
immediately previous personal `microharness` namespace and the older
`superharness` namespace so existing wrappers and symlink installs still count
as managed during upgrade. Validation passed with `go test ./tests/smoke -run
'TestInstallDevHarness(ReplacesLegacySymlinkedBinaryWithoutForce|WrapperDispatchesToCurrentWorktree)|TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary' -count=1`,
a host-platform `scripts/build-release --version v0.1.0-alpha.4 --output-dir
.local/release-org-transfer-check --platform $(go env GOOS)/$(go env GOARCH)`,
`unzip -l` on the generated archive confirming the packaged `harness` binary,
and a fresh `go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: once the repository transfer and org-hosted prerelease
joined the same candidate, a Step-2-only delta review would have been
misleading. The broader Step 3 review and final full review cover this
namespace migration in more realistic branch context.

### Step 3: Transfer the repository and publish an org-hosted prerelease

- Done: [x]

#### Objective

Finish the public move by transferring the repository into `catu-ai` and
proving the recommended release path works from the org namespace.

#### Details

Execution should transfer the GitHub repository to `catu-ai/microharness`,
update the local remote, and then publish a fresh alpha from the transferred
repository rather than relying only on redirects from earlier assets. The final
validation should prove that the org-owned repository URL resolves, PR/release
automation still works after transfer, and the downloaded release assets still
produce the `harness` binary with the expected version information.

#### Expected Files

- GitHub repository metadata for `catu-ai/microharness`
- GitHub prerelease/tag metadata for the next alpha after the current latest
  prerelease
- `README.md`
- `docs/releasing.md`

#### Validation

- The GitHub repository URL resolves under `catu-ai/microharness`.
- The release workflow succeeds for the post-transfer prerelease tag.
- Downloaded org-hosted release assets verify successfully, and the unpacked
  executable still reports the expected version via `./harness --version`.
- The README and release guide point external testers at the org namespace
  rather than the personal namespace.

#### Execution Notes

Transferred the repository from `yzhang1918/microharness` to
`catu-ai/microharness`, updated the local `origin` remote to
`git@github.com:catu-ai/microharness.git`, pushed
`codex/move-to-catu-ai-org`, and opened PR #49 in the org-owned repository. A
fresh prerelease tag `v0.1.0-alpha.4` was pushed from the transferred repo,
the release workflow succeeded, and the downloaded `darwin_arm64` archive from
`https://github.com/catu-ai/microharness/releases/tag/v0.1.0-alpha.4` verified
cleanly: the local checksum matched the published `SHA256SUMS`, the archive
contained the expected `microharness_v0.1.0-alpha.4_darwin_arm64/` root with
`harness`, and the unpacked binary reported `version: v0.1.0-alpha.4`,
`mode: release`, and commit `98f4fc2c0b75de3dbb238ac833e50ca3c3492bc3`. After
`review-001-full` flagged that the GitHub/org verification only lived in
`.local` artifacts, the branch gained a repo-owned
`scripts/verify-release-namespace` verifier backed by Go logic in
`scripts/releaseverify/` plus fake-`gh` smoke coverage. The same verifier now
also ran live against `catu-ai/microharness@v0.1.0-alpha.4`, downloading
`SHA256SUMS` and the `darwin_arm64` archive into the release-verification
artifact directory and proving the org-owned release path through a durable
checked-in command rather than ad hoc shell snippets alone. After
`review-005-full` asked for durable live coverage beyond the fake-`gh` smoke,
the branch added an opt-in live GitHub smoke test plus a `Release` workflow
step that enables it automatically after publishing assets. Local default
`go test ./...` remains offline because the live path only runs when
`MICROHARNESS_RUN_LIVE_GH_SMOKE=1`, while this execution also proved the live
test manually against `catu-ai/microharness@v0.1.0-alpha.4`. After
`review-006-full` requested one more repair, the live smoke now goes past
download and checksum verification by unpacking the published archive and
running the extracted `./harness --version`, so the durable automated proof
matches the original manual release-verification expectation end to end.

#### Review Notes

`review-001-full` requested changes because the org transfer and release proof
only existed in `.local` execution artifacts, not in a repo-owned verification
path. The follow-up added `scripts/verify-release-namespace`, fake-`gh` smoke
coverage in `tests/smoke/verify_release_namespace_test.go`, and a live rerun
against `catu-ai/microharness@v0.1.0-alpha.4`. `review-002-delta` then passed
cleanly with no findings, closing the first repair loop for Step 3.
`review-005-full` later requested one more fix because the live GitHub release
path still lacked durable automated coverage. The follow-up added an opt-in
live smoke test and wired it into `.github/workflows/release.yml`; a fresh full
review was required after that branch-level repair. `review-006-full` then
reduced the remaining gap to one tests finding: the live smoke downloaded the
published assets but still stopped short of unpacking the archive and running
the shipped binary. The current finalize repair closes that gap by extracting
the downloaded zip and asserting that the packaged `harness --version` reports
the expected prerelease version and `release` mode before the next fresh full
review.

## Validation Strategy

- Use focused grep/search checks to keep live references aligned on
  `catu-ai/microharness` while leaving historical archived context untouched.
- Run `go test ./... -count=1` after the namespace move and again before final
  handoff if release or transfer follow-up changes tracked files.
- Use a host-platform `scripts/build-release` smoke plus downloaded release
  verification to confirm that transfer-related metadata changes do not regress
  the packaged `harness` binary.
- Verify remote state directly with `gh repo view`, `gh release view`, and
  post-transfer CI evidence so archive handoff reflects the real org-owned
  repository.

## Risks

- Risk: The acting account may hit org transfer restrictions, permission gaps,
  or repo-name conflicts during the GitHub transfer.
  - Mitigation: Confirm transfer prerequisites up front in Step 1 and keep the
    repository-owned namespace changes separate from the actual transfer so the
    repair surface stays narrow if GitHub blocks the move.
- Risk: Some live docs or release/install references may keep pointing at the
  old personal namespace and confuse early testers.
  - Mitigation: Use repo-scoped live-reference searches plus release/download
    verification before archive.
- Risk: Publishing immediately after transfer may expose hidden workflow or
  permissions assumptions in Actions/release automation.
  - Mitigation: Treat a fresh org-hosted prerelease as part of the acceptance
    criteria rather than assuming redirects are sufficient.

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
