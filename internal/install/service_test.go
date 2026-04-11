package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	versioninfo "github.com/catu-ai/easyharness/internal/version"
)

func testService(root string) Service {
	return Service{
		Workdir: root,
		Version: versioninfo.Info{Version: "v9.9.9", Mode: "release"},
		LookupEnv: func(key string) (string, bool) {
			return "", false
		},
		UserHomeDir: func() (string, error) {
			return filepath.Join(root, "home"), nil
		},
	}
}

func TestInitCreatesManagedInstructionsAndSkills(t *testing.T) {
	root := t.TempDir()

	result := testService(root).Init(Options{})
	if !result.OK {
		t.Fatalf("expected init success, got %#v", result)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	agentsBody := string(agentsData)
	if !strings.Contains(agentsBody, `<!-- easyharness:begin version="v9.9.9" -->`) {
		t.Fatalf("expected versioned managed marker, got:\n%s", agentsBody)
	}
	if !strings.Contains(agentsBody, ".agents/skills") {
		t.Fatalf("expected default repo skills path in managed block, got:\n%s", agentsBody)
	}

	skillData, err := os.ReadFile(filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md"))
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	skillBody := string(skillData)
	if !strings.Contains(skillBody, "easyharness-managed: \"true\"") {
		t.Fatalf("expected managed metadata in skill frontmatter, got:\n%s", skillBody)
	}
	if !strings.Contains(skillBody, "easyharness-version: v9.9.9") {
		t.Fatalf("expected version metadata in skill frontmatter, got:\n%s", skillBody)
	}
}

func TestInitRefreshesManagedBlockWithoutTouchingUserContent(t *testing.T) {
	root := t.TempDir()
	original := strings.Join([]string{
		"# AGENTS.md",
		"",
		"User-owned intro.",
		"",
		"<!-- easyharness:begin -->",
		"old managed content",
		"<!-- easyharness:end -->",
		"",
		"User-owned footer.",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(original), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := testService(root).Init(Options{})
	if !result.OK {
		t.Fatalf("expected init success, got %#v", result)
	}

	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	rendered := string(data)
	if !strings.Contains(rendered, "User-owned intro.") || !strings.Contains(rendered, "User-owned footer.") {
		t.Fatalf("expected user content to survive, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "old managed content") {
		t.Fatalf("expected managed block refresh, got:\n%s", rendered)
	}
	if strings.Count(rendered, "<!-- easyharness:begin") != 1 || strings.Count(rendered, "<!-- easyharness:end -->") != 1 {
		t.Fatalf("expected exactly one managed block, got:\n%s", rendered)
	}
}

func TestInstallSkillsRejectsNonManagedConflicts(t *testing.T) {
	root := t.TempDir()
	conflictPath := filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(conflictPath), 0o755); err != nil {
		t.Fatalf("mkdir conflict skill dir: %v", err)
	}
	conflictBody := strings.Join([]string{
		"---",
		"name: harness-discovery",
		"description: Custom user-owned skill.",
		"---",
		"",
		"# Custom",
		"",
	}, "\n")
	if err := os.WriteFile(conflictPath, []byte(conflictBody), 0o644); err != nil {
		t.Fatalf("write conflict skill: %v", err)
	}

	result := testService(root).InstallSkills(Options{})
	if result.OK {
		t.Fatalf("expected managed conflict failure, got %#v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected conflict errors, got %#v", result)
	}
}

func TestUninstallSkillsRemovesManagedPackagesButLeavesCustomOnes(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)
	if result := svc.InstallSkills(Options{}); !result.OK {
		t.Fatalf("install skills failed: %#v", result)
	}

	customPath := filepath.Join(root, ".agents/skills/custom/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customPath), 0o755); err != nil {
		t.Fatalf("mkdir custom skill dir: %v", err)
	}
	customBody := strings.Join([]string{
		"---",
		"name: custom",
		"description: User-owned custom skill.",
		"---",
		"",
		"# Custom",
		"",
	}, "\n")
	if err := os.WriteFile(customPath, []byte(customBody), 0o644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	result := svc.UninstallSkills(Options{})
	if !result.OK {
		t.Fatalf("expected uninstall success, got %#v", result)
	}

	if _, err := os.Stat(filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("expected managed skill removal, err=%v", err)
	}
	if _, err := os.Stat(customPath); err != nil {
		t.Fatalf("expected custom skill to survive, err=%v", err)
	}
}

func TestUninstallInstructionsDeletesFileWhenOnlyManagedContentRemains(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)
	if result := svc.InstallInstructions(Options{}); !result.OK {
		t.Fatalf("install instructions failed: %#v", result)
	}

	result := svc.UninstallInstructions(Options{})
	if !result.OK {
		t.Fatalf("expected uninstall instructions success, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected AGENTS.md deletion, err=%v", err)
	}
}

func TestUninstallInstructionsPreservesUserContentAroundManagedBlock(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)
	if result := svc.InstallInstructions(Options{}); !result.OK {
		t.Fatalf("install instructions failed: %#v", result)
	}

	agentsPath := filepath.Join(root, "AGENTS.md")
	installed, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read installed AGENTS.md: %v", err)
	}
	customized := strings.Join([]string{
		"# AGENTS.md",
		"",
		"User-owned intro.",
		"",
		strings.TrimSpace(string(installed)),
		"",
		"User-owned footer.",
		"",
	}, "\n")
	if err := os.WriteFile(agentsPath, []byte(customized), 0o644); err != nil {
		t.Fatalf("write mixed-content AGENTS.md: %v", err)
	}

	result := svc.UninstallInstructions(Options{})
	if !result.OK {
		t.Fatalf("expected uninstall instructions success, got %#v", result)
	}

	rendered, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read preserved AGENTS.md: %v", err)
	}
	body := string(rendered)
	if !strings.Contains(body, "User-owned intro.") || !strings.Contains(body, "User-owned footer.") {
		t.Fatalf("expected user content to survive, got:\n%s", body)
	}
	if strings.Contains(body, "<!-- easyharness:begin") || strings.Contains(body, "<!-- easyharness:end -->") {
		t.Fatalf("expected managed block removal, got:\n%s", body)
	}
}

func TestInstallInstructionsDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()

	result := testService(root).InstallInstructions(Options{DryRun: true})
	if !result.OK {
		t.Fatalf("expected dry-run success, got %#v", result)
	}
	if result.Mode != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to avoid writing AGENTS.md, err=%v", err)
	}
}

func TestInstallSkillsRejectsExistingUnmanagedDirectoryWithoutSkillFile(t *testing.T) {
	root := t.TempDir()
	conflictFile := filepath.Join(root, ".agents/skills/harness-discovery/custom.txt")
	if err := os.MkdirAll(filepath.Dir(conflictFile), 0o755); err != nil {
		t.Fatalf("mkdir unmanaged dir: %v", err)
	}
	if err := os.WriteFile(conflictFile, []byte("custom"), 0o644); err != nil {
		t.Fatalf("write unmanaged file: %v", err)
	}

	result := testService(root).InstallSkills(Options{})
	if result.OK {
		t.Fatalf("expected unmanaged-directory conflict, got %#v", result)
	}
}

func TestSkillsUserScopeUsesCodexHomeAndUninstallsManagedPackages(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)

	installResult := svc.InstallSkills(Options{Scope: ScopeUser})
	if !installResult.OK {
		t.Fatalf("user-scope install failed: %#v", installResult)
	}
	userSkill := filepath.Join(root, "home/.codex/skills/harness-discovery/SKILL.md")
	if _, err := os.Stat(userSkill); err != nil {
		t.Fatalf("expected user-scope skill install, err=%v", err)
	}

	uninstallResult := svc.UninstallSkills(Options{Scope: ScopeUser})
	if !uninstallResult.OK {
		t.Fatalf("user-scope uninstall failed: %#v", uninstallResult)
	}
	if _, err := os.Stat(userSkill); !os.IsNotExist(err) {
		t.Fatalf("expected managed user-scope skill removal, err=%v", err)
	}
}

func TestInstructionsUserScopeUsesCodexHomeAndUninstallsManagedBlock(t *testing.T) {
	root := t.TempDir()
	svc := testService(root)

	installResult := svc.InstallInstructions(Options{Scope: ScopeUser})
	if !installResult.OK {
		t.Fatalf("user-scope instructions install failed: %#v", installResult)
	}
	userAgents := filepath.Join(root, "home/.codex/AGENTS.md")
	data, err := os.ReadFile(userAgents)
	if err != nil {
		t.Fatalf("read user-scope AGENTS.md: %v", err)
	}
	if !strings.Contains(string(data), `<!-- easyharness:begin version="v9.9.9" -->`) {
		t.Fatalf("expected versioned managed block, got:\n%s", data)
	}

	uninstallResult := svc.UninstallInstructions(Options{Scope: ScopeUser})
	if !uninstallResult.OK {
		t.Fatalf("user-scope instructions uninstall failed: %#v", uninstallResult)
	}
	if _, err := os.Stat(userAgents); !os.IsNotExist(err) {
		t.Fatalf("expected managed user-scope instructions removal, err=%v", err)
	}
}

func TestInitSupportsExplicitOverrideTargetsForUnknownAgents(t *testing.T) {
	root := t.TempDir()
	result := testService(root).Init(Options{
		Agent:            "claude",
		SkillsDir:        ".claude/skills",
		InstructionsFile: "CLAUDE.md",
	})
	if !result.OK {
		t.Fatalf("expected explicit override init success, got %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "CLAUDE.md")); err != nil {
		t.Fatalf("expected explicit instructions file, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude/skills/harness-discovery/SKILL.md")); err != nil {
		t.Fatalf("expected explicit skills dir install, err=%v", err)
	}
}

func TestInitRefreshesVersionMarkersAcrossVersionChanges(t *testing.T) {
	root := t.TempDir()
	first := testService(root)
	if result := first.Init(Options{}); !result.OK {
		t.Fatalf("initial init failed: %#v", result)
	}

	second := testService(root)
	second.Version = versioninfo.Info{Version: "v10.0.0", Mode: "release"}
	if result := second.Init(Options{}); !result.OK {
		t.Fatalf("refresh init failed: %#v", result)
	}

	agentsData, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md after refresh: %v", err)
	}
	if !strings.Contains(string(agentsData), `<!-- easyharness:begin version="v10.0.0" -->`) {
		t.Fatalf("expected refreshed block version marker, got:\n%s", agentsData)
	}

	skillData, err := os.ReadFile(filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md"))
	if err != nil {
		t.Fatalf("read skill after refresh: %v", err)
	}
	if !strings.Contains(string(skillData), "easyharness-version: v10.0.0") {
		t.Fatalf("expected refreshed skill metadata version, got:\n%s", skillData)
	}
}
