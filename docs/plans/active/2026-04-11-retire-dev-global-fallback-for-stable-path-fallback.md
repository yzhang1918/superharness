---
template_version: 0.2.0
created_at: "2026-04-11T23:24:00+08:00"
source_type: direct_request
source_refs:
    - chat://current-session
size: M
---

# Retire Dev Global Fallback For Stable PATH Fallback

## Goal

Replace the development wrapper's out-of-tree fallback model so
`scripts/install-dev-harness` no longer maintains a separate dev-owned global
fallback binary under the user's home directory.

Inside an easyharness source tree, the wrapper should keep the current strict
worktree-local behavior and require `<repo>/.local/bin/harness`. Outside an
easyharness source tree, the wrapper should instead dispatch to a stable
`harness` already available on `PATH`, which is expected to be the Homebrew
release install in the normal operator flow.

## Scope

### In Scope

- Remove the `--global` installer path and the dev-owned global fallback file
  management from `scripts/install-dev-harness`.
- Change the generated wrapper so out-of-tree invocation resolves a stable
  `harness` from `PATH` while skipping the managed dev wrapper itself.
- Keep easyharness source-tree detection authoritative so source-tree
  invocations still require the current worktree's `.local/bin/harness`.
- Update development docs and installer smoke coverage to describe and verify
  the new release-backed fallback contract.

### Out of Scope

- Changing the release packaging or Homebrew publication flow itself.
- Replacing the worktree-aware wrapper model inside easyharness source trees.
- Adding Homebrew-specific hardcoded filesystem probing when a stable
  `harness` is not already present on `PATH`.

## Acceptance Criteria

- [x] `scripts/install-dev-harness` no longer accepts or documents `--global`,
      and no longer writes or repairs a dev-owned global fallback binary.
- [x] Inside an easyharness source tree, the managed wrapper still resolves the
      current worktree's `.local/bin/harness` and fails locally when that
      binary is missing.
- [x] Outside easyharness source trees, the managed wrapper dispatches to a
      stable `harness` found on `PATH` and emits a clear actionable error when
      none exists.
- [x] The wrapper avoids recursively selecting the managed dev wrapper itself
      when searching `PATH` for the stable fallback.
- [x] Development docs and installer smoke tests describe and verify the new
      contract centered on Homebrew or other stable PATH installs.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Replace wrapper fallback resolution

- Done: [x]

#### Objective

Refactor `scripts/install-dev-harness` and the generated wrapper so the
out-of-tree dispatch path uses a stable `PATH` binary instead of the dev-owned
global fallback.

#### Details

The clean target contract is:

- inside easyharness source trees: use `<repo>/.local/bin/harness` only
- outside easyharness source trees: resolve a stable `harness` from `PATH`

This step removes the installer's `--global` option, the
`~/.local/share/easyharness/dev/harness` fallback management, and any ordinary
install self-heal behavior tied to that path. The generated wrapper must skip
its own managed dev wrapper entry when searching `PATH` so an out-of-tree call
does not recurse back into itself. The out-of-tree error should point operators
at the stable release install path rather than suggesting a dev-only global
refresh.

#### Expected Files

- `scripts/install-dev-harness`

#### Validation

- `bash -n scripts/install-dev-harness`
- Focused smoke coverage that proves source-tree resolution still wins and that
  out-of-tree resolution now selects a stable `PATH` binary rather than a
  dev-managed global file.

#### Execution Notes

Removed the installer's `--global` flag, deleted the dev-owned
`~/.local/share/easyharness/dev/harness` fallback path management, and changed
the generated wrapper so out-of-tree execution resolves a stable `harness`
from `PATH` while skipping managed dev wrappers and its own installed path.
Validated with `bash -n scripts/install-dev-harness`,
`gofmt -w tests/smoke/install_dev_harness_test.go`, and
`go test ./tests/smoke -run InstallDevHarness -count=1`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: Step 1 and Step 2 form one bounded installer-contract
slice, so separate Step 1 closeout review would be artificial and the real
review boundary is the integrated Step 2 candidate.

### Step 2: Rebaseline docs and smoke coverage

- Done: [x]

#### Objective

Update the operator-facing docs and installer smoke suite to match the new
stable PATH fallback contract.

#### Details

Tests should stop asserting `--global` behavior and instead cover:

- success when a stable `harness` is available on `PATH` outside source trees
- failure with a clear message when no stable `harness` is available on `PATH`
- continued refusal to leave source-tree execution for a stable out-of-tree
  binary when the local worktree binary is missing

Docs should explain that development installs expose a worktree-aware wrapper
for source-tree work while ordinary out-of-tree usage should come from the
stable release install, with Homebrew as the default supported path.

#### Expected Files

- `docs/development.md`
- `tests/smoke/install_dev_harness_test.go`

#### Validation

- `go test ./tests/smoke -run InstallDevHarness -count=1`
- Review the updated dev-install docs for consistency with the wrapper's
  actual runtime behavior.

#### Execution Notes

