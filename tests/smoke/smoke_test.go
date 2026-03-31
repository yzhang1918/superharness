package smoke_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

type statusResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type lintResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Artifacts struct {
		PlanPath string `json:"plan_path"`
	} `json:"artifacts"`
}

type installResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	Mode    string `json:"mode"`
	Scope   string `json:"scope"`
	Actions []struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	} `json:"actions"`
}

func TestHelpShowsTopLevelUsage(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
	support.RequireContains(t, result.CombinedOutput(), "--version       Print concise debug information for the running harness binary")
	support.RequireContains(t, result.CombinedOutput(), "plan template   Render the packaged plan template")
	support.RequireContains(t, result.CombinedOutput(), "plan lint       Validate a tracked plan")
	support.RequireContains(t, result.CombinedOutput(), "execute start   Record the execution-start milestone")
	support.RequireContains(t, result.CombinedOutput(), "evidence submit Record append-only CI, publish, or sync evidence")
	support.RequireContains(t, result.CombinedOutput(), "review start    Create a deterministic review round")
	support.RequireContains(t, result.CombinedOutput(), "review submit   Record one reviewer submission")
	support.RequireContains(t, result.CombinedOutput(), "review aggregate Aggregate reviewer submissions")
	support.RequireContains(t, result.CombinedOutput(), "land            Record merge confirmation for the archived candidate")
	support.RequireContains(t, result.CombinedOutput(), "land complete   Record post-merge cleanup completion")
	support.RequireContains(t, result.CombinedOutput(), "archive         Freeze the current active plan")
	support.RequireContains(t, result.CombinedOutput(), "reopen          Restore the current archived plan")
	support.RequireContains(t, result.CombinedOutput(), "status          Summarize the current plan and local execution state")
	support.RequireContains(t, result.CombinedOutput(), "install         Install or refresh the harness-managed repository bootstrap")
}

func TestVersionPrintsHumanReadableBuildInfo(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "--version")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	if mode := requireVersionField(t, result.Stdout, "mode"); mode != "release" {
		t.Fatalf("expected release mode, got %q\noutput:\n%s", mode, result.Stdout)
	}
	expectedCommit := gitHeadCommit(t, support.RepoRoot(t))
	if commit := requireVersionField(t, result.Stdout, "commit"); commit != expectedCommit {
		t.Fatalf("expected release version commit %q, got %q\noutput:\n%s", expectedCommit, commit, result.Stdout)
	}
	if strings.Contains(result.Stdout, "path: ") {
		t.Fatalf("expected release build version output to omit path, got %q", result.Stdout)
	}
	if strings.HasPrefix(strings.TrimSpace(result.Stdout), "{") {
		t.Fatalf("expected plain-text version output, got %q", result.Stdout)
	}
}

func TestStatusReportsIdleWorkspace(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[statusResult](t, result)
	if !payload.OK {
		t.Fatalf("expected ok status payload, got %#v", payload)
	}
	if payload.Command != "status" {
		t.Fatalf("expected status command, got %#v", payload)
	}
	if payload.State.CurrentNode != "idle" {
		t.Fatalf("expected idle state, got %#v", payload)
	}
	if payload.Summary != "No current plan is active in this worktree." {
		t.Fatalf("expected idle summary, got %#v", payload)
	}
	if len(payload.NextAction) == 0 {
		t.Fatalf("expected idle status to include next-action guidance, got %#v", payload)
	}
	if payload.NextAction[0].Command != nil {
		t.Fatalf("expected idle next action to be descriptive only, got %#v", payload)
	}
	if payload.NextAction[0].Description != "Start discovery or create a new tracked plan when the next slice is ready." {
		t.Fatalf("expected idle handoff guidance, got %#v", payload)
	}
}

func TestPlanTemplatePrintsToStdoutByDefault(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", "Stdout Plan",
		"--timestamp", "2026-03-22T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
	)
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	support.RequireContains(t, result.Stdout, "# Stdout Plan")
	support.RequireContains(t, result.Stdout, "created_at: 2026-03-22T00:00:00Z")
	support.RequireContains(t, result.Stdout, "source_type: issue")
	support.RequireContains(t, result.Stdout, "source_refs: [\"#6\"]")
}

func TestInstallBootstrapsFreshRepository(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "install")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[installResult](t, result)
	if !payload.OK || payload.Command != "install" {
		t.Fatalf("expected install payload, got %#v", payload)
	}
	if payload.Mode != "apply" || payload.Scope != "all" {
		t.Fatalf("unexpected install mode/scope: %#v", payload)
	}

	agentsPath := workspace.Path("AGENTS.md")
	support.RequireFileExists(t, agentsPath)
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	support.RequireContains(t, string(agentsData), "<!-- easyharness:begin -->")
	support.RequireContains(t, string(agentsData), "<!-- easyharness:end -->")

	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-execute/SKILL.md"))
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
}

func TestInstallDryRunDoesNotWriteRepositoryFiles(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "install", "--dry-run")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[installResult](t, result)
	if payload.Mode != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", payload)
	}
	if len(payload.Actions) == 0 {
		t.Fatalf("expected planned actions, got %#v", payload)
	}

	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents"))
}

