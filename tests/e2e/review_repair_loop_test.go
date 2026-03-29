package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	reviewRepairPlanTitle = "Review Repair Loop Plan"
	reviewRepairStepOne   = "Repair step-level review findings"
	reviewRepairStepTwo   = "Finish implementation before finalize rerun"
)

func TestReviewRepairLoopsWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-review-repair-loop.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", reviewRepairPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, reviewRepairPlanTitle, reviewRepairLoopPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	initialStatus := runStatus(t, workspace.Root)
	assertNode(t, initialStatus, "execution/step-1/implement")

	blockingStepRound := runBlockingStepReview(t, workspace, reviewRepairStepOne, 1)
	postStepFailure := runStatus(t, workspace.Root)
	assertNode(t, postStepFailure, "execution/step-1/implement")
	if postStepFailure.Facts.ReviewStatus != "changes_requested" || postStepFailure.Facts.ReviewTitle != trackedStepTitle(1, reviewRepairStepOne) {
		t.Fatalf("expected step-review failure facts after %s, got %#v", blockingStepRound, postStepFailure)
	}
	if !strings.Contains(postStepFailure.Summary, "requested changes") {
		t.Fatalf("expected step-review failure summary, got %#v", postStepFailure)
	}

	passingStepRound := runPassingDeltaReview(t, workspace, reviewRepairStepOne, 1)
	postStepRepair := runStatus(t, workspace.Root)
	assertNode(t, postStepRepair, "execution/step-1/implement")
	if postStepRepair.Facts.ReviewStatus != "pass" {
		t.Fatalf("expected repaired step-review facts after %s, got %#v", passingStepRound, postStepRepair)
	}

	support.CompleteStep(
		t,
		planPath,
		1,
		fmt.Sprintf("Repaired the step-local issue raised by %s and reran a clean delta review.", blockingStepRound),
		fmt.Sprintf("Blocking delta review %s was repaired and clean delta review %s passed before advancing.", blockingStepRound, passingStepRound),
	)

	secondStepStatus := runStatus(t, workspace.Root)
	assertNode(t, secondStepStatus, "execution/step-2/implement")

	passingSecondStepRound := runPassingDeltaReview(t, workspace, reviewRepairStepTwo, 2)
	support.CompleteStep(
		t,
		planPath,
		2,
		"Finished the remaining work needed before finalize review.",
		fmt.Sprintf("Clean delta review %s passed for %q before entering finalize review.", passingSecondStepRound, reviewRepairStepTwo),
	)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	blockingFinalizeRound := runBlockingFinalizeReview(t, workspace)
	postFinalizeFailure := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeFailure, "execution/finalize/fix")
	if postFinalizeFailure.Facts.ReviewStatus != "changes_requested" {
		t.Fatalf("expected finalize changes-requested facts after %s, got %#v", blockingFinalizeRound, postFinalizeFailure)
	}
	if !strings.Contains(postFinalizeFailure.Summary, "finalize-scope repair") {
		t.Fatalf("expected finalize-repair summary, got %#v", postFinalizeFailure)
	}

	passingFinalizeRound := runPassingFinalizeReview(t, workspace)
	postFinalizeRepair := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeRepair, "execution/finalize/archive")
	if len(postFinalizeRepair.NextAction) == 0 || postFinalizeRepair.NextAction[0].Description == "" {
		t.Fatalf("expected archive-stage guidance after %s, got %#v", passingFinalizeRound, postFinalizeRepair)
	}
}

