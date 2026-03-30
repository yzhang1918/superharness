package smoke_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"debug/macho"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestBuildReleaseProducesSupportedAlphaArchivesAndVersionedBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	firstOutputDir := newReleaseOutputDir(t, "supported-alpha-a")
	secondOutputDir := newReleaseOutputDir(t, "supported-alpha-b")
	version := "v0.1.0-alpha.1"
	expectedCommit := gitHeadCommit(t, support.RepoRoot(t))
	expectedArchiveTime := gitCommitTimestampUTC(t, support.RepoRoot(t), expectedCommit)

	firstResult := runReleaseBuild(t, version, firstOutputDir)
	secondResult := runReleaseBuild(t, version, secondOutputDir)

	expectedPlatforms := []string{
		"darwin/amd64",
		"darwin/arm64",
		"linux/amd64",
		"linux/arm64",
	}

	firstChecksums := parseChecksums(t, readFile(t, filepath.Join(firstOutputDir, "SHA256SUMS")))
	secondChecksums := parseChecksums(t, readFile(t, filepath.Join(secondOutputDir, "SHA256SUMS")))
	for _, platform := range expectedPlatforms {
		goos, goarch := splitPlatform(t, platform)
		archiveName := "easyharness_" + version + "_" + goos + "_" + goarch + ".zip"
		firstArchivePath := filepath.Join(firstOutputDir, archiveName)
		secondArchivePath := filepath.Join(secondOutputDir, archiveName)
		if _, err := os.Stat(firstArchivePath); err != nil {
			t.Fatalf("expected archive %s: %v\n%s", firstArchivePath, err, firstResult)
		}
		if _, err := os.Stat(secondArchivePath); err != nil {
			t.Fatalf("expected archive %s: %v\n%s", secondArchivePath, err, secondResult)
		}
		firstChecksum := checksumFile(t, firstArchivePath)
		secondChecksum := checksumFile(t, secondArchivePath)
		if got := firstChecksums[archiveName]; got != firstChecksum {
			t.Fatalf("expected first checksum for %s to match file contents, got %q want %q", archiveName, got, firstChecksum)
		}
		if got := secondChecksums[archiveName]; got != secondChecksum {
			t.Fatalf("expected second checksum for %s to match file contents, got %q want %q", archiveName, got, secondChecksum)
		}
		if firstChecksum != secondChecksum {
			t.Fatalf("expected deterministic archive checksum for %s, got %q and %q", archiveName, firstChecksum, secondChecksum)
		}
		if !bytes.Equal(readFileBytes(t, firstArchivePath), readFileBytes(t, secondArchivePath)) {
			t.Fatalf("expected deterministic archive bytes for %s across identical builds", archiveName)
		}
		verifyArchiveContents(t, workspace, firstArchivePath, version, goos, goarch, expectedCommit, expectedArchiveTime)
	}
}

func TestBuildReleaseCleansReusedOutputDirectory(t *testing.T) {
	outputDir := newReleaseOutputDir(t, "reuse-output")

	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	version := "v0.1.0-alpha.1"
	runReleaseBuildForPlatforms(t, version, outputDir, hostPlatform)

	staleFile := filepath.Join(outputDir, "stale.txt")
	if err := os.WriteFile(staleFile, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	staleArchive := filepath.Join(outputDir, "easyharness_stale.zip")
	if err := os.WriteFile(staleArchive, []byte("stale archive"), 0o644); err != nil {
		t.Fatalf("write stale archive: %v", err)
	}

	runReleaseBuildForPlatforms(t, version, outputDir, hostPlatform)

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed, got err=%v", err)
	}
	if _, err := os.Stat(staleArchive); !os.IsNotExist(err) {
		t.Fatalf("expected stale archive to be removed, got err=%v", err)
	}

	goos, goarch := splitPlatform(t, hostPlatform)
	wantEntries := map[string]bool{
		"SHA256SUMS": true,
		"easyharness_" + version + "_" + goos + "_" + goarch + ".zip": true,
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("read output dir: %v", err)
	}
	if len(entries) != len(wantEntries) {
		t.Fatalf("expected %d release outputs, found %d", len(wantEntries), len(entries))
	}
	for _, entry := range entries {
		if !wantEntries[entry.Name()] {
			t.Fatalf("unexpected leftover release output %q", entry.Name())
		}
	}
}

