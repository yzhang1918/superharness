package resilience_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

type commandError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type statusResult struct {
	OK      bool           `json:"ok"`
	Command string         `json:"command"`
	Summary string         `json:"summary"`
	Warnings []string      `json:"warnings"`
	Errors  []commandError `json:"errors"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type lifecycleResult struct {
	OK      bool           `json:"ok"`
	Command string         `json:"command"`
	Summary string         `json:"summary"`
	Errors  []commandError `json:"errors"`
	Artifacts struct {
		FromPlanPath    string `json:"from_plan_path"`
		ToPlanPath      string `json:"to_plan_path"`
		LocalStatePath  string `json:"local_state_path"`
		CurrentPlanPath string `json:"current_plan_path"`
	} `json:"artifacts"`
}

func writePlanFixture(t *testing.T, workspace *support.Workspace, relPath, title string, mutate func(string) string) string {
	t.Helper()

	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 4, 11, 13, 0, 0, 0, time.UTC),
		SourceType: "issue",
		SourceRefs: []string{"#37"},
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.Replace(rendered, "size: REPLACE_WITH_PLAN_SIZE", "size: M", 1)
	if mutate != nil {
		rendered = mutate(rendered)
	}
	return workspace.WriteFile(t, relPath, []byte(rendered))
}

func writeCurrentPlan(t *testing.T, workspace *support.Workspace, relPath string) {
	t.Helper()
	if _, err := runstate.SaveCurrentPlan(workspace.Root, relPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
}

func writeState(t *testing.T, workspace *support.Workspace, planStem string, state *runstate.State) {
	t.Helper()
	if _, err := runstate.SaveState(workspace.Root, planStem, state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

func writeReviewManifest(t *testing.T, workspace *support.Workspace, planStem, roundID string, payload map[string]any) string {
	t.Helper()
	return workspace.WriteJSON(t, filepath.ToSlash(filepath.Join(".local", "harness", "plans", planStem, "reviews", roundID, "manifest.json")), payload)
}

func writeReviewAggregate(t *testing.T, workspace *support.Workspace, planStem, roundID string, payload map[string]any) string {
	t.Helper()
	return workspace.WriteJSON(t, filepath.ToSlash(filepath.Join(".local", "harness", "plans", planStem, "reviews", roundID, "aggregate.json")), payload)
}

func writePublishRecord(t *testing.T, workspace *support.Workspace, planStem, relPlanPath, recordID, prURL string, revision int) string {
	t.Helper()
	return workspace.WriteJSON(t, filepath.ToSlash(filepath.Join(".local", "harness", "plans", planStem, "evidence", "publish", recordID+".json")), map[string]any{
		"record_id":   recordID,
		"kind":        "publish",
		"plan_path":   relPlanPath,
		"plan_stem":   planStem,
		"revision":    revision,
		"recorded_at": "2026-04-11T13:05:00Z",
		"status":      "recorded",
		"pr_url":      prURL,
		"branch":      "codex/resilience-fixture",
		"base":        "main",
		"commit":      "abc123def456",
	})
}

func writeCIRecord(t *testing.T, workspace *support.Workspace, planStem, relPlanPath, recordID, status string, revision int) string {
	t.Helper()
	return workspace.WriteJSON(t, filepath.ToSlash(filepath.Join(".local", "harness", "plans", planStem, "evidence", "ci", recordID+".json")), map[string]any{
		"record_id":   recordID,
		"kind":        "ci",
		"plan_path":   relPlanPath,
		"plan_stem":   planStem,
		"revision":    revision,
		"recorded_at": "2026-04-11T13:06:00Z",
		"status":      status,
		"provider":    "github-actions",
		"url":         "https://ci.example/build/42",
	})
}

func writeArchivedArchiveCandidate(t *testing.T, workspace *support.Workspace, relPath string) string {
	t.Helper()
	return writePlanFixture(t, workspace, relPath, "Resilience Archived Candidate", func(content string) string {
		return completeAllSteps(content, true)
	})
}

func writeActiveArchiveCandidate(t *testing.T, workspace *support.Workspace, relPath string) string {
	t.Helper()
	return writePlanFixture(t, workspace, relPath, "Resilience Active Candidate", func(content string) string {
		return completeAllSteps(content, true)
	})
}

func completeFirstStep(content string) string {
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 1)
	content = strings.Replace(content, "PENDING_STEP_EXECUTION", "Done.", 1)
	content = strings.Replace(content, "PENDING_STEP_REVIEW", "Reviewed.", 1)
	return content
}

func completeAllSteps(content string, archiveReady bool) string {
	content = strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")
	content = strings.ReplaceAll(content, "- [ ]", "- [x]")
	content = strings.ReplaceAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = strings.ReplaceAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: Fixture relies on explicit review artifacts.")
	if archiveReady {
		content = strings.Replace(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the candidate through deterministic repository-level fixtures.", 1)
		content = strings.Replace(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nNo unresolved blocking review findings remain.", 1)
		content = strings.Replace(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate is ready for archive.\n- Merge Handoff: Commit and push the archive move before merge approval.", 1)
		content = strings.Replace(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the planned resilience fixture.", 1)
		content = strings.Replace(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.", 1)
	}
	return content
}

func findError(errors []commandError, path string) bool {
	for _, issue := range errors {
		if issue.Path == path {
			return true
		}
	}
	return false
}

func findWarning(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}
	return false
}

func readCurrentPlan(t *testing.T, workspace *support.Workspace) map[string]any {
	t.Helper()
	data, err := os.ReadFile(workspace.Path(".local/harness/current-plan.json"))
	if err != nil {
		t.Fatalf("read current-plan.json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode current-plan.json: %v", err)
	}
	return payload
}
