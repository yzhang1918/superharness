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
	"github.com/catu-ai/easyharness/internal/review"
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
	if state != nil {
		t.Fatalf("expected status read to avoid caching plan node, got %#v", state)
	}

	doc, err := plan.LoadFile(filepath.Join(root, "docs/plans/active/2026-03-18-status-plan.md"))
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if got := doc.DerivedLifecycle(nil); got != "awaiting_plan_approval" {
		t.Fatalf("expected lifecycle to derive from the plan alone, got %q", got)
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
		"revision": 1,
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
	if state == nil || state.ExecutionStartedAt != "2026-03-18T10:05:00+08:00" {
		t.Fatalf("expected execution-start state to remain available, got %#v", state)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "current_node")
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

func TestStatusWarnsWhenHistoricalCloseoutMissingRevisionIsIgnored(t *testing.T) {
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
		"review_title": "mystery historical target",
		"step":         1,
		"revision":     0,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	foundIgnoredWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") && strings.Contains(warning, "review-002-delta") {
			foundIgnoredWarning = true
		}
	}
	if !foundIgnoredWarning {
		t.Fatalf("expected ignored malformed-round warning, got %#v", result.Warnings)
	}
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "review-002-delta") {
			t.Fatalf("did not expect missing-revision history to add repair guidance, got %#v", result.NextAction)
		}
	}
}

