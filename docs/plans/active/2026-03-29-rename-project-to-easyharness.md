---
template_version: 0.2.0
created_at: "2026-03-29T23:59:00+08:00"
source_type: direct_request
source_refs: []
---

# Rename the project and repository to easyharness

## Goal

Evaluate and, if approved, execute a second public rename from
`microharness` to `easyharness` so the project name communicates the user value
more directly on first contact. The rename should leave the project with a
clearer user-facing identity while preserving the existing workflow model:
git-tracked plans, disposable local trajectory, evidence-first review, and the
stable `harness` executable name.

The outcome should be a coherent public identity: the repository and module
path move to `github.com/catu-ai/easyharness`, release archives are published
as `easyharness_<version>_<goos>_<goarch>.zip`, live docs explain that users
still run `harness`, and the durable proposal memo records why the rename is
being pursued from a user-mental-model perspective rather than from a purely
architectural naming preference.

## Scope

### In Scope

- Keep a durable naming rationale in tracked docs that explains why
  `easyharness` is favored from a user-mental-model perspective.
- Rename the GitHub repository branding from `microharness` to
  `easyharness`.
- Update the Go module path and all live in-repo imports from
  `github.com/catu-ai/microharness` to `github.com/catu-ai/easyharness`.
- Update README, specs, release docs, tests, and packaging metadata so live
  references use `easyharness`.
- Update release packaging and smoke coverage so archives are published as
  `easyharness_<version>_<goos>_<goarch>.zip` while the packaged executable
  remains `harness`.
- Publish and verify a fresh prerelease from the renamed repository after the
  rename lands.

### Out of Scope

- Renaming the CLI executable from `harness` to `easyharness`.
- Rewriting archived plans or old release notes solely to erase historical
  `superharness` or `microharness` references.
- Homebrew distribution or other new install channels unrelated to the rename.
- Website, domain, or broader marketing work beyond the repository-owned docs
  and release surfaces.
- Any further rename exploration after `easyharness`; this slice assumes the
  naming direction has already converged.

## Acceptance Criteria

- [ ] A durable tracked proposal explains the user-facing rationale for
      preferring `easyharness`, and the tracked execution plan stays aligned
      with that rationale.
- [ ] The GitHub repository, live tracked docs, module path, and live codepaths
      align on `easyharness`, with only intentional historical references left
      under archived materials.
- [ ] `go.mod` and all live in-repo imports use
      `github.com/catu-ai/easyharness`, and the repository still builds and
      tests successfully after the rename.
- [ ] Release packaging, installer behavior, workflow docs, and smoke coverage
      publish and validate `easyharness_*` archives while preserving `harness`
      as the executable name.
- [ ] A fresh prerelease from the renamed repository is published and verified
      as the recommended public test artifact after the rename.

## Deferred Items

- `#42` Homebrew distribution and tap naming or install flow.
- Any website, domain, or broader brand rollout outside repository-owned docs
  and release surfaces.
- Any future revisit of the `harness` executable name; this slice
  intentionally keeps it stable.
- Cleanup of historical archived docs, releases, or issue text that mention
  `microharness` as past context.

## Work Breakdown

### Step 1: Lock the rename rationale and live-name boundaries

- Done: [x]

#### Objective

Turn the current naming recommendation into a durable execution contract so a
future agent can explain both why `easyharness` was chosen and what must stay
unchanged during the rename.

#### Details

This step should connect the proposal memo, tracked plan, and live docs around
the same boundary: the project, repository, module path, and release asset
names become `easyharness`, while the executable remains `harness` and
historical archived material stays intact. The step should also make the
user-facing rationale explicit so the rename does not look like a cosmetic
preference with hidden scope, and it should confirm the concrete remote
prerequisites before later execution assumes the GitHub rename will succeed.

#### Expected Files

- `docs/specs/proposals/project-name-easyharness.md`
- `docs/specs/index.md`
- `docs/plans/active/2026-03-29-rename-project-to-easyharness.md`
- `README.md`
- `AGENTS.md`
- `docs/releasing.md`
- `docs/specs/cli-contract.md`
- `docs/specs/plan-schema.md`

#### Validation

- The proposal memo and tracked plan describe the same rename rationale.
- Live docs explicitly state that `easyharness` is the project name while
  `harness` remains the executable name.
