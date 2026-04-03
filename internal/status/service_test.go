package status_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
)

const (
	stepOneTitle = "Step 1: Replace with first step title"
	stepTwoTitle = "Step 2: Replace with second step title"
)

func TestStatusPlanNodeForActivePlan(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness execute start" {
		t.Fatalf("expected execute-start guidance, got %#v", result.NextAction)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "plan" {
		t.Fatalf("expected cached plan node, got %#v", state)
	}

	doc, err := plan.LoadFile(filepath.Join(root, "docs/plans/active/2026-03-18-status-plan.md"))
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if got := doc.DerivedLifecycle(state); got != "awaiting_plan_approval" {
		t.Fatalf("expected cached plan node to preserve awaiting_plan_approval, got %q", got)
	}
}

func TestStatusPlanNodeForTrackedLightweightPlan(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		return strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected OK status result, got %#v", result)
	}
	if result.State.CurrentNode != "plan" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.PlanPath, relPath) {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusLightweightPublishPromptsForBreadcrumb(t *testing.T) {
	root := t.TempDir()
	relPath := ".local/harness/plans/archived/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, relPath)
	writeState(t, root, "2026-03-18-status-lightweight", map[string]any{
		"plan_path": relPath,
		"plan_stem": "2026-03-18-status-lightweight",
		"revision":  1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance first, got %#v", result.NextAction)
	}
	foundCommitPush := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Commit and push the tracked plan change created by archiving") {
			foundCommitPush = true
			break
		}
	}
	if !foundCommitPush {
		t.Fatalf("expected publish guidance to mention commit/push for the tracked archive change, got %#v", result.NextAction)
	}
	if !strings.Contains(result.Summary, "repo-visible breadcrumb") {
		t.Fatalf("expected summary to mention breadcrumb, got %q", result.Summary)
	}
}

func TestStatusExecutionStepImplementNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected cached execution node, got %#v", state)
	}
}

func TestStatusRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := status.Service{Workdir: root}.Read()
	if result.OK {
		t.Fatalf("expected status failure while state lock is held, got %#v", result)
	}
	if result.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Path != "state" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestStatusIgnoresNonStructuralReviewFactsForCurrentStep(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-011-delta",
			"kind":       "delta",
			"aggregated": true,
			"decision":   "pass",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "" || result.Facts.ReviewTrigger != "" {
		t.Fatalf("expected non-structural review facts to be ignored, got %#v", result.Facts)
	}
	if strings.Contains(result.Summary, "clean review") {
		t.Fatalf("expected non-structural review round to stay out of summary, got %q", result.Summary)
	}
}

func TestStatusExecutionStepReviewNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "in_progress" || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewTitle != stepOneTitle || result.Facts.ReviewTrigger != "step_closeout" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.ReviewRoundID != "review-001-delta" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusStepReviewMatchesTargetWithoutMarkdownPunctuation(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return strings.Replace(content, "### Step 1: Replace with first step title", "### Step 1: Resolve `current_node`", 1)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": "Step 1: Resolve current_node",
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected normalized target match to avoid warnings, got %#v", result.Warnings)
	}
}

func TestStatusDoesNotWarnForEarlierCompletedStepWithCleanFullReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-full", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-full", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected clean full step-closeout review to allow step 2, got %#v", result.State)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings for a clean earlier full review, got %#v", result.Warnings)
	}
}

func TestStatusWarnsWhenEarlierCompletedStepLacksReviewCompleteCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected later-step node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected warning for missing earlier step-closeout review, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 2 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected earliest repair guidance first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("expected review-start guidance after the warning action, got %#v", result.NextAction)
	}
}

func TestStatusWarnsWhenLatestHistoricalStepCloseoutRoundIsNotClean(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected latest non-clean historical closeout to restore the warning, got %#v", result.Warnings)
	}
}

func TestStatusDoesNotWarnWhenLatestHistoricalStepCloseoutRepairsEarlierFailure(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "changes_requested",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected clean latest historical closeout to allow step 2, got %#v", result.State)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected latest clean historical closeout to suppress warnings, got %#v", result.Warnings)
	}
}

