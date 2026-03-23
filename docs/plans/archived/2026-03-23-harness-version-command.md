---
template_version: 0.2.0
created_at: "2026-03-23T00:20:00+08:00"
source_type: direct_request
source_refs: []
---

# Add harness --version command

## Goal

Add a top-level `harness --version` diagnostic entrypoint so operators can tell
which `harness` binary is actually running without digging through wrapper
paths or local build state by hand.

This slice should keep the existing agent-facing workflow commands JSON-first
while treating `--version` as a human-oriented debug surface. The command
should report the running binary's build commit, identify whether the binary is
running in dev or release mode, and print the binary path only for dev mode.

## Scope

### In Scope

- Add root-level `harness --version` handling distinct from `harness --help`
  and the workflow subcommand surface.
- Report the running binary's build commit as the primary commit identity for
  the command.
- Distinguish dev vs release mode and print the resolved binary path in dev
  mode only.
- Update root help and tracked docs where the command surface or output
  contract changes.
- Add unit and smoke coverage for the new flag and its output contract.

### Out of Scope

- Adding a `harness version` subcommand alongside the root `--version` flag.
- Making `harness --version` return JSON by default in this slice.
- Reworking workflow subcommands away from their current JSON-first behavior.
- Broader installer-policy changes beyond any metadata plumbing needed to make
  dev-mode version reporting work correctly.

## Acceptance Criteria

- [x] `harness --version` exits zero and prints concise human-readable debug
      output rather than a JSON envelope.
- [x] The output includes the running binary's build commit and an explicit
      mode indicator, and includes the resolved binary path only when the
      binary is in dev mode.
- [x] Root help and tracked docs describe `--version` as a debug-oriented
      exception to the JSON-first workflow commands without absorbing it into
      `--help` itself.
- [x] Automated coverage proves the new root flag works without regressing
      existing help or subcommand parsing behavior.

## Deferred Items

- Add an opt-in JSON form such as `harness --version --json` if a later slice
  needs machine-readable version output.
- Expand version output with richer build metadata such as dirty state, build
  time, Go version, or wrapper/install provenance if those become useful.
- Revisit installer default-directory discovery separately if the current
  allowlist policy for user-local wrapper dirs still feels too rigid after more
  dogfooding.

## Work Breakdown

### Step 1: Capture discovery decisions for the version command

- Done: [x]

#### Objective

Record the command shape, output model, and adjacent installer-policy decisions
already settled in discovery so execution does not depend on chat memory.

#### Details

Discovery converged on a root `harness --version` flag rather than a
`harness version` subcommand or a `--help` expansion. The command is treated as
a debug-oriented diagnostic surface, so plain text output is intentional even
though stateful workflow commands remain JSON-first. The current working
assumption is that `commit` means the running binary's build commit because
that remains meaningful even when the binary is invoked outside a
`superharness` worktree. Discovery also converged on keeping the wrapper
installer's default directory policy conservative: prefer allowlisted
user-local wrapper dirs such as `~/.local/bin` and `~/bin`, not arbitrary
writable `PATH` entries.

#### Expected Files

- `docs/plans/active/2026-03-23-harness-version-command.md`

#### Validation

- The tracked plan captures the accepted `--version` direction, non-goals, and
  the current build-commit assumption without relying on hidden chat context.

#### Execution Notes

Discovery completed before planning. The accepted direction is:
`harness --version`, plain text output, build-commit identity, and dev-mode
path reporting only. The installer-policy discussion is included here as
adjacent context, not as committed execution scope for this slice.

#### Review Notes

NO_STEP_REVIEW_NEEDED: discovery-only closeout recorded directly in the plan.

### Step 2: Implement the root --version flag and metadata plumbing

- Done: [x]

#### Objective

Add root-level `--version` handling and wire enough build/runtime metadata into
the binary to report commit, mode, and dev path correctly.

#### Details

