package review_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/review"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
)

func TestStartCreatesRoundAndUpdatesState(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 123456789, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check the state and artifact contract."},
			{Name: "agent_ux", Instructions: "Check that outputs are agent-friendly."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected start success, got %#v", result)
	}
	if result.Artifacts == nil || len(result.Artifacts.Slots) != 2 {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
	if result.Artifacts.RoundID != "review-001-delta" {
		t.Fatalf("expected compact first round id, got %#v", result.Artifacts)
	}
	if _, err := os.Stat(result.Artifacts.ManifestPath); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated {
		t.Fatalf("unexpected state: %#v", state)
	}
	if state.ActiveReviewRound.Decision != "" {
		t.Fatalf("expected empty decision before aggregate, got %#v", state.ActiveReviewRound)
	}
	if state.ActiveReviewRound.Step == nil || *state.ActiveReviewRound.Step != 1 || state.ActiveReviewRound.Revision != 1 {
		t.Fatalf("expected inferred step 1 on revision 1, got %#v", state.ActiveReviewRound)
	}
}

func TestStartAcceptsExplicitEarlierStepFromLaterExecutionFrontier(t *testing.T) {
	root := t.TempDir()
	path := writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")
	markFirstPlanStepDone(t, path)

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Step: intPtr(1),
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Repair the earlier step closeout."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected explicit earlier-step start success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Step == nil || *state.ActiveReviewRound.Step != 1 {
		t.Fatalf("expected active review state to bind explicit step 1, got %#v", state)
	}

	var manifest review.Manifest
	data, err := os.ReadFile(result.Artifacts.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Step == nil || *manifest.Step != 1 {
		t.Fatalf("expected manifest to record explicit step 1, got %#v", manifest)
	}
	if manifest.ReviewTitle != "Step 1: Replace with first step title" {
		t.Fatalf("expected explicit earlier-step review title to default to step 1 title, got %#v", manifest)
	}
}

func TestStartAcceptsExplicitEarlierStepFromFinalizeContext(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-18-review-contract.md"
	writeExecutingFinalizePlan(t, root, relPath)
	if _, err := runstate.SaveState(root, "2026-03-18-review-contract", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:00:00Z",
		Revision:           1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "changes_requested",
		},
	}); err != nil {
		t.Fatalf("save finalize-fix state: %v", err)
	}
	statusResult := status.Service{Workdir: root}.Read()
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("expected fixture to resolve the real finalize-fix node before explicit repair, got %#v", statusResult.State)
	}

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 20, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Step: intPtr(1),
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Repair the earlier closeout from finalize scope."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected explicit earlier-step start success from finalize context, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Step == nil || *state.ActiveReviewRound.Step != 1 {
		t.Fatalf("expected explicit step-bound review round in finalize context, got %#v", state)
	}

	var manifest review.Manifest
	data, err := os.ReadFile(result.Artifacts.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Step == nil || *manifest.Step != 1 {
		t.Fatalf("expected manifest to stay step-bound in finalize context, got %#v", manifest)
	}
	if manifest.ReviewTitle == "Branch candidate before archive" || manifest.ReviewTitle == "Full branch candidate before archive" {
		t.Fatalf("expected explicit step binding to avoid finalize default title, got %#v", manifest)
	}
}

func TestStartRejectsDefaultFinalizeReviewWhenEarlierCloseoutDebtExists(t *testing.T) {
	root := t.TempDir()
	writeExecutingFinalizePlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 25, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check the finalize candidate."},
		},
	}))
	if result.OK {
		t.Fatalf("expected default finalize review start to reject unresolved earlier-step closeout debt, got %#v", result)
	}
	assertStartError(t, result, "spec")
	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0].Message, "Step 1: Replace with first step title") || !strings.Contains(result.Errors[0].Message, "spec.step") {
		t.Fatalf("expected explicit earlier-step repair guidance in the start error, got %#v", result.Errors)
	}
}

func TestStartAllowsDefaultFinalizeReviewWhenEarlierCloseoutDebtIsSatisfied(t *testing.T) {
	root := t.TempDir()
	path := writeExecutingFinalizePlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")
	markAllPlanStepsNoReviewNeeded(t, path)

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 30, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check the finalize candidate."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected default finalize review start success once earlier-step closeout debt is satisfied, got %#v", result)
	}
	if result.Artifacts == nil {
		t.Fatalf("expected finalize review artifacts, got %#v", result)
	}
	var manifest review.Manifest
	data, err := os.ReadFile(result.Artifacts.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Step != nil {
		t.Fatalf("expected finalize-bound review manifest after debt is cleared, got %#v", manifest)
	}
}

