package status_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/status"
)

func TestStatusMinimalActivePlan(t *testing.T) {
	root := t.TempDir()
	planPath := writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.PlanStatus != "active" || result.State.Lifecycle != "awaiting_plan_approval" {
		t.Fatalf("unexpected state: %#v", result.State)
	}
	if result.State.Step != "Step 1: Replace with first step title" {
		t.Fatalf("unexpected step: %#v", result.State)
	}
	if result.State.StepState != "" {
		t.Fatalf("expected no step_state outside executing, got %#v", result.State)
	}
	if result.Artifacts.PlanPath != planPath {
		t.Fatalf("unexpected plan path: %#v", result.Artifacts)
	}
}

func TestStatusReviewInProgress(t *testing.T) {
	root := t.TempDir()
	planPath := writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = replaceOnce(t, content, "lifecycle: awaiting_plan_approval", "lifecycle: executing")
		content = replaceOnce(t, content, "- Status: pending", "- Status: in_progress")
		return content
	})
	writeCurrentPlan(t, root, "docs/plans/active/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"active_review_round": map[string]any{
			"round_id":   "round-1",
			"kind":       "delta",
			"aggregated": false,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK || result.State.StepState != "reviewing" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Artifacts.PlanPath != planPath || result.Artifacts.ReviewRoundID != "round-1" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusWaitingCI(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = replaceOnce(t, content, "lifecycle: awaiting_plan_approval", "lifecycle: executing")
		content = replaceOnce(t, content, "- Status: pending", "- Status: in_progress")
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"latest_ci": map[string]any{
			"snapshot_id": "ci-1",
			"status":      "pending",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.StepState != "waiting_ci" {
		t.Fatalf("unexpected step state: %#v", result.State)
	}
}

func TestStatusResolvingConflicts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = replaceOnce(t, content, "lifecycle: awaiting_plan_approval", "lifecycle: executing")
		content = replaceOnce(t, content, "- Status: pending", "- Status: in_progress")
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"sync": map[string]any{
			"freshness": "stale",
			"conflicts": true,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.StepState != "resolving_conflicts" {
		t.Fatalf("unexpected step state: %#v", result.State)
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("expected remote freshness warning")
	}
}

func TestStatusReadyForArchive(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = replaceOnce(t, content, "lifecycle: awaiting_plan_approval", "lifecycle: executing")
		content = stringsReplaceAll(content, "- Status: pending", "- Status: completed")
		content = stringsReplaceAll(content, "- [ ]", "- [x]")
		content = stringsReplaceAll(content, "PENDING_STEP_EXECUTION", "Done.")
		content = stringsReplaceAll(content, "PENDING_STEP_REVIEW", "Reviewed.")
		content = stringsReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Ready.")
		return content
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.StepState != "ready_for_archive" {
		t.Fatalf("unexpected step state: %#v", result.State)
	}
}

func TestStatusArchivedPlan(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		content = replaceOnce(t, content, "status: active", "status: archived")
		content = replaceOnce(t, content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval")
		content = stringsReplaceAll(content, "- Status: pending", "- Status: completed")
		content = stringsReplaceAll(content, "- [ ]", "- [x]")
		content = stringsReplaceAll(content, "PENDING_STEP_EXECUTION", "Done.")
		content = stringsReplaceAll(content, "PENDING_STEP_REVIEW", "Reviewed.")
		content = stringsReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Ready.")
		return content
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK || result.State.PlanStatus != "archived" || result.State.Lifecycle != "awaiting_merge_approval" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.State.StepState != "" {
		t.Fatalf("expected no step_state for archived plan, got %#v", result.State)
	}
}

func writePlan(t *testing.T, root, relPath string, mutate func(string) string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Status Plan",
		Timestamp:  time.Date(2026, 3, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	content := mutate(rendered)
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return path
}

func writeCurrentPlan(t *testing.T, root, relPath string) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}
	payload, err := json.Marshal(map[string]any{"plan_path": relPath})
	if err != nil {
		t.Fatalf("marshal current-plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "current-plan.json"), payload, 0o644); err != nil {
		t.Fatalf("write current-plan: %v", err)
	}
}

func writeState(t *testing.T, root, planStem string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), data, 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

func replaceOnce(t *testing.T, content, old, new string) string {
	t.Helper()
	updated := stringsReplaceOnce(content, old, new)
	if updated == content {
		t.Fatalf("expected replacement %q -> %q", old, new)
	}
	return updated
}

func stringsReplaceOnce(content, old, new string) string {
	return strings.Replace(content, old, new, 1)
}

func stringsReplaceAll(content, old, new string) string {
	return strings.ReplaceAll(content, old, new)
}