func TestStatusWarnsWhenLatestHistoricalStepCloseoutRoundIsStillInFlight(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected in-flight latest historical closeout to restore the warning, got %#v", result.Warnings)
	}
}

func TestStatusWarnsWhenLatestHistoricalStepCloseoutManifestIsUnreadable(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepOneTitle,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) < 2 || !strings.Contains(result.Warnings[0], "Unable to read historical review manifest") || !strings.Contains(result.Warnings[1], "is invalid and cannot be mapped to a tracked step") {
		t.Fatalf("expected unreadable latest manifest to preserve a warning, got %#v", result.Warnings)
	}
}

func TestStatusWarnsWhenLatestUnreadableHistoricalCloseoutCannotBeMapped(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "Unable to read historical review manifest") {
		t.Fatalf("expected unreadable-history warning to remain visible, got %#v", result.Warnings)
	}
	foundUnscopedWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect an unmappable unreadable round to unsatisfy Step 1, got %#v", result.Warnings)
		}
		if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") && strings.Contains(warning, "review-002-delta") {
			foundUnscopedWarning = true
		}
	}
	if !foundUnscopedWarning {
		t.Fatalf("expected a conservative unmapped-round warning, got %#v", result.Warnings)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command != nil || strings.Contains(result.NextAction[0].Description, "review-002-delta") {
		t.Fatalf("expected ordinary next action guidance without unmapped-round repair work, got %#v", result.NextAction)
	}
}

func TestStatusFinalizeArchiveSuppressesArchiveActionForUnscopedUnreadableHistory(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/archive" {
		t.Fatalf("expected archive node to stay stable, got %#v", result.State)
	}
	if strings.Contains(result.Summary, "review-002-delta") {
		t.Fatalf("did not expect archive summary to mention ignored unmapped history, got %q", result.Summary)
	}
	foundUnscopedWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") && strings.Contains(warning, "review-002-delta") {
			foundUnscopedWarning = true
		}
	}
	if !foundUnscopedWarning {
		t.Fatalf("expected conservative unmapped-round warning, got %#v", result.Warnings)
	}
	foundArchiveAction := false
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness archive" {
			foundArchiveAction = true
		}
		if strings.Contains(action.Description, "review-002-delta") {
			t.Fatalf("did not expect archive next actions to mention ignored unmapped history, got %#v", result.NextAction)
		}
	}
	if !foundArchiveAction {
		t.Fatalf("expected ordinary archive action to remain available, got %#v", result.NextAction)
	}
}

func TestStatusDoesNotLetUnreadableHistoryForOneStepUnsatisfyAnother(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected finalize node to stay stable, got %#v", result.State)
	}
	foundStepTwo := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepTwoTitle) {
			foundStepTwo = true
		}
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect unreadable Step 2 history to unsatisfy Step 1, got %#v", result.Warnings)
		}
	}
	if !foundStepTwo {
		t.Fatalf("expected Step 2 closeout debt to remain, got %#v", result.Warnings)
	}
}

func TestStatusWarnsInFinalizeWhenCompletedStepStillLacksCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected finalize node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected exactly one finalize warning for the unresolved earlier step, got %#v", result.Warnings)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "Finalize progression is continuing") || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected finalize warning for missing earlier closeout, got %#v", result.Warnings)
	}
	if strings.Contains(result.Summary, "needs finalize review before archive") {
		t.Fatalf("expected finalize summary to stop claiming readiness once reminder debt exists, got %q", result.Summary)
	}
	if strings.Contains(result.Warnings[0], stepTwoTitle) {
		t.Fatalf("expected clean historical Step 2 closeout to stay out of finalize warnings, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 2 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected finalize warning guidance to come first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("expected finalize review guidance to remain available after the reminder, got %#v", result.NextAction)
	}
}

