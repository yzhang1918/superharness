package smoke_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestRenderHomebrewFormulaFromChecksums(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	workdir := repoRoot
	tempDir := t.TempDir()
	checksumsPath := filepath.Join(tempDir, "SHA256SUMS")
	outputPath := filepath.Join(tempDir, "Formula", "easyharness.rb")
	checksumContents := "" +
		"1111111111111111111111111111111111111111111111111111111111111111  easyharness_v0.1.0-alpha.5_darwin_arm64.zip\n" +
		"2222222222222222222222222222222222222222222222222222222222222222  easyharness_v0.1.0-alpha.5_darwin_amd64.zip\n" +
		"3333333333333333333333333333333333333333333333333333333333333333  easyharness_v0.1.0-alpha.5_linux_arm64.zip\n" +
		"4444444444444444444444444444444444444444444444444444444444444444  easyharness_v0.1.0-alpha.5_linux_amd64.zip\n"
	if err := os.WriteFile(checksumsPath, []byte(checksumContents), 0o644); err != nil {
		t.Fatalf("write SHA256SUMS: %v", err)
	}

	result := runCommand(
		t,
		workdir,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "render-homebrew-formula"),
		"--repo", "catu-ai/easyharness",
		"--tag", "v0.1.0-alpha.5",
		"--checksums", checksumsPath,
		"--output", outputPath,
	)
	if result.ExitCode != 0 {
		t.Fatalf("render-homebrew-formula failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	formulaData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read rendered formula: %v", err)
	}
	formula := string(formulaData)
	support.RequireContains(t, formula, "class Easyharness < Formula")
	support.RequireContains(t, formula, `homepage "https://github.com/catu-ai/easyharness"`)
	support.RequireContains(t, formula, `version "0.1.0-alpha.5"`)
	support.RequireContains(t, formula, `url "https://github.com/catu-ai/easyharness/releases/download/v0.1.0-alpha.5/easyharness_v0.1.0-alpha.5_darwin_arm64.zip"`)
	support.RequireContains(t, formula, `sha256 "1111111111111111111111111111111111111111111111111111111111111111"`)
	support.RequireContains(t, formula, `url "https://github.com/catu-ai/easyharness/releases/download/v0.1.0-alpha.5/easyharness_v0.1.0-alpha.5_darwin_amd64.zip"`)
	support.RequireContains(t, formula, `sha256 "2222222222222222222222222222222222222222222222222222222222222222"`)
	support.RequireContains(t, formula, `url "https://github.com/catu-ai/easyharness/releases/download/v0.1.0-alpha.5/easyharness_v0.1.0-alpha.5_linux_arm64.zip"`)
	support.RequireContains(t, formula, `sha256 "3333333333333333333333333333333333333333333333333333333333333333"`)
	support.RequireContains(t, formula, `url "https://github.com/catu-ai/easyharness/releases/download/v0.1.0-alpha.5/easyharness_v0.1.0-alpha.5_linux_amd64.zip"`)
	support.RequireContains(t, formula, `sha256 "4444444444444444444444444444444444444444444444444444444444444444"`)
	support.RequireContains(t, formula, `bin.install Dir["**/harness"].fetch(0) => "harness"`)
	support.RequireContains(t, formula, `assert_match "version: v#{version}", output`)
	support.RequireContains(t, formula, `assert_match "mode: release", output`)
}

