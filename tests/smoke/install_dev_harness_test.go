package smoke_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"

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
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+expectedWrapper)
	support.RequireFileExists(t, expectedWrapper)
	support.RequireFileMissing(t, filepath.Join(firstPathDir, "harness"))

	info, err := os.Lstat(expectedWrapper)
	if err != nil {
		t.Fatalf("lstat wrapper: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected %s to be a wrapper file, not a symlink", expectedWrapper)
	}
}

func TestInstallDevHarnessGlobalDefaultsToUserLocalBinAndRefreshesFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness --global failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	expectedWrapper := filepath.Join(tempHome, ".local", "bin", "harness")
	support.RequireFileExists(t, expectedWrapper)
	support.RequireContains(t, result.Stdout, "Installed harness wrapper at "+expectedWrapper)

	expectedGlobalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	support.RequireFileExists(t, expectedGlobalFallback)
	support.RequireContains(t, result.Stdout, "Updated global fallback binary at "+expectedGlobalFallback)

	writeFixtureFile(t, expectedGlobalFallback, "#!/bin/sh\nprintf 'stale-fallback\\n'\n", 0o755)

	refreshResult := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if refreshResult.ExitCode != 0 {
		t.Fatalf("second install-dev-harness --global failed with exit %d\nstdout:\n%s\nstderr:\n%s", refreshResult.ExitCode, refreshResult.Stdout, refreshResult.Stderr)
	}

	fallbackResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		expectedWrapper,
		"--help",
	)
	if fallbackResult.ExitCode != 0 {
		t.Fatalf("wrapper using refreshed global fallback failed with exit %d\nstdout:\n%s\nstderr:\n%s", fallbackResult.ExitCode, fallbackResult.Stdout, fallbackResult.Stderr)
	}
	if strings.Contains(fallbackResult.CombinedOutput(), "stale-fallback") {
		t.Fatalf("expected --global refresh to overwrite stale fallback contents\nstdout:\n%s\nstderr:\n%s", fallbackResult.Stdout, fallbackResult.Stderr)
	}
	support.RequireContains(t, refreshResult.Stdout, "Updated global fallback binary at "+expectedGlobalFallback)
}

func TestInstallDevHarnessGlobalRefreshReplacesFallbackFileObject(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()

	initialResult := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if initialResult.ExitCode != 0 {
		t.Fatalf("initial install-dev-harness --global failed with exit %d\nstdout:\n%s\nstderr:\n%s", initialResult.ExitCode, initialResult.Stdout, initialResult.Stderr)
	}

	globalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	initialInode := fileInode(t, globalFallback)
	writeFixtureFile(t, globalFallback, "#!/bin/sh\nprintf 'stale-global\\n'\n", 0o755)

	refreshResult := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
	)
	if refreshResult.ExitCode != 0 {
		t.Fatalf("refresh install-dev-harness --global failed with exit %d\nstdout:\n%s\nstderr:\n%s", refreshResult.ExitCode, refreshResult.Stdout, refreshResult.Stderr)
	}

	refreshedInode := fileInode(t, globalFallback)
	if refreshedInode == initialInode {
		t.Fatalf("expected --global refresh to replace the fallback file object, but inode stayed %d", refreshedInode)
	}

	versionResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		globalFallback,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("refreshed fallback version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}
	support.RequireContains(t, versionResult.Stdout, "mode: dev")
}

func TestInstallDevHarnessGlobalRejectsWrapperInstallDirConflict(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()
	conflictingDir := filepath.Join(tempHome, ".local", "share", "easyharness", "dev")
	conflictingBinary := filepath.Join(conflictingDir, "harness")
	if err := os.MkdirAll(conflictingDir, 0o755); err != nil {
		t.Fatalf("mkdir conflicting dir: %v", err)
	}
	writeFixtureFile(t, conflictingBinary, "#!/bin/sh\nprintf 'old-global\\n'\n", 0o755)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
		"--install-dir", conflictingDir,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected conflicting --install-dir to fail\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
	}
	support.RequireContains(t, result.Stderr, "Refusing to install the wrapper over the global fallback binary")

	fallbackContents, err := os.ReadFile(conflictingBinary)
	if err != nil {
		t.Fatalf("read conflicting fallback after failed install: %v", err)
	}
	if string(fallbackContents) != "#!/bin/sh\nprintf 'old-global\\n'\n" {
		t.Fatalf("expected failed install to leave existing fallback untouched, got:\n%s", string(fallbackContents))
	}
}

