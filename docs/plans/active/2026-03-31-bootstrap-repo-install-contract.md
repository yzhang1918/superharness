---
template_version: 0.2.0
created_at: 2026-03-31T00:08:51+08:00
source_type: issue
source_refs:
    - '#68'
---

# Add a repeatable repository install flow for AGENTS delta and skills

## Goal

Add a first-run repository bootstrap flow that turns an ordinary repository
into a harness-aware repository without forcing users to reverse-engineer this
repo's local setup. The result should give `easyharness` one user-facing entry
command, `harness install`, that can be run safely more than once and that
manages the minimum repo-owned assets needed for the harness workflow.

This slice should keep the bootstrap intentionally narrow. It should install or
refresh the harness-managed `AGENTS.md` delta and the minimum repo-local skill
pack, while preserving user-owned repository instructions outside the managed
region. Richer customization models, remote template catalogs, and optional
skill packs remain deferred.

## Scope

### In Scope

- Define the first public repository-bootstrap contract around
  `harness install`.
- Split the bootstrap surfaces into one managed `AGENTS.md` delta and one
  managed repo-local skill pack, while exposing them through a single command
  with selectable scope.
- Decide and document repeat-run behavior: install on first run, refresh on
  later runs, and report no-op when the managed content is already current.
- Add packaged bootstrap assets for the minimum distributable harness contract,
  separate from this repository's easyharness-specific dogfood guidance.
- Dogfood the split by refactoring this repo so its top-level `AGENTS.md`
  contains both repo-specific guidance and the same harness-managed delta the
  command would install elsewhere.
- Update public docs and CLI contracts for release users who install
  `easyharness` from Homebrew or GitHub Releases and then need to bootstrap a
  repository.

### Out of Scope

- A full customization framework for arbitrary repo-specific bootstrap
  templates.
- Installing optional or remote skill packs from GitHub or a central catalog as
  part of the first-run path.
- Managing user-owned `AGENTS.md` content outside the harness-managed block.
- A web UI or interactive wizard for bootstrap.
- Automatic migration of every historical dogfood repo to the new contract.

## Acceptance Criteria

- [x] The CLI exposes `harness install` with a default direct-write mode plus a
      `--dry-run` preview mode, and the command is safe to rerun.
- [x] `harness install` can target `agents`, `skills`, or `all`, but the
      product entrypoint remains a single command rather than separate top-level
      subcommands.
- [x] The installed `AGENTS.md` content is a harness-managed delta block with
      stable markers; reruns update that block in place and never rewrite
      user-owned content outside it.
- [x] The installed skill pack is repository-owned content managed by the CLI,
      and reruns refresh the known managed files without deleting unrelated
      user-added files.
- [x] This repository's tracked docs and dogfood setup distinguish
      easyharness-specific repo guidance from the distributable harness
      bootstrap contract.
- [x] README, CLI help, and durable specs explain how a release-installed user
      bootstraps a fresh repository and what happens on repeated installs.

## Deferred Items

- Optional remote skill-pack installation or agent-assisted downloads.
- User-selectable bootstrap templates beyond the minimum default contract.
- Automatic merging or semantic editing of arbitrary user-authored
  `AGENTS.md` structures beyond the managed block protocol.
- Versioned upgrade prompts or background reminders to rerun `harness install`
  after every release upgrade.

## Work Breakdown

### Step 1: Define the distributable bootstrap contract and dogfood split

- Done: [x]

#### Objective

Lock the minimum harness-managed bootstrap assets and separate them from this
repo's easyharness-specific dogfood guidance.

#### Details

This step should capture the accepted product decisions from discovery so the
implementation does not rely on chat memory. The command name should be
`harness install`, not `init`, because the bootstrap must be safe to rerun for
refreshes as well as first installs. The CLI should expose a single user-facing
command with internal scope selection rather than multiple separate top-level
install commands. `AGENTS.md` should be managed as a delta block with stable
markers, while `.agents/skills/` should be managed as a repo-owned asset set.
This repo must dogfood the same split by separating easyharness-specific
development guidance from the distributable harness workflow contract.

#### Expected Files

- `docs/plans/active/2026-03-31-bootstrap-repo-install-contract.md`
- `AGENTS.md`
- bootstrap asset files under a new packaged-assets location if the chosen
  implementation records the split there

#### Validation

- The tracked plan and repo docs clearly distinguish user-owned repo guidance
  from the distributable harness-managed contract.
- A future agent can tell which `AGENTS.md` content is CLI-managed and which is
  easyharness-specific dogfood content.

#### Execution Notes

Split the distributable bootstrap contract out of repo-specific guidance by
adding packaged bootstrap assets under `assets/bootstrap/` and rewriting this
repo's `AGENTS.md` into a repo-specific wrapper plus the same managed block
that `harness install --scope agents` installs elsewhere. The distributable
contract now covers the harness working agreement, source-of-truth split,
workflow, review execution, and start points, while easyharness-specific
mission, development setup, and git rules stay outside the managed markers.

Dogfood validation included refreshing this repository's managed block through
`harness install --scope agents` and confirming the repo then reports a no-op
dry run for the managed block.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the contract split landed as part of one integrated
bootstrap slice and is covered by the branch-level finalize review.

### Step 2: Implement `harness install` and packaged bootstrap assets

- Done: [x]

#### Objective