func TestInstallRepeatRunReportsNoopActions(t *testing.T) {
	workspace := support.NewWorkspace(t)

	first := support.Run(t, workspace.Root, "install")
	support.RequireSuccess(t, first)
	support.RequireNoStderr(t, first)

	second := support.Run(t, workspace.Root, "install")
	support.RequireSuccess(t, second)
	support.RequireNoStderr(t, second)

	payload := support.RequireJSONResult[installResult](t, second)
	if !strings.Contains(payload.Summary, "already up to date") {
		t.Fatalf("expected no-op summary, got %#v", payload)
	}
	for _, action := range payload.Actions {
		if action.Kind != "noop" {
			t.Fatalf("expected noop repeat install actions, got %#v", payload.Actions)
		}
	}
}

func TestInstallRejectsInvalidScopeViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "install", "--scope", "bogus")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[installResult](t, result)
	if payload.OK {
		t.Fatalf("expected install failure payload, got %#v", payload)
	}
	if payload.Command != "install" || payload.Scope != "bogus" {
		t.Fatalf("unexpected invalid-scope payload: %#v", payload)
	}
}

func TestInstallRejectsDuplicateManagedBlocksViaCLI(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsPath := workspace.Path("AGENTS.md")
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"<!-- easyharness:begin -->",
		"one",
		"<!-- easyharness:end -->",
		"",
		"<!-- easyharness:begin -->",
		"two",
		"<!-- easyharness:end -->",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := support.Run(t, workspace.Root, "install", "--scope", "agents")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[installResult](t, result)
	if payload.OK {
		t.Fatalf("expected duplicate-block install failure, got %#v", payload)
	}
	if payload.Command != "install" || payload.Scope != "agents" {
		t.Fatalf("unexpected duplicate-block payload: %#v", payload)
	}
}

func TestInstallSkillsScopeBootstrapsOnlySkills(t *testing.T) {
	workspace := support.NewWorkspace(t)

	result := support.Run(t, workspace.Root, "install", "--scope", "skills")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	payload := support.RequireJSONResult[installResult](t, result)
	if !payload.OK || payload.Scope != "skills" {
		t.Fatalf("unexpected skills-scope payload: %#v", payload)
	}
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-discovery/SKILL.md"))
	support.RequireFileMissing(t, workspace.Path("AGENTS.md"))
}

func TestInstallSkillsScopeRecoversAfterApplyWriteFailure(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsRootPath := workspace.Path(".agents")
	if err := os.WriteFile(agentsRootPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write blocking .agents file: %v", err)
	}

	failed := support.Run(t, workspace.Root, "install", "--scope", "skills")
	support.RequireExitCode(t, failed, 1)
	support.RequireNoStderr(t, failed)

	failedPayload := support.RequireJSONResult[installResult](t, failed)
	if failedPayload.OK {
		t.Fatalf("expected apply-mode write failure, got %#v", failedPayload)
	}

	if err := os.Remove(agentsRootPath); err != nil {
		t.Fatalf("remove blocking .agents file: %v", err)
	}

	retry := support.Run(t, workspace.Root, "install", "--scope", "skills")
	support.RequireSuccess(t, retry)
	support.RequireNoStderr(t, retry)

	retryPayload := support.RequireJSONResult[installResult](t, retry)
	if !retryPayload.OK || retryPayload.Scope != "skills" {
		t.Fatalf("expected successful retry payload, got %#v", retryPayload)
	}
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
}

func TestInstallDefaultScopeRecoversAfterMidFlightFailure(t *testing.T) {
	workspace := support.NewWorkspace(t)
	blockedSkillPath := workspace.Path(".agents/skills/harness-discovery/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(blockedSkillPath), 0o755); err != nil {
		t.Fatalf("mkdir blocked skill dir: %v", err)
	}
	if err := os.WriteFile(blockedSkillPath, []byte("stale skill"), 0o400); err != nil {
		t.Fatalf("write blocked skill file: %v", err)
	}

	failed := support.Run(t, workspace.Root, "install")
	support.RequireExitCode(t, failed, 1)
	support.RequireNoStderr(t, failed)

	failedPayload := support.RequireJSONResult[installResult](t, failed)
	if failedPayload.OK {
		t.Fatalf("expected default install failure, got %#v", failedPayload)
	}
	support.RequireFileExists(t, workspace.Path("AGENTS.md"))
	support.RequireFileMissing(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))

	if err := os.Chmod(blockedSkillPath, 0o644); err != nil {
		t.Fatalf("chmod blocked skill file: %v", err)
	}

	retry := support.Run(t, workspace.Root, "install")
	support.RequireSuccess(t, retry)
	support.RequireNoStderr(t, retry)

	retryPayload := support.RequireJSONResult[installResult](t, retry)
	if !retryPayload.OK || retryPayload.Scope != "all" {
		t.Fatalf("expected successful default-scope retry payload, got %#v", retryPayload)
	}
	support.RequireFileExists(t, workspace.Path("AGENTS.md"))
	support.RequireFileExists(t, workspace.Path(".agents/skills/harness-reviewer/SKILL.md"))
}