func TestStartAcceptsExecutionStartMilestoneWithoutLegacyExecutingLifecycle(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-03-18-review-contract.md"
	writePlainReviewPlan(t, root, relPath)
	if _, err := runstate.SaveState(root, "2026-03-18-review-contract", &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:01:00Z",
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 2, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected start success, got %#v", result)
	}
}

func TestStartIgnoresLegacyTimestampReviewDirectoriesForCompactSequence(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeArchiveReadyFinalizePlan(t, root, "docs/plans/active/"+planStem+".md")

	legacyRoundDir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", "review-20260318t010000000000000z-delta")
	if err := os.MkdirAll(legacyRoundDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy round dir: %v", err)
	}

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 30, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected start success, got %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.RoundID != "review-001-full" {
		t.Fatalf("expected first compact round id when only legacy history exists, got %#v", result.Artifacts)
	}
}

func TestStartUsesMaxExistingCompactReviewSequence(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeExecutingPlan(t, root, "docs/plans/active/"+planStem+".md")

	sparseCompactDirs := []string{
		"review-001-delta",
		"review-003-full",
		"review-20260318t010000000000000z-delta",
	}
	for _, dir := range sparseCompactDirs {
		if err := os.MkdirAll(filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", dir), 0o755); err != nil {
			t.Fatalf("mkdir review round dir %q: %v", dir, err)
		}
	}

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 45, 0, 0, time.UTC)
		},
	}

	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !result.OK {
		t.Fatalf("expected start success, got %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.RoundID != "review-004-delta" {
		t.Fatalf("expected next round after max compact sequence, got %#v", result.Artifacts)
	}
}

func TestStartRejectsInvalidSpec(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{Workdir: root}
	result := svc.Start(mustJSON(t, map[string]any{
		"kind":       "delta",
		"dimensions": []any{},
	}))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertStartError(t, result, "spec.dimensions")
}

func TestStartRejectsUnknownSchemaProperty(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	result := review.Service{Workdir: root}.Start([]byte(`{
		"kind": "delta",
		"dimensions": [
			{"name": "correctness", "instructions": "Check behavior."}
		],
		"unexpected": true
	}`))
	if result.OK {
		t.Fatalf("expected schema validation failure, got %#v", result)
	}
	assertStartError(t, result, "spec.unexpected")
}

func TestStartRejectsUnknownTopLevelSpecField(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{Workdir: root}
	result := svc.Start([]byte(`{
		"kind":"delta",
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}],
		"unexpected":true
	}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertStartError(t, result, "spec.unexpected")
}

func TestStartRejectsWrongTypeBeforeSemanticValidation(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{Workdir: root}
	result := svc.Start([]byte(`{
		"kind":1,
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}]
	}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertStartError(t, result, "spec.kind")
}

