package smoke_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestRepositoryVersionFileUsesUnprefixedReleaseVersion(t *testing.T) {
	repoRoot := support.RepoRoot(t)

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
	)
	if result.ExitCode != 0 {
		t.Fatalf("read-release-version failed with exit %d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}

	version := strings.TrimSpace(result.Stdout)
	if version == "" {
		t.Fatalf("expected repository VERSION file to contain a release version")
	}
	if strings.HasPrefix(version, "v") {
		t.Fatalf("expected repository VERSION file to omit the v prefix, got %q", version)
	}
}

func TestReadReleaseVersionOutputsVersionAndTag(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	versionFile := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(versionFile, []byte("0.2.0-alpha.1\n"), 0o644); err != nil {
		t.Fatalf("write VERSION file: %v", err)
	}

	versionResult := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
	)
	if versionResult.ExitCode != 0 {
		t.Fatalf("read-release-version failed with exit %d\nstdout:\n%s\nstderr:\n%s", versionResult.ExitCode, versionResult.Stdout, versionResult.Stderr)
	}
	if got := strings.TrimSpace(versionResult.Stdout); got != "0.2.0-alpha.1" {
		t.Fatalf("expected raw version output %q, got %q", "0.2.0-alpha.1", got)
	}

	tagResult := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
		"--tag",
	)
	if tagResult.ExitCode != 0 {
		t.Fatalf("read-release-version --tag failed with exit %d\nstdout:\n%s\nstderr:\n%s", tagResult.ExitCode, tagResult.Stdout, tagResult.Stderr)
	}
	if got := strings.TrimSpace(tagResult.Stdout); got != "v0.2.0-alpha.1" {
		t.Fatalf("expected tag output %q, got %q", "v0.2.0-alpha.1", got)
	}
}

func TestReadReleaseVersionRejectsPrefixedVersionFile(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	versionFile := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(versionFile, []byte("v0.2.0\n"), 0o644); err != nil {
		t.Fatalf("write VERSION file: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected prefixed VERSION file to be rejected")
	}
	support.RequireContains(t, result.Stderr, "VERSION must not include the leading v prefix")
}

func TestReadReleaseVersionRejectsMissingVersionFile(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	versionFile := filepath.Join(t.TempDir(), "missing-version")

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected missing VERSION file to be rejected")
	}
	support.RequireContains(t, result.Stderr, "VERSION file does not exist")
}

func TestReadReleaseVersionRejectsEmptyVersionFile(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	versionFile := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(versionFile, []byte("\n \n"), 0o644); err != nil {
		t.Fatalf("write VERSION file: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected empty VERSION file to be rejected")
	}
	support.RequireContains(t, result.Stderr, "VERSION file must not be empty")
}

func TestReadReleaseVersionRejectsVersionThatCannotFormGitTag(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	versionFile := filepath.Join(t.TempDir(), "VERSION")
	if err := os.WriteFile(versionFile, []byte("0.2.0..1\n"), 0o644); err != nil {
		t.Fatalf("write VERSION file: %v", err)
	}

	result := runCommand(
		t,
		repoRoot,
		envWithOverrides(t, nil),
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "read-release-version"),
		"--version-file", versionFile,
	)
	if result.ExitCode == 0 {
		t.Fatalf("expected invalid git tag ref VERSION to be rejected")
	}
	support.RequireContains(t, result.Stderr, "VERSION must map to a valid git tag ref")
}

func TestCreateReleaseTagFromVersionSkipsWhenTagAlreadyMatchesCommit(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	remoteDir, cloneDir := newTaggedReleaseRepo(t)

	writeReleaseVersionFile(t, cloneDir, "0.2.0")
	mustRunGit(t, cloneDir, "add", "VERSION")
	mustRunGit(t, cloneDir, "commit", "-m", "Add VERSION")
	mustRunGit(t, cloneDir, "push", "-u", "origin", "main")

	env := envWithOverrides(t, nil)
	firstResult := runCommand(
		t,
		cloneDir,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "create-release-tag-from-version"),
	)
	if firstResult.ExitCode != 0 {
		t.Fatalf("create-release-tag-from-version failed with exit %d\nstdout:\n%s\nstderr:\n%s", firstResult.ExitCode, firstResult.Stdout, firstResult.Stderr)
	}
	support.RequireContains(t, firstResult.Stdout, "Created and pushed tag v0.2.0")

	headCommit := strings.TrimSpace(runGitOutput(t, cloneDir, "rev-parse", "HEAD"))
	secondResult := runCommand(
		t,
		cloneDir,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "create-release-tag-from-version"),
	)
	if secondResult.ExitCode != 0 {
		t.Fatalf("expected second create-release-tag-from-version run to skip cleanly, got exit %d\nstdout:\n%s\nstderr:\n%s", secondResult.ExitCode, secondResult.Stdout, secondResult.Stderr)
	}
	support.RequireContains(t, secondResult.Stdout, "Tag v0.2.0 already exists at "+headCommit+"; skipping.")

	remoteTagCommit := strings.TrimSpace(runGitOutput(t, remoteDir, "rev-list", "-n", "1", "refs/tags/v0.2.0"))
	if remoteTagCommit != headCommit {
		t.Fatalf("expected remote tag to remain at %s, got %s", headCommit, remoteTagCommit)
	}
}

