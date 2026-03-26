package smoke_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
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