func TestInstallDevHarnessRepairsInvalidExistingGlobalFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()
	globalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	if err := os.MkdirAll(filepath.Dir(globalFallback), 0o755); err != nil {
		t.Fatalf("mkdir global fallback dir: %v", err)
	}
	writeFixtureFile(t, globalFallback, "#!/bin/sh\nkill -9 $$\n", 0o755)

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(tempHome, ".local", "bin", "harness")
	support.RequireContains(t, result.Stdout, "Repaired invalid global fallback binary at "+globalFallback)

	versionResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("wrapper version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}
	support.RequireContains(t, versionResult.Stdout, "mode: dev")
	if strings.Contains(versionResult.CombinedOutput(), "killed") {
		t.Fatalf("expected repaired fallback to avoid killed output\nstdout:\n%s\nstderr:\n%s", versionResult.Stdout, versionResult.Stderr)
	}
}

func TestInstallDevHarnessRepairsBrokenSymlinkGlobalFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()
	globalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	if err := os.MkdirAll(filepath.Dir(globalFallback), 0o755); err != nil {
		t.Fatalf("mkdir global fallback dir: %v", err)
	}
	if err := os.Symlink(filepath.Join(tempHome, "missing-fallback"), globalFallback); err != nil {
		t.Fatalf("create broken fallback symlink: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	support.RequireContains(t, result.Stdout, "Repaired invalid global fallback binary at "+globalFallback)

	info, err := os.Lstat(globalFallback)
	if err != nil {
		t.Fatalf("lstat repaired fallback: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected repaired fallback to replace the broken symlink")
	}

	versionResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		globalFallback,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("repaired broken-symlink fallback version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}
	support.RequireContains(t, versionResult.Stdout, "mode: dev")
}

func TestInstallDevHarnessRepairsDirectorySymlinkGlobalFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	tempHome := t.TempDir()
	globalFallback := filepath.Join(tempHome, ".local", "share", "easyharness", "dev", "harness")
	symlinkTargetDir := filepath.Join(tempHome, "fallback-dir")
	if err := os.MkdirAll(filepath.Dir(globalFallback), 0o755); err != nil {
		t.Fatalf("mkdir global fallback dir: %v", err)
	}
	if err := os.MkdirAll(symlinkTargetDir, 0o755); err != nil {
		t.Fatalf("mkdir symlink target dir: %v", err)
	}
	if err := os.Symlink(symlinkTargetDir, globalFallback); err != nil {
		t.Fatalf("create directory fallback symlink: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": tempHome,
			"PATH": installerPath(t, filepath.Join(tempHome, ".local", "bin")),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	support.RequireContains(t, result.Stdout, "Repaired invalid global fallback binary at "+globalFallback)

	info, err := os.Lstat(globalFallback)
	if err != nil {
		t.Fatalf("lstat repaired directory-symlink fallback: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected repaired fallback to replace the directory symlink")
	}

	targetEntries, err := os.ReadDir(symlinkTargetDir)
	if err != nil {
		t.Fatalf("read repaired symlink target dir: %v", err)
	}
	if len(targetEntries) != 0 {
		t.Fatalf("expected repaired directory-symlink target dir to stay empty, found %d entries", len(targetEntries))
	}

	versionResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		globalFallback,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("repaired directory-symlink fallback version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}
	support.RequireContains(t, versionResult.Stdout, "mode: dev")
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

func TestInstallDevHarnessWrapperDispatchesToCurrentWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
	homeDir := t.TempDir()

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)
	globalFallback := filepath.Join(homeDir, ".local", "share", "easyharness", "dev", "harness")
	writeFixtureFile(t, globalFallback, "#!/bin/sh\nprintf 'unexpected-global\\n'\n", 0o755)

	_, nestedDir := newFakeWorktree(t)
	wrapperResult := runCommand(
		t,
		nestedDir,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode != 0 {
		t.Fatalf("wrapper failed with exit %d\nstdout:\n%s\nstderr:\n%s", wrapperResult.ExitCode, wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stdout, "fake worktree harness")
	support.RequireContains(t, wrapperResult.Stdout, "args=status")
	if strings.Contains(wrapperResult.CombinedOutput(), "unexpected-global") {
		t.Fatalf("expected wrapper to prefer the worktree-local binary over the global fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}
}

func TestInstallDevHarnessWrapperRequiresExplicitGlobalFallbackOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")

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

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)

	otherProject := t.TempDir()
	wrapperResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"--help",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected wrapper without --global fallback to fail outside easyharness source trees\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "Could not find an easyharness source tree")
	support.RequireContains(t, wrapperResult.Stderr, "scripts/install-dev-harness --global")
}

func TestInstallDevHarnessWrapperUsesExplicitGlobalFallbackOutsideWorktree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
	homeDir := t.TempDir()

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	support.RequireFileExists(t, wrapperPath)
	globalFallback := filepath.Join(homeDir, ".local", "share", "easyharness", "dev", "harness")
	support.RequireFileExists(t, globalFallback)
	support.RequireContains(t, result.Stdout, "Updated global fallback binary at "+globalFallback)

	otherProject := t.TempDir()
	helpResult := runCommand(
		t,
		otherProject,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"--help",
	)
	if helpResult.ExitCode != 0 {
		t.Fatalf("wrapper global fallback failed with exit %d\nstdout:\n%s\nstderr:\n%s", helpResult.ExitCode, helpResult.Stdout, helpResult.Stderr)
	}

	support.RequireContains(t, helpResult.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
}

func TestInstallDevHarnessVersionReportsDevModeAndPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
	expectedCommit := "0123456789abcdef0123456789abcdef01234567"
	fakeGitDir := fakeGitDirForHeadCommit(t, repoRoot, expectedCommit)
	homeDir := t.TempDir()

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t, fakeGitDir),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
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
			"PATH": installerPath(t, fakeGitDir),
		}),
		wrapperPath,
		"--version",
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("wrapper version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}

	if mode := requireVersionField(t, versionResult.Stdout, "mode"); mode != "dev" {
		t.Fatalf("expected dev mode, got %q\noutput:\n%s", mode, versionResult.Stdout)
	}
	if commit := requireVersionField(t, versionResult.Stdout, "commit"); commit != expectedCommit {
		t.Fatalf("expected injected dev commit %q, got %q\noutput:\n%s", expectedCommit, commit, versionResult.Stdout)
	}
	expectedPath := filepath.Join(homeDir, ".local", "share", "easyharness", "dev", "harness")
	if path := requireVersionField(t, versionResult.Stdout, "path"); path != expectedPath {
		t.Fatalf("expected dev path %q, got %q\noutput:\n%s", expectedPath, path, versionResult.Stdout)
	}
	if strings.HasPrefix(strings.TrimSpace(versionResult.Stdout), "{") {
		t.Fatalf("expected plain-text version output, got %q", versionResult.Stdout)
	}
}

func TestInstallDevHarnessNormalInstallDoesNotReplaceExistingGlobalFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoOne := copyInstallerFixture(t)
	repoTwo := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
	homeDir := t.TempDir()

	firstInstall := runCommand(
		t,
		repoOne,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoOne, "scripts", "install-dev-harness"),
		"--global",
		"--install-dir", installDir,
	)
	if firstInstall.ExitCode != 0 {
		t.Fatalf("first install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", firstInstall.ExitCode, firstInstall.Stdout, firstInstall.Stderr)
	}

	globalFallback := filepath.Join(homeDir, ".local", "share", "easyharness", "dev", "harness")
	writeFixtureFile(t, globalFallback, "#!/bin/sh\nprintf 'global-from-first\\n'\n", 0o755)
	initialInode := fileInode(t, globalFallback)

	secondInstall := runCommand(
		t,
		repoTwo,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoTwo, "scripts", "install-dev-harness"),
		"--install-dir", installDir,
	)
	if secondInstall.ExitCode != 0 {
		t.Fatalf("second install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", secondInstall.ExitCode, secondInstall.Stdout, secondInstall.Stderr)
	}

	support.RequireContains(t, secondInstall.Stdout, "Global fallback binary remains at "+globalFallback)
	if refreshedInode := fileInode(t, globalFallback); refreshedInode != initialInode {
		t.Fatalf("expected healthy global fallback inode to remain %d, got %d", initialInode, refreshedInode)
	}

	wrapperPath := filepath.Join(installDir, "harness")
	wrapperResult := runCommand(
		t,
		t.TempDir(),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"--help",
	)
	if wrapperResult.ExitCode != 0 {
		t.Fatalf("wrapper with preserved global fallback failed with exit %d\nstdout:\n%s\nstderr:\n%s", wrapperResult.ExitCode, wrapperResult.Stdout, wrapperResult.Stderr)
	}
	support.RequireContains(t, wrapperResult.Stdout, "global-from-first")
}

