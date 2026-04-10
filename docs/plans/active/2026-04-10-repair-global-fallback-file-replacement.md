---
template_version: 0.2.0
created_at: "2026-04-10T09:43:51+08:00"
source_type: direct_request
source_refs:
  - chat://current-session
---

# Repair Global Fallback Replacement And Health Checks

## Goal

Fix the dev installer so `scripts/install-dev-harness --global` truly replaces
the global fallback binary used outside easyharness source trees instead of
rewriting bytes in place on top of an existing file object that may remain
unhealthy.

This slice addresses the observed failure where
`~/.local/share/easyharness/dev/harness` was byte-identical to a healthy
worktree-local binary but still got killed when executed from an unrelated
repository until the old path was removed and recreated with a new inode. The
installer should replace the fallback atomically, verify that the final target
is runnable, and repair an invalid existing fallback even during ordinary
non-`--global` installs without regressing the explicit global ownership model
introduced on 2026-04-08.

## Scope

### In Scope

- Change global fallback writes in `scripts/install-dev-harness` from in-place
  overwrite to same-directory temp-file replacement with an atomic rename.
- Add a health check for the final fallback path so the installer validates the
  executable it just installed, not only the worktree-local build output.
- Repair an invalid existing global fallback during ordinary installs while
  preserving the existing rule that healthy fallbacks are not refreshed unless
  `--global` is requested.
- Add smoke coverage for atomic replacement, invalid-fallback self-healing, and
  continued preservation of healthy fallbacks.

### Out of Scope

- Changing wrapper dispatch rules between easyharness source trees and unrelated
  repositories.
- Broad redesign of development installation locations or fallback discovery.
- Release-binary, Homebrew, or hosted distribution changes.

## Acceptance Criteria

- [ ] `scripts/install-dev-harness --global` replaces the global fallback via a
      new file object rather than an in-place overwrite, and the final fallback
      path passes a direct runnable health check.
- [ ] Ordinary `scripts/install-dev-harness` keeps an existing healthy global
      fallback untouched.
- [ ] Ordinary `scripts/install-dev-harness` repairs an existing invalid global
      fallback so the wrapper works again from a non-easyharness repository.
- [ ] Installer smoke coverage demonstrates the atomic replacement path, the
      invalid-fallback self-heal path, and the healthy-fallback preservation
      path.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Make fallback refresh replace the final file object

- Done: [x]

#### Objective

Refactor installer fallback writes so `--global` installs a newly created final
file object and validates the resulting executable path after replacement.

#### Details

The current installer copies the worktree-local binary directly onto the global
fallback path, which preserves the existing file object and was shown to leave
an apparently unhealthy executable in place even when its bytes match a healthy
binary. Write the fallback into a temp path inside the destination directory,
set executable permissions there, atomically rename it into place, then run the
health check against the final target path. If validation fails, the installer
should exit with a clear error instead of silently reporting success.

#### Expected Files

- `scripts/install-dev-harness`

#### Validation

- `bash -n scripts/install-dev-harness`
- Manual or automated proof that `--global` replaces the existing fallback with
  a new file object and leaves the final path directly runnable.

#### Execution Notes

Changed `scripts/install-dev-harness` so global fallback refresh now writes to a
same-directory temp file and atomically renames it into place, then validates
the final fallback path with `--version` before reporting success. Focused
validation passed with `bash -n scripts/install-dev-harness`,
`go test ./tests/smoke -run TestInstallDevHarnessGlobalRefreshReplacesFallbackFileObject -count=1`,
and a real reinstall plus `harness --version` from
`/Users/yaozhang/Workspace/HEJI`, which now succeeds against
`~/.local/share/easyharness/dev/harness`.

#### Review Notes

NO_STEP_REVIEW_NEEDED: This installer slice is reviewable only as the combined
Step 1 + Step 2 candidate, so closeout review is deferred to Step 2.

### Step 2: Cover self-healing and preserved healthy fallback behavior

- Done: [x]

#### Objective

Extend smoke coverage so the installer contract explicitly covers invalid
fallback repair without regressing the existing healthy-fallback preservation
rule.

#### Details

Add a smoke test that seeds an invalid global fallback, runs ordinary
`scripts/install-dev-harness`, and proves the wrapper works from an unrelated
repository afterward. Keep or expand the existing preservation coverage so a
healthy fallback still remains untouched without `--global`. If the execution
flow or user-facing output changes materially, update README/help text to match
the repaired installer semantics.

#### Expected Files

- `tests/smoke/install_dev_harness_test.go`
- `README.md`

#### Validation

- `go test ./tests/smoke -run InstallDevHarness -count=1`
- Review any updated install guidance for consistency with the actual script
  behavior.

#### Execution Notes

Expanded installer smoke coverage with a new inode-replacement assertion for
`--global` refresh and an invalid-fallback self-heal case for ordinary installs.
The repair path passed
`go test ./tests/smoke -run TestInstallDevHarnessRepairsInvalidExistingGlobalFallback -count=1`.
After step-closeout review surfaced symlink edge cases, added focused smoke
coverage for both broken symlink and directory-symlink global fallbacks. Those
repairs passed
`go test ./tests/smoke -run TestInstallDevHarnessRepairsBrokenSymlinkGlobalFallback -count=1`,
`go test ./tests/smoke -run TestInstallDevHarnessRepairsDirectorySymlinkGlobalFallback -count=1`,
and a focused regression run over the fallback-repair smoke subset.
Updated `README.md` to clarify the final contract: healthy outside-source-tree
fallbacks still only refresh on `--global`, while ordinary installs now
self-heal invalid fallback paths.

#### Review Notes

`review-001-delta` requested one blocking correctness fix because ordinary
installs still skipped self-healing when the fallback path was a broken symlink.
The repair broadened the invalid-fallback gate to include symlink paths and
added `TestInstallDevHarnessRepairsBrokenSymlinkGlobalFallback`.
`review-002-delta` then requested one blocking correctness fix because a
directory-symlink fallback still caused `mv -f` to write into the linked
directory instead of replacing the symlink entry. The repair now removes
symlink targets before the final rename and adds
`TestInstallDevHarnessRepairsDirectorySymlinkGlobalFallback`.
`review-003-delta` passed cleanly after those fixes.

## Validation Strategy

- Run shell validation for the installer script after the fallback-write
  changes.
- Run the installer-focused smoke suite covering fallback refresh, fallback
  preservation, and invalid-fallback repair.
- Re-run `scripts/install-dev-harness` after Go or installer changes before
  relying on direct `harness` invocations in this worktree.

## Risks

- Risk: The replacement flow could accidentally break the explicit 2026-04-08
  contract that healthy global fallbacks are preserved unless `--global` is
  used.
  - Mitigation: Keep the refresh gate unchanged for healthy fallbacks and add
    smoke coverage for preservation.
- Risk: A temp-file-and-rename flow could leave partial artifacts behind or
  validate the wrong path.
  - Mitigation: Create the temp file in the destination directory, chmod before
    rename, and run the health check against the final fallback path after the
    rename.

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