func TestStatusFinalizeArchiveSummaryAndActionsDoNotPretendReadyWhenCloseoutDebtExists(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/archive" {
		t.Fatalf("expected archive node to stay stable, got %#v", result.State)
	}
	if !strings.Contains(result.Summary, "still need review-complete closeout") || !strings.Contains(result.Summary, stepOneTitle) {
		t.Fatalf("expected archive summary to mention the missing closeout debt, got %q", result.Summary)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected archive warning to keep surfacing the missing closeout debt, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 2 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected archive repair guidance to mention the missing closeout debt, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("expected archive repair flow to keep the step-closeout review action, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness archive" {
			t.Fatalf("did not expect archive action while missing closeout debt remains, got %#v", result.NextAction)
		}
	}
}

func TestStatusFinalizeArchiveKeepsBlockerGuidanceWhenCloseoutDebtAndArchiveBlockersCoexist(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/archive" {
		t.Fatalf("expected archive node to stay stable, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ArchiveBlockerCount == 0 {
		t.Fatalf("expected archive blockers to remain visible, got %#v", result.Facts)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected mixed-debt archive warning to remain visible, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 2 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected reminder guidance first for mixed-debt archive state, got %#v", result.NextAction)
	}
	foundBlockerGuidance := false
	for _, action := range result.NextAction[1:] {
		if action.Command == nil && strings.Contains(action.Description, "Fix the archive blockers surfaced below") {
			foundBlockerGuidance = true
			break
		}
	}
	if !foundBlockerGuidance {
		t.Fatalf("expected archive blocker guidance to remain after the reminder, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness archive" {
			t.Fatalf("did not expect archive action while mixed debt remains, got %#v", result.NextAction)
		}
	}
}

func TestStatusDoesNotSuggestSecondReviewWhileStepReviewIsInFlight(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-002-delta",
			"kind":       "delta",
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/review" {
		t.Fatalf("expected in-flight step review node, got %#v", result.State)
	}
	if len(result.NextAction) < 2 {
		t.Fatalf("expected repair guidance plus aggregate action, got %#v", result.NextAction)
	}
	if result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected earlier-step repair guidance first, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "aggregate the active review round first") {
		t.Fatalf("expected in-flight repair guidance to mention aggregating first, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness review start --spec <path>" {
			t.Fatalf("did not expect a second review-start action while step review is active, got %#v", result.NextAction)
		}
	}
	if result.NextAction[1].Command == nil || !strings.Contains(*result.NextAction[1].Command, "harness review aggregate --round review-002-delta") {
		t.Fatalf("expected aggregate action to remain available, got %#v", result.NextAction)
	}
}

func TestStatusDoesNotSuggestSecondReviewWhileFinalizeReviewIsInFlight(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-003-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-003-full", map[string]any{
		"review_title": "Full branch candidate before archive",
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected in-flight finalize review node, got %#v", result.State)
	}
	if len(result.NextAction) < 2 {
		t.Fatalf("expected repair guidance plus aggregate action, got %#v", result.NextAction)
	}
	if result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected earlier-step repair guidance first, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "aggregate the active review round first") {
		t.Fatalf("expected finalize in-flight repair guidance to mention aggregating first, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness review start --spec <path>" {
			t.Fatalf("did not expect a second review-start action while finalize review is active, got %#v", result.NextAction)
		}
	}
	if result.NextAction[1].Command == nil || !strings.Contains(*result.NextAction[1].Command, "harness review aggregate --round review-003-full") {
		t.Fatalf("expected aggregate action to remain available, got %#v", result.NextAction)
	}
}

func TestStatusSuppressesMissingReviewWarningWithNoReviewNeededMarker(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeFirstStep(content)
		return replaceOnce(content, "Reviewed.", "NO_STEP_REVIEW_NEEDED: Doc-only wording cleanup with no contract or behavior change.")
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node with suppressed warning, got %#v", result.State)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected NO_STEP_REVIEW_NEEDED to suppress reminder warnings, got %#v", result.Warnings)
	}
}

func TestStatusNoReviewNeededMarkerDoesNotHideLaterFailedCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeFirstStep(content)
		return replaceOnce(content, "Reviewed.", "NO_STEP_REVIEW_NEEDED: Doc-only wording cleanup with no contract or behavior change.")
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected a later failed closeout to restore the reminder even with NO_STEP_REVIEW_NEEDED, got %#v", result.Warnings)
	}
}