func reviewRepairLoopPlanBody() string {
	return strings.TrimSpace(fmt.Sprintf(`
## Goal

Exercise step-review and finalize-review repair loops through the real built
binary so status, review artifacts, and derived transitions stay aligned when
review first fails and later passes.

## Scope

### In Scope

- Fail and rerun one step-closeout review.
- Fail and rerun one finalize review.

### Out of Scope

- Archive, publish, and land follow-up.

## Acceptance Criteria

- [ ] Step review findings can send the workflow back to implement and a later clean review can still advance the plan.
- [ ] Finalize review findings can send the workflow to finalize repair and a later clean review can still return the plan to archive closeout.

## Deferred Items

- None.

## Work Breakdown

### Step 1: %s

- Done: [ ]

#### Objective

Drive a step-closeout review failure and later clean rerun.

#### Details

Keep the first tracked step focused on the step-review repair loop itself.

#### Expected Files

- tests/e2e/review_repair_loop_test.go

#### Validation

- Run a blocking delta review, repair it, then rerun a clean delta review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: %s

- Done: [ ]

#### Objective

Finish the remaining work needed to enter finalize review and then exercise the finalize repair loop.

#### Details

Use a clean step review before entering a blocking finalize review.

#### Expected Files

- tests/e2e/review_repair_loop_test.go

#### Validation

- Run a clean delta review before finalize review.
- Prove finalize review can fail, return to repair, and later pass.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: Review-loop assertions could accidentally skip the failure path and only prove the final success case.
  - Mitigation: Assert the intermediate changes-requested status before the clean rerun.

## Validation Summary

PENDING_UNTIL_ARCHIVE

## Review Summary

PENDING_UNTIL_ARCHIVE

## Archive Summary

PENDING_UNTIL_ARCHIVE

## Outcome Summary

### Delivered

PENDING_UNTIL_ARCHIVE

### Not Delivered

PENDING_UNTIL_ARCHIVE

### Follow-Up Issues

NONE
`, reviewRepairStepOne, reviewRepairStepTwo))
}

func runBlockingStepReview(t *testing.T, workspace *support.Workspace, stepTitle string, stepNumber int) string {
	t.Helper()

	aggregatePayload := runSingleSlotReviewWithFindings(
		t,
		workspace,
		fmt.Sprintf("tmp/step-%d-blocking-review-spec.json", stepNumber),
		map[string]any{
			"kind": "delta",
			"dimensions": []map[string]any{
				{
					"name":         "correctness",
					"instructions": "Check that the tracked step is ready to close out cleanly.",
				},
			},
		},
		"Found a step-level blocker.",
		[]map[string]any{
			{
				"severity": "important",
				"title":    "Repair needed before closeout",
				"details":  "The tracked step still has one issue that should block closeout.",
			},
		},
	)
	if aggregatePayload.Review.Decision != "changes_requested" {
		t.Fatalf("expected blocking step review to request changes, got %#v", aggregatePayload)
	}
	return aggregatePayload.Artifacts.AggregatePath
}

func runBlockingFinalizeReview(t *testing.T, workspace *support.Workspace) string {
	t.Helper()

	aggregatePayload := runSingleSlotReviewWithFindings(
		t,
		workspace,
		"tmp/finalize-blocking-review-spec.json",
		map[string]any{
			"kind": "full",
			"dimensions": []map[string]any{
				{
					"name":         "correctness",
					"instructions": "Check that the full branch candidate is archive-ready.",
				},
			},
		},
		"Found a finalize blocker.",
		[]map[string]any{
			{
				"severity": "important",
				"title":    "Finalize follow-up required",
				"details":  "The branch still needs one archive-scope fix before closeout.",
			},
		},
	)
	if aggregatePayload.Review.Decision != "changes_requested" {
		t.Fatalf("expected blocking finalize review to request changes, got %#v", aggregatePayload)
	}
	return aggregatePayload.Artifacts.AggregatePath
}

func runSingleSlotReviewWithFindings(t *testing.T, workspace *support.Workspace, specRelPath string, spec map[string]any, summary string, findings []map[string]any) aggregateResult {
	t.Helper()

	startPayload := startReviewRound(t, workspace, specRelPath, spec)
	if len(startPayload.Artifacts.Slots) != 1 {
		t.Fatalf("expected one review slot, got %#v", startPayload)
	}
	submitReviewSlot(t, workspace, startPayload.Artifacts.RoundID, startPayload.Artifacts.Slots[0], summary, findings)
	return aggregateReviewRound(t, workspace, startPayload.Artifacts.RoundID)
}
