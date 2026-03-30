package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
)

const (
	internalStepOneTitle = "Step 1: Replace with first step title"
	internalStepTwoTitle = "Step 2: Replace with second step title"
)

func TestLoadSatisfiedStepCloseoutTargetsUsesActiveReviewContextForUnreadableCurrentRound(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-status-plan"
	doc := &plan.Document{
		Steps: []plan.DocumentStep{
			{Title: internalStepOneTitle},
			{Title: internalStepTwoTitle},
		},
	}

	writeHistoricalReviewJSON(t, root, planStem, "review-001-delta", "manifest.json", map[string]any{
		"review_title": internalStepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeHistoricalReviewJSON(t, root, planStem, "review-001-delta", "aggregate.json", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}
	writeHistoricalReviewJSON(t, root, planStem, "review-002-delta", "aggregate.json", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	reviewCtx := &reviewContext{
		RoundID:         "review-002-delta",
		Trigger:         "step_closeout",
		TargetStepIndex: 0,
	}
	satisfied, warnings := loadSatisfiedStepCloseoutTargets(root, planStem, doc, reviewCtx)
	if satisfied[normalizeReviewTitle(internalStepOneTitle)] {
		t.Fatalf("expected active reviewCtx fallback to keep step 1 unsatisfied, got %#v", satisfied)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected unreadable manifest warning, got none")
	}
}

func TestLoadSatisfiedStepCloseoutTargetsUsesActiveInFlightReviewContextForUnreadableCurrentRound(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-status-plan"
	doc := &plan.Document{
		Steps: []plan.DocumentStep{
			{Title: internalStepOneTitle},
			{Title: internalStepTwoTitle},
		},
	}

	writeHistoricalReviewJSON(t, root, planStem, "review-001-delta", "manifest.json", map[string]any{
		"review_title": internalStepOneTitle,
		"step":         1,
		"revision":     1,
	})
	writeHistoricalReviewJSON(t, root, planStem, "review-001-delta", "aggregate.json", map[string]any{
		"decision": "pass",
	})

	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", "review-002-delta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir unreadable manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write unreadable manifest: %v", err)
	}

	reviewCtx := &reviewContext{
		RoundID:         "review-002-delta",
		Trigger:         "step_closeout",
		TargetStepIndex: 0,
		InFlight:        true,
	}
	satisfied, warnings := loadSatisfiedStepCloseoutTargets(root, planStem, doc, reviewCtx)
	if satisfied[normalizeReviewTitle(internalStepOneTitle)] {
		t.Fatalf("expected active in-flight reviewCtx fallback to keep step 1 unsatisfied, got %#v", satisfied)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected unreadable manifest warning, got none")
	}
}

func writeHistoricalReviewJSON(t *testing.T, root, planStem, roundID, name string, payload any) {
	t.Helper()

	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