func TestBuildReleaseRejectsUnsafeOutputDirectory(t *testing.T) {
	cases := []struct {
		name      string
		outputDir string
		wantText  string
	}{
		{
			name:      "repo root",
			outputDir: ".",
			wantText:  "refusing to use repository root as the release output directory",
		},
		{
			name:      "relative parent escape",
			outputDir: "../release-out",
			wantText:  "output directory must not contain parent-directory segments",
		},
		{
			name:      "tracked repo directory",
			outputDir: "docs/release-out",
			wantText:  "output directory must stay within repo-owned dist/ or .local/ subdirectories",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(
				"scripts/build-release",
				"--version", "v0.1.0-alpha.1",
				"--output-dir", tc.outputDir,
				"--platform", runtime.GOOS+"/"+runtime.GOARCH,
			)
			cmd.Dir = support.RepoRoot(t)
			result, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected build-release to reject output dir %q", tc.outputDir)
			}
			if !strings.Contains(string(result), tc.wantText) {
				t.Fatalf("expected output for %q to contain %q, got:\n%s", tc.outputDir, tc.wantText, result)
			}
		})
	}
}

func TestBuildReleaseSupportsOutputDirectoryWithSpaces(t *testing.T) {
	outputDir := newReleaseOutputDir(t, "space output")

	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	version := "v0.1.0-alpha.1"
	runReleaseBuildForPlatforms(t, version, outputDir, hostPlatform)

	goos, goarch := splitPlatform(t, hostPlatform)
	archiveName := "easyharness_" + version + "_" + goos + "_" + goarch + ".zip"
	checksums := parseChecksums(t, readFile(t, filepath.Join(outputDir, "SHA256SUMS")))
	archivePath := filepath.Join(outputDir, archiveName)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected archive in spaced output dir %s: %v", archivePath, err)
	}
	if got := checksums[archiveName]; got != checksumFile(t, archivePath) {
		t.Fatalf("expected checksum for %s to match file contents, got %q", archiveName, got)
	}
}

func TestBuildReleaseCreatesMissingSafeRootInFreshCheckout(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	checkoutRoot := newReleaseBuildCheckout(t)
	outputDir := filepath.Join("dist", "release")
	version := "v0.1.0-alpha.1"

	runReleaseBuildInDir(t, checkoutRoot, version, outputDir, hostPlatform)

	goos, goarch := splitPlatform(t, hostPlatform)
	archiveName := "easyharness_" + version + "_" + goos + "_" + goarch + ".zip"
	archivePath := filepath.Join(checkoutRoot, outputDir, archiveName)
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("expected archive in fresh checkout output dir %s: %v", archivePath, err)
	}
	if _, err := os.Stat(filepath.Join(checkoutRoot, outputDir, "SHA256SUMS")); err != nil {
		t.Fatalf("expected SHA256SUMS in fresh checkout output dir: %v", err)
	}
}