Updated `docs/development.md` to remove the obsolete `--global` guidance and
describe the new stable PATH fallback behavior outside source trees. Replaced
installer smoke coverage with assertions for the removed flag, source-tree
precedence over stable PATH fallback, out-of-tree failure without a stable
`harness` on `PATH`, successful out-of-tree stable fallback dispatch, stable
`--version` forwarding, and continued refusal to leave a source tree when the
local binary is missing. Reinstalled the current worktree with
`scripts/install-dev-harness` after the script change and reran
`harness status`. After `review-001-full` requested changes, hardened the PATH
fallback search so symlink aliases to managed wrappers resolve to their real
targets before candidate selection, and extended smoke coverage to prove both
symlink-alias skipping and skipping other managed wrappers already on `PATH`.
Revalidated with `bash -n scripts/install-dev-harness`,
`gofmt -w tests/smoke/install_dev_harness_test.go`, and
`go test ./tests/smoke -run InstallDevHarness -count=1`. After finalize
`review-003-full` raised one remaining tests finding, added an explicit smoke
assertion that ordinary installs leave the retired
`~/.local/share/easyharness/dev/harness` path absent and reran
`gofmt -w tests/smoke/install_dev_harness_test.go` plus
`go test ./tests/smoke -run InstallDevHarness -count=1`. After
`review-005-full` requested further changes, tightened out-of-tree PATH
selection so only `mode: release` candidates qualify as stable fallbacks, and
extended managed-wrapper coverage to exercise both the legacy wrapper-signature
branch and the case where another checkout's repo-local dev binary appears on
`PATH` ahead of the stable install. Revalidated with
`bash -n scripts/install-dev-harness`, `gofmt -w tests/smoke/install_dev_harness_test.go`,
and `go test ./tests/smoke -run InstallDevHarness -count=1`.

#### Review Notes

`review-001-full` requested two blocking findings: symlinked aliases to the
managed wrapper could recurse forever during out-of-tree PATH fallback, and
the smoke suite did not exercise the separate branch that skips other managed
wrappers already on `PATH`. The repair resolves wrapper candidates to their
real paths before self/managed-wrapper checks and adds focused smoke coverage
for both the symlink-alias path and the separate managed-wrapper-skip branch.
`review-002-delta` then passed cleanly with no remaining correctness or test
findings for the repair. Finalize `review-003-full` later requested one tests
finding because the suite no longer proved the retired dev-owned global
fallback path stayed absent during ordinary installs; the repair added that
assertion and `review-004-delta` passed cleanly for that narrow fix.
`review-005-full` then requested two more blocking findings: restrict
out-of-tree fallback selection to stable release-mode binaries rather than any
non-wrapper `harness` on `PATH`, and add smoke coverage for the legacy
managed-wrapper detection branch that still remains supported. The repair now
filters PATH candidates by `mode: release` and extends the smoke suite to
cover both the legacy-wrapper branch and repo-local dev binaries on `PATH`.
Fresh full finalize review is now pending for that repaired candidate.

## Validation Strategy

- Run shell validation for the installer script after changing the wrapper
  template and option parsing.
- Run the installer-focused smoke suite covering source-tree precedence,
  out-of-tree stable PATH fallback success, and out-of-tree failure without a
  stable PATH binary.
- Re-run `scripts/install-dev-harness` in this worktree after installer changes
  before relying on direct `harness` commands locally.

## Risks

- Risk: PATH-based lookup could accidentally select the managed dev wrapper
  again and recurse instead of reaching a stable install.
  - Mitigation: Make the wrapper detect and skip its own managed wrapper path,
    and add smoke coverage for the out-of-tree PATH-selection path.
- Risk: Removing `--global` could leave docs or tests describing an obsolete
  dev-fallback recovery path.
  - Mitigation: Update development docs and installer smoke assertions in the
    same slice so the contract changes atomically.
- Risk: Source-tree enforcement could regress and silently use a stable PATH
  binary when the local worktree binary is missing.
  - Mitigation: Preserve source-tree detection order and keep the explicit
    missing-local-binary smoke coverage.

## Validation Summary

- `bash -n scripts/install-dev-harness`
- `gofmt -w tests/smoke/install_dev_harness_test.go`
- `go test ./tests/smoke -run InstallDevHarness -count=1`
- `scripts/install-dev-harness`
- `harness status`

## Review Summary

- `review-001-full` requested two blocking findings: fix PATH-fallback
  recursion through symlinked wrapper aliases and add smoke coverage for
  skipping other managed wrappers already on `PATH`.
- `review-002-delta` passed cleanly after the wrapper real-path resolution fix
  and the added managed-wrapper and alias-chain smoke coverage.
- `review-003-full` requested one blocking tests finding: prove the retired
  `~/.local/share/easyharness/dev/harness` path stays absent during ordinary
  installs.
- `review-004-delta` passed cleanly after adding that retired-path absence
  assertion to the normal-install smoke.

## Archive Summary

- PR: pending creation after archive closeout.
- Ready: awaiting a fresh passing full finalize review for revision `1`, after
  which the candidate should be archive-ready.
- Merge Handoff: rerun full finalize review, archive the active plan, commit
  the archive move and closeout summaries, push branch
  `codex/retire-dev-global-fallback-path`, open or update the PR, and record
  publish/CI/sync evidence until `harness status` reaches
  `execution/finalize/await_merge`.

## Outcome Summary

### Delivered

- Removed the dev installer's explicit `--global` path and the dev-owned global
  fallback management under `~/.local/share/easyharness/dev/harness`.
- Kept the managed wrapper model, but changed out-of-tree dispatch to use a
  stable `harness` already on `PATH` while preserving strict repo-local binary
  enforcement inside easyharness source trees.
- Hardened PATH fallback selection so managed wrappers and symlink aliases are
  skipped rather than recursively re-entering another dev wrapper.
- Updated development docs and installer smoke coverage to match the new
  release-backed PATH fallback contract, including proof that the retired
  global fallback path stays absent during ordinary installs.

### Not Delivered

NONE

### Follow-Up Issues

NONE
