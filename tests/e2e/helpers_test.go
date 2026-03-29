package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

type commandError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type reviewSlot struct {
	Name           string `json:"name"`
	Slot           string `json:"slot"`
	Instructions   string `json:"instructions,omitempty"`
	SubmissionPath string `json:"submission_path"`
}

type reviewDimension struct {
	Name         string `json:"name"`
	Slot         string `json:"slot"`
	Instructions string `json:"instructions"`
}

type executeStartResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Artifacts struct {
		LocalStatePath string `json:"local_state_path"`
	} `json:"artifacts"`
}

type lifecycleCommandResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		PlanStatus string `json:"plan_status"`
		Lifecycle  string `json:"lifecycle"`
		Revision   int    `json:"revision"`
	} `json:"state"`
	Artifacts struct {
		FromPlanPath    string `json:"from_plan_path"`
		ToPlanPath      string `json:"to_plan_path"`
		LocalStatePath  string `json:"local_state_path"`
		CurrentPlanPath string `json:"current_plan_path"`
	} `json:"artifacts"`
}

type reviewStartResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Artifacts struct {
		RoundID       string       `json:"round_id"`
		ManifestPath  string       `json:"manifest_path"`
		LedgerPath    string       `json:"ledger_path"`
		AggregatePath string       `json:"aggregate_path"`
		Slots         []reviewSlot `json:"slots"`
	} `json:"artifacts"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type evidenceSubmitResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Summary   string `json:"summary"`
	Artifacts struct {
		PlanPath       string `json:"plan_path"`
		LocalStatePath string `json:"local_state_path"`
		RecordID       string `json:"record_id"`
		RecordPath     string `json:"record_path"`
		Kind           string `json:"kind"`
	} `json:"artifacts"`
}

type submitResult struct {
	OK        bool   `json:"ok"`
	Command   string `json:"command"`
	Artifacts struct {
		Slot           string `json:"slot"`
		SubmissionPath string `json:"submission_path"`
		LedgerPath     string `json:"ledger_path"`
	} `json:"artifacts"`
}

type aggregateResult struct {
	OK        bool           `json:"ok"`
	Command   string         `json:"command"`
	Summary   string         `json:"summary"`
	Errors    []commandError `json:"errors"`
	Artifacts struct {
		AggregatePath  string `json:"aggregate_path"`
		LocalStatePath string `json:"local_state_path"`
	} `json:"artifacts"`
	Review struct {
		Decision            string `json:"decision"`
		NonBlockingFindings []struct {
			Severity string `json:"severity"`
			Title    string `json:"title"`
			Details  string `json:"details"`
		} `json:"non_blocking_findings"`
	} `json:"review"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type aggregateArtifact struct {
	RoundID             string `json:"round_id"`
	Kind                string `json:"kind"`
	Step                *int   `json:"step,omitempty"`
	Revision            int    `json:"revision"`
	ReviewTitle         string `json:"review_title"`
	Decision            string `json:"decision"`
	AggregatedAt        string `json:"aggregated_at"`
	NonBlockingFindings []struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Details  string `json:"details"`
	} `json:"non_blocking_findings"`
}

type reviewManifest struct {
	RoundID     string            `json:"round_id"`
	Step        *int              `json:"step,omitempty"`
	Revision    int               `json:"revision"`
	ReviewTitle string            `json:"review_title"`
	PlanPath    string            `json:"plan_path"`
	Dimensions  []reviewDimension `json:"dimensions"`
}

type reviewLedger struct {
	Slots []struct {
		Slot   string `json:"slot"`
		Status string `json:"status"`
	} `json:"slots"`
}

type reviewSubmission struct {
	RoundID   string `json:"round_id"`
	Slot      string `json:"slot"`
	Dimension string `json:"dimension"`
	Summary   string `json:"summary"`
	Findings  []struct {
		Severity string `json:"severity"`
		Title    string `json:"title"`
		Details  string `json:"details"`
	} `json:"findings"`
}