func TestBuildReleaseOnlyCleansPreparedLeafForNestedOutputDirectories(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	checkoutRoot := newReleaseBuildCheckout(t)
	outputDir := filepath.Join("dist", "nested-release", "a", "b")
	leafDir := filepath.Join(checkoutRoot, outputDir)
	keepDir := filepath.Join(checkoutRoot, "dist", "nested-release", "keep")
	rootSibling := filepath.Join(checkoutRoot, "dist", "dist-sibling.txt")
	if err := os.MkdirAll(leafDir, 0o755); err != nil {
		t.Fatalf("mkdir leaf dir: %v", err)
	}
	if err := os.MkdirAll(keepDir, 0o755); err != nil {
		t.Fatalf("mkdir keep dir: %v", err)
	}
	staleLeafFile := filepath.Join(leafDir, "stale.txt")
	if err := os.WriteFile(staleLeafFile, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale leaf file: %v", err)
	}
	keepSentinel := filepath.Join(keepDir, "sentinel.txt")
	if err := os.WriteFile(keepSentinel, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write keep sentinel: %v", err)
	}
	if err := os.WriteFile(rootSibling, []byte("root"), 0o644); err != nil {
		t.Fatalf("write dist-root sibling: %v", err)
	}

	version := "v0.1.0-alpha.1"
	runReleaseBuildInDir(t, checkoutRoot, version, outputDir, hostPlatform)

	if _, err := os.Stat(keepSentinel); err != nil {
		t.Fatalf("expected nested sibling sentinel to survive leaf cleanup, got: %v", err)
	}
	if _, err := os.Stat(rootSibling); err != nil {
		t.Fatalf("expected dist-root sibling to survive leaf cleanup, got: %v", err)
	}
	if _, err := os.Stat(staleLeafFile); !os.IsNotExist(err) {
		t.Fatalf("expected stale leaf file to be removed from prepared output directory, got err=%v", err)
	}

	goos, goarch := splitPlatform(t, hostPlatform)
	archiveName := "easyharness_" + version + "_" + goos + "_" + goarch + ".zip"
	if _, err := os.Stat(filepath.Join(leafDir, archiveName)); err != nil {
		t.Fatalf("expected release archive in prepared nested leaf: %v", err)
	}
	if _, err := os.Stat(filepath.Join(leafDir, "SHA256SUMS")); err != nil {
		t.Fatalf("expected SHA256SUMS in prepared nested leaf: %v", err)
	}
}

func TestBuildReleaseRejectsSymlinkEscapesFromAllowedOutputRoots(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	repoRoot := support.RepoRoot(t)
	distRoot := newReleaseDistFixtureDir(t, "symlink")
	externalRoot := t.TempDir()
	externalOutputDir := filepath.Join(externalRoot, "release-out")
	if err := os.MkdirAll(externalOutputDir, 0o755); err != nil {
		t.Fatalf("mkdir external output dir: %v", err)
	}
	sentinelPath := filepath.Join(externalOutputDir, "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	symlinkPath := filepath.Join(distRoot, "symlink-root")
	if err := os.Symlink(externalRoot, symlinkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(symlinkPath) })

	cmd := exec.Command(
		"scripts/build-release",
		"--version", "v0.1.0-alpha.1",
		"--output-dir", filepath.Join(symlinkPath, "release-out"),
		"--platform", hostPlatform,
	)
	cmd.Dir = repoRoot
	result, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build-release to reject symlinked output dir, output:\n%s", result)
	}
	if !strings.Contains(string(result), "output directory path must not traverse symlink segments") {
		t.Fatalf("expected symlink rejection message, got:\n%s", result)
	}
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("expected sentinel outside allowed roots to survive rejected build, got: %v", err)
	}
}

func TestBuildReleaseRejectsUnsafeVersion(t *testing.T) {
	outputDir := newReleaseOutputDir(t, "unsafe-version")

	cmd := exec.Command(
		"scripts/build-release",
		"--version", "v0.1.0/../../escape",
		"--output-dir", outputDir,
		"--platform", runtime.GOOS+"/"+runtime.GOARCH,
	)
	cmd.Dir = support.RepoRoot(t)
	result, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build-release to reject unsafe version, output:\n%s", result)
	}
	if !strings.Contains(string(result), "version must use only ASCII letters, digits, dot, underscore, plus, or hyphen") {
		t.Fatalf("expected unsafe-version rejection message, got:\n%s", result)
	}
}

