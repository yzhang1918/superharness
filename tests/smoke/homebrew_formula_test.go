package smoke_test

import (
	"os"
	"path/filepath"
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