type currentPlan struct {
	PlanPath           string `json:"plan_path"`
	LastLandedPlanPath string `json:"last_landed_plan_path"`
	LastLandedAt       string `json:"last_landed_at"`
}

type statusResult struct {
	OK      bool   `json:"ok"`
	Command string `json:"command"`
	Summary string `json:"summary"`
	State   struct {
		CurrentNode string `json:"current_node"`
	} `json:"state"`
	Facts struct {
		CurrentStep   string `json:"current_step"`
		ReviewStatus  string `json:"review_status"`
		ReviewTitle   string `json:"review_title"`
		ReopenMode    string `json:"reopen_mode"`
		Revision      int    `json:"revision"`
		PublishStatus string `json:"publish_status"`
		CIStatus      string `json:"ci_status"`
		SyncStatus    string `json:"sync_status"`
		LandPRURL     string `json:"land_pr_url"`
	} `json:"facts"`
	Artifacts struct {
		ReviewRoundID      string `json:"review_round_id"`
		PublishRecordID    string `json:"publish_record_id"`
		CIRecordID         string `json:"ci_record_id"`
		SyncRecordID       string `json:"sync_record_id"`
		LastLandedPlanPath string `json:"last_landed_plan_path"`
		LastLandedAt       string `json:"last_landed_at"`
	} `json:"artifacts"`
	NextAction []struct {
		Command     *string `json:"command"`
		Description string  `json:"description"`
	} `json:"next_actions"`
}

type runState struct {
	ExecutionStartedAt string `json:"execution_started_at"`
	PlanPath           string `json:"plan_path"`
	Revision           int    `json:"revision"`
	CurrentNode        string `json:"current_node"`
	ActiveReviewRound  struct {
		RoundID    string `json:"round_id"`
		Aggregated bool   `json:"aggregated"`
		Decision   string `json:"decision"`
		Step       *int   `json:"step,omitempty"`
		Revision   int    `json:"revision"`
		Kind       string `json:"kind"`
	} `json:"active_review_round"`
}

func runStatus(t *testing.T, workdir string) statusResult {
	t.Helper()

	status := support.Run(t, workdir, "status")
	support.RequireSuccess(t, status)
	support.RequireNoStderr(t, status)
	return support.RequireJSONResult[statusResult](t, status)
}

func assertNode(t *testing.T, status statusResult, want string) {
	t.Helper()
	if status.State.CurrentNode != want {
		t.Fatalf("expected current node %q, got %#v", want, status)
	}
}

func submitReviewSlot(t *testing.T, workspace *support.Workspace, roundID string, slot reviewSlot, summary string, findings []map[string]any) {
	t.Helper()

	submissionPath := workspace.WriteJSON(t, fmt.Sprintf("tmp/%s-%s.json", roundID, slot.Slot), map[string]any{
		"summary":  summary,
		"findings": findings,
	})

	submit := support.Run(
		t,
		workspace.Root,
		"review", "submit",
		"--round", roundID,
		"--slot", slot.Slot,
		"--input", submissionPath,
	)
	support.RequireSuccess(t, submit)
	support.RequireNoStderr(t, submit)
	submitPayload := support.RequireJSONResult[submitResult](t, submit)
	if !submitPayload.OK || submitPayload.Command != "review submit" {
		t.Fatalf("unexpected review-submit payload: %#v", submitPayload)
	}
	if submitPayload.Artifacts.Slot != slot.Slot || submitPayload.Artifacts.SubmissionPath != slot.SubmissionPath {
		t.Fatalf("unexpected submit artifacts for slot %#v: %#v", slot, submitPayload)
	}
	support.RequireFileExists(t, submitPayload.Artifacts.SubmissionPath)
}

func slotMap(slots []reviewSlot) map[string]reviewSlot {
	byName := make(map[string]reviewSlot, len(slots))
	for _, slot := range slots {
		byName[slot.Name] = slot
	}
	return byName
}