func TestBuildReleaseRejectsOutputDirectoryReplacedBySymlinkAfterValidation(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	repoRoot := support.RepoRoot(t)
	distRoot := newReleaseDistFixtureDir(t, "race")
	outputDir := filepath.Join(distRoot, "output")
	redirectedOutputDir := filepath.Join(distRoot, "redirected")
	if err := os.MkdirAll(redirectedOutputDir, 0o755); err != nil {
		t.Fatalf("mkdir redirected output dir: %v", err)
	}

	realMkdir, err := exec.LookPath("mkdir")
	if err != nil {
		t.Fatalf("look up mkdir: %v", err)
	}
	fakeBin := t.TempDir()
	fakeMkdir := filepath.Join(fakeBin, "mkdir")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
if [ "$#" -eq 1 ] && [ "$1" = %q ] && [ ! -e "$1" ]; then
  %q "$@"
  rmdir "$1"
  ln -s %q "$1"
  exit 0
fi
exec %q "$@"
`, filepath.Base(outputDir), realMkdir, filepath.Base(redirectedOutputDir), realMkdir)
	if err := os.WriteFile(fakeMkdir, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake mkdir: %v", err)
	}

	cmd := exec.Command(
		"scripts/build-release",
		"--version", "v0.1.0-alpha.1",
		"--output-dir", outputDir,
		"--platform", hostPlatform,
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build-release to reject mkdir-time symlink replacement into a different safe-root directory, output:\n%s", result)
	}
	if !strings.Contains(string(result), "output directory path must stay on the requested directory path during preparation") {
		t.Fatalf("expected mkdir-race rejection message, got:\n%s", result)
	}
	entries, err := os.ReadDir(redirectedOutputDir)
	if err != nil {
		t.Fatalf("read redirected output dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected rejected build to avoid writing into the wrong safe-root directory, found %d entries", len(entries))
	}
}

func TestBuildReleaseRejectsPreparedOutputDirectoryBeingReplacedDuringBuild(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	repoRoot := support.RepoRoot(t)
	distRoot := newReleaseDistFixtureDir(t, "swap")
	outputDir := filepath.Join(distRoot, "output")
	redirectedOutputDir := filepath.Join(distRoot, "redirected")
	liveOutputDir := filepath.Join(distRoot, "live")
	if err := os.MkdirAll(redirectedOutputDir, 0o755); err != nil {
		t.Fatalf("mkdir redirected output dir: %v", err)
	}

	realZip, err := exec.LookPath("zip")
	if err != nil {
		t.Fatalf("look up zip: %v", err)
	}
	fakeBin := t.TempDir()
	fakeZip := filepath.Join(fakeBin, "zip")
	markerPath := filepath.Join(fakeBin, "zip-raced")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
if [ ! -e %q ]; then
  mv %q %q
  ln -s %q %q
  : > %q
fi
exec %q "$@"
`, markerPath, outputDir, liveOutputDir, redirectedOutputDir, outputDir, markerPath, realZip)
	if err := os.WriteFile(fakeZip, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake zip: %v", err)
	}

	cmd := exec.Command(
		"scripts/build-release",
		"--version", "v0.1.0-alpha.1",
		"--output-dir", outputDir,
		"--platform", hostPlatform,
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build-release to reject prepared output dir replacement during build, output:\n%s", result)
	}
	if !strings.Contains(string(result), "prepared output directory changed unexpectedly during build") {
		t.Fatalf("expected output-dir replacement rejection message, got:\n%s", result)
	}
	redirectedEntries, err := os.ReadDir(redirectedOutputDir)
	if err != nil {
		t.Fatalf("read redirected output dir: %v", err)
	}
	if len(redirectedEntries) != 0 {
		t.Fatalf("expected redirected output dir to stay empty, found %d entries", len(redirectedEntries))
	}
	liveEntries, err := os.ReadDir(liveOutputDir)
	if err != nil {
		t.Fatalf("read live output dir: %v", err)
	}
	if len(liveEntries) != 0 {
		t.Fatalf("expected prepared output dir to stay empty after rejection, found %d entries", len(liveEntries))
	}
}

