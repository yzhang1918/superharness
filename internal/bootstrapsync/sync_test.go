package bootstrapsync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/install"
)

func TestSyncRefreshesManagedOutputsFromBootstrapAssets(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	rewriteLegacyManagedOutputs(t, root, "harness-discovery")

	if _, err := Sync(root); err != nil {
		t.Fatalf("sync: %v", err)
	}

	agentsData, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	rendered := string(agentsData)
	if !strings.Contains(rendered, "Repo-specific intro.") || !strings.Contains(rendered, "Repo-specific footer.") {
		t.Fatalf("expected repo-specific AGENTS content to survive, got:\n%s", rendered)
	}
	if strings.Contains(rendered, "stale managed content") {
		t.Fatalf("expected managed block refresh, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, `<!-- easyharness:begin version="dev" -->`) {
		t.Fatalf("expected versioned managed block marker, got:\n%s", rendered)
	}

	skillPath := filepath.Join(root, ".agents/skills/harness-discovery/SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if !strings.Contains(string(skillData), "easyharness-managed: \"true\"") {
		t.Fatalf("expected managed skill metadata, got:\n%s", skillData)
	}
}

func TestCheckReportsDriftWhenManagedOutputsAreStale(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}
	rewriteLegacyManagedOutputs(t, root, "harness-reviewer")

	result, err := Check(root)
	if err == nil {
		t.Fatalf("expected drift error, got success %#v", result)
	}
	driftErr, ok := err.(*DriftError)
	if !ok {
		t.Fatalf("expected DriftError, got %T: %v", err, err)
	}
	if len(driftErr.Actions) == 0 {
		t.Fatalf("expected drift actions, got %#v", driftErr)
	}
}

func TestCheckReportsStaleManagedSkillPackages(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	orphanPath := filepath.Join(root, ".agents/skills/orphan/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o755); err != nil {
		t.Fatalf("mkdir orphan dir: %v", err)
	}
	orphanBody := strings.Join([]string{
		"---",
		"name: orphan",
		"description: stale easyharness-managed skill.",
		"metadata:",
		"  easyharness-managed: \"true\"",
		"  easyharness-version: dev",
		"---",
		"",
		"# Orphan",
		"",
	}, "\n")
	if err := os.WriteFile(orphanPath, []byte(orphanBody), 0o644); err != nil {
		t.Fatalf("write orphan skill: %v", err)
	}

	_, err := Check(root)
	if err == nil {
		t.Fatalf("expected drift error for orphaned file")
	}
	driftErr, ok := err.(*DriftError)
	if !ok {
		t.Fatalf("expected DriftError, got %T: %v", err, err)
	}

	found := false
	for _, action := range driftErr.Actions {
		if action.Path == ".agents/skills/orphan/SKILL.md" && action.Kind == install.ActionDelete {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected orphaned file drift action, got %#v", driftErr.Actions)
	}
}

func TestSyncRemovesStaleManagedSkillPackages(t *testing.T) {
	root := t.TempDir()

	if _, err := Sync(root); err != nil {
		t.Fatalf("initial sync: %v", err)
	}

	orphanPath := filepath.Join(root, ".agents/skills/orphan/SKILL.md")
	if err := os.MkdirAll(filepath.Dir(orphanPath), 0o755); err != nil {
		t.Fatalf("mkdir orphan dir: %v", err)
	}
	orphanBody := strings.Join([]string{
		"---",
		"name: orphan",
		"description: stale easyharness-managed skill.",
		"metadata:",
		"  easyharness-managed: \"true\"",
		"  easyharness-version: dev",
		"---",
		"",
		"# Orphan",
		"",
	}, "\n")
	if err := os.WriteFile(orphanPath, []byte(orphanBody), 0o644); err != nil {
		t.Fatalf("write orphan skill: %v", err)
	}

	result, err := Sync(root)
	if err != nil {
		t.Fatalf("sync with orphan: %v", err)
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("expected orphaned file removal, err=%v", err)
	}

	found := false
	for _, action := range result.Actions {
		if action.Path == ".agents/skills/orphan/SKILL.md" && action.Kind == install.ActionDelete {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected delete action in sync result, got %#v", result.Actions)
	}
}

func rewriteLegacyManagedOutputs(t *testing.T, root, skillName string) {
	t.Helper()

	agentsPath := filepath.Join(root, "AGENTS.md")
	agentsData, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	legacyAgents := strings.Replace(string(agentsData), `<!-- easyharness:begin version="dev" -->`, "<!-- easyharness:begin -->", 1)
	legacyAgents = strings.Replace(legacyAgents, "# AGENTS.md", "# AGENTS.md\n\nRepo-specific intro.", 1)
	legacyAgents = strings.TrimRight(legacyAgents, "\n") + "\n\nRepo-specific footer.\n"
	if err := os.WriteFile(agentsPath, []byte(legacyAgents), 0o644); err != nil {
		t.Fatalf("write legacy AGENTS.md: %v", err)
	}

	skillPath := filepath.Join(root, ".agents/skills", skillName, "SKILL.md")
	skillData, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read managed skill: %v", err)
	}
	legacySkill := regexp.MustCompile(`(?ms)\nmetadata:\n(?:  easyharness-managed: "true"\n  easyharness-version: [^\n]+\n)`).ReplaceAllString(string(skillData), "\n")
	if err := os.WriteFile(skillPath, []byte(legacySkill), 0o644); err != nil {
		t.Fatalf("write legacy skill: %v", err)
	}
}