func TestStatusNoReviewNeededMarkerDoesNotHideLaterInFlightCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeFirstStep(content)
		return replaceOnce(content, "Reviewed.", "NO_STEP_REVIEW_NEEDED: Doc-only wording cleanup with no contract or behavior change.")
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected an in-flight later closeout to restore the reminder even with NO_STEP_REVIEW_NEEDED, got %#v", result.Warnings)
	}
}

func TestStatusNoReviewNeededMarkerAllowsLaterCleanCloseoutToStaySatisfied(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeFirstStep(content)
		return replaceOnce(content, "Reviewed.", "NO_STEP_REVIEW_NEEDED: Doc-only wording cleanup with no contract or behavior change.")
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect a later clean closeout to restore reminder debt, got %#v", result.Warnings)
		}
	}
}

func TestStatusUnreadableFinalizeManifestDoesNotMasqueradeAsStepDebt(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-003-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-003-full")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable finalize manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable finalize manifest: %v", err)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected finalize review node to stay stable, got %#v", result.State)
	}
	if !strings.Contains(result.Summary, "waiting for the active review round to be aggregated") {
		t.Fatalf("expected ordinary finalize-review summary, got %q", result.Summary)
	}
	if len(result.NextAction) != 1 || result.NextAction[0].Command == nil || !strings.Contains(*result.NextAction[0].Command, "harness review aggregate --round review-003-full") {
		t.Fatalf("expected only aggregate guidance for unreadable finalize manifest, got %#v", result.NextAction)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") {
			t.Fatalf("did not expect unreadable finalize manifest to create unscoped step debt, got %#v", result.Warnings)
		}
	}
}

func TestStatusFinalizeReviewUsesAggregateFirstGuidanceForUnscopedUnreadableHistory(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-003-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-003-full", map[string]any{
		"review_title": "Full branch candidate before archive",
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected finalize review node to stay stable, got %#v", result.State)
	}
	if !strings.Contains(result.Summary, "Plan is in finalize review and waiting for the active review round to be aggregated.") || strings.Contains(result.Summary, "review-002-delta") {
		t.Fatalf("expected finalize review summary to ignore unreadable historical round, got %q", result.Summary)
	}
	if len(result.NextAction) < 1 || result.NextAction[0].Command == nil || !strings.Contains(*result.NextAction[0].Command, "harness review aggregate --round review-003-full") {
		t.Fatalf("expected ordinary aggregate action to remain first, got %#v", result.NextAction)
	}
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "review-002-delta") {
			t.Fatalf("did not expect finalize review guidance to mention ignored unmapped history, got %#v", result.NextAction)
		}
	}
}

func TestStatusFinalizeReviewSummaryForUnscopedUnreadableHistoryWithoutActiveRound(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected finalize review node to stay stable, got %#v", result.State)
	}
	if strings.Contains(result.Summary, "review-002-delta") {
		t.Fatalf("expected finalize-review summary to ignore the unreadable historical round, got %q", result.Summary)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness review start --spec <path>" {
		t.Fatalf("expected ordinary finalize review start guidance to remain available, got %#v", result.NextAction)
	}
}

func TestStatusFinalizeFixSummaryForUnscopedUnreadableHistory(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("expected finalize fix node to stay stable, got %#v", result.State)
	}
	if strings.Contains(result.Summary, "review-002-delta") {
		t.Fatalf("expected finalize-fix summary to ignore the unreadable historical round, got %q", result.Summary)
	}
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "review-002-delta") {
			t.Fatalf("did not expect finalize-fix guidance to mention ignored unmapped history, got %#v", result.NextAction)
		}
	}
	foundFinalizeRestart := false
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness review start --spec <path>" {
			foundFinalizeRestart = true
			break
		}
	}
	if !foundFinalizeRestart {
		t.Fatalf("expected finalize review restart guidance to remain available, got %#v", result.NextAction)
	}
}