func TestBuildReleaseDoesNotFollowSymlinkedOutputEntryDuringPublish(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	outputDir := newReleaseOutputDir(t, "publish-entry")
	redirectedOutputDir := t.TempDir()
	checksumToolName, checksumToolPath := releaseChecksumTool(t)
	goos, goarch := splitPlatform(t, hostPlatform)
	version := "v0.1.0-alpha.1"
	archiveName := "easyharness_" + version + "_" + goos + "_" + goarch + ".zip"
	archiveEntryPath := filepath.Join(outputDir, archiveName)

	fakeBin := t.TempDir()
	fakeChecksum := filepath.Join(fakeBin, checksumToolName)
	markerPath := filepath.Join(fakeBin, "checksum-raced")
	script := fmt.Sprintf(`#!/bin/sh
set -eu
if [ ! -e %q ]; then
  ln -s %q %q
  : > %q
fi
exec %q "$@"
`, markerPath, redirectedOutputDir, archiveEntryPath, markerPath, checksumToolPath)
	if err := os.WriteFile(fakeChecksum, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake checksum tool: %v", err)
	}

	cmd := exec.Command(
		"scripts/build-release",
		"--version", version,
		"--output-dir", outputDir,
		"--platform", hostPlatform,
	)
	cmd.Dir = support.RepoRoot(t)
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected build-release to safely replace symlinked output entry during publish, got:\n%s", result)
	}
	redirectedEntries, err := os.ReadDir(redirectedOutputDir)
	if err != nil {
		t.Fatalf("read redirected output dir: %v", err)
	}
	if len(redirectedEntries) != 0 {
		t.Fatalf("expected redirected output dir to stay empty, found %d entries", len(redirectedEntries))
	}
	info, err := os.Lstat(archiveEntryPath)
	if err != nil {
		t.Fatalf("stat published archive entry: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("expected archive entry to be replaced with a regular file, got mode %v", info.Mode())
	}
}

func TestBuildReleaseRejectsPreparedOutputDirectoryReplacementDuringPublish(t *testing.T) {
	hostPlatform := runtime.GOOS + "/" + runtime.GOARCH
	if !isSupportedAlphaPlatform(hostPlatform) {
		t.Skipf("host platform %s is outside the supported alpha target set", hostPlatform)
	}

	repoRoot := support.RepoRoot(t)
	outputDir := newReleaseOutputDir(t, "publish-dir-race")
	redirectedOutputDir := t.TempDir()
	liveOutputDir := outputDir + "-live"
	markerPath := filepath.Join(t.TempDir(), "publish-helper-raced")

	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("look up go: %v", err)
	}
	fakeBin := t.TempDir()
	fakeGo := filepath.Join(fakeBin, "go")
	wrapper := fmt.Sprintf(`#!/bin/sh
set -eu
real_go=%q
helper_source=%q
output_dir=%q
live_output_dir=%q
redirected_output_dir=%q
marker_path=%q
if [ "$#" -ge 4 ] && [ "$1" = build ] && [ "$2" = -trimpath ]; then
  prev=
  out=
  last=
  for arg in "$@"; do
    if [ "$prev" = -o ]; then
      out="$arg"
    fi
    prev="$arg"
    last="$arg"
  done
  if [ "$last" = "$helper_source" ]; then
    "$real_go" "$@"
    mv "$out" "$out.real"
    cat > "$out" <<EOF2
#!/bin/sh
set -eu
if [ -e %q ]; then
  mv %q %q
  ln -s %q %q
fi
if [ ! -e %q ]; then
  : > %q
else
  : > %q
fi
exec "$out.real" "\$@"
EOF2
    chmod +x "$out"
    exit 0
  fi
fi
exec "$real_go" "$@"
`, realGo, filepath.Join(repoRoot, "scripts", "release_publish.go"), outputDir, liveOutputDir, redirectedOutputDir, markerPath, markerPath, outputDir, liveOutputDir, redirectedOutputDir, outputDir, markerPath, markerPath, markerPath)
	if err := os.WriteFile(fakeGo, []byte(wrapper), 0o755); err != nil {
		t.Fatalf("write fake go wrapper: %v", err)
	}

	cmd := exec.Command(
		"scripts/build-release",
		"--version", "v0.1.0-alpha.1",
		"--output-dir", outputDir,
		"--platform", hostPlatform,
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	result, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected build-release to reject prepared output directory replacement during publish, output:\n%s", result)
	}
	if !strings.Contains(string(result), "prepared output directory changed unexpectedly during build") {
		t.Fatalf("expected publish-directory replacement message, got:\n%s", result)
	}
	redirectedEntries, err := os.ReadDir(redirectedOutputDir)
	if err != nil {
		t.Fatalf("read redirected output dir: %v", err)
	}
	if len(redirectedEntries) != 0 {
		t.Fatalf("expected redirected output dir to stay empty, found %d entries", len(redirectedEntries))
	}
}