func assertLedgerStatuses(t *testing.T, ledger reviewLedger, want map[string]string) {
	t.Helper()

	got := make(map[string]string, len(ledger.Slots))
	for _, slot := range ledger.Slots {
		got[slot.Slot] = slot.Status
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d ledger slots, got %#v", len(want), ledger)
	}
	for slot, status := range want {
		if got[slot] != status {
			t.Fatalf("expected ledger slot %q to be %q, got %#v", slot, status, ledger)
		}
	}
}

func trackedStepTitle(stepNumber int, stepTitle string) string {
	return fmt.Sprintf("Step %d: %s", stepNumber, stepTitle)
}

func startReviewRound(t *testing.T, workspace *support.Workspace, specRelPath string, spec map[string]any) reviewStartResult {
	t.Helper()

	specPath := workspace.WriteJSON(t, specRelPath, spec)
	start := support.Run(t, workspace.Root, "review", "start", "--spec", specPath)
	support.RequireSuccess(t, start)
	support.RequireNoStderr(t, start)
	payload := support.RequireJSONResult[reviewStartResult](t, start)
	if !payload.OK || payload.Command != "review start" {
		t.Fatalf("unexpected review-start payload: %#v", payload)
	}
	return payload
}

func aggregateReviewRound(t *testing.T, workspace *support.Workspace, roundID string) aggregateResult {
	t.Helper()

	aggregate := support.Run(t, workspace.Root, "review", "aggregate", "--round", roundID)
	support.RequireSuccess(t, aggregate)
	support.RequireNoStderr(t, aggregate)
	payload := support.RequireJSONResult[aggregateResult](t, aggregate)
	if !payload.OK || payload.Command != "review aggregate" {
		t.Fatalf("unexpected review aggregate payload: %#v", payload)
	}
	return payload
}

func runPassingDeltaReview(t *testing.T, workspace *support.Workspace, stepTitle string, stepNumber int) string {
	t.Helper()

	target := trackedStepTitle(stepNumber, stepTitle)
	startPayload := startReviewRound(t, workspace, fmt.Sprintf("tmp/step-%d-review-spec.json", stepNumber), map[string]any{
		"kind": "delta",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Check that the tracked step is ready to close out cleanly.",
			},
		},
	})
	if !strings.HasSuffix(startPayload.Artifacts.RoundID, "-delta") {
		t.Fatalf("expected delta round id shape, got %#v", startPayload)
	}
	if len(startPayload.Artifacts.Slots) != 1 {
		t.Fatalf("expected one delta review slot, got %#v", startPayload)
	}

	reviewStatus := runStatus(t, workspace.Root)
	assertNode(t, reviewStatus, fmt.Sprintf("execution/step-%d/review", stepNumber))
	if reviewStatus.Facts.CurrentStep != target || reviewStatus.Facts.ReviewStatus != "in_progress" || reviewStatus.Facts.ReviewTitle != target {
		t.Fatalf("expected active step-review facts for %q, got %#v", target, reviewStatus)
	}

	slot := startPayload.Artifacts.Slots[0]
	submitReviewSlot(t, workspace, startPayload.Artifacts.RoundID, slot, fmt.Sprintf("Step %d is ready to close out.", stepNumber), nil)

	stillInReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, stillInReviewStatus, fmt.Sprintf("execution/step-%d/review", stepNumber))
	if stillInReviewStatus.Facts.CurrentStep != target || stillInReviewStatus.Facts.ReviewStatus != "in_progress" || stillInReviewStatus.Facts.ReviewTitle != target {
		t.Fatalf("expected submission-only update to preserve active step-review facts for %q, got %#v", target, stillInReviewStatus)
	}

	aggregatePayload := aggregateReviewRound(t, workspace, startPayload.Artifacts.RoundID)
	if aggregatePayload.Review.Decision != "pass" {
		t.Fatalf("expected clean delta aggregate for %q, got %#v", stepTitle, aggregatePayload)
	}

	return startPayload.Artifacts.RoundID
}