func TestStatusFailedStepReviewUsesCachedStepWhenManifestIsMissing(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"current_node":         "execution/step-1/implement",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"step":       1,
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected cache to stay pinned to the reviewed step, got %#v", state)
	}
}

func TestStatusUnknownAggregatedReviewDecisionStaysConservative(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"aggregated": true,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "unknown" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Recover or rerun review-001-delta") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusFailedStepReviewPinsReviewedStep(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-002-delta",
			"kind":       "delta",
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewStatus != "changes_requested" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusAdvancesToNextStepAfterCleanStepReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-003-delta",
			"kind":       "delta",
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-003-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepTwoTitle {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusFinalizeReviewNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness review start --spec <path>" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusFinalizeReviewClearsPriorStepReviewFacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, false)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-003-delta",
			"kind":       "delta",
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-003-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts != nil && (result.Facts.ReviewStatus != "" || result.Facts.ReviewTitle != "" || result.Facts.ReviewTrigger != "" || result.Facts.ReviewKind != "") {
		t.Fatalf("expected prior step-review facts to be cleared at finalize review, got %#v", result.Facts)
	}
	if result.Artifacts != nil && result.Artifacts.ReviewRoundID != "" {
		t.Fatalf("expected prior step-review artifact pointer to be cleared at finalize review, got %#v", result.Artifacts)
	}
}

func TestStatusFinalizeReviewInFlightIncludesReviewFacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": false,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewTrigger != "pre_archive" || result.Facts.ReviewStatus != "in_progress" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusFinalizeFixNodeAfterFailedFinalizeReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "changes_requested" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusFinalizeArchiveNodeAfterCleanFinalizeReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "pass",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/archive" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.Blockers) != 0 {
		t.Fatalf("expected no archive blockers, got %#v", result.Blockers)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness archive" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanNeedsPublishEvidence(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	foundPublishSubmit := false
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness evidence submit --kind publish --input <json>" {
			foundPublishSubmit = true
			break
		}
	}
	if !foundPublishSubmit {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusWarnsInArchivedPublishWhenCompletedStepStillLacksCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected publish node to stay stable, got %#v", result.State)
	}
	if !strings.Contains(result.Summary, "reopen the candidate") || !strings.Contains(result.Summary, stepOneTitle) {
		t.Fatalf("expected publish summary to require reopen before merge-ready handoff, got %q", result.Summary)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected archived publish warning for unresolved Step 1 closeout, got %#v", result.Warnings)
	}
	if strings.Contains(result.Warnings[0], stepTwoTitle) {
		t.Fatalf("expected clean historical Step 2 closeout to stay out of archived publish warnings, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 3 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected archived publish repair guidance first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness reopen --mode finalize-fix" {
		t.Fatalf("expected archived publish flow to require reopen before repair, got %#v", result.NextAction)
	}
	foundPublishGuidance := false
	for _, action := range result.NextAction[2:] {
		if strings.Contains(action.Description, "Open or update the PR for the archived candidate") {
			foundPublishGuidance = true
			break
		}
	}
	if !foundPublishGuidance {
		t.Fatalf("expected ordinary publish follow-up to remain after the reopen guidance, got %#v", result.NextAction)
	}
}

func TestStatusLightweightPublishPrioritizesRepairDebtBeforeBreadcrumb(t *testing.T) {
	root := t.TempDir()
	relPath := ".local/harness/plans/archived/2026-03-18-status-lightweight.md"
	writePlan(t, root, relPath, func(content string) string {
		content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
		return completeAllStepsWithoutCloseout(content, true)
	})
	writeCurrentPlan(t, root, relPath)
	writeReviewManifest(t, root, "2026-03-18-status-lightweight", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-lightweight", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected publish node to stay stable, got %#v", result.State)
	}
	if len(result.NextAction) < 2 || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected reopen guidance to remain first when lightweight publish still has closeout debt, got %#v", result.NextAction)
	}
	if strings.Contains(result.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("did not expect breadcrumb guidance to outrank repair-first actions, got %#v", result.NextAction)
	}
}