- Deferred items stay explicit instead of being implied by the rename.
- The plan captures remote rename prerequisites such as repository-name
  availability, acting-account permissions, and the public surfaces that must
  move together.

#### Execution Notes

Confirmed the remote rename prerequisites before the mechanical rename: `gh`
auth is active for `yzhang1918`, the token carries `repo` and `read:org`
scopes, `gh repo view catu-ai/microharness --json ...` reports
`viewerPermission: ADMIN`, and `gh repo view catu-ai/easyharness` currently
fails with "Could not resolve to a Repository", which confirms the target name
is unclaimed. Locked the live naming boundary in tracked docs by adding the
durable `easyharness` naming proposal/index entries and updating README,
AGENTS, the release guide, and the CLI/plan specs so they state that the
project name becomes `easyharness` while the executable remains `harness`.
TDD was not practical for this step because it only changes tracked
documentation and execution planning rather than runtime behavior.

#### Review Notes

NO_STEP_REVIEW_NEEDED: this step only tightened tracked docs, naming rationale,
and remote prerequisites, and the broader branch review after the mechanical
rename will cover these boundary changes in more realistic context.

### Step 2: Rename the codebase, module path, and packaging metadata

- Done: [ ]

#### Objective

Update the repository-owned code, imports, packaging outputs, and live docs so
the working tree becomes internally consistent with the new `easyharness`
identity while preserving the stable `harness` CLI.

#### Details

This step covers the mechanical rename inside the repository: move `go.mod`,
update imports, refresh repo URLs, rename release package roots, and keep tests
aligned with the new expectations. Validation should target live codepaths and
current docs rather than archived historical content. Compatibility behavior in
the installer should continue to recognize legacy managed installs that still
point at earlier `microharness` or `superharness` checkouts when that matters
for upgrade flow.

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

- Update or add targeted checks that enforce the new module path and release
  asset naming.
- Run `go test ./... -count=1`.
- Run a repo-scoped search to confirm live references moved to `easyharness`
  while historical references remain only where intentionally preserved.
- Run `scripts/build-release --version <candidate> --output-dir .local/... --platform $(go env GOOS)/$(go env GOARCH)`
  and verify the produced archive name uses `easyharness_...` while the
  packaged executable is still `harness`.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: Rename the GitHub repository and publish the renamed prerelease

- Done: [ ]

#### Objective

Finish the public rename by aligning the remote repository identity and the
recommended external test release with the `easyharness` name after the remote
rename prerequisites are confirmed.

#### Details

Execution should coordinate the actual GitHub repository rename with the
already-updated tracked docs and module path so the public URLs settle around
`easyharness` before further distribution work hardens around `microharness`.
The release follow-up should publish a new alpha from the renamed repository
instead of rewriting earlier release history. Final validation should prove
that the renamed release assets, checksums, and version output all match the
new repo identity while the binary name remains `harness`.

#### Expected Files

- GitHub repository metadata for `catu-ai/easyharness`
- GitHub prerelease/tag metadata for the next alpha after the current latest
  prerelease
- `README.md`
- `docs/releasing.md`

#### Validation

- The GitHub repository URL resolves under `catu-ai/easyharness`.
- The release workflow succeeds for the post-rename prerelease tag.
- Downloaded release assets use `easyharness_*` naming, checksum verification
  passes, and the unpacked executable still reports the expected version via
  `./harness --version`.
- The README and release guide point external testers at the renamed release
  path rather than the old project name.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Validate the tracked plan with `harness plan lint`.
- Use repo-scoped searches to confirm that live references move together while
  historical archived context remains intentionally untouched.
- Run focused release and installer smoke coverage before relying on the new
  public name.
- Run a full `go test ./... -count=1` pass before archive.
- Verify a fresh prerelease download after the repo rename rather than relying
  only on redirects or existing assets.

## Risks

- Risk: `easyharness` improves first-read clarity but sounds more generic than
  `microharness`.
  - Mitigation: keep the proposal memo and README language specific about the
    thin, git-native, evidence-first workflow contract.
- Risk: a second rename in quick succession can leave stale references in
  tests, release scripts, or install compatibility paths.
  - Mitigation: require repo-wide search checks plus focused release and
    installer smoke coverage before archive.
- Risk: external users may be confused if the project name changes again while
  the executable still stays `harness`.
  - Mitigation: document the naming boundary clearly in README, release docs,
    and installer guidance.

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
