package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestSyncContractArtifactsCheckPassesForCurrentRepo(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cmd := exec.Command(filepath.Join(repoRoot, "scripts", "sync-contract-artifacts"), "--check")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync-contract-artifacts --check: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Contract schemas are in sync.") {
		t.Fatalf("unexpected check output:\n%s", output)
	}
}

func TestSyncContractArtifactsCheckFailsOnStaleGeneratedFiles(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cloneRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cloneRoot, 0o755); err != nil {
		t.Fatalf("mkdir clone root: %v", err)
	}
	copyCurrentRepo(t, repoRoot, cloneRoot)

	stalePath := filepath.Join(cloneRoot, "schema", "index.json")
	if err := os.WriteFile(stalePath, []byte("{\"stale\":true}\n"), 0o644); err != nil {
		t.Fatalf("write stale schema: %v", err)
	}

	checkCmd := exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	checkCmd.Dir = cloneRoot
	output, err := checkCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected stale generated file check to fail:\n%s", output)
	}
	if !strings.Contains(string(output), "stale generated file") {
		t.Fatalf("expected stale-file error, got:\n%s", output)
	}

	syncCmd := exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"))
	syncCmd.Dir = cloneRoot
	output, err = syncCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync-contract-artifacts repair run: %v\n%s", err, output)
	}

	checkCmd = exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	checkCmd.Dir = cloneRoot
	output, err = checkCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-repair sync-contract-artifacts --check: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Contract schemas are in sync.") {
		t.Fatalf("unexpected post-repair check output:\n%s", output)
	}
}

func TestSyncContractArtifactsCheckFailsOnDeprecatedGeneratedDocs(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	cloneRoot := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(cloneRoot, 0o755); err != nil {
		t.Fatalf("mkdir clone root: %v", err)
	}
	copyCurrentRepo(t, repoRoot, cloneRoot)

	stalePath := filepath.Join(cloneRoot, "docs", "reference", "contracts", "README.md")
	if err := os.MkdirAll(filepath.Dir(stalePath), 0o755); err != nil {
		t.Fatalf("mkdir stale docs dir: %v", err)
	}
	if err := os.WriteFile(stalePath, []byte("# stale generated docs\n"), 0o644); err != nil {
		t.Fatalf("write stale docs: %v", err)
	}

	checkCmd := exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	checkCmd.Dir = cloneRoot
	output, err := checkCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected deprecated generated docs check to fail:\n%s", output)
	}
	if !strings.Contains(string(output), "unexpected generated file") {
		t.Fatalf("expected unexpected-file error, got:\n%s", output)
	}

	syncCmd := exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"))
	syncCmd.Dir = cloneRoot
	output, err = syncCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sync-contract-artifacts cleanup run: %v\n%s", err, output)
	}

	checkCmd = exec.Command(filepath.Join(cloneRoot, "scripts", "sync-contract-artifacts"), "--check")
	checkCmd.Dir = cloneRoot
	output, err = checkCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("post-repair sync-contract-artifacts --check: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Contract schemas are in sync.") {
		t.Fatalf("unexpected post-repair check output:\n%s", output)
	}

	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected deprecated generated docs to be removed, got err=%v", err)
	}
}

func copyCurrentRepo(t *testing.T, src, dst string) {
	t.Helper()
	archive := exec.Command("tar", "-cf", "-", "--exclude=.git", "--exclude=.local", ".")
	archive.Dir = src
	extract := exec.Command("tar", "-xf", "-", "-C", dst)

	pipe, err := archive.StdoutPipe()
	if err != nil {
		t.Fatalf("archive stdout pipe: %v", err)
	}
	extract.Stdin = pipe
	extract.Stderr = os.Stderr

	if err := archive.Start(); err != nil {
		t.Fatalf("start archive: %v", err)
	}
	if err := extract.Start(); err != nil {
		t.Fatalf("start extract: %v", err)
	}
	if err := archive.Wait(); err != nil {
		t.Fatalf("archive repo: %v", err)
	}
	if err := extract.Wait(); err != nil {
		t.Fatalf("extract repo: %v", err)
	}
}