func TestStartRejectsMissingRequiredKind(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	result := review.Service{Workdir: root}.Start([]byte(`{
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}]
	}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertStartError(t, result, "spec.kind")
}

func TestSubmitStoresSubmissionAndUpdatesLedger(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 5, 0, 0, time.UTC)
	}
	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Found a targeted issue.",
		Findings: []review.Finding{
			{
				Severity:  "important",
				Title:     "Missing location preservation",
				Details:   "The submission should preserve reviewer-provided locations.",
				Locations: []string{"internal/review/service.go#L235", "schema/artifacts/review-submission.schema.json"},
			},
		},
	}))
	if !result.OK {
		t.Fatalf("expected submit success, got %#v", result)
	}
	if _, err := os.Stat(result.Artifacts.SubmissionPath); err != nil {
		t.Fatalf("submission missing: %v", err)
	}
	var submission review.Submission
	data, err := os.ReadFile(result.Artifacts.SubmissionPath)
	if err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if err := json.Unmarshal(data, &submission); err != nil {
		t.Fatalf("unmarshal submission: %v", err)
	}
	if len(submission.Findings) != 1 || len(submission.Findings[0].Locations) != 2 {
		t.Fatalf("expected persisted locations, got %#v", submission.Findings)
	}
	if len(result.NextAction) != 1 || result.NextAction[0].Description != "Report the submission receipt to the controller agent and end the reviewer thread. If the same slot later needs a narrow follow-up for the same tracked step or the same finalize review title in the same revision, the controller may reopen this reviewer through the runtime's native resume mechanism only after this submission is verified and only while the slot instructions still materially match." {
		t.Fatalf("unexpected submit next action: %#v", result.NextAction)
	}
}

func TestSubmitRejectsUnknownSchemaProperty(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", []byte(`{
		"summary": "Looks good.",
		"unexpected": true
	}`))
	if result.OK {
		t.Fatalf("expected schema validation failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.unexpected")
}

func TestSubmitRejectsUnknownSlot(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "missing", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if result.OK {
		t.Fatalf("expected submit failure, got %#v", result)
	}
	assertSubmitError(t, result, "slot")
}

func TestSubmitRejectsEmptyLocationString(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Found one issue.",
		Findings: []review.Finding{
			{
				Severity:  "important",
				Title:     "Blank location",
				Details:   "Locations should not include blank strings.",
				Locations: []string{"   "},
			},
		},
	}))
	if result.OK {
		t.Fatalf("expected submit failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.findings[0].locations[0]")
}

func TestSubmitRejectsNullLocations(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, map[string]any{
		"summary": "Found one issue.",
		"findings": []any{
			map[string]any{
				"severity":  "important",
				"title":     "Null locations are invalid",
				"details":   "The contract only allows omission or an array of strings.",
				"locations": nil,
			},
		},
	}))
	if result.OK {
		t.Fatalf("expected submit failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.findings[0].locations")
}

func TestSubmitRejectsUnknownTopLevelField(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", []byte(`{
		"summary":"Found one issue.",
		"unexpected":true
	}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.unexpected")
}

func TestSubmitRejectsWrongFindingSeverityType(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", []byte(`{
		"summary":"Found one issue.",
		"findings":[{"severity":1,"title":"Wrong type","details":"Severity must be a string."}]
	}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.findings[0].severity")
}

func TestSubmitRejectsMissingRequiredSummary(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", []byte(`{"findings":[]}`))
	if result.OK {
		t.Fatalf("expected failure, got %#v", result)
	}
	assertSubmitError(t, result, "submission.summary")
}

func TestSubmitPreservesExplicitEmptyLocationsArray(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, map[string]any{
		"summary": "Found one issue.",
		"findings": []any{
			map[string]any{
				"severity":  "important",
				"title":     "Empty locations still matter",
				"details":   "An explicit empty array should round-trip.",
				"locations": []any{},
			},
		},
	}))
	if !result.OK {
		t.Fatalf("expected submit success, got %#v", result)
	}

	data, err := os.ReadFile(result.Artifacts.SubmissionPath)
	if err != nil {
		t.Fatalf("read submission: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw submission: %v", err)
	}
	findings := raw["findings"].([]any)
	finding := findings[0].(map[string]any)
	locations, ok := finding["locations"].([]any)
	if !ok || len(locations) != 0 {
		t.Fatalf("expected explicit empty locations array, got %#v", finding["locations"])
	}
}

func TestSubmitAcceptsFindingWithoutLocations(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Found one issue.",
		Findings: []review.Finding{
			{
				Severity: "important",
				Title:    "Locations remain optional",
				Details:  "The old payload shape still works.",
			},
		},
	}))
	if !result.OK {
		t.Fatalf("expected submit success, got %#v", result)
	}

	data, err := os.ReadFile(result.Artifacts.SubmissionPath)
	if err != nil {
		t.Fatalf("read submission: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw submission: %v", err)
	}
	findings := raw["findings"].([]any)
	finding := findings[0].(map[string]any)
	if _, ok := finding["locations"]; ok {
		t.Fatalf("expected omitted locations field, got %#v", finding)
	}
}

func TestAggregateRejectsMissingSubmission(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	result := svc.Aggregate(start.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected aggregate failure, got %#v", result)
	}
	assertAggregateError(t, result, "submissions")
}

