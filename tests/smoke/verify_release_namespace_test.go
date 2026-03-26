package smoke_test

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/catu-ai/microharness/tests/support"
)

func TestVerifyReleaseNamespaceWithFakeGHDownloadsAndChecksums(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("verify-release-namespace smoke test requires a POSIX shell")
	}

	repo := "catu-ai/microharness"
	tag := "v0.1.0-alpha.4"
	archiveName := "microharness_v0.1.0-alpha.4_darwin_arm64.zip"
	archiveBody := []byte("fake archive bytes")
	checksum := sha256.Sum256(archiveBody)
	fakeGHDir := fakeGHReleaseDir(
		t,
		repo,
		tag,
		map[string][]byte{
			archiveName:  archiveBody,
			"SHA256SUMS": []byte(fmt.Sprintf("%s  %s\n", hex.EncodeToString(checksum[:]), archiveName)),
		},
	)

	downloadDir := filepath.Join(t.TempDir(), "downloads")
	result := runCommand(
		t,
		support.RepoRoot(t),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, fakeGHDir),
		}),
		"/bin/bash",
		filepath.Join(support.RepoRoot(t), "scripts", "verify-release-namespace"),
		"--repo", repo,
		"--tag", tag,
		"--asset", "SHA256SUMS",
		"--asset", archiveName,
		"--download-dir", downloadDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("verify-release-namespace failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stdout, "Verified repo: catu-ai/microharness")
	support.RequireContains(t, result.Stdout, "Verified release: v0.1.0-alpha.4")
	support.RequireContains(t, result.Stdout, "Verified downloaded assets in "+downloadDir)
	support.RequireFileExists(t, filepath.Join(downloadDir, "SHA256SUMS"))
	support.RequireFileExists(t, filepath.Join(downloadDir, archiveName))

	logData, err := os.ReadFile(filepath.Join(fakeGHDir, "gh.log"))
	if err != nil {
		t.Fatalf("read fake gh log: %v", err)
	}
	support.RequireContains(t, string(logData), "repo view catu-ai/microharness --json nameWithOwner,url")
	support.RequireContains(t, string(logData), "release view v0.1.0-alpha.4 -R catu-ai/microharness --json url,tagName,assets")
	support.RequireContains(t, string(logData), "release download v0.1.0-alpha.4 -R catu-ai/microharness -D "+downloadDir+" --clobber -p SHA256SUMS -p "+archiveName)
}

func TestVerifyReleaseNamespaceFailsWhenAssetIsMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("verify-release-namespace smoke test requires a POSIX shell")
	}

	fakeGHDir := fakeGHReleaseDir(
		t,
		"catu-ai/microharness",
		"v0.1.0-alpha.4",
		map[string][]byte{
			"SHA256SUMS": []byte(""),
		},
	)

	result := runCommand(
		t,
		support.RepoRoot(t),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, fakeGHDir),
		}),
		"/bin/bash",
		filepath.Join(support.RepoRoot(t), "scripts", "verify-release-namespace"),
		"--repo", "catu-ai/microharness",
		"--tag", "v0.1.0-alpha.4",
		"--asset", "microharness_v0.1.0-alpha.4_darwin_arm64.zip",
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected verify-release-namespace to fail when the requested asset is missing")
	}
	support.RequireContains(t, result.Stderr, "is missing required asset microharness_v0.1.0-alpha.4_darwin_arm64.zip")
}

func TestVerifyReleaseNamespaceFailsWhenChecksumDoesNotMatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("verify-release-namespace smoke test requires a POSIX shell")
	}

	repo := "catu-ai/microharness"
	tag := "v0.1.0-alpha.4"
	archiveName := "microharness_v0.1.0-alpha.4_darwin_arm64.zip"
	archiveBody := []byte("tampered archive bytes")
	wrongChecksum := sha256.Sum256([]byte("different bytes"))
	fakeGHDir := fakeGHReleaseDir(
		t,
		repo,
		tag,
		map[string][]byte{
			archiveName:  archiveBody,
			"SHA256SUMS": []byte(fmt.Sprintf("%s  %s\n", hex.EncodeToString(wrongChecksum[:]), archiveName)),
		},
	)

	downloadDir := filepath.Join(t.TempDir(), "downloads")
	result := runCommand(
		t,
		support.RepoRoot(t),
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t, fakeGHDir),
		}),
		"/bin/bash",
		filepath.Join(support.RepoRoot(t), "scripts", "verify-release-namespace"),
		"--repo", repo,
		"--tag", tag,
		"--asset", "SHA256SUMS",
		"--asset", archiveName,
		"--download-dir", downloadDir,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected verify-release-namespace to fail when the downloaded asset checksum does not match SHA256SUMS")
	}
	support.RequireContains(t, result.Stderr, "checksum mismatch for "+archiveName)
}

