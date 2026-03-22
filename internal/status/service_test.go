package status_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/evidence"
	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/runstate"
	"github.com/yzhang1918/superharness/internal/status"
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
			"trigger":    "review_fix",
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
			"trigger":    "step_closeout",
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"target":  stepOneTitle,
		"trigger": "step_closeout",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if result.Facts == nil || result.Facts.ReviewStatus != "in_progress" || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewTarget != stepOneTitle || result.Facts.ReviewTrigger != "step_closeout" {
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
			"trigger":    "step_closeout",
			"aggregated": false,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"target":  "Step 1: Resolve current_node",
		"trigger": "step_closeout",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/review" {
		t.Fatalf("unexpected node: %#v", result.State)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected normalized target match to avoid warnings, got %#v", result.Warnings)
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
			"trigger":    "step_closeout",
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

func TestStatusMissingReviewTriggerStaysConservativeOnCachedStep(t *testing.T) {
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
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected missing trigger metadata to stay on the cached step, got %#v", result.State)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "trigger metadata was missing") {
		t.Fatalf("expected conservative trigger warning, got %#v", result.Warnings)
	}
}

func TestStatusMissingReviewTriggerWithTargetSkipsCacheWrite(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"target": stepOneTitle,
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected conservative fallback to the reviewed step, got %#v", result.State)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "" {
		t.Fatalf("expected unsafe fallback to skip current_node cache refresh, got %#v", state)
	}
}

func TestStatusStepReviewTargetMismatchSkipsCacheWrite(t *testing.T) {
	root := t.TempDir()
	writePlan(t, root, "docs/plans/active/2026-03-18-status-plan.md", func(content string) string {
		return completeFirstStep(content)
	})
	writeState(t, root, "2026-03-18-status-plan", map[string]any{
		"execution_started_at": "2026-03-18T10:05:00+08:00",
		"active_review_round": map[string]any{
			"round_id":   "review-001-delta",
			"kind":       "delta",
			"trigger":    "step_closeout",
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"target":  "Step 99: Unknown reviewed step",
		"trigger": "step_closeout",
	})

	result := status.Service{Workdir: root}.Read()
	if result.State.CurrentNode != "execution/step-1/implement" {
		t.Fatalf("expected target mismatch to stay conservative on the fallback step, got %#v", result.State)
	}
	if result.Facts == nil || result.Facts.CurrentStep != stepOneTitle || result.Facts.ReviewStatus != "" || result.Facts.ReviewTrigger != "" {
		t.Fatalf("expected unsafe fallback to hide structural review facts, got %#v", result.Facts)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "did not match a tracked step title") {
		t.Fatalf("expected target mismatch warning, got %#v", result.Warnings)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-status-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "" {
		t.Fatalf("expected unsafe fallback to skip current_node cache refresh, got %#v", state)
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
			"trigger":    "step_closeout",
			"aggregated": true,
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-001-delta", map[string]any{
		"target":  stepOneTitle,
		"trigger": "step_closeout",
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
			"trigger":    "step_closeout",
			"aggregated": true,
			"decision":   "changes_requested",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-002-delta", map[string]any{
		"target":  stepOneTitle,
		"trigger": "step_closeout",
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
			"trigger":    "step_closeout",
			"aggregated": true,
			"decision":   "pass",
		},
	})
	writeReviewManifest(t, root, "2026-03-18-status-plan", "review-003-delta", map[string]any{
		"target":  stepOneTitle,
		"trigger": "step_closeout",
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
			"trigger":    "pre_archive",
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
			"trigger":    "pre_archive",
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
			"trigger":    "pre_archive",
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
	if len(result.NextAction) < 2 || result.NextAction[1].Command == nil || *result.NextAction[1].Command != "harness evidence submit --kind publish --input <json>" {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
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
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/yzhang1918/superharness/pull/13"}`)); !result.OK {
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
	if result.Facts == nil || result.Facts.PRURL != "https://github.com/yzhang1918/superharness/pull/13" || result.Facts.CIStatus != "not_applied" || result.Facts.SyncStatus != "fresh" {
		t.Fatalf("unexpected facts: %#v", result.Facts)
	}
	if result.Artifacts == nil || result.Artifacts.PublishRecordID == "" || result.Artifacts.CIRecordID == "" || result.Artifacts.SyncRecordID == "" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
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
			"pr_url":     "https://github.com/yzhang1918/superharness/pull/13",
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
	if result.Facts == nil || result.Facts.PRURL != "https://github.com/yzhang1918/superharness/pull/13" || result.Facts.CIStatus != "success" || result.Facts.SyncStatus != "fresh" {
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
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/yzhang1918/superharness/pull/13"}`)); !result.OK {
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
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/yzhang1918/superharness/pull/13"}`)); !result.OK {
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
			"pr_url":     "https://github.com/yzhang1918/superharness/pull/13",
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
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Fix the CI failures") {
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
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/yzhang1918/superharness/pull/13"}`)); !result.OK {
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
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Wait for the relevant post-archive CI") {
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
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/yzhang1918/superharness/pull/13"}`)); !result.OK {
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
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Resolve merge conflicts") {
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
			"pr_url":    "https://github.com/yzhang1918/superharness/pull/99",
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

func completeFirstStep(content string) string {
	content = replaceOnce(content, "- Done: [ ]", "- Done: [x]")
	content = replaceOnce(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceOnce(content, "PENDING_STEP_REVIEW", "Reviewed.")
	return content
}

func completeAllSteps(content string, archiveReady bool) string {
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