func TestAggregateDeltaPassUpdatesState(t *testing.T) {
	root := t.TempDir()
	writeExecutingPlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
	}
	result := svc.Aggregate(start.Artifacts.RoundID)
	if !result.OK || result.Review == nil || result.Review.Decision != "pass" {
		t.Fatalf("unexpected aggregate result: %#v", result)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || !state.ActiveReviewRound.Aggregated {
		t.Fatalf("expected aggregated state, got %#v", state)
	}
	if state.ActiveReviewRound.Decision != "pass" {
		t.Fatalf("expected passing decision in state, got %#v", state.ActiveReviewRound)
	}
}

func TestAggregateRejectsNonActiveRound(t *testing.T) {
	root := t.TempDir()
	writeArchiveReadyFinalizePlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	stale := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !stale.OK {
		t.Fatalf("stale round start failed: %#v", stale)
	}
	submit := svc.Submit(stale.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 5, 0, 0, time.UTC)
	}
	active := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !active.OK {
		t.Fatalf("active round start failed: %#v", active)
	}

	result := svc.Aggregate(stale.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected stale aggregate failure, got %#v", result)
	}
	assertAggregateError(t, result, "round")

	if _, err := os.Stat(filepath.Join(root, ".local", "harness", "plans", "2026-03-18-review-contract", "reviews", stale.Artifacts.RoundID, "aggregate.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no stale aggregate artifact, got %v", err)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil {
		t.Fatalf("expected active round state, got %#v", state)
	}
	if state.ActiveReviewRound.RoundID != active.Artifacts.RoundID {
		t.Fatalf("expected active round %q to remain current, got %#v", active.Artifacts.RoundID, state.ActiveReviewRound)
	}
	if state.ActiveReviewRound.Aggregated {
		t.Fatalf("expected newer active round to remain in flight, got %#v", state.ActiveReviewRound)
	}
}

func TestStartRejectsWhenReviewMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeExecutingPlan(t, root, "docs/plans/active/"+planStem+".md")
	holdReviewMutationLock(t, root, planStem)

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if result.OK {
		t.Fatalf("expected start failure while lock is held, got %#v", result)
	}
	assertStartError(t, result, "review")
}

func TestStartRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeExecutingPlan(t, root, "docs/plans/active/"+planStem+".md")
	release, err := runstate.AcquireStateMutationLock(root, planStem)
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	result := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if result.OK {
		t.Fatalf("expected start failure while state lock is held, got %#v", result)
	}
	assertStartError(t, result, "state")
}

func TestAggregateRejectsWhenReviewMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeExecutingPlan(t, root, "docs/plans/active/"+planStem+".md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	holdReviewMutationLock(t, root, planStem)

	result := svc.Aggregate(start.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected aggregate failure while lock is held, got %#v", result)
	}
	assertAggregateError(t, result, "review")
	if _, err := os.Stat(filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", start.Artifacts.RoundID, "aggregate.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no aggregate artifact while lock is held, got %v", err)
	}
}

func TestAggregateRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-18-review-contract"
	writeExecutingPlan(t, root, "docs/plans/active/"+planStem+".md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "delta",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	release, err := runstate.AcquireStateMutationLock(root, planStem)
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := svc.Aggregate(start.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected aggregate failure while state lock is held, got %#v", result)
	}
	assertAggregateError(t, result, "state")
	if _, err := os.Stat(filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", start.Artifacts.RoundID, "aggregate.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no aggregate artifact while state lock is held, got %v", err)
	}
}

func TestAggregateFullWithBlockingFindings(t *testing.T) {
	root := t.TempDir()
	writeArchiveReadyFinalizePlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, review.SubmissionInput{
		Summary: "Found a blocker.",
		Findings: []review.Finding{
			{
				Severity:  "important",
				Title:     "Missing validation",
				Details:   "The archive path is missing one required validation.",
				Locations: []string{"internal/lifecycle/service.go#L10-L18"},
			},
		},
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 12, 0, 0, time.UTC)
	}
	result := svc.Aggregate(start.Artifacts.RoundID)
	if !result.OK || result.Review == nil || result.Review.Decision != "changes_requested" {
		t.Fatalf("unexpected aggregate result: %#v", result)
	}
	if len(result.Review.BlockingFindings) != 1 {
		t.Fatalf("expected one blocking finding, got %#v", result.Review)
	}
	if got := result.Review.BlockingFindings[0].Locations; len(got) != 1 || got[0] != "internal/lifecycle/service.go#L10-L18" {
		t.Fatalf("expected aggregate to preserve locations, got %#v", result.Review.BlockingFindings[0])
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-review-contract")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Decision != "changes_requested" {
		t.Fatalf("expected failing decision in state, got %#v", state)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "Fix the blocking findings before archive") {
		t.Fatalf("unexpected next actions: %#v", result.NextAction)
	}
}

