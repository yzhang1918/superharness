package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
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
	mustRunGit(t, tempDir, "clone", remoteDir, tapDir)

	headCommit := strings.TrimSpace(runGitOutput(t, tapDir, "rev-parse", "HEAD"))
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
		"--version", "v0.1.0-alpha.5",
	)
	if result.ExitCode != 0 {
		t.Fatalf("update-homebrew-tap failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	verifyDir := filepath.Join(tempDir, "verify")
	mustRunGit(t, tempDir, "clone", remoteDir, verifyDir)
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
	support.RequireContains(t, workflow, `if: ${{ env.EASYHARNESS_HOMEBREW_TAP_TOKEN != '' }}`)
	support.RequireContains(t, workflow, `scripts/render-homebrew-formula \`)
	support.RequireContains(t, workflow, `--repo "${{ github.repository }}"`)
	support.RequireContains(t, workflow, `--tag "${{ steps.release-version.outputs.version }}"`)
	support.RequireContains(t, workflow, `--checksums dist/release/SHA256SUMS`)
	support.RequireContains(t, workflow, `--output dist/homebrew/easyharness.rb`)
	support.RequireContains(t, workflow, `scripts/update-homebrew-tap \`)
	support.RequireContains(t, workflow, `--formula dist/homebrew/easyharness.rb`)
	support.RequireContains(t, workflow, `--tap-dir dist/homebrew-tap`)
	support.RequireContains(t, workflow, `--version "${{ steps.release-version.outputs.version }}"`)
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