func TestCreateReleaseTagFromVersionFailsWhenTagPointsAtDifferentCommit(t *testing.T) {
	repoRoot := support.RepoRoot(t)
	_, cloneDir := newTaggedReleaseRepo(t)

	writeReleaseVersionFile(t, cloneDir, "0.2.0")
	mustRunGit(t, cloneDir, "add", "VERSION")
	mustRunGit(t, cloneDir, "commit", "-m", "Add VERSION")
	mustRunGit(t, cloneDir, "push", "-u", "origin", "main")

	env := envWithOverrides(t, nil)
	firstResult := runCommand(
		t,
		cloneDir,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "create-release-tag-from-version"),
	)
	if firstResult.ExitCode != 0 {
		t.Fatalf("create-release-tag-from-version failed with exit %d\nstdout:\n%s\nstderr:\n%s", firstResult.ExitCode, firstResult.Stdout, firstResult.Stderr)
	}

	if err := os.WriteFile(filepath.Join(cloneDir, "README.md"), []byte("# changed\n"), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}
	mustRunGit(t, cloneDir, "add", "README.md")
	mustRunGit(t, cloneDir, "commit", "-m", "Change README")
	mustRunGit(t, cloneDir, "push", "origin", "main")

	secondResult := runCommand(
		t,
		cloneDir,
		env,
		"/bin/bash",
		filepath.Join(repoRoot, "scripts", "create-release-tag-from-version"),
	)
	if secondResult.ExitCode == 0 {
		t.Fatalf("expected create-release-tag-from-version to fail when the tag already points at another commit")
	}
	support.RequireContains(t, secondResult.Stderr, "Tag v0.2.0 already exists at")
}

func TestVersionTagWorkflowUsesRepositoryVersionFile(t *testing.T) {
	workflowPath := filepath.Join(support.RepoRoot(t), ".github", "workflows", "tag-release-from-version.yml")
	workflowData, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read tag-release-from-version workflow: %v", err)
	}
	workflow := string(workflowData)

	support.RequireContains(t, workflow, `branches:`)
	support.RequireContains(t, workflow, `- "main"`)
	support.RequireContains(t, workflow, `paths:`)
	support.RequireContains(t, workflow, `- "VERSION"`)
	support.RequireContains(t, workflow, `contents: write`)
	support.RequireContains(t, workflow, `uses: actions/checkout@v4`)
	support.RequireContains(t, workflow, `fetch-depth: 0`)
	support.RequireContains(t, workflow, `- name: Resolve release tag from VERSION`)
	support.RequireContains(t, workflow, `tag="$(scripts/read-release-version --tag)"`)
	support.RequireContains(t, workflow, `- name: Create release tag when missing`)
	support.RequireContains(t, workflow, `scripts/create-release-tag-from-version --commit "${GITHUB_SHA}"`)
	support.RequireContains(t, workflow, `- name: Dispatch Release workflow for resolved tag`)
	support.RequireContains(t, workflow, `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}`)
	support.RequireContains(t, workflow, `gh workflow run release.yml --ref "${GITHUB_REF_NAME}" -f version="${{ steps.release-version.outputs.tag }}"`)

	checkoutIndex := strings.Index(workflow, `uses: actions/checkout@v4`)
	resolveIndex := strings.Index(workflow, `- name: Resolve release tag from VERSION`)
	createIndex := strings.Index(workflow, `- name: Create release tag when missing`)
	dispatchIndex := strings.Index(workflow, `- name: Dispatch Release workflow for resolved tag`)
	if checkoutIndex == -1 || resolveIndex == -1 || createIndex == -1 || dispatchIndex == -1 {
		t.Fatalf("expected checkout, resolve, create, and dispatch steps to exist in workflow")
	}
	if !(checkoutIndex < resolveIndex && resolveIndex < createIndex && createIndex < dispatchIndex) {
		t.Fatalf("expected workflow step order checkout -> resolve -> create -> dispatch, got checkout=%d resolve=%d create=%d dispatch=%d", checkoutIndex, resolveIndex, createIndex, dispatchIndex)
	}
}

func newTaggedReleaseRepo(t *testing.T) (string, string) {
	t.Helper()

	tempDir := t.TempDir()
	remoteDir := filepath.Join(tempDir, "remote.git")
	cloneDir := filepath.Join(tempDir, "repo")

	mustRunGit(t, tempDir, "init", "--bare", remoteDir)
	mustRunGit(t, tempDir, "clone", remoteDir, cloneDir)
	mustRunGit(t, cloneDir, "config", "user.name", "Test User")
	mustRunGit(t, cloneDir, "config", "user.email", "test@example.com")
	writeFixtureFile(t, filepath.Join(cloneDir, "README.md"), "# repo\n", 0o644)
	mustRunGit(t, cloneDir, "add", "README.md")
	mustRunGit(t, cloneDir, "commit", "-m", "Initial commit")
	mustRunGit(t, cloneDir, "branch", "-M", "main")
	mustRunGit(t, cloneDir, "push", "-u", "origin", "main")

	return remoteDir, cloneDir
}

func writeReleaseVersionFile(t *testing.T, repoDir, version string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, "VERSION"), []byte(version+"\n"), 0o644); err != nil {
		t.Fatalf("write VERSION: %v", err)
	}
}