Keep the new flag outside the workflow subcommand tree so it behaves like a
binary-identity probe rather than a stateful command. Prefer metadata that
describes the running binary itself, not the caller's current working tree.
Use Go build information when it is sufficient and deterministic; if that
cannot cleanly express build commit or dev-mode detection across installed dev
binaries and tests, add explicit build variables or a small internal package
that centralizes version data instead of scattering logic through the CLI.

#### Expected Files

- `cmd/harness/main.go`
- `internal/cli/app.go`
- `internal/cli/app_test.go`
- `scripts/install-dev-harness`
- optional new internal package if version/build metadata needs a dedicated home

#### Validation

- Root-flag unit tests cover `--version`, `--help`, and ordinary subcommand
  parsing without ambiguity.
- Tests pin the expected labeled fields in the version output without relying
  on an unstable exact local path or commit string fixture.
- Dev installs still produce a binary whose version output can identify itself
  correctly when run from the worktree or through the installed wrapper.

#### Execution Notes

Added a small `internal/version` package to centralize build commit, mode, and
dev-path reporting, then wired root-level `harness --version` handling through
`internal/cli/app.go` as plain-text debug output rather than the shared JSON
envelope. Dev installs now build with explicit version metadata so the command
can report `mode: dev`, while ordinary binary builds continue to default to
`release`. Step-closeout review then exposed that the repo-built release binary
and the dev installer smoke path could still degrade to `commit: unknown`
without failing coverage, so the slice was tightened by explicitly injecting
`BuildCommit` into the repo-level test binary helper and by pinning smoke
expectations to the current `HEAD` commit for release builds plus a
deterministic fake-git commit for dev installs. Validated the implementation
with `go test ./internal/version ./internal/cli -count=1`,
`go test ./tests/smoke -count=1`, `go test ./... -count=1`,
`bash -n scripts/install-dev-harness`, `scripts/install-dev-harness`, and a
direct `PATH="$HOME/.local/bin:$PATH" harness --version` run that reported the
current worktree binary path.

#### Review Notes

Step-closeout review `review-001-delta` requested changes because the initial
smoke coverage only asserted that `commit:` labels existed and did not fail
when the version output degraded to `commit: unknown`. After the test binary
builder and smoke assertions were tightened, step-closeout review
`review-002-delta` passed. One non-blocking follow-up remains tracked in
`#33`: add dedicated smoke coverage for the installer's PATH-verified branch.

### Step 3: Document the contract and add repo-level validation

- Done: [x]

#### Objective

Document the new command surface clearly and add high-signal validation that
protects the debug contract from regressions.

#### Details

Update the command-surface docs and any CLI contract wording needed to explain
why `--version` stays plain text while workflow commands remain JSON-first.
Extend the smoke suite so the built binary covers the new root flag alongside
existing top-level help behavior. Keep the docs explicit that `--version` is a
debug surface and not a stateful workflow command.

#### Expected Files

- `README.md`
- `docs/specs/cli-contract.md`
- `tests/smoke/smoke_test.go`
- any other docs/help locations that list the root command surface

#### Validation

- Smoke coverage proves the built binary responds to `harness --version`
  successfully.
- Updated docs and help text stay consistent about where JSON is required and
  where plain text is intentional.
- Full Go test coverage still passes after the new flag and docs land.

#### Execution Notes

Updated README and the CLI contract to document `harness --version` as a
plain-text debug exception outside the JSON-first workflow commands. Extended
repo-level smoke coverage for root help, plain-text release-mode `--version`
output, dev-mode `--version` through the installed wrapper, managed-wrapper
refresh, and legacy-wrapper replacement without `--force`. Validated this
slice with `go test ./tests/smoke -count=1`, `go test ./...`, and
`bash -n scripts/install-dev-harness`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 2 step-closeout review covers the tightly coupled
docs and smoke-test follow-up for this slice.

## Validation Strategy

- Run `harness plan lint docs/plans/active/2026-03-23-harness-version-command.md`
  before execution starts and whenever the plan wording changes materially.