func TestStatusWarnsWhenHistoricalCloseoutStepIsOutOfRangeIsIgnored(t *testing.T) {
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
		"review_title": "mystery historical target",
		"step":         99,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to stay stable, got %#v", result.State)
	}
	foundIgnoredWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "is invalid and cannot be mapped to a tracked step") && strings.Contains(warning, "review-002-delta") {
			foundIgnoredWarning = true
		}
	}
	if !foundIgnoredWarning {
		t.Fatalf("expected ignored malformed-round warning, got %#v", result.Warnings)
	}
	for _, action := range result.NextAction {
		if strings.Contains(action.Description, "review-002-delta") {
			t.Fatalf("did not expect out-of-range-step history to add repair guidance, got %#v", result.NextAction)
		}
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

func TestStatusShowsExplicitEarlierStepRepairAsInFlightReview(t *testing.T) {
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
		"active_review_round": map[string]any{
			"round_id":   "review-004-full",
			"kind":       "full",
			"step":       1,
			"revision":   1,
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-004-full", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/review" {
		t.Fatalf("expected explicit earlier-step repair to show step 1 review in flight, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewStatus != "in_progress" {
		t.Fatalf("expected in-flight review facts for step 1, got %#v", result.Facts)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect passive debt warnings while explicit repair review is active, got %#v", result.Warnings)
		}
	}
	for _, action := range result.NextAction {
		if action.Command != nil && *action.Command == "harness review start --spec <path>" {
			t.Fatalf("did not expect a second review-start action while explicit repair review is active, got %#v", result.NextAction)
		}
	}
	if len(result.NextAction) == 0 || result.NextAction[len(result.NextAction)-1].Command == nil || !strings.Contains(*result.NextAction[len(result.NextAction)-1].Command, "harness review aggregate --round review-004-full") {
		t.Fatalf("expected aggregate action for the explicit repair round, got %#v", result.NextAction)
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

func TestStatusFailedStepReviewUsesActiveReviewRoundWhenManifestIsMissing(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
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
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.RoundID != "review-001-delta" {
		t.Fatalf("expected review control state to remain intact without a node cache, got %#v", state)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "current_node")
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
			if tc.name == "await-merge" {
				foundRequiredBookkeeping := false
				for _, action := range result.NextAction {
					if strings.Contains(action.Description, "enter required post-merge bookkeeping") {
						foundRequiredBookkeeping = true
						break
					}
				}
				if !foundRequiredBookkeeping {
					t.Fatalf("expected await_merge guidance to mention required post-merge bookkeeping, got %#v", result.NextAction)
				}
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

func TestStatusArchivedPlanReadyForAwaitMergeFromEvidenceArtifacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
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
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/await_merge" {
		t.Fatalf("expected evidence artifacts to reach await_merge, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.PRURL != "https://github.com/catu-ai/easyharness/pull/13" || result.Facts.CIStatus != "success" || result.Facts.SyncStatus != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PublishRecordID == "" || result.Artifacts.CIRecordID == "" || result.Artifacts.SyncRecordID == "" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "latest_publish", "latest_ci", "latest_evidence")
}

func TestStatusArchivedPlanIgnoresOlderRevisionEvidenceArtifacts(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
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
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 2,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected older revision evidence to keep publish state, got %#v", result.State)
	}
	if result.Facts != nil && (result.Facts.PRURL != "" || result.Facts.CIStatus != "" || result.Facts.SyncStatus != "") {
		t.Fatalf("expected older revision evidence to stay hidden, got %#v", result.Facts)
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

func TestStatusArchivedPlanStaysInPublishFromEvidenceArtifactsWhenDirty(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md", func(content string) string {
		return completeAllSteps(content, true)
	})
	writeCurrentPlan(t, root, "docs/plans/archived/2026-03-18-status-plan.md")
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"revision": 1,
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
	if result := svc.Submit("ci", []byte(`{"status":"failed","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("sync evidence: %#v", result)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected dirty evidence artifacts to stay in publish, got %#v", result.State)
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
	assertStateJSONLacksKeys(t, root, "2026-03-18-status-plan", "latest_publish", "latest_ci", "latest_evidence")
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
	if !strings.Contains(result.Summary, "required post-merge bookkeeping is still in progress") {
		t.Fatalf("expected land summary to mention required bookkeeping, got %#v", result)
	}
	if result.Facts == nil || result.Facts.LandPRURL == "" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness land complete" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "required post-merge bookkeeping") {
		t.Fatalf("expected bookkeeping guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "final PR comment") {
		t.Fatalf("expected final PR comment guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[0].Description, "follow-up references") {
		t.Fatalf("expected linked issue follow-up guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "only after the required PR and issue bookkeeping is done") {
		t.Fatalf("expected land complete gate guidance, got %#v", result.NextAction)
	}
	if !strings.Contains(result.NextAction[1].Description, "required post-merge bookkeeping completion") {
		t.Fatalf("expected land complete action to mention required bookkeeping completion, got %#v", result.NextAction)
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

func TestStatusReopenedNewStepKeepsLaterFrontierWhileEarlierCloseoutDebtExists(t *testing.T) {
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
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"review_title": stepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"decision": "changes_requested",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-3/implement" {
		t.Fatalf("expected reopened step 3 frontier to stay stable, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != "Step 3: Follow-up reopened work" {
		t.Fatalf("expected current frontier facts for step 3, got %#v", result.Facts)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), stepOneTitle) {
		t.Fatalf("expected earlier step debt warning while staying on step 3, got %#v", result.Warnings)
	}
	if len(result.NextAction) < 2 || !strings.Contains(result.NextAction[0].Description, stepOneTitle) {
		t.Fatalf("expected repair guidance for the earlier step first, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("expected explicit step-closeout review guidance to remain available, got %#v", result.NextAction)
	}
	foundLaterFrontierAction := false
	for _, action := range result.NextAction[2:] {
		if action.Command == nil && strings.Contains(action.Description, "Continue the current step") {
			foundLaterFrontierAction = true
			break
		}
	}
	if !foundLaterFrontierAction {
		t.Fatalf("expected ordinary later-frontier guidance to remain after the earlier-step reminder, got %#v", result.NextAction)
	}
}

func TestStatusReentersReviewedStepAfterFailedExplicitEarlierStepRepair(t *testing.T) {
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

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 10, 0, 0, time.FixedZone("CST", 8*60*60))
		},
	}
	start := svc.Start(mustJSONBytes(t, review.Spec{
		Step: reviewIntPtr(1),
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Repair the earlier step closeout from the later frontier."},
		},
	}))
	if !start.OK {
		t.Fatalf("expected explicit earlier-step review start, got %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 12, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSONBytes(t, review.SubmissionInput{
		Summary: "The repair still needs work.",
		Findings: []review.Finding{
			{
				Severity: "important",
				Title:    "Earlier-step contract drift",
				Details:  "The explicit repair still misses one frontier-stability assertion.",
			},
		},
	}))
	if !submit.OK {
		t.Fatalf("expected review submission, got %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 14, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	aggregate := svc.Aggregate(start.Artifacts.RoundID)
	if !aggregate.OK || aggregate.Review == nil || aggregate.Review.Decision != "changes_requested" {
		t.Fatalf("expected failed explicit earlier-step repair aggregate, got %#v", aggregate)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected failed explicit earlier-step repair to reenter step 1, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewStatus != "changes_requested" {
		t.Fatalf("expected reviewed step facts after failed explicit repair, got %#v", result.Facts)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect passive debt warnings once explicit repair has reentered step 1, got %#v", result.Warnings)
		}
	}
	if len(result.NextAction) < 2 || !strings.Contains(result.NextAction[0].Description, "Address the findings from review-001-full") {
		t.Fatalf("expected ordinary repair guidance after reentering step 1, got %#v", result.NextAction)
	}
	if result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness review start --spec <path>" {
		t.Fatalf("expected another explicit step-closeout review to remain available, got %#v", result.NextAction)
	}
}

func TestStatusReturnsToLaterFrontierAfterCleanExplicitEarlierStepRepair(t *testing.T) {
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

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 20, 0, 0, time.FixedZone("CST", 8*60*60))
		},
	}
	start := svc.Start(mustJSONBytes(t, review.Spec{
		Step: reviewIntPtr(1),
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Repair the earlier step closeout from the later frontier."},
		},
	}))
	if !start.OK {
		t.Fatalf("expected explicit earlier-step review start, got %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 22, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSONBytes(t, review.SubmissionInput{
		Summary:  "The earlier-step repair is now clean.",
		Findings: nil,
	}))
	if !submit.OK {
		t.Fatalf("expected review submission, got %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 24, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	aggregate := svc.Aggregate(start.Artifacts.RoundID)
	if !aggregate.OK || aggregate.Review == nil || aggregate.Review.Decision != "pass" {
		t.Fatalf("expected clean explicit earlier-step repair aggregate, got %#v", aggregate)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-3/implement" {
		t.Fatalf("expected clean explicit earlier-step repair to return to the later frontier, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != "Step 3: Follow-up reopened work" || result.Facts.ReviewStatus != "pass" {
		t.Fatalf("expected later-frontier facts after clean explicit repair, got %#v", result.Facts)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect repaired step debt to remain after clean explicit repair, got %#v", result.Warnings)
		}
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command != nil || !strings.Contains(result.NextAction[0].Description, "Continue the current step") {
		t.Fatalf("expected ordinary later-frontier guidance after clean explicit repair, got %#v", result.NextAction)
	}
}

func TestStatusReturnsToFinalizeReviewAfterCleanExplicitEarlierStepRepair(t *testing.T) {
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

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 11, 30, 0, 0, time.FixedZone("CST", 8*60*60))
		},
	}
	start := svc.Start(mustJSONBytes(t, review.Spec{
		Step: reviewIntPtr(1),
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Repair the earlier step closeout from finalize review."},
		},
	}))
	if !start.OK {
		t.Fatalf("expected explicit earlier-step review start from finalize review, got %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 32, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSONBytes(t, review.SubmissionInput{
		Summary:  "The finalize-context earlier-step repair is clean.",
		Findings: nil,
	}))
	if !submit.OK {
		t.Fatalf("expected review submission, got %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 11, 34, 0, 0, time.FixedZone("CST", 8*60*60))
	}
	aggregate := svc.Aggregate(start.Artifacts.RoundID)
	if !aggregate.OK || aggregate.Review == nil || aggregate.Review.Decision != "pass" {
		t.Fatalf("expected clean explicit earlier-step repair aggregate from finalize review, got %#v", aggregate)
	}

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/finalize/review" {
		t.Fatalf("expected clean explicit earlier-step repair to return to finalize review, got %#v", result.State)
	}
	if result.Facts != nil && result.Facts.CurrentStep == stepOneTitle {
		t.Fatalf("did not expect repaired step facts to stay pinned after clean finalize-context repair, got %#v", result.Facts)
	}
	for _, warning := range result.Warnings {
		if strings.Contains(warning, stepOneTitle) {
			t.Fatalf("did not expect repaired step debt to remain after clean finalize-context repair, got %#v", result.Warnings)
		}
	}
	if len(result.NextAction) == 0 || result.NextAction[0].Command == nil || *result.NextAction[0].Command != "harness review start --spec <path>" {
		t.Fatalf("expected ordinary finalize-review guidance after clean explicit repair, got %#v", result.NextAction)
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

func assertStateJSONLacksKeys(t *testing.T, root, planStem string, keys ...string) {
	t.Helper()
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state json: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("parse state json: %v", err)
	}
	requiredAbsent := []string{
		"current_node",
		"plan_path",
		"plan_stem",
		"latest_evidence",
		"latest_ci",
		"sync",
		"latest_publish",
	}
	keys = append(requiredAbsent, keys...)
	for _, key := range keys {
		if _, ok := payload[key]; ok {
			t.Fatalf("expected state.json to omit %q, got %#v", key, payload)
		}
	}
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func reviewIntPtr(value int) *int {
	return &value
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
