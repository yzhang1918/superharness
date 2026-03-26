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

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Update the codebase and live docs for the catu-ai namespace

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Transfer the repository and publish an org-hosted prerelease

- Done: [ ]

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

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

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