func TestAggregatePreservesExplicitEmptyLocationsArray(t *testing.T) {
	root := t.TempDir()
	writeArchiveReadyFinalizePlan(t, root, "docs/plans/active/2026-03-18-review-contract.md")

	svc := review.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check correctness."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSON(t, map[string]any{
		"summary": "Found one issue.",
		"findings": []any{
			map[string]any{
				"severity":  "important",
				"title":     "Empty locations still matter",
				"details":   "An explicit empty array should round-trip.",
				"locations": []any{},
			},
		},
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	result := svc.Aggregate(start.Artifacts.RoundID)
	if !result.OK {
		t.Fatalf("aggregate failed: %#v", result)
	}

	data, err := os.ReadFile(start.Artifacts.AggregatePath)
	if err != nil {
		t.Fatalf("read aggregate: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw aggregate: %v", err)
	}
	findings := raw["blocking_findings"].([]any)
	finding := findings[0].(map[string]any)
	locations, ok := finding["locations"].([]any)
	if !ok || len(locations) != 0 {
		t.Fatalf("expected explicit empty locations array in aggregate, got %#v", finding["locations"])
	}
}

func writeExecutingPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	path := writePlainReviewPlan(t, root, relPath)
	if _, err := runstate.SaveState(root, strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)), &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save execute-start state: %v", err)
	}
	return path
}

func writeExecutingFinalizePlan(t *testing.T, root, relPath string) string {
	t.Helper()
	path := writePlainReviewPlan(t, root, relPath)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	content := strings.ReplaceAll(string(data), "- Done: [ ]", "- Done: [x]")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write finalized plan: %v", err)
	}
	if _, err := runstate.SaveState(root, strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)), &runstate.State{
		ExecutionStartedAt: "2026-03-18T01:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save execute-start state: %v", err)
	}
	return path
}

func writeArchiveReadyFinalizePlan(t *testing.T, root, relPath string) string {
	t.Helper()
	path := writeExecutingFinalizePlan(t, root, relPath)
	markAllPlanStepsNoReviewNeeded(t, path)
	return path
}

func writePlainReviewPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Review Contract Plan",
		Timestamp:  time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return path
}

func writeReviewPlanFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
}

func markFirstPlanStepDone(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	content := strings.Replace(string(data), "- Done: [ ]", "- Done: [x]", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write updated plan: %v", err)
	}
}

func markAllPlanStepsNoReviewNeeded(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read plan: %v", err)
	}
	content := strings.ReplaceAll(
		string(data),
		"#### Review Notes\n\nPENDING_STEP_REVIEW",
		"#### Review Notes\n\nNO_STEP_REVIEW_NEEDED: Test fixture uses explicit review-complete closeout.",
	)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write updated plan: %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func holdReviewMutationLock(t *testing.T, root, planStem string) {
	t.Helper()
	lockPath := filepath.Join(root, ".local", "harness", "plans", planStem, ".review-mutation.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir lock parent: %v", err)
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open lock: %v", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		t.Fatalf("flock lock: %v", err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	})
}

func assertStartError(t *testing.T, result review.StartResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected start error for %s, got %#v", path, result.Errors)
}

func assertSubmitError(t *testing.T, result review.SubmitResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected submit error for %s, got %#v", path, result.Errors)
}

func assertAggregateError(t *testing.T, result review.AggregateResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected aggregate error for %s, got %#v", path, result.Errors)
}

func TestStartUsesReviewStartCommandIdentifierOnPlanDetectionFailure(t *testing.T) {
	root := t.TempDir()

	result := review.Service{Workdir: root}.Start(mustJSON(t, review.Spec{
		Kind: "full",
		Dimensions: []review.Dimension{
			{Name: "correctness", Instructions: "Check setup failures."},
		},
	}))
	if result.OK {
		t.Fatalf("expected start failure when no current plan exists, got %#v", result)
	}
	if result.Command != "review start" {
		t.Fatalf("expected review start command identifier, got %#v", result)
	}
}