func runPassingFinalizeReview(t *testing.T, workspace *support.Workspace) string {
	t.Helper()

	startPayload := startReviewRound(t, workspace, "tmp/finalize-passing-review-spec.json", map[string]any{
		"kind": "full",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Check that the full branch candidate is archive-ready.",
			},
		},
	})
	if len(startPayload.Artifacts.Slots) != 1 {
		t.Fatalf("expected one finalize review slot, got %#v", startPayload)
	}

	inReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, inReviewStatus, "execution/finalize/review")
	if inReviewStatus.Facts.ReviewStatus != "in_progress" || inReviewStatus.Facts.ReviewTitle != "Full branch candidate before archive" {
		t.Fatalf("expected active finalize-review facts, got %#v", inReviewStatus)
	}

	submitReviewSlot(t, workspace, startPayload.Artifacts.RoundID, startPayload.Artifacts.Slots[0], "Finalize issues were repaired.", nil)

	stillInReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, stillInReviewStatus, "execution/finalize/review")
	if stillInReviewStatus.Facts.ReviewStatus != "in_progress" || stillInReviewStatus.Facts.ReviewTitle != "Full branch candidate before archive" {
		t.Fatalf("expected submission-only update to preserve active finalize-review facts, got %#v", stillInReviewStatus)
	}

	aggregatePayload := aggregateReviewRound(t, workspace, startPayload.Artifacts.RoundID)
	if aggregatePayload.Review.Decision != "pass" {
		t.Fatalf("expected passing finalize review, got %#v", aggregatePayload)
	}
	return startPayload.Artifacts.RoundID
}

func runPassingDeltaReviewAndComplete(t *testing.T, workspace *support.Workspace, planPath, stepTitle string, stepNumber int) {
	t.Helper()

	roundID := runPassingDeltaReview(t, workspace, stepTitle, stepNumber)
	support.CompleteStep(
		t,
		planPath,
		stepNumber,
		fmt.Sprintf("Completed %q and prepared the next tracked step.", stepTitle),
		fmt.Sprintf("Clean delta review %s passed for %q before advancing.", roundID, stepTitle),
	)
}

func drivePlanToArchivedPublishNode(t *testing.T, workspace *support.Workspace, planPath string, stepTitles ...string) lifecycleCommandResult {
	t.Helper()

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	for index, stepTitle := range stepTitles {
		runPassingDeltaReviewAndComplete(t, workspace, planPath, stepTitle, index+1)
	}

	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	runPassingFinalizeReview(t, workspace)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")
	stillArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, stillArchiveStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	payload := support.RequireJSONResult[lifecycleCommandResult](t, archive)
	if !payload.OK || payload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", payload)
	}

	postArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postArchiveStatus, "execution/finalize/publish")

	return payload
}

func drivePlanToAwaitMergeNode(t *testing.T, workspace *support.Workspace, planPath string, stepTitles ...string) {
	t.Helper()

	drivePlanToArchivedPublishNode(t, workspace, planPath, stepTitles...)

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/99",
		"branch": "codex/e2e-lifecycle-handoff-coverage",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/2",
	})
	submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status":   "fresh",
		"base_ref": "main",
		"head_ref": "codex/e2e-lifecycle-handoff-coverage",
	})

	postSyncStatus := runStatus(t, workspace.Root)
	assertNode(t, postSyncStatus, "execution/finalize/await_merge")
}

func submitEvidence(t *testing.T, workspace *support.Workspace, kind, inputRelPath string, payload map[string]any) evidenceSubmitResult {
	t.Helper()

	inputPath := workspace.WriteJSON(t, inputRelPath, payload)
	result := support.Run(t, workspace.Root, "evidence", "submit", "--kind", kind, "--input", inputPath)
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)
	parsed := support.RequireJSONResult[evidenceSubmitResult](t, result)
	if !parsed.OK || parsed.Command != "evidence submit" {
		t.Fatalf("unexpected evidence-submit payload: %#v", parsed)
	}
	return parsed
}