func runReleaseBuild(t *testing.T, version, outputDir string) string {
	t.Helper()
	return runReleaseBuildForPlatforms(t, version, outputDir)
}

func runReleaseBuildInDir(t *testing.T, repoDir, version, outputDir string, platforms ...string) string {
	t.Helper()

	args := []string{"--version", version, "--output-dir", outputDir}
	for _, platform := range platforms {
		args = append(args, "--platform", platform)
	}
	cmd := exec.Command("scripts/build-release", args...)
	cmd.Dir = repoDir
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build release: %v\n%s", err, result)
	}
	return string(result)
}

func runReleaseBuildForPlatforms(t *testing.T, version, outputDir string, platforms ...string) string {
	t.Helper()
	return runReleaseBuildInDir(t, support.RepoRoot(t), version, outputDir, platforms...)
}

func verifyArchiveContents(t *testing.T, workspace *support.Workspace, archivePath, version, goos, goarch, expectedCommit string, expectedArchiveTime time.Time) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer reader.Close()

	packageRoot := "easyharness_" + version + "_" + goos + "_" + goarch + "/"
	binaryName := packageRoot + "harness"
	readmeName := packageRoot + "README.md"
	licenseName := packageRoot + "LICENSE"

	var sawBinary bool
	var sawReadme bool
	var sawLicense bool
	for _, file := range reader.File {
		archiveTime := file.Modified.UTC()
		delta := archiveTime.Sub(expectedArchiveTime)
		if delta < 0 {
			delta = -delta
		}
		if delta > time.Second {
			t.Fatalf("expected archive entry %s to stay within ZIP timestamp precision of commit time %s, got %s", file.Name, expectedArchiveTime.Format(time.RFC3339), archiveTime.Format(time.RFC3339))
		}
		if archiveTime.Year() == 2000 {
			t.Fatalf("expected archive entry %s to stop using the fixed year-2000 timestamp", file.Name)
		}
		switch file.Name {
		case binaryName:
			sawBinary = true
		case readmeName:
			sawReadme = true
		case licenseName:
			sawLicense = true
		}
	}
	if !sawBinary {
		t.Fatalf("expected archive to include %s", binaryName)
	}
	if !sawReadme {
		t.Fatalf("expected archive to include %s", readmeName)
	}
	if !sawLicense {
		t.Fatalf("expected archive to include %s", licenseName)
	}

	extractDir := workspace.Path(filepath.Join("extract", goos+"-"+goarch))
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		t.Fatalf("mkdir extract: %v", err)
	}
	if err := unzipArchive(archivePath, extractDir); err != nil {
		t.Fatalf("unzip archive: %v", err)
	}

	binaryPath := filepath.Join(extractDir, binaryName)
	verifyBinaryMetadata(t, binaryPath, version, goos, goarch, expectedCommit)
	if goos == runtime.GOOS && goarch == runtime.GOARCH {
		versionCmd := exec.Command(binaryPath, "--version")
		versionCmd.Dir = extractDir
		versionOutput, err := versionCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run packaged binary --version: %v\n%s", err, versionOutput)
		}

		output := string(versionOutput)
		if got := requireVersionField(t, output, "version"); got != version {
			t.Fatalf("expected packaged version %q, got %q\noutput:\n%s", version, got, output)
		}
		if got := requireVersionField(t, output, "mode"); got != "release" {
			t.Fatalf("expected packaged mode release, got %q\noutput:\n%s", got, output)
		}
		if got := requireVersionField(t, output, "commit"); got != expectedCommit {
			t.Fatalf("expected packaged commit %q, got %q\noutput:\n%s", expectedCommit, got, output)
		}
		if strings.Contains(output, "path: ") {
			t.Fatalf("expected packaged release output to omit path, got %q", output)
		}

		statusCmd := exec.Command(binaryPath, "status")
		statusCmd.Dir = workspace.Root
		statusOutput, err := statusCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run packaged binary status: %v\n%s", err, statusOutput)
		}

		if !strings.Contains(string(statusOutput), `"current_node": "idle"`) {
			t.Fatalf("expected packaged binary status output to report idle workspace, got:\n%s", statusOutput)
		}
	}
}