- During implementation, keep `go test ./internal/cli -count=1` and
  `go test ./tests/smoke -count=1` green as the fastest contract checks for the
  new root flag.
- Before archive, run `go test ./...` so the new flag, metadata plumbing, and
  smoke coverage do not regress the wider CLI surface.

## Risks

- Risk: The reported commit could be misread as the current worktree's `HEAD`
  rather than the running binary's build identity.
  - Mitigation: Make the plan and docs explicit that `--version` reports the
    running binary's build commit, and pair it with a mode label plus dev path
    when relevant.
- Risk: Dev-mode path reporting and build metadata may be awkward to test if
  the output bakes in local absolute paths or environment-specific VCS data.
  - Mitigation: Assert labeled fields and presence/absence rules in unit and
    smoke tests instead of snapshotting the entire output verbatim.
- Risk: Root flag parsing could accidentally interfere with existing `--help`
  behavior or unknown-command handling.
  - Mitigation: Add root-level parser coverage that exercises `--version`,
    `--help`, and ordinary subcommands side by side.

## Validation Summary

- `harness plan lint docs/plans/active/2026-03-23-harness-version-command.md`
- `go test ./internal/version ./internal/cli -count=1`
- `go test ./tests/smoke -count=1`
- `go test ./... -count=1`
- `bash -n scripts/install-dev-harness`
- `scripts/install-dev-harness`
- `PATH="$HOME/.local/bin:$PATH" harness --version`

## Review Summary

- `review-001-delta` requested changes because the first smoke pass only
  checked for `commit:` labels and allowed release or dev outputs to degrade
  to `commit: unknown`.
- `review-002-delta` passed after explicit build-commit injection was added to
  the repo-level test binary helper and the smoke suite pinned release/dev
  commit expectations. It left one non-blocking installer-coverage note that
  is now tracked in `#33`.
- `review-003-full` passed as the `pre_archive` gate. Correctness and
  docs-consistency reviews were clean; the same non-blocking installer PATH
  verification coverage gap remains deferred to `#33`.

## Archive Summary

- Archived At: 2026-03-23T01:30:58+08:00
- Revision: 1
- PR: not created yet; publish evidence will record the PR URL after archive.
- Ready: `review-003-full` passed as the structural `pre_archive` gate, all
  acceptance criteria are satisfied, and the remaining work is the archive move
  plus publish/CI/sync evidence.
- Merge Handoff: After archive, commit and push the archived plan move plus the
  tracked code/doc changes, open the PR, record publish/CI/sync evidence, and
  carry deferred follow-up scope in `#31`, `#32`, and `#33`.

## Outcome Summary

### Delivered

- Added a root-level `harness --version` flag that prints plain-text debug
  output for the running binary instead of the JSON envelope used by workflow
  commands.
- Added centralized version/build metadata handling in `internal/version`,
  including explicit dev-mode path reporting and build-commit resolution for
  both installed dev binaries and repo-built validation binaries.
- Updated root help, README, and the CLI contract to document `--version` as a
  debug-oriented exception to the JSON-first workflow surface.
- Added unit and smoke coverage that pins release-mode commit reporting to the
  current repository `HEAD`, pins dev-mode commit reporting through a
  deterministic fake-git installer path, and protects the wrapper refresh and
  legacy-wrapper replacement behavior already touched by this slice.

### Not Delivered

- An opt-in machine-readable form such as `harness --version --json`.
- Richer version/build metadata such as dirty state, build time, Go version,
  or wrapper/install provenance.
- Broader installer default-directory discovery beyond the current
  `~/.local/bin` / `~/bin` allowlist.
- Dedicated smoke coverage for the installer's on-PATH verification branch.

### Follow-Up Issues

- `#32` Extend `harness --version` with optional JSON and richer build
  metadata.
- `#31` Revisit installer default wrapper-directory discovery.
- `#33` Cover the PATH-verified install branch in `install-dev-harness` smoke
  tests.