func TestRenderHomebrewFormulaFailsWhenChecksumIsMissing(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	tempDir := t.TempDir()
	checksumsPath := filepath.Join(tempDir, "SHA256SUMS")
	if err := os.WriteFile(checksumsPath, []byte(""+
		"1111111111111111111111111111111111111111111111111111111111111111  easyharness_v0.1.0-alpha.5_darwin_arm64.zip\n"+
		"2222222222222222222222222222222222222222222222222222222222222222  easyharness_v0.1.0-alpha.5_darwin_amd64.zip\n"+
		"3333333333333333333333333333333333333333333333333333333333333333  easyharness_v0.1.0-alpha.5_linux_arm64.zip\n"), 0o644); err != nil {
		t.Fatalf("write SHA256SUMS: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "render-homebrew-formula"),
		"--repo", "catu-ai/easyharness",
		"--tag", "v0.1.0-alpha.5",
		"--checksums", checksumsPath,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected render-homebrew-formula to fail when a required checksum entry is missing")
	}
	support.RequireContains(t, result.Stderr, "SHA256SUMS is missing required checksum entry for easyharness_v0.1.0-alpha.5_linux_amd64.zip")
}

func TestUpdateHomebrewTapWarnsWithoutToken(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	formulaPath := filepath.Join(t.TempDir(), "easyharness.rb")
	if err := os.WriteFile(formulaPath, []byte("class Easyharness < Formula\nend\n"), 0o644); err != nil {
		t.Fatalf("write formula file: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, map[string]string{
			"PATH": installerPath(t),
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "update-homebrew-tap"),
		"--formula", formulaPath,
		"--tap-dir", filepath.Join(t.TempDir(), "missing-tap"),
		"--branch", "main",
		"--version", "v0.1.0-alpha.5",
	)
	if result.ExitCode != 0 {
		t.Fatalf("expected update-homebrew-tap to skip cleanly without a token, got exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	support.RequireContains(t, result.Stdout, "::warning title=Homebrew tap update skipped::EASYHARNESS_HOMEBREW_TAP_TOKEN is not set; skipping Homebrew tap update.")
}

func TestUpdateHomebrewTapPushesFromDetachedCheckout(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote.git")
	seedDir := filepath.Join(tempDir, "seed")
	tapDir := filepath.Join(tempDir, "tap")
	formulaPath := filepath.Join(tempDir, "easyharness.rb")
	formulaBody := "class Easyharness < Formula\n  desc \"tap test\"\nend\n"

	mustRunGit(t, tempDir, "init", "--bare", remoteDir)
	mustRunGit(t, tempDir, "clone", remoteDir, seedDir)
	mustRunGit(t, seedDir, "config", "user.name", "Test User")
	mustRunGit(t, seedDir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("# tap\n"), 0o644); err != nil {
		t.Fatalf("write seed README: %v", err)
	}
	mustRunGit(t, seedDir, "add", "README.md")
	mustRunGit(t, seedDir, "commit", "-m", "Initial tap commit")
	mustRunGit(t, seedDir, "branch", "-M", "main")
	mustRunGit(t, seedDir, "push", "-u", "origin", "main")
	mustRunGit(t, tempDir, "init", tapDir)
	mustRunGit(t, tapDir, "remote", "add", "origin", remoteDir)
	mustRunGit(t, tapDir, "fetch", "--depth=1", "origin", "main")
	headCommit := strings.TrimSpace(runGitOutput(t, tapDir, "rev-parse", "FETCH_HEAD"))
	mustRunGit(t, tapDir, "checkout", "--detach", headCommit)

	if err := os.WriteFile(formulaPath, []byte(formulaBody), 0o644); err != nil {
		t.Fatalf("write formula file: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, map[string]string{
			"PATH":                           installerPath(t),
			"EASYHARNESS_HOMEBREW_TAP_TOKEN": "dummy-token",
		}),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "update-homebrew-tap"),
		"--formula", formulaPath,
		"--tap-dir", tapDir,
		"--branch", "main",
		"--version", "v0.1.0-alpha.5",
	)
	if result.ExitCode != 0 {
		t.Fatalf("update-homebrew-tap failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	verifyDir := filepath.Join(tempDir, "verify")
	mustRunGit(t, tempDir, "clone", "--branch", "main", remoteDir, verifyDir)
	renderedPath := filepath.Join(verifyDir, "Formula", "easyharness.rb")
	renderedData, err := os.ReadFile(renderedPath)
	if err != nil {
		t.Fatalf("read pushed formula: %v", err)
	}
	if string(renderedData) != formulaBody {
		t.Fatalf("expected pushed formula to match rendered contents, got:\n%s", renderedData)
	}
}

func TestReleaseWorkflowWiresHomebrewTapPublishing(t *testing.T) {
	workflowPath := filepath.Join(support.RepoRoot(t), ".github", "workflows", "release.yml")
	workflowData, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	workflow := string(workflowData)

	support.RequireContains(t, workflow, `EASYHARNESS_HOMEBREW_TAP_TOKEN: ${{ secrets.EASYHARNESS_HOMEBREW_TAP_TOKEN }}`)
	support.RequireContains(t, workflow, `EASYHARNESS_HOMEBREW_TAP_BRANCH: main`)
	support.RequireContains(t, workflow, `if: ${{ env.EASYHARNESS_HOMEBREW_TAP_TOKEN != '' }}`)
	support.RequireContains(t, workflow, `repository: catu-ai/homebrew-tap`)
	support.RequireContains(t, workflow, `ref: ${{ env.EASYHARNESS_HOMEBREW_TAP_BRANCH }}`)
	support.RequireContains(t, workflow, `token: ${{ env.EASYHARNESS_HOMEBREW_TAP_TOKEN }}`)
	support.RequireContains(t, workflow, `path: dist/homebrew-tap`)
	support.RequireContains(t, workflow, `fetch-depth: 0`)
	support.RequireContains(t, workflow, `scripts/render-homebrew-formula \`)
	support.RequireContains(t, workflow, `--repo "${{ github.repository }}"`)
	support.RequireContains(t, workflow, `--tag "${{ steps.release-version.outputs.version }}"`)
	support.RequireContains(t, workflow, `--checksums dist/release/SHA256SUMS`)
	support.RequireContains(t, workflow, `--output dist/homebrew/easyharness.rb`)
	support.RequireContains(t, workflow, `scripts/update-homebrew-tap \`)
	support.RequireContains(t, workflow, `--formula dist/homebrew/easyharness.rb`)
	support.RequireContains(t, workflow, `--tap-dir dist/homebrew-tap`)
	support.RequireContains(t, workflow, `--branch "${{ env.EASYHARNESS_HOMEBREW_TAP_BRANCH }}"`)
	support.RequireContains(t, workflow, `--version "${{ steps.release-version.outputs.version }}"`)
	support.RequireContains(t, workflow, `verify-homebrew-install:`)
	support.RequireContains(t, workflow, `runs-on: macos-latest`)
	support.RequireContains(t, workflow, `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`)
	support.RequireContains(t, workflow, `EASYHARNESS_RUN_LIVE_BREW_SMOKE: "1"`)
	support.RequireContains(t, workflow, `EASYHARNESS_LIVE_GH_REPO: ${{ github.repository }}`)
	support.RequireContains(t, workflow, `EASYHARNESS_LIVE_GH_TAG: ${{ steps.release-version.outputs.version }}`)
	support.RequireContains(t, workflow, `go test ./tests/smoke -run TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled -count=1`)
}

func TestVerifyHomebrewTapInstallAgainstGitHubWhenEnabled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Homebrew smoke test requires a POSIX environment")
	}
	if os.Getenv("EASYHARNESS_RUN_LIVE_BREW_SMOKE") != "1" {
		t.Skip("set EASYHARNESS_RUN_LIVE_BREW_SMOKE=1 to enable live Homebrew verification")
	}

	repo := requiredEnv(t, "EASYHARNESS_LIVE_GH_REPO")
	tag := requiredEnv(t, "EASYHARNESS_LIVE_GH_TAG")
	previousTag := os.Getenv("EASYHARNESS_LIVE_PREVIOUS_GH_TAG")

	brewPath, err := exec.LookPath("brew")
	if err != nil {
		t.Skip("brew not available on PATH")
	}
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		t.Fatalf("find gh on PATH: %v", err)
	}

	repoRoot := support.RepoRoot(t)
	env := envWithOverrides(t, map[string]string{
		"PATH": strings.Join([]string{filepath.Dir(brewPath), filepath.Dir(ghPath), installerPath(t)}, string(os.PathListSeparator)),
	})
	if previousTag == "" {
		previousTag = resolvePreviousHomebrewReleaseTag(t, repoRoot, env, repo, tag)
	}

	brewRepoResult := runCommand(t, repoRoot, env, brewPath, "--repository")
	if brewRepoResult.ExitCode != 0 {
		t.Fatalf("brew --repository failed with exit %d\nstdout:\n%s\nstderr:\n%s", brewRepoResult.ExitCode, brewRepoResult.Stdout, brewRepoResult.Stderr)
	}
	brewRepo := strings.TrimSpace(brewRepoResult.Stdout)
	tapRoot := filepath.Join(brewRepo, "Library", "Taps", "catu-ai", "homebrew-tap")
	if _, err := os.Stat(tapRoot); err == nil {
		t.Skipf("tap path already exists at %s; refusing to clobber a real tap checkout", tapRoot)
	}

	t.Cleanup(func() {
		_ = exec.Command(brewPath, "uninstall", "--force", "easyharness").Run()
		_ = os.RemoveAll(tapRoot)
	})

	formulaPath := filepath.Join(tapRoot, "Formula", "easyharness.rb")
	if err := os.MkdirAll(filepath.Dir(formulaPath), 0o755); err != nil {
		t.Fatalf("mkdir staged tap formula dir: %v", err)
	}

	currentChecksumsPath := downloadReleaseChecksums(t, repoRoot, env, repo, tag)
	if previousTag != "" {
		previousChecksumsPath := downloadReleaseChecksums(t, repoRoot, env, repo, previousTag)
		renderHomebrewFormula(t, repoRoot, env, repo, previousTag, previousChecksumsPath, formulaPath)

		installResult := runCommand(t, repoRoot, env, brewPath, "install", "catu-ai/tap/easyharness")
		if installResult.ExitCode != 0 {
			t.Fatalf("brew install catu-ai/tap/easyharness failed with exit %d\nstdout:\n%s\nstderr:\n%s", installResult.ExitCode, installResult.Stdout, installResult.Stderr)
		}
		requireInstalledHarnessVersion(t, repoRoot, env, brewPath, previousTag)

		renderHomebrewFormula(t, repoRoot, env, repo, tag, currentChecksumsPath, formulaPath)
		upgradeResult := runCommand(t, repoRoot, env, brewPath, "upgrade", "catu-ai/tap/easyharness")
		if upgradeResult.ExitCode != 0 {
			t.Fatalf("brew upgrade catu-ai/tap/easyharness failed with exit %d\nstdout:\n%s\nstderr:\n%s", upgradeResult.ExitCode, upgradeResult.Stdout, upgradeResult.Stderr)
		}
	} else {
		renderHomebrewFormula(t, repoRoot, env, repo, tag, currentChecksumsPath, formulaPath)

		installResult := runCommand(t, repoRoot, env, brewPath, "install", "catu-ai/tap/easyharness")
		if installResult.ExitCode != 0 {
			t.Fatalf("brew install catu-ai/tap/easyharness failed with exit %d\nstdout:\n%s\nstderr:\n%s", installResult.ExitCode, installResult.Stdout, installResult.Stderr)
		}
	}
	requireInstalledHarnessVersion(t, repoRoot, env, brewPath, tag)

	testResult := runCommand(t, repoRoot, env, brewPath, "test", "easyharness")
	if testResult.ExitCode != 0 {
		t.Fatalf("brew test easyharness failed with exit %d\nstdout:\n%s\nstderr:\n%s", testResult.ExitCode, testResult.Stdout, testResult.Stderr)
	}
}

func downloadReleaseChecksums(t *testing.T, repoRoot string, env []string, repo, tag string) string {
	t.Helper()

	downloadDir := filepath.Join(t.TempDir(), "downloads")
	verifyResult := runCommand(
		t,
		repoRoot,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "verify-release-namespace"),
		"--repo", repo,
		"--tag", tag,
		"--asset", "SHA256SUMS",
		"--download-dir", downloadDir,
	)
	if verifyResult.ExitCode != 0 {
		t.Fatalf("verify-release-namespace failed for %s with exit %d\nstdout:\n%s\nstderr:\n%s", tag, verifyResult.ExitCode, verifyResult.Stdout, verifyResult.Stderr)
	}
	return filepath.Join(downloadDir, "SHA256SUMS")
}

func renderHomebrewFormula(t *testing.T, repoRoot string, env []string, repo, tag, checksumsPath, formulaPath string) {
	t.Helper()

	renderResult := runCommand(
		t,
		repoRoot,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "render-homebrew-formula"),
		"--repo", repo,
		"--tag", tag,
		"--checksums", checksumsPath,
		"--output", formulaPath,
	)
	if renderResult.ExitCode != 0 {
		t.Fatalf("render-homebrew-formula failed for %s with exit %d\nstdout:\n%s\nstderr:\n%s", tag, renderResult.ExitCode, renderResult.Stdout, renderResult.Stderr)
	}
}

func requireInstalledHarnessVersion(t *testing.T, repoRoot string, env []string, brewPath string, tag string) {
	t.Helper()

	prefixResult := runCommand(t, repoRoot, env, brewPath, "--prefix")
	if prefixResult.ExitCode != 0 {
		t.Fatalf("brew --prefix failed with exit %d\nstdout:\n%s\nstderr:\n%s", prefixResult.ExitCode, prefixResult.Stdout, prefixResult.Stderr)
	}
	binaryPath := filepath.Join(strings.TrimSpace(prefixResult.Stdout), "bin", "harness")
	versionCmd := exec.Command(binaryPath, "--version")
	versionCmd.Dir = repoRoot
	versionCmd.Env = env
	versionOutput, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run installed harness --version: %v\n%s", err, versionOutput)
	}
	support.RequireContains(t, string(versionOutput), "version: "+tag)
	support.RequireContains(t, string(versionOutput), "mode: release")
}

func resolvePreviousHomebrewReleaseTag(t *testing.T, repoRoot string, env []string, repo, tag string) string {
	t.Helper()

	releases := listGitHubReleases(t, repoRoot, env, repo)
	for i, release := range releases {
		if release.TagName != tag {
			continue
		}
		for _, previous := range releases[i+1:] {
			if previous.TagName == "" || previous.Draft {
				continue
			}
			checksumsPath := downloadReleaseChecksums(t, repoRoot, env, repo, previous.TagName)
			if releaseSupportsHomebrewFormula(t, checksumsPath, previous.TagName) {
				return previous.TagName
			}
		}
		return ""
	}

	t.Fatalf("release tag %s not found in GitHub release list", tag)
	return ""
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Draft   bool   `json:"draft"`
}

func listGitHubReleases(t *testing.T, repoRoot string, env []string, repo string) []githubRelease {
	t.Helper()

	ghPath, err := exec.LookPath("gh")
	if err != nil {
		t.Fatalf("find gh on PATH: %v", err)
	}

	releases := make([]githubRelease, 0, 100)
	for page := 1; ; page++ {
		cmd := exec.Command(ghPath, "api", fmt.Sprintf("repos/%s/releases?per_page=100&page=%d", repo, page))
		cmd.Dir = repoRoot
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("gh api releases page %d failed: %v\n%s", page, err, output)
		}

		var pageReleases []githubRelease
		if err := json.Unmarshal(output, &pageReleases); err != nil {
			t.Fatalf("parse gh release response page %d: %v\n%s", page, err, output)
		}
		releases = append(releases, pageReleases...)
		if len(pageReleases) < 100 {
			break
		}
	}
	return releases
}

func releaseSupportsHomebrewFormula(t *testing.T, checksumsPath string, tag string) bool {
	t.Helper()

	checksumsData, err := os.ReadFile(checksumsPath)
	if err != nil {
		t.Fatalf("read %s: %v", checksumsPath, err)
	}
	checksums := string(checksumsData)
	for _, asset := range expectedReleaseAssets(tag) {
		if !strings.Contains(checksums, "  "+asset+"\n") {
			return false
		}
	}
	return true
}

func mustRunGit(t *testing.T, workdir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func runGitOutput(t *testing.T, workdir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
	return string(output)
}
