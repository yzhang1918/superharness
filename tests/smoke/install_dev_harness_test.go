package smoke_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/tests/support"
)

type commandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

var installerCacheDirs struct {
	once       sync.Once
	goCache    string
	goModCache string
	err        error
}

func (r commandResult) CombinedOutput() string {
	return r.Stdout + r.Stderr
}

func TestInstallDevHarnessDefaultsToUserLocalBin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()
	firstPathDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(firstPathDir, 0o755); err != nil {
		t.Fatalf("mkdir first PATH dir: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, firstPathDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	expectedWrapper := filepath.Join(tempHome, ".local", "bin", "harness")
	retiredGlobalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+expectedWrapper)
	support.RequireFileExists(t, expectedWrapper)
	support.RequireFileMissing(t, filepath.Join(firstPathDir, "harness"))
	support.RequireFileMissing(t, retiredGlobalFallback)

	info, err := os.Lstat(expectedWrapper)
	if err != nil {
		t.Fatalf("lstat wrapper: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to be a wrapper file, not a symlink", expectedWrapper)
	}
}

func TestInstallDevHarnessHelpDoesNotMentionGlobalFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--help",
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness --help failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	if strings.Contains(result.CombinedOutput(), "--global") {
		t.Fatalf("expected help output to omit removed --global flag\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
}

func TestInstallDevHarnessRejectsRemovedGlobalFlag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected removed --global flag to fail\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stderr, "Unknown argument: --global")
}

func TestInstallDevHarnessVerifiesPATHResolvedWrapperWhenInstallDirIsAlreadyOnPATH(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+wrapperPath)
	support.RequireContains(t, result.Stdout, "Verified harness on PATH at "+wrapperPath)
}

func TestInstallDevHarnessWrapperDispatchesToCurrentWorktreeOverStablePathFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	_, nestedDir := newFakeWorktree(t)
	wrapperResult := runCommand(
		t,
		nestedDir,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode != 0 {
		t.Fatalf("wrapper failed with exit %d\nstdout:\n%s\nstderr:\n%s", wrapperResult.ExitCode, wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stdout, "fake worktree harness")
	support.RequireContains(t, wrapperResult.Stdout, "args=status")
	if strings.Contains(wrapperResult.CombinedOutput(), "stable fallback harness") {
		t.Fatalf("expected wrapper to prefer the worktree-local binary over the stable PATH fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}
}

func TestInstallDevHarnessWrapperRequiresStableHarnessOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	wrapperResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir),
		}),
		wrapperPath,
		"--help",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected wrapper without a stable PATH fallback to fail outside easyharness source trees\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "Could not find an easyharness source tree")
	support.RequireContains(t, wrapperResult.Stderr, "no stable harness binary is available on PATH")
	support.RequireContains(t, wrapperResult.Stderr, "Install the stable easyharness release with Homebrew")
}

func TestInstallDevHarnessWrapperUsesStableHarnessOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	helpResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper stable PATH fallback failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
}

func TestInstallDevHarnessWrapperSkipsOtherManagedWrappersOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	managedDir := t.TempDir()
	stableDir, _ := newFakeStableHarness(t)
	writeFixtureFile(t, filepath.Join(managedDir, "harness"), fakeManagedWrapperScript("unexpected managed wrapper"), 0o755)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, managedDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	otherProject := t.TempDir()
	helpResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir, managedDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper with other managed wrapper on PATH failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
	if strings.Contains(helpResult.CombinedOutput(), "unexpected managed wrapper") {
		t.Fatalf("expected wrapper to skip other managed wrappers on PATH\nstdout:\n%s\nstderr:\n%s", helpResult.Stdout, helpResult.Stderr)
	}
}

func TestInstallDevHarnessWrapperSkipsSymlinkAliasesOnPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	aliasOneDir := t.TempDir()
	aliasTwoDir := t.TempDir()
	stableDir, _ := newFakeStableHarness(t)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	if err := os.Symlink(wrapperPath, filepath.Join(aliasOneDir, "harness")); err != nil {
		t.Fatalf("create first wrapper alias: %v", err)
	}
	if err := os.Symlink(filepath.Join(aliasOneDir, "harness"), filepath.Join(aliasTwoDir, "harness")); err != nil {
		t.Fatalf("create second wrapper alias: %v", err)
	}

	otherProject := t.TempDir()
	helpResult := runCommandWithTimeout(
		t,
		5*time.Second,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, aliasOneDir, aliasTwoDir, installDir, stableDir),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper with symlink aliases on PATH failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.Stdout, "stable fallback harness help")
}