func TestStatusArchivedNodesRequireReopenForUnscopedUnreadableHistory(t *testing.T) {
	cases := []struct {
		name                string
		withMergeEvidence   bool
		expectedNode        string
		expectedOrdinaryCue string
	}{
		{
			name:                "publish",
			withMergeEvidence:   false,
			expectedNode:        "execution/finalize/publish",
			expectedOrdinaryCue: "Open or update the PR for the archived candidate",
		},
		{
			name:                "await-merge",
			withMergeEvidence:   true,
			expectedNode:        "execution/finalize/await_merge",
			expectedOrdinaryCue: "Wait for explicit human approval before merging the PR.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
				return completeAllSteps(content, true)
			})
			writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

			dir := filepath.Join(root, ".local", "harness", "plans", "2026-03-18-status-plan", "reviews", "review-002-delta")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("mkdir unreadable manifest dir: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
				t.Fatalf("write unreadable manifest: %v", err)
			}
			writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
				"review_title": "mystery historical target",
				"revision":     1,
				"decision":     "changes_requested",
			})

			if tc.withMergeEvidence {
				svc := evidence.Service{
					Workdir: root,
					Now: func() time.Time {
						return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
					},
				}
				if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
					t.Fatalf("publish evidence: %#v", result)
				}
				if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
					t.Fatalf("ci evidence: %#v", result)
				}
				if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
					t.Fatalf("sync evidence: %#v", result)
				}
			}

			result := status.Service{Workdir: root}.Read()
			if result.State.CurrentNode != tc.expectedNode {
				t.Fatalf("expected archived node %q, got %#v", tc.expectedNode, result.State)
			}
			if strings.Contains(result.Summary, "review-002-delta") {
				t.Fatalf("expected archived summary to ignore unmapped history, got %q", result.Summary)
			}
			foundUnscopedWarning := false
			for _, warning := range result.Warnings {
				if strings.Contains(warning, stepOneTitle) || strings.Contains(warning, stepTwoTitle) {
					t.Fatalf("did not expect unmapped unreadable history to invent step-specific debt, got %#v", result.Warnings)
				}
				if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") && strings.Contains(warning, "review-002-delta") {
					foundUnscopedWarning = true
				}
			}
			if !foundUnscopedWarning {
				t.Fatalf("expected conservative unmapped-round warning, got %#v", result.Warnings)
			}
			for _, action := range result.NextAction {
				if strings.Contains(action.Description, "review-002-delta") {
					t.Fatalf("did not expect archived next actions to mention ignored unmapped history, got %#v", result.NextAction)
				}
			}
			foundOrdinaryGuidance := false
			for _, action := range result.NextAction {
				if strings.Contains(action.Description, tc.expectedOrdinaryCue) {
					foundOrdinaryGuidance = true
					break
				}
			}
			if !foundOrdinaryGuidance {
				t.Fatalf("expected ordinary archived follow-up to remain after reopen guidance, got %#v", result.NextAction)
			}
		})
	}
}