func TestInstallDevHarnessReplacesLegacyManagedWrapperWithoutForce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
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
			installDir := filepath.Join(t.TempDir(), "global-bin")
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

func TestInstallDevHarnessWrapperDoesNotUseGlobalFallbackInsideSourceTreeWithoutLocalBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("installer smoke tests require a POSIX shell")
	}

	repoRoot := copyInstallerFixture(t)
	installDir := filepath.Join(t.TempDir(), "global-bin")
	homeDir := t.TempDir()

	result := runCommand(
		t,
		repoRoot,
		installerEnv(t, map[string]string{
			"HOME": homeDir,
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "install-dev-harness"),
		"--global",
		"--install-dir", installDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("install-dev-harness failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	_, nestedDir := newFakeWorktreeWithoutLocalBinary(t)
	globalFallback := filepath.Join(homeDir, ".local", "share", "easyharness", "dev", "harness")
	writeFixtureFile(t, globalFallback, "#!/bin/sh\nprintf 'unexpected-global-fallback\\n'\n", 0o755)

	wrapperPath := filepath.Join(installDir, "harness")
	wrapperResult := runCommand(
		t,
		nestedDir,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		wrapperPath,
		"status",
	)
	if wrapperResult.ExitCode == 0 {
		t.Fatalf("expected source-tree invocation without local binary to fail\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
	}

	support.RequireContains(t, wrapperResult.Stderr, "No repo-local harness binary found at ")
	support.RequireContains(t, wrapperResult.Stderr, filepath.Join(".local", "bin", "harness"))
	if strings.Contains(wrapperResult.CombinedOutput(), "unexpected-global-fallback") {
		t.Fatalf("expected source-tree invocation to refuse the global fallback\nstdout:\n%s\nstderr:\n%s", wrapperResult.Stdout, wrapperResult.Stderr)
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

func writeFixtureFile(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func fileInode(t *testing.T, path string) uint64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("expected syscall.Stat_t for %s, got %T", path, info.Sys())
	}
	return stat.Ino
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

func fakeGitDirForHeadCommit(t *testing.T, repoRoot, commit string) string {
	t.Helper()

	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("find git on PATH: %v", err)
	}

	dir := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
set -eu

repo_root=%q
fake_commit=%q
real_git=%q

if [ "$#" -ge 4 ] && [ "$1" = "-C" ] && [ "$2" = "$repo_root" ] && [ "$3" = "rev-parse" ] && [ "$4" = "HEAD" ]; then
  printf '%%s\n' "$fake_commit"
  exit 0
fi

exec "$real_git" "$@"
`, repoRoot, commit, realGit)
	writeFixtureFile(t, filepath.Join(dir, "git"), script, 0o755)
	return dir
}

func runCommand(t *testing.T, workdir string, env []string, argv ...string) commandResult {
	t.Helper()

	cmd := exec.Command(argv[0], argv[1:]...)
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
		t.Fatalf("run command %v: %v", argv, err)
	}
	result.ExitCode = exitErr.ExitCode()
	return result
}