func TestInstallDevHarnessVersionReportsStableModeAndPathOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, stablePath := newFakeStableHarness(t)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	versionResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("wrapper version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}

	if mode := requireVersionField(t, versionResult.Stdout, "mode"); mode != "release" {
		t.Fatalf("expected release mode from stable PATH fallback, got %q\noutput:\n%s", mode, versionResult.Stdout)
	}
	if commit := requireVersionField(t, versionResult.Stdout, "commit"); commit != "stable-test-commit" {
		t.Fatalf("expected stable fallback commit %q, got %q\noutput:\n%s", "stable-test-commit", commit, versionResult.Stdout)
	}
	if path := requireVersionField(t, versionResult.Stdout, "path"); path != stablePath {
		t.Fatalf("expected stable fallback path %q, got %q\noutput:\n%s", stablePath, path, versionResult.Stdout)
	}
	if strings.HasPrefix(strings.TrimSpace(versionResult.Stdout), "{") {
		t.Fatalf("expected plain-text version output, got %q", versionResult.Stdout)
	}
}

func TestInstallDevHarnessReplacesLegacyManagedWrapperWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	legacyWrapper := `#!/usr/bin/env bash
set -euo pipefail

find_repo_root() {
  local root=""
  if command -v git >/dev/null 2>&1; then
    root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
    if [[ -n "${root}" && -f "${root}/scripts/install-dev-harness" && -f "${root}/cmd/harness/main.go" ]]; then
      printf '%s\n' "${root}"
      return 0
    fi
  fi

  local dir="${PWD}"
  while :; do
    if [[ -f "${dir}/scripts/install-dev-harness" && -f "${dir}/cmd/harness/main.go" ]]; then
      printf '%s\n' "${dir}"
      return 0
    fi
    if [[ "${dir}" == "/" ]]; then
      break
    fi
    dir="$(dirname "${dir}")"
  done

  return 1
}

if ! repo_root="$(find_repo_root)"; then
  echo "Could not find a microharness worktree from ${PWD}." >&2
  echo "Run harness from inside a microharness checkout, or call a repo-local binary directly." >&2
  exit 1
fi

binary_path="${repo_root}/.local/bin/harness"
if [[ ! -x "${binary_path}" ]]; then
  echo "No repo-local harness binary found at ${binary_path}." >&2
  echo "Run scripts/install-dev-harness from this worktree first." >&2
  exit 1
fi

exec "${binary_path}" "$@"
`
	writeFixtureFile(t, wrapperPath, legacyWrapper, 0o755)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	refreshed, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read refreshed wrapper: %v", err)
	}
	support.RequireContains(t, string(refreshed), "# easyharness-install-dev-wrapper")
}

func TestInstallDevHarnessReplacesLegacySymlinkedBinaryWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	cases := []struct {
		name       string
		moduleLine string
	}{
		{
			name:       "superharness namespace",
			moduleLine: "module github.com/yzhang1918/superharness\n",
		},
		{
			name:       "personal microharness namespace",
			moduleLine: "module github.com/yzhang1918/microharness\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := copyInstallerFixture(t)
			installDir := filepath.Join(t.TempDir(), "path-bin")
			if err := os.MkdirAll(installDir, 0o755); err != nil {
				t.Fatalf("mkdir install dir: %v", err)
			}

			legacyRoot := t.TempDir()
			for _, dir := range []string{
				filepath.Join(legacyRoot, "scripts"),
				filepath.Join(legacyRoot, "cmd", "harness"),
				filepath.Join(legacyRoot, ".local", "bin"),
			} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("mkdir legacy dir %s: %v", dir, err)
				}
			}
			writeFixtureFile(t, filepath.Join(legacyRoot, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
			writeFixtureFile(t, filepath.Join(legacyRoot, "cmd", "harness", "main.go"), "package main\n", 0o644)
			writeFixtureFile(t, filepath.Join(legacyRoot, "go.mod"), tc.moduleLine, 0o644)
			writeFixtureFile(t, filepath.Join(legacyRoot, ".local", "bin", "harness"), "#!/bin/sh\nexit 0\n", 0o755)

			wrapperPath := filepath.Join(installDir, "harness")
			if err := os.Symlink(filepath.Join(legacyRoot, ".local", "bin", "harness"), wrapperPath); err != nil {
				t.Fatalf("create legacy symlink: %v", err)
			}

			result := runCommand(
				t,
				repoRoot,
				installerEnv(t, map[string]string{
					"HOME": t.TempDir(),
					"PATH": installerPath(t),
				}),
				"/bin/bash",
				filepath.Join(repoRoot, "scripts", "install-dev-harness"),
				"--install-dir", installDir,
			)
			if result.ExitCode != 0 {
				t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
			}

			info, err := os.Lstat(wrapperPath)
			if err != nil {
				t.Fatalf("lstat refreshed wrapper: %v", err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				t.Fatalf("expected refreshed wrapper to replace the legacy symlink")
			}
			refreshed, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read refreshed wrapper: %v", err)
			}
			support.RequireContains(t, string(refreshed), "# easyharness-install-dev-wrapper")
		})
	}
}