func TestStatusArchivedPlanReadyForAwaitMerge(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.PRURL != "https://github.com/catu-ai/easyharness/pull/13" || result.Facts.CIStatus != "not_applied" || result.Facts.SyncStatus != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PublishRecordID == "" || result.Artifacts.CIRecordID == "" || result.Artifacts.SyncRecordID == "" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusWarnsInAwaitMergeWhenCompletedStepStillLacksCloseout(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, false)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": stepTwoTitle,
		"step":         2,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"decision": "pass",
	})

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("expected await_merge node to stay stable, got %#v", result.State)
	}
	if !strings.Contains(result.Summary, "reopen the candidate") || !strings.Contains(result.Summary, stepOneTitle) {
		t.Fatalf("expected await_merge summary to require reopen before merge-ready handoff, got %q", result.Summary)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], stepOneTitle) {
		t.Fatalf("expected await_merge warning for unresolved Step 1 closeout, got %#v", result.Warnings)
	}
	if strings.Contains(result.Warnings[0], stepTwoTitle) {
		t.Fatalf("expected clean historical Step 2 closeout to stay out of await_merge warnings, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 3 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected await_merge repair guidance first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness reopen --mode finalize-fix" {
		t.Fatalf("expected await_merge flow to require reopen before repair, got %#v", result.NextAction)
	}
	foundAwaitMergeGuidance := false
	for _, action := range result.NextAction[2:] {
		if action.Command == nil && strings.Contains(action.Description, "Wait for explicit human approval before merging the PR.") {
			foundAwaitMergeGuidance = true
			break
		}
	}
	if !foundAwaitMergeGuidance {
		t.Fatalf("expected ordinary await_merge follow-up to remain after the reopen guidance, got %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeFromLegacyEvidenceCache(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"latest_publish": map[string]any{
			"attempt_id": "publish-legacy-001",
			"pr_url":     "https://github.com/catu-ai/easyharness/pull/13",
		},
		"latest_ci": map[string]any{
			"snapshot_id": "ci-legacy-001",
			"status":      "success",
		},
		"sync": map[string]any{
			"freshness": "fresh",
			"conflicts": false,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("expected legacy evidence fallback to reach await_merge, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.PRURL != "https://github.com/catu-ai/easyharness/pull/13" || result.Facts.CIStatus != "success" || result.Facts.SyncStatus != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PublishRecordID != "publish-legacy-001" || result.Artifacts.CIRecordID != "ci-legacy-001" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeWithSyncNotApplied(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no meaningful merge-base freshness signal in this test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CIStatus != "success" || result.Facts.SyncStatus != "not_applied" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusArchivedPlanReadyForAwaitMergeWhenCIAndSyncAreBothNotApplied(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"not_applied","reason":"repository has no hosted CI in this test"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no meaningful merge-base freshness signal in this test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CIStatus != "not_applied" || result.Facts.SyncStatus != "not_applied" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusArchivedPlanStaysInPublishFromLegacyEvidenceCacheWhenDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"latest_publish": map[string]any{
			"attempt_id": "publish-legacy-001",
			"pr_url":     "https://github.com/catu-ai/easyharness/pull/13",
		},
		"latest_ci": map[string]any{
			"snapshot_id": "ci-legacy-001",
			"status":      "failed",
		},
		"sync": map[string]any{
			"freshness": "fresh",
			"conflicts": false,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty legacy evidence fallback to stay in publish, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CIStatus != "failed" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundFixCI := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Fix the CI failures") {
			foundFixCI = true
			break
		}
	}
	if !foundFixCI {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanStaysInPublishWhenEvidenceIsDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"pending","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty evidence to stay in publish, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CIStatus != "pending" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundPendingCI := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Wait for the relevant post-archive CI") {
			foundPendingCI = true
			break
		}
	}
	if !foundPendingCI {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusArchivedPlanStaysInPublishWhenSyncIsDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")

	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 0, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/13"}`)); !result.OK {
		t.Fatalf("publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"conflicted","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty sync evidence to stay in publish, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.SyncStatus != "conflicted" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	foundResolveConflicts := false
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "Resolve merge conflicts") {
			foundResolveConflicts = true
			break
		}
	}
	if !foundResolveConflicts {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusLandNode(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"current_node": "land",
		"land": map[string]any{
			"pr_url":    "https://github.com/catu-ai/easyharness/pull/99",
			"commit":    "abc123",
			"landed_at": "2026-03-18T12:00:00Z",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "land" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.LandPRURL == "" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness land complete" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusIdleNodeAfterLand(t *testing.T) {
	root := t.TempDir()
	writeCurrentPlanPayload(t, root, map[string]any{
		"last_landed_plan_path": "docs/plans/archived/2026-03-18-status-plan.md",
		"last_landed_at":        "2026-03-19T12:00:00Z",
	})

	result := status.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected idle result, got %#v", result)
	}
	if result.State.CurrentNode != "idle" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Artifacts == nil || result.Artifacts.LastLandedPlanPath != "docs/plans/archived/2026-03-18-status-plan.md" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
}

func TestStatusReopenedFinalizeFixNeedsReview(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "finalize-fix",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusReopenedNewStepPendingPromptsForNewStep(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if !strings.Contains(result.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestStatusReopenedNewStepPendingKeepsNewStepCueEvenWithMissingCloseoutWarnings(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeAllStepsWithoutCloseout(content, true)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if !strings.Contains(result.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "still lack review-complete closeout") {
		t.Fatalf("expected missing closeout warnings to remain visible, got %#v", result.Warnings)
	}
}

func TestStatusReopenedNewStepContinuesOnceStepExists(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeAllSteps(content, true)
		return appendThirdStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-3/implement" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != "Step 3: Follow-up reopened work" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
}

func TestStatusConsumedReopenedNewStepDoesNotForceAnotherStepAfterLaterFinding(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		content = completeAllSteps(content, true)
		content = appendThirdStep(content)
		content = replaceOnce(content, "- Done: [ ]", "- Done: [x]")
		content = replaceOnce(content, "PENDING_STEP_EXECUTION", "Done.")
		content = replaceOnce(content, "PENDING_STEP_REVIEW", "Reviewed.")
		return content
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"reopen": map[string]any{
			"mode":            "new-step",
			"reopened_at":     "2026-03-18T11:00:00+08:00",
			"base_step_count": 2,
		},
		"active_review_round": map[string]any{
			"round_id":   "review-005-full",
			"kind":       "full",
			"revision":   1,
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("expected finalize fix node, got %#v", result.State)
	}
	if strings.Contains(result.Summary, "needs a new unfinished step") {
		t.Fatalf("expected consumed new-step reopen mode to stop forcing another step, got %q", result.Summary)
	}
	if len(result.NextAction) == 0 || strings.Contains(result.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected finalize repair guidance instead of another new-step demand, got %#v", result.NextAction)
	}
	if result.Facts != nil && result.Facts.ReopenMode == "new-step" {
		t.Fatalf("expected consumed reopen mode to stop surfacing raw new-step guidance, got %#v", result.Facts)
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
	writeCurrentPlanPayload(t, root, map[string]any{"plan_path": relPath})
}

func writeCurrentPlanPayload(t *testing.T, root string, payloadMap map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}
	payload, err := json.Marshal(payloadMap)
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

func writeReviewManifest(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func writeReviewAggregate(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir aggregate dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal aggregate: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aggregate.json"), data, 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func completeFirstStep(content string) string {
	content = replaceOnce(content, "- Done: [ ]", "- Done: [x]")
	content = replaceOnce(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceOnce(content, "PENDING_STEP_REVIEW", "Reviewed.")
	return content
}

func completeAllSteps(content string, archiveReady bool) string {
	content = completeAllStepsWithoutCloseout(content, archiveReady)
	return stringsReplaceAll(content, "Reviewed.", "NO_STEP_REVIEW_NEEDED: Test fixture uses explicit review-complete closeout.")
}

func completeAllStepsWithoutCloseout(content string, archiveReady bool) string {
	content = stringsReplaceAll(content, "- Done: [ ]", "- Done: [x]")
	content = stringsReplaceAll(content, "- [ ]", "- [x]")
	content = stringsReplaceAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = stringsReplaceAll(content, "PENDING_STEP_REVIEW", "Reviewed.")
	if archiveReady {
		content = stringsReplaceAll(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the implementation.")
		content = stringsReplaceAll(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nNo blocking review findings remain.")
		content = stringsReplaceAll(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate is ready for archive.\n- Merge Handoff: Commit and push the archive move before merge approval.")
		content = stringsReplaceAll(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the planned slice.")
		content = stringsReplaceAll(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.")
	}
	return content
}

func appendThirdStep(content string) string {
	insert := `### Step 3: Follow-up reopened work

- Done: [ ]

#### Objective

Carry the reopened follow-up work as a proper third step.

#### Details

NONE

#### Expected Files

- ` + "`path/to/file`" + `

#### Validation

- Verify the reopened scope is complete.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy`
	return strings.Replace(content, "## Validation Strategy", insert, 1)
}

func replaceOnce(content, old, new string) string {
	return strings.Replace(content, old, new, 1)
}

func stringsReplaceAll(content, old, new string) string {
	return strings.ReplaceAll(content, old, new)
}