func TestVerifyReleaseNamespaceAgainstGitHubWhenEnabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("verify-release-namespace smoke test requires a POSIX shell")
	}
	if os.Getenv("MICROHARNESS_RUN_LIVE_GH_SMOKE") != "1" {
		t.Skip("set MICROHARNESS_RUN_LIVE_GH_SMOKE=1 to enable live GitHub verification")
	}

	repo := requiredEnv(t, "MICROHARNESS_LIVE_GH_REPO")
	tag := requiredEnv(t, "MICROHARNESS_LIVE_GH_TAG")
	asset := requiredEnv(t, "MICROHARNESS_LIVE_GH_ASSET")

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		t.Fatalf("find gh on PATH: %v", err)
	}

	downloadDir := filepath.Join(t.TempDir(), "downloads")
	result := runCommand(
		t,
		support.RepoRoot(t),
		envWithOverrides(t, map[string]string{
			"PATH": strings.Join([]string{filepath.Dir(ghPath), installerPath(t)}, string(os.PathListSeparator)),
		}),
		"/bin/bash",
		filepath.Join(support.RepoRoot(t), "scripts", "verify-release-namespace"),
		"--repo", repo,
		"--tag", tag,
		"--asset", "SHA256SUMS",
		"--asset", asset,
		"--download-dir", downloadDir,
	)
	if result.ExitCode != 0 {
		t.Fatalf("live verify-release-namespace failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	support.RequireContains(t, result.Stdout, "Verified repo: "+repo)
	support.RequireContains(t, result.Stdout, "Verified release: "+tag)
	support.RequireContains(t, result.Stdout, "Verified downloaded assets in "+downloadDir)
	support.RequireFileExists(t, filepath.Join(downloadDir, "SHA256SUMS"))
	support.RequireFileExists(t, filepath.Join(downloadDir, asset))

	extractDir := filepath.Join(t.TempDir(), "extract")
	extractZipAsset(t, filepath.Join(downloadDir, asset), extractDir)
	binaryPath := filepath.Join(extractDir, strings.TrimSuffix(asset, ".zip"), "harness")
	versionCmd := exec.Command(binaryPath, "--version")
	versionCmd.Dir = support.RepoRoot(t)
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run downloaded harness --version: %v\n%s", err, versionOutput)
	}
	support.RequireContains(t, string(versionOutput), "version: "+tag)
	support.RequireContains(t, string(versionOutput), "mode: release")
}

func fakeGHReleaseDir(t *testing.T, repo, tag string, assets map[string][]byte) string {
	t.Helper()

	dir := t.TempDir()
	assetsDir := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets dir: %v", err)
	}
	for name, contents := range assets {
		if err := os.WriteFile(filepath.Join(assetsDir, name), contents, 0o644); err != nil {
			t.Fatalf("write fake asset %s: %v", name, err)
		}
	}

	assetJSON := make([]string, 0, len(assets))
	for name := range assets {
		assetJSON = append(assetJSON, fmt.Sprintf(`{"name":"%s"}`, name))
	}
	releaseJSON := fmt.Sprintf(`{"url":"https://github.com/%s/releases/tag/%s","tagName":"%s","assets":[%s]}`, repo, tag, tag, strings.Join(assetJSON, ","))
	repoJSON := fmt.Sprintf(`{"nameWithOwner":"%s","url":"https://github.com/%s"}`, repo, repo)
	if err := os.WriteFile(filepath.Join(dir, "repo.json"), []byte(repoJSON), 0o644); err != nil {
		t.Fatalf("write repo json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "release.json"), []byte(releaseJSON), 0o644); err != nil {
		t.Fatalf("write release json: %v", err)
	}

	script := fmt.Sprintf(`#!/usr/bin/env bash
set -euo pipefail

log_file=%q
repo_json=%q
release_json=%q
assets_dir=%q

printf '%%s\n' "$*" >> "$log_file"

if [[ "$#" -ge 4 && "$1" == "repo" && "$2" == "view" ]]; then
  cat "$repo_json"
  exit 0
fi

if [[ "$#" -ge 6 && "$1" == "release" && "$2" == "view" ]]; then
  cat "$release_json"
  exit 0
fi

if [[ "$#" -ge 6 && "$1" == "release" && "$2" == "download" ]]; then
  dest=""
  patterns=()
  shift 2
  while (($#)); do
    case "$1" in
      -D)
        dest="$2"
        shift 2
        ;;
      -p)
        patterns+=("$2")
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done
  mkdir -p "$dest"
  for pattern in "${patterns[@]}"; do
    cp "$assets_dir/$pattern" "$dest/$pattern"
  done
  exit 0
fi

echo "unexpected gh invocation: $*" >&2
exit 1
`, filepath.Join(dir, "gh.log"), filepath.Join(dir, "repo.json"), filepath.Join(dir, "release.json"), assetsDir)
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	return dir
}

func requiredEnv(t *testing.T, key string) string {
	t.Helper()

	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("expected %s to be set when live GitHub verification is enabled", key)
	}
	return value
}

func extractZipAsset(t *testing.T, archivePath, destDir string) {
	t.Helper()

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("open zip %s: %v", archivePath, err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		targetPath := filepath.Join(destDir, file.Name)
		cleanTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(cleanTarget, filepath.Clean(destDir)+string(os.PathSeparator)) && cleanTarget != filepath.Clean(destDir) {
			t.Fatalf("zip entry escaped extract dir: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(cleanTarget, 0o755); err != nil {
				t.Fatalf("mkdir extracted dir %s: %v", cleanTarget, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(cleanTarget), 0o755); err != nil {
			t.Fatalf("mkdir extracted parent for %s: %v", cleanTarget, err)
		}

		in, err := file.Open()
		if err != nil {
			t.Fatalf("open zip entry %s: %v", file.Name, err)
		}

		mode := file.Mode()
		if mode == 0 {
			mode = 0o644
		}
		if filepath.Base(cleanTarget) == "harness" {
			mode = 0o755
		}

		out, err := os.OpenFile(cleanTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			in.Close()
			t.Fatalf("create extracted file %s: %v", cleanTarget, err)
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			in.Close()
			t.Fatalf("extract zip entry %s: %v", file.Name, err)
		}
		if err := out.Close(); err != nil {
			in.Close()
			t.Fatalf("close extracted file %s: %v", cleanTarget, err)
		}
		if err := in.Close(); err != nil {
			t.Fatalf("close zip entry %s: %v", file.Name, err)
		}
	}
}