func TestInstallDevHarnessWrapperDoesNotUseStablePathFallbackInsideSourceTreeWithoutLocalBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "path-bin")
	stableDir, _ := newFakeStableHarness(t)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": t.TempDir(),
			"PATH": installerPath(t, installDir, stableDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	_, nestedDir := newFakeWorktreeWithoutLocalBinary(t)
	wrapperPath := filepath.Join(installDir, "harness")
	wrapperResult := runCommand(
		t,
		nestedDir,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, installDir, stableDir),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected source-tree invocation without local binary to fail\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "No repo-local harness binary found at ")
	support.RequireContains(t, wrapperResult.Stderr, filepath.Join(".local", "bin", "harness"))
	if strings.Contains(wrapperResult.CombinedOutput(), "stable fallback harness") {
		t.Fatalf("expected source-tree invocation to refuse the stable PATH fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}
}

func copyInstallerFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	sourceRoot := support.RepoRoot(t)
	for _, rel := range []string{
		"go.mod",
		"go.sum",
		"assets",
		"cmd",
		"internal",
		"scripts/install-dev-harness",
	} {
		copyPath(t, filepath.Join(sourceRoot, rel), filepath.Join(root, rel))
	}
	return root
}

func copyPath(t *testing.T, src, dst string) {
	t.Helper()

	info, err := os.Stat(src)
	if err != nil {
		t.Fatalf("stat %s: %v", src, err)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			t.Fatalf("mkdir %s: %v", dst, err)
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			t.Fatalf("read dir %s: %v", src, err)
		}
		for _, entry := range entries {
			copyPath(t, filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
		}
		return
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir parent for %s: %v", dst, err)
	}

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

func newFakeWorktree(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "scripts"),
		filepath.Join(root, "cmd", "harness"),
		filepath.Join(root, ".local", "bin"),
		filepath.Join(root, "nested", "dir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeFixtureFile(t, filepath.Join(root, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
	writeFixtureFile(t, filepath.Join(root, "cmd", "harness", "main.go"), "package main\n", 0o644)
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module github.com/catu-ai/easyharness\n", 0o644)
	writeFixtureFile(
		t,
		filepath.Join(root, ".local", "bin", "harness"),
		"#!/bin/sh\nprintf 'fake worktree harness\\n'\nprintf 'args=%s\\n' \"$*\"\n",
		0o755,
	)

	return root, filepath.Join(root, "nested", "dir")
}

func newFakeWorktreeWithoutLocalBinary(t *testing.T) (string, string) {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "scripts"),
		filepath.Join(root, "cmd", "harness"),
		filepath.Join(root, "nested", "dir"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeFixtureFile(t, filepath.Join(root, "scripts", "install-dev-harness"), "#!/usr/bin/env bash\n", 0o755)
	writeFixtureFile(t, filepath.Join(root, "cmd", "harness", "main.go"), "package main\n", 0o644)
	writeFixtureFile(t, filepath.Join(root, "go.mod"), "module github.com/catu-ai/easyharness\n", 0o644)

	return root, filepath.Join(root, "nested", "dir")
}

func newFakeStableHarness(t *testing.T) (string, string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "harness")
	writeFixtureFile(
		t,
		path,
		`#!/bin/sh
set -eu

case "${1:-}" in
  --help)
    printf 'stable fallback harness help\n'
    ;;
  --version)
    printf 'version: 0.2.0\n'
    printf 'mode: release\n'
    printf 'commit: stable-test-commit\n'
    printf 'path: %s\n' "$0"
    ;;
  *)
    printf 'stable fallback harness\n'
    printf 'args=%s\n' "$*"
    ;;
esac
`,
		0o755,
	)
	return dir, path
}