func gitCommitTimestampUTC(t *testing.T, repoRoot, commit string) time.Time {
	t.Helper()

	cmd := exec.Command("git", "-C", repoRoot, "show", "-s", "--format=%ct", commit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git show commit timestamp: %v\n%s", err, output)
	}

	secondsText := strings.TrimSpace(string(output))
	seconds, err := strconv.ParseInt(secondsText, 10, 64)
	if err != nil {
		t.Fatalf("parse commit timestamp %q: %v", secondsText, err)
	}

	ts := time.Unix(seconds, 0).UTC()
	return ts
}

func verifyBinaryMetadata(t *testing.T, binaryPath, version, goos, goarch, expectedCommit string) {
	t.Helper()

	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read Go build info for %s: %v", binaryPath, err)
	}
	if info.GoVersion == "" {
		t.Fatalf("expected Go build info in %s", binaryPath)
	}
	if info.Main.Path != "github.com/catu-ai/easyharness" {
		t.Fatalf("expected binary %s to record module path %q, got %q", binaryPath, "github.com/catu-ai/easyharness", info.Main.Path)
	}
	if info.Path != "github.com/catu-ai/easyharness/cmd/harness" {
		t.Fatalf("expected binary %s to record main package path %q, got %q", binaryPath, "github.com/catu-ai/easyharness/cmd/harness", info.Path)
	}

	binaryData := readFileBytes(t, binaryPath)
	if !bytes.Contains(binaryData, []byte(version)) {
		t.Fatalf("expected binary %s to contain release version %q", binaryPath, version)
	}
	if !bytes.Contains(binaryData, []byte(expectedCommit)) {
		t.Fatalf("expected binary %s to contain build commit %q", binaryPath, expectedCommit)
	}

	switch goos {
	case "darwin":
		file, err := macho.Open(binaryPath)
		if err != nil {
			t.Fatalf("open Mach-O binary %s: %v", binaryPath, err)
		}
		defer file.Close()

		wantCPU := macho.CpuAmd64
		if goarch == "arm64" {
			wantCPU = macho.CpuArm64
		}
		if file.Cpu != wantCPU {
			t.Fatalf("expected Mach-O CPU %v for %s, got %v", wantCPU, binaryPath, file.Cpu)
		}
	case "linux":
		file, err := elf.Open(binaryPath)
		if err != nil {
			t.Fatalf("open ELF binary %s: %v", binaryPath, err)
		}
		defer file.Close()

		wantMachine := elf.EM_X86_64
		if goarch == "arm64" {
			wantMachine = elf.EM_AARCH64
		}
		if file.FileHeader.Machine != wantMachine {
			t.Fatalf("expected ELF machine %v for %s, got %v", wantMachine, binaryPath, file.FileHeader.Machine)
		}
	default:
		t.Fatalf("unsupported target OS %q", goos)
	}
}