Add the CLI command, packaged bootstrap assets, and safe repeat-run behavior
for installing or refreshing the harness-managed repo contract.

#### Details

The implementation should embed or otherwise package the minimum bootstrap
assets with the release binary so the first-run path does not depend on network
access or a separate agent-side installer. `harness install` should default to
writing changes directly, while `--dry-run` reports the intended file actions
clearly enough for a human or another agent to apply them deliberately. For
`AGENTS.md`, the command should insert the managed block on first install,
replace exactly one valid existing managed block on rerun, and error when the
managed markers are duplicated or structurally unsafe. For skills, the command
should create or refresh the known managed files while leaving unrelated
user-added files alone. If `--scope` is introduced, its values should cover
`agents`, `skills`, and `all`.

#### Expected Files

- `internal/cli/app.go`
- `internal/cli/*_test.go`
- implementation packages for bootstrap asset loading and install behavior
- packaged bootstrap assets for `AGENTS.md` delta and skill files
- optional supporting tests under `internal/` or `tests/`

#### Validation

- Add or update focused tests for CLI parsing, dry-run output, safe rerun
  semantics, marker-conflict failures, and skill-pack refresh behavior.
- Verify the packaged bootstrap assets can be rendered or installed without
  depending on network access.
- Run the relevant test suites for the new command and affected support
  packages.

#### Execution Notes

Added `assets/bootstrap/` as the packaged source of truth for the managed
`AGENTS.md` block and repo-local skill files, then introduced
`internal/install/` as the install engine for first-run bootstrap and repeat
refreshes. The CLI now exposes `harness install` with `--scope` and `--dry-run`
support, reports JSON actions for both preview and apply flows, and treats
`AGENTS.md` plus the packaged skill pack as CLI-managed assets with safe rerun
semantics.

Focused TDD covered fresh installs, managed-block refresh, duplicate-marker
errors, managed-skill refresh without deleting user files, repeat-run no-op
behavior, and CLI help/JSON output. A repo-wrapper regression test fixed a
newline-normalization bug so repo-specific `AGENTS.md` wrappers become stable
no-op reruns after refresh.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the implementation was developed and validated as one
integrated bootstrap slice and will receive a full finalize review before
archive.

### Step 3: Publish the first-run install story and dogfood it end to end

- Done: [x]

#### Objective

Document the release-user bootstrap flow and prove the new install contract is
usable in this repository and in a fresh repo-like environment.

#### Details

The docs should explain that installing the `easyharness` binary does not by
itself modify a repository; users then run `harness install` in the repository
they want to bootstrap. The README, specs, and any release-facing maintainer
docs should explain direct-write behavior, `--dry-run`, repeat-run semantics,
and the managed-block boundary in `AGENTS.md`. Dogfood validation should cover
this repo's split `AGENTS.md` contract and at least one deterministic fresh
repo bootstrap path that exercises `agents`, `skills`, and repeat-run behavior.

#### Expected Files

- `README.md`
- `docs/specs/cli-contract.md`
- `docs/specs/index.md`
- `AGENTS.md`
- smoke or integration tests if needed for fresh-repo bootstrap coverage

#### Validation

- Update docs so a release-installed user can follow the first-run bootstrap
  flow without hidden context from this repo.
- Add or run deterministic validation for a fresh repository bootstrap and a
  repeat install that produces either the expected refresh or a no-op result.
- Run the broader affected test suite after docs and dogfood changes land.

#### Execution Notes

Updated the public and durable docs so release-installed users now have an
explicit repository bootstrap story after installing the binary. `README.md`
documents `harness install`, the managed `AGENTS.md` block, repo-local skills,
and repeat-run behavior. The CLI contract now includes `harness install`, and
the smoke suite exercises fresh-repo bootstrap, dry-run non-writing behavior,
and repeat-run no-op results.

Validation passed with `go test ./internal/install ./internal/cli ./tests/smoke
-run 'TestInstall|TestHelpShowsTopLevelUsage' -count=1`, a repo-local
`harness install --scope agents --dry-run` no-op after dogfooding, and
`go test ./... -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: the docs and smoke updates close out the same integrated
slice and will be covered by full finalize review before archive.

## Validation Strategy

- Use focused TDD around the install engine first, especially for `AGENTS.md`
  managed-block insertion, replacement, duplicate-marker errors, and skill-pack
  refresh semantics.
- Add or extend integration coverage for fresh-repo bootstrap and repeat-run
  behavior so the user-facing command contract is exercised end to end.
- Re-run the broader Go test suite after the CLI, docs, and dogfood split are
  aligned.

## Risks

- Risk: The command could accidentally overwrite user-owned `AGENTS.md`
  content or delete repo-local customizations in `.agents/skills/`.
  - Mitigation: Restrict writes to one managed block for `AGENTS.md`, update
    only known managed skill files, and add focused failure-path tests for
    ambiguous or unsafe file layouts.
- Risk: The bootstrap contract could keep easyharness dogfood assumptions mixed
  into the distributable assets, making the public install flow misleading.
  - Mitigation: Split the repo-specific guidance from the distributable assets
    first and dogfood the packaged contract in this repository before archive.
- Risk: `install` could still read as a one-time initializer and confuse users
  about reruns after upgrades.
  - Mitigation: Make repeat-run semantics explicit in help text, docs, dry-run
    output, and test coverage for safe refresh/no-op behavior.

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