func fakeManagedWrapperScript(marker string) string {
	return "#!/bin/sh\n" +
		"# easyharness-install-dev-wrapper\n" +
		"printf '" + marker + "\\n'\n"
}

func writeFixtureFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func envWithOverrides(t *testing.T, overrides map[string]string) []string {
	t.Helper()

	env := append([]string(nil), os.Environ()...)
	for key, value := range overrides {
		prefix := key + "="
		replaced := false
		for i, entry := range env {
			if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
				env[i] = prefix + value
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, prefix+value)
		}
	}
	return env
}

func installerEnv(t *testing.T, overrides map[string]string) []string {
	t.Helper()

	if overrides == nil {
		overrides = map[string]string{}
	}
	if _, ok := overrides["GOCACHE"]; !ok {
		overrides["GOCACHE"] = sharedInstallerGoCache(t)
	}
	if _, ok := overrides["GOMODCACHE"]; !ok {
		overrides["GOMODCACHE"] = sharedInstallerGoModCache(t)
	}
	if _, ok := overrides["GOFLAGS"]; !ok {
		overrides["GOFLAGS"] = "-modcacherw"
	}
	return envWithOverrides(t, overrides)
}

func sharedInstallerGoCache(t *testing.T) string {
	t.Helper()
	initializeInstallerCaches(t)
	return installerCacheDirs.goCache
}

func sharedInstallerGoModCache(t *testing.T) string {
	t.Helper()
	initializeInstallerCaches(t)
	return installerCacheDirs.goModCache
}

func initializeInstallerCaches(t *testing.T) {
	t.Helper()

	installerCacheDirs.once.Do(func() {
		root, err := os.MkdirTemp("", "easyharness-install-smoke-cache-*")
		if err != nil {
			installerCacheDirs.err = err
			return
		}
		installerCacheDirs.goCache = filepath.Join(root, "go-build")
		installerCacheDirs.goModCache = filepath.Join(root, "gomod")
		for _, dir := range []string{installerCacheDirs.goCache, installerCacheDirs.goModCache} {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				installerCacheDirs.err = err
				return
			}
		}
	})

	if installerCacheDirs.err != nil {
		t.Fatalf("initialize shared installer caches: %v", installerCacheDirs.err)
	}
}

func installerPath(t *testing.T, extraDirs ...string) string {
	t.Helper()

	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("find go on PATH: %v", err)
	}

	seen := map[string]bool{}
	dirs := make([]string, 0, len(extraDirs)+5)
	addDir := func(dir string) {
		if dir == "" || seen[dir] {
			return
		}
		seen[dir] = true
		dirs = append(dirs, dir)
	}

	for _, dir := range extraDirs {
		addDir(dir)
	}
	addDir(filepath.Dir(goPath))
	addDir("/usr/bin")
	addDir("/bin")
	addDir("/usr/sbin")
	addDir("/sbin")

	return strings.Join(dirs, string(os.PathListSeparator))
}

func runCommand(t *testing.T, workdir string, env []string, argv ...string) commandResult {
	t.Helper()

	return runCommandWithTimeout(t, 0, workdir, env, argv...)
}

func runCommandWithTimeout(t *testing.T, timeout time.Duration, workdir string, env []string, argv ...string) commandResult {
	t.Helper()

	var (
		cmd    *exec.Cmd
		cancel func()
	)
	if timeout > 0 {
		var ctx context.Context
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
	} else {
		cmd = exec.Command(argv[0], argv[1:]...)
	}

	cmd.Dir = workdir
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := commandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		return result
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("run command %v timed out after %s\nstdout:\n%s\nstderr:\n%s", argv, timeout, stdout.String(), stderr.String())
		}
		t.Fatalf("run command %v: %v", argv, err)
	}
	result.ExitCode = exitErr.ExitCode()
	return result
}