func parseChecksums(t *testing.T, data string) map[string]string {
	t.Helper()

	checksums := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			t.Fatalf("malformed checksum line %q", line)
		}
		checksums[fields[len(fields)-1]] = fields[0]
	}
	return checksums
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	return string(readFileBytes(t, path))
}

func readFileBytes(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}

func checksumFile(t *testing.T, path string) string {
	t.Helper()

	data := readFileBytes(t, path)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func splitPlatform(t *testing.T, platform string) (string, string) {
	t.Helper()

	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid platform %q", platform)
	}
	return parts[0], parts[1]
}

func isSupportedAlphaPlatform(platform string) bool {
	switch platform {
	case "darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64":
		return true
	default:
		return false
	}
}

func newReleaseOutputDir(t *testing.T, prefix string) string {
	t.Helper()

	baseDir := filepath.Join(support.RepoRoot(t), ".local", "release-smoke")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir release smoke base dir: %v", err)
	}
	outputDir, err := os.MkdirTemp(baseDir, prefix+"-*")
	if err != nil {
		t.Fatalf("mktemp release output dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(outputDir)
	})
	return outputDir
}

func newReleaseDistFixtureDir(t *testing.T, prefix string) string {
	t.Helper()

	baseDir := filepath.Join(support.RepoRoot(t), "dist", "release-smoke-fixtures")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir dist fixture base dir: %v", err)
	}
	fixtureDir, err := os.MkdirTemp(baseDir, prefix+"-*")
	if err != nil {
		t.Fatalf("mktemp dist fixture dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(fixtureDir)
	})
	return fixtureDir
}

func newReleaseBuildCheckout(t *testing.T) string {
	t.Helper()

	repoRoot := support.RepoRoot(t)
	checkoutRoot := filepath.Join(t.TempDir(), "checkout")
	cmd := exec.Command("git", "-C", repoRoot, "worktree", "add", "--detach", checkoutRoot, "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git worktree add: %v\n%s", err, output)
	}
	t.Cleanup(func() {
		removeCmd := exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", checkoutRoot)
		if cleanupOutput, cleanupErr := removeCmd.CombinedOutput(); cleanupErr != nil {
			t.Errorf("git worktree remove: %v\n%s", cleanupErr, cleanupOutput)
		}
	})

	scriptPath := filepath.Join(repoRoot, "scripts", "build-release")
	scriptData := readFileBytes(t, scriptPath)
	checkoutScriptPath := filepath.Join(checkoutRoot, "scripts", "build-release")
	if err := os.WriteFile(checkoutScriptPath, scriptData, 0o755); err != nil {
		t.Fatalf("write checkout build-release script: %v", err)
	}
	publishHelperPath := filepath.Join(repoRoot, "scripts", "release_publish.go")
	publishHelperData := readFileBytes(t, publishHelperPath)
	checkoutPublishHelperPath := filepath.Join(checkoutRoot, "scripts", "release_publish.go")
	if err := os.WriteFile(checkoutPublishHelperPath, publishHelperData, 0o644); err != nil {
		t.Fatalf("write checkout release_publish helper: %v", err)
	}

	return checkoutRoot
}

func releaseChecksumTool(t *testing.T) (string, string) {
	t.Helper()

	if path, err := exec.LookPath("sha256sum"); err == nil {
		return "sha256sum", path
	}
	if path, err := exec.LookPath("shasum"); err == nil {
		return "shasum", path
	}
	t.Fatal("expected sha256sum or shasum to be available")
	return "", ""
}

func unzipArchive(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath := filepath.Join(destDir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(targetPath, destDir+string(os.PathSeparator)) && targetPath != destDir {
			return os.ErrPermission
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return err
		}
		if err := os.WriteFile(targetPath, data, file.Mode()); err != nil {
			return err
		}
	}

	return nil
}
