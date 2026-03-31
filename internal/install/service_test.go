package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCreatesManagedAgentsFileWhenMissing(t *testing.T) {
	root := t.TempDir()

	result := Service{Workdir: root}.Install(Options{})
	if !result.OK {
		t.Fatalf("expected install success, got %#v", result)
	}

	path := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# AGENTS.md") {
		t.Fatalf("expected AGENTS heading, got:\n%s", content)
	}
	if !strings.Contains(content, agentsManagedBlockBegin) || !strings.Contains(content, agentsManagedBlockEnd) {
		t.Fatalf("expected managed markers, got:\n%s", content)
	}
}

func TestInstallUpdatesManagedBlockWithoutTouchingUserContent(t *testing.T) {
	root := t.TempDir()
	original := strings.Join([]string{
		"# AGENTS.md",
		"",
		"User-owned intro.",
		"",
		agentsManagedBlockBegin,
		"outdated managed content",
		agentsManagedBlockEnd,
		"",
		"User-owned footer.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := Service{Workdir: root}.Install(Options{Scope: ScopeAgents})
	if !result.OK {
		t.Fatalf("expected install success, got %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "User-owned intro.") || !strings.Contains(content, "User-owned footer.") {
		t.Fatalf("expected user content to survive, got:\n%s", content)
	}
	if strings.Contains(content, "outdated managed content") {
		t.Fatalf("expected managed block refresh, got:\n%s", content)
	}
	if strings.Count(content, agentsManagedBlockBegin) != 1 || strings.Count(content, agentsManagedBlockEnd) != 1 {
		t.Fatalf("expected exactly one managed block, got:\n%s", content)
	}
}

func TestInstallRejectsDuplicateManagedBlocks(t *testing.T) {
	root := t.TempDir()
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		agentsManagedBlockBegin,
		"one",
		agentsManagedBlockEnd,
		"",
		agentsManagedBlockBegin,
		"two",
		agentsManagedBlockEnd,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := Service{Workdir: root}.Install(Options{Scope: ScopeAgents})
	if result.OK {
		t.Fatalf("expected duplicate-block install failure, got %#v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected duplicate-block errors, got %#v", result)
	}
}

func TestInstallIgnoresLiteralMarkerMentionsInUserOwnedProse(t *testing.T) {
	root := t.TempDir()
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"User-owned note mentioning markers inline: `<!-- easyharness:begin -->` and `<!-- easyharness:end -->`.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := Service{Workdir: root}.Install(Options{Scope: ScopeAgents})
	if !result.OK {
		t.Fatalf("expected install success, got %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	rendered := string(data)
	if !strings.Contains(rendered, "User-owned note mentioning markers inline") {
		t.Fatalf("expected inline marker prose to survive, got:\n%s", rendered)
	}
	if len(agentsManagedBlockBeginPattern.FindAllStringIndex(rendered, -1)) != 1 || len(agentsManagedBlockEndPattern.FindAllStringIndex(rendered, -1)) != 1 {
		t.Fatalf("expected a single managed block append, got:\n%s", rendered)
	}
}

func TestInstallRefreshesManagedSkillsWithoutRemovingUserFiles(t *testing.T) {
	root := t.TempDir()
	managedPath := filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o755); err != nil {
		t.Fatalf("mkdir managed skill dir: %v", err)
	}
	if err := os.WriteFile(managedPath, []byte("outdated skill content"), 0o644); err != nil {
		t.Fatalf("write managed skill: %v", err)
	}

	customPath := filepath.Join(root, ".agents/skills/custom/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatalf("mkdir custom skill dir: %v", err)
	}
	if err := os.WriteFile(customPath, []byte("user custom skill"), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	result := Service{Workdir: root}.Install(Options{Scope: ScopeSkills})
	if !result.OK {
		t.Fatalf("expected skill install success, got %#v", result)
	}

	managedData, err := os.ReadFile(managedPath)
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	if string(managedData) == "outdated skill content" {
		t.Fatalf("expected managed skill refresh, got:\n%s", managedData)
	}

	customData, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("read custom skill: %v", err)
	}
	if string(customData) != "user custom skill" {
		t.Fatalf("expected custom skill to survive untouched, got %q", customData)
	}
}

func TestInstallDryRunReportsPlannedActionsWithoutWriting(t *testing.T) {
	root := t.TempDir()

	result := Service{Workdir: root}.Install(Options{DryRun: true})
	if !result.OK {
		t.Fatalf("expected dry-run success, got %#v", result)
	}
	if len(result.Actions) == 0 {
		t.Fatalf("expected planned actions, got %#v", result)
	}

	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry run to avoid writing AGENTS.md, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents")); !os.IsNotExist(err) {
		t.Fatalf("expected dry run to avoid writing skills, err=%v", err)
	}
}

func TestInstallNoopsWhenManagedAssetsAreCurrent(t *testing.T) {
	root := t.TempDir()
	svc := Service{Workdir: root}
	first := svc.Install(Options{})
	if !first.OK {
		t.Fatalf("first install failed: %#v", first)
	}

	second := svc.Install(Options{})
	if !second.OK {
		t.Fatalf("second install failed: %#v", second)
	}
	for _, action := range second.Actions {
		if action.Kind != ActionNoop {
			t.Fatalf("expected noop actions on repeat install, got %#v", second.Actions)
		}
	}
	if len(second.NextAction) != 0 {
		t.Fatalf("expected noop repeat install to have no next actions, got %#v", second.NextAction)
	}
}

func TestInstallNoopsForRepoSpecificAgentsWrapperAfterRefresh(t *testing.T) {
	root := t.TempDir()
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"Repo-specific intro.",
		"",
		agentsManagedBlockBegin,
		"old managed content",
		agentsManagedBlockEnd,
		"",
		"## Repo Rules",
		"",
		"- Keep commits small.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	svc := Service{Workdir: root}
	first := svc.Install(Options{Scope: ScopeAgents})
	if !first.OK {
		t.Fatalf("first install failed: %#v", first)
	}

	second := svc.Install(Options{Scope: ScopeAgents})
	if !second.OK {
		t.Fatalf("second install failed: %#v", second)
	}
	if len(second.Actions) != 1 || second.Actions[0].Kind != ActionNoop {
		t.Fatalf("expected repo-specific wrapper to become noop, got %#v", second.Actions)
	}
}

func TestInstallRecognizesManagedBlockWithCRLFLineEndings(t *testing.T) {
	root := t.TempDir()
	content := strings.Join([]string{
		"# AGENTS.md",
		"",
		"Repo-specific intro.",
		"",
		agentsManagedBlockBegin,
		"old managed content",
		agentsManagedBlockEnd,
		"",
	}, "\r\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	svc := Service{Workdir: root}
	first := svc.Install(Options{Scope: ScopeAgents})
	if !first.OK {
		t.Fatalf("first install failed: %#v", first)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read refreshed AGENTS.md: %v", err)
	}
	normalized := strings.ReplaceAll(string(data), "\r\n", "")
	if strings.Contains(normalized, "\n") || strings.Contains(normalized, "\r") {
		t.Fatalf("expected refreshed AGENTS.md to preserve consistent CRLF line endings, got:\n%q", string(data))
	}

	second := svc.Install(Options{Scope: ScopeAgents})
	if !second.OK {
		t.Fatalf("second install failed: %#v", second)
	}
	if len(second.Actions) != 1 || second.Actions[0].Kind != ActionNoop {
		t.Fatalf("expected CRLF rerun to noop, got %#v", second.Actions)
	}
}