func TestInstallRefreshesExistingManagedWrapperAndThenNoops(t *testing.T) {
	workspace := support.NewWorkspace(t)
	agentsPath := workspace.Path("AGENTS.md")
	initial := strings.Join([]string{
		"# AGENTS.md",
		"",
		"Repo-owned intro.",
		"",
		"<!-- easyharness:begin -->",
		"outdated managed content",
		"<!-- easyharness:end -->",
		"",
		"## Repo Rules",
		"",
		"- Keep commits reviewable.",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	refresh := support.Run(t, workspace.Root, "install", "--scope", "agents")
	support.RequireSuccess(t, refresh)
	support.RequireNoStderr(t, refresh)

	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read refreshed AGENTS.md: %v", err)
	}
	agentsBody := string(agentsData)
	support.RequireContains(t, agentsBody, "Repo-owned intro.")
	support.RequireContains(t, agentsBody, "## Repo Rules")
	support.RequireContains(t, agentsBody, "## Harness Working Agreement")
	if strings.Contains(agentsBody, "outdated managed content") {
		t.Fatalf("expected refreshed managed block, got:\n%s", agentsBody)
	}

	second := support.Run(t, workspace.Root, "install", "--scope", "agents")
	support.RequireSuccess(t, second)
	support.RequireNoStderr(t, second)

	payload := support.RequireJSONResult[installResult](t, second)
	if !strings.Contains(payload.Summary, "already up to date") {
		t.Fatalf("expected noop wrapper rerun summary, got %#v", payload)
	}
	if len(payload.Actions) != 1 || payload.Actions[0].Kind != "noop" {
		t.Fatalf("expected noop wrapper rerun action, got %#v", payload.Actions)
	}
}

func TestSupportRunUsesBuiltBinaryInsteadOfPATH(t *testing.T) {
	workspace := support.NewWorkspace(t)
	poisonDir := workspace.Path("tmp/poison-bin")
	if err := os.MkdirAll(poisonDir, 0o755); err != nil {
		t.Fatalf("mkdir poison dir: %v", err)
	}

	name := "harness"
	script := "#!/bin/sh\necho poisoned harness\nexit 97\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		name += ".exe"
		script = "@echo poisoned harness\r\nexit /b 97\r\n"
		mode = 0o644
	}
	poisonPath := filepath.Join(poisonDir, name)
	if err := os.WriteFile(poisonPath, []byte(script), mode); err != nil {
		t.Fatalf("write poison harness: %v", err)
	}

	// Build once before poisoning PATH so the runner can only succeed by using
	// the cached absolute binary path instead of resolving `harness` from PATH.
	support.BuildBinary(t)
	t.Setenv("PATH", poisonDir)

	result := support.Run(t, workspace.Root, "--help")
	support.RequireSuccess(t, result)
	support.RequireContains(t, result.CombinedOutput(), "Usage: harness <command> [subcommand] [flags]")
	if result.CombinedOutput() == "poisoned harness\n" || result.CombinedOutput() == "poisoned harness\r\n" {
		t.Fatalf("expected support runner to bypass PATH and invoke the built binary, got %q", result.CombinedOutput())
	}
}

func TestPlanTemplateAndLintRoundTrip(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-22-smoke-plan.md"

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", "Smoke Plan",
		"--timestamp", "2026-03-22T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)

	planPath := workspace.Path(planRelPath)
	support.RequireFileExists(t, planPath)
	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("read rendered plan: %v", err)
	}
	support.RequireContains(t, string(data), "# Smoke Plan")
	support.RequireContains(t, string(data), "created_at: 2026-03-22T00:00:00Z")
	support.RequireContains(t, string(data), "source_type: issue")
	support.RequireContains(t, string(data), "source_refs: [\"#6\"]")

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	payload := support.RequireJSONResult[lintResult](t, lint)
	if !payload.OK {
		t.Fatalf("expected lint success, got %#v", payload)
	}
	if payload.Command != "plan lint" {
		t.Fatalf("expected lint command, got %#v", payload)
	}
	if payload.Artifacts.PlanPath != planRelPath {
		t.Fatalf("expected lint plan path %q, got %#v", planRelPath, payload)
	}
}

func requireVersionField(t *testing.T, output, field string) string {
	t.Helper()

	prefix := field + ": "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(line, prefix))
			if value == "" {
				t.Fatalf("expected version field %q to be non-empty\noutput:\n%s", field, output)
			}
			return value
		}
	}

	t.Fatalf("expected version field %q in output:\n%s", field, output)
	return ""
}

func gitHeadCommit(t *testing.T, repoRoot string) string {
	t.Helper()

	cmd := exec.Command("git", "-C", repoRoot, "rev-parse", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v\n%s", err, output)
	}

	commit := strings.TrimSpace(string(output))
	if commit == "" {
		t.Fatalf("expected git HEAD commit for %s", repoRoot)
	}
	return commit
}
