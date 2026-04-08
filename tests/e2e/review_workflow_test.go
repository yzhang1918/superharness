package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	reviewWorkflowTitle = "Review Workflow Plan"
	stepOneTitle        = "Build repo-level test support"
	stepTwoTitle        = "Validate multi-slot review workflow"
)

func TestReviewWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-22-review-workflow.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", reviewWorkflowTitle,
		"--timestamp", "2026-03-22T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RequireFileExists(t, planPath)

	// Smoke covers the default template body. This E2E rewrites the generated
	// file into a deterministic fixture so workflow assertions follow the
	// state-model contract instead of incidental template copy.
	support.RewritePlanPreservingFrontmatter(t, planPath, reviewWorkflowTitle, reviewWorkflowPlanBody())
	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")
	stillPlannedStatus := runStatus(t, workspace.Root)
	assertNode(t, stillPlannedStatus, "plan")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)
	executePayload := support.RequireJSONResult[executeStartResult](t, execute)
	if !executePayload.OK || executePayload.Command != "execute start" {
		t.Fatalf("unexpected execute-start payload: %#v", executePayload)
	}
	support.RequireFileExists(t, executePayload.Artifacts.LocalStatePath)

	currentPlanPath := workspace.Path(".local/harness/current-plan.json")
	support.RequireFileExists(t, currentPlanPath)
	current := support.ReadJSONFile[currentPlan](t, currentPlanPath)
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q, got %#v", planRelPath, current)
	}

	initialStatus := runStatus(t, workspace.Root)
	assertNode(t, initialStatus, "execution/step-1/implement")
	if initialStatus.Facts.CurrentStep != trackedStepTitle(1, stepOneTitle) {
		t.Fatalf("expected current step %q after execute start, got %#v", trackedStepTitle(1, stepOneTitle), initialStatus)
	}

	stepOneRound := runPassingDeltaReview(t, workspace, stepOneTitle, 1)
	postStepOneReview := runStatus(t, workspace.Root)
	assertNode(t, postStepOneReview, "execution/step-1/implement")
	if postStepOneReview.Facts.ReviewStatus != "pass" || postStepOneReview.Facts.ReviewTitle != trackedStepTitle(1, stepOneTitle) {
		t.Fatalf("expected clean step-one review facts after aggregate, got %#v", postStepOneReview)
	}
	support.CompleteStep(
		t,
		planPath,
		1,
		"Built the repo-level binary/workspace/assertion helpers used by smoke and E2E coverage.",
		fmt.Sprintf("Clean delta review %s passed for %q before advancing to step 2.", stepOneRound, stepOneTitle),
	)

	secondStepStatus := runStatus(t, workspace.Root)
	assertNode(t, secondStepStatus, "execution/step-2/implement")
	if secondStepStatus.Facts.CurrentStep != trackedStepTitle(2, stepTwoTitle) {
		t.Fatalf("expected current step %q after step-one closeout, got %#v", trackedStepTitle(2, stepTwoTitle), secondStepStatus)
	}

	stepTwoRound := runPassingDeltaReview(t, workspace, stepTwoTitle, 2)
	postStepTwoReview := runStatus(t, workspace.Root)
	assertNode(t, postStepTwoReview, "execution/step-2/implement")
	if postStepTwoReview.Facts.ReviewStatus != "pass" || postStepTwoReview.Facts.ReviewTitle != trackedStepTitle(2, stepTwoTitle) {
		t.Fatalf("expected clean step-two review facts after aggregate, got %#v", postStepTwoReview)
	}

	support.CheckAllAcceptanceCriteria(t, planPath)
	support.CompleteStep(
		t,
		planPath,
		2,
		"Exercised finalize review orchestration, submission persistence, and aggregate gating across multiple slots.",
		fmt.Sprintf("Clean delta review %s passed for %q before entering finalize review.", stepTwoRound, stepTwoTitle),
	)

	preReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, preReviewStatus, "execution/finalize/review")
	if preReviewStatus.Summary != "Plan has finished its tracked steps and needs finalize review before archive." {
		t.Fatalf("expected finalize-review preflight summary, got %#v", preReviewStatus)
	}
	if preReviewStatus.Facts.ReviewStatus != "" || preReviewStatus.Facts.ReviewTitle != "" {
		t.Fatalf("expected finalize preflight to clear prior step-review facts, got %#v", preReviewStatus)
	}
	if preReviewStatus.Artifacts.ReviewRoundID != "" {
		t.Fatalf("expected finalize preflight to clear prior step-review artifacts, got %#v", preReviewStatus)
	}
	if len(preReviewStatus.NextAction) == 0 || preReviewStatus.NextAction[0].Command == nil || *preReviewStatus.NextAction[0].Command != "harness review start --spec <path>" {
		t.Fatalf("expected finalize-review next action guidance, got %#v", preReviewStatus)
	}

	invalidSpecPath := workspace.WriteJSON(t, "tmp/review-invalid-spec.json", map[string]any{
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Check that schema validation runs from the built binary.",
			},
		},
	})
	invalidStart := support.Run(t, workspace.Root, "review", "start", "--spec", invalidSpecPath)
	support.RequireExitCode(t, invalidStart, 1)
	support.RequireNoStderr(t, invalidStart)
	invalidStartPayload := support.RequireJSONResult[struct {
		OK      bool           `json:"ok"`
		Command string         `json:"command"`
		Summary string         `json:"summary"`
		Errors  []commandError `json:"errors"`
	}](t, invalidStart)
	if invalidStartPayload.OK || invalidStartPayload.Command != "review start" {
		t.Fatalf("expected failed review-start payload for invalid schema input, got %#v", invalidStartPayload)
	}
	if invalidStartPayload.Summary != "Review spec is invalid." {
		t.Fatalf("expected schema-invalid summary, got %#v", invalidStartPayload)
	}
	if len(invalidStartPayload.Errors) != 1 || invalidStartPayload.Errors[0].Path != "spec.kind" {
		t.Fatalf("expected schema-invalid kind error from built binary, got %#v", invalidStartPayload.Errors)
	}

	specPath := workspace.WriteJSON(t, "tmp/review-spec.json", map[string]any{
		"kind": "full",
		"dimensions": []map[string]any{
			{
				"name":         "correctness",
				"instructions": "Check that the repo-level binary workflow is wired correctly.",
			},
			{
				"name":         "tests",
				"instructions": "Check that aggregate waits for every expected reviewer submission.",
			},
		},
	})

	start := support.Run(t, workspace.Root, "review", "start", "--spec", specPath)
	support.RequireSuccess(t, start)
	support.RequireNoStderr(t, start)
	startPayload := support.RequireJSONResult[reviewStartResult](t, start)
	if !startPayload.OK || startPayload.Command != "review start" {
		t.Fatalf("unexpected review-start payload: %#v", startPayload)
	}
	if !strings.HasPrefix(startPayload.Artifacts.RoundID, "review-") || !strings.HasSuffix(startPayload.Artifacts.RoundID, "-full") {
		t.Fatalf("expected full review round id shape, got %#v", startPayload)
	}
	if len(startPayload.Artifacts.Slots) != 2 {
		t.Fatalf("expected two review slots for finalize review, got %#v", startPayload)
	}
	support.RequireFileExists(t, startPayload.Artifacts.ManifestPath)
	support.RequireFileExists(t, startPayload.Artifacts.LedgerPath)
	if len(startPayload.NextAction) < 2 || startPayload.NextAction[1].Command == nil || *startPayload.NextAction[1].Command != "harness review aggregate --round "+startPayload.Artifacts.RoundID {
		t.Fatalf("expected review-start next actions to point at aggregate, got %#v", startPayload)
	}

	inReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, inReviewStatus, "execution/finalize/review")
	if inReviewStatus.Summary != "Plan is in finalize review and waiting for the active review round to be aggregated." {
		t.Fatalf("expected finalize-review summary to reflect active round, got %#v", inReviewStatus)
	}
	if inReviewStatus.Facts.ReviewStatus != "in_progress" {
		t.Fatalf("expected active finalize review status, got %#v", inReviewStatus)
	}
	if inReviewStatus.Artifacts.ReviewRoundID != startPayload.Artifacts.RoundID {
		t.Fatalf("expected active review round %q in status artifacts, got %#v", startPayload.Artifacts.RoundID, inReviewStatus)
	}
	if len(inReviewStatus.NextAction) == 0 || inReviewStatus.NextAction[0].Command == nil || *inReviewStatus.NextAction[0].Command != "harness review aggregate --round "+startPayload.Artifacts.RoundID {
		t.Fatalf("expected status guidance to point at aggregate for the active round, got %#v", inReviewStatus)
	}

	slots := slotMap(startPayload.Artifacts.Slots)
	correctnessSlot, ok := slots["correctness"]
	if !ok {
		t.Fatalf("missing correctness slot in %#v", startPayload.Artifacts.Slots)
	}
	if correctnessSlot.Instructions != "Check that the repo-level binary workflow is wired correctly." {
		t.Fatalf("expected correctness instructions in review-start receipt, got %#v", correctnessSlot)
	}
	testsSlot, ok := slots["tests"]
	if !ok {
		t.Fatalf("missing tests slot in %#v", startPayload.Artifacts.Slots)
	}
	if testsSlot.Instructions != "Check that aggregate waits for every expected reviewer submission." {
		t.Fatalf("expected tests instructions in review-start receipt, got %#v", testsSlot)
	}

	preSubmitLedger := support.ReadJSONFile[reviewLedger](t, startPayload.Artifacts.LedgerPath)
	assertLedgerStatuses(t, preSubmitLedger, map[string]string{
		correctnessSlot.Slot: "pending",
		testsSlot.Slot:       "pending",
	})

	submitReviewSlot(t, workspace, startPayload.Artifacts.RoundID, correctnessSlot, "Core workflow artifacts look correct.", nil)

	postFirstSubmitLedger := support.ReadJSONFile[reviewLedger](t, startPayload.Artifacts.LedgerPath)
	assertLedgerStatuses(t, postFirstSubmitLedger, map[string]string{
		correctnessSlot.Slot: "submitted",
		testsSlot.Slot:       "pending",
	})

	blockedAggregate := support.Run(t, workspace.Root, "review", "aggregate", "--round", startPayload.Artifacts.RoundID)
	support.RequireExitCode(t, blockedAggregate, 1)
	support.RequireNoStderr(t, blockedAggregate)
	blockedAggregatePayload := support.RequireJSONResult[aggregateResult](t, blockedAggregate)
	if blockedAggregatePayload.OK || blockedAggregatePayload.Command != "review aggregate" {
		t.Fatalf("expected failed aggregate payload, got %#v", blockedAggregatePayload)
	}
	if blockedAggregatePayload.Summary != "Review round is missing required submissions." {
		t.Fatalf("expected missing-submission summary, got %#v", blockedAggregatePayload)
	}
	if len(blockedAggregatePayload.Errors) != 1 || blockedAggregatePayload.Errors[0].Path != "submissions" || !strings.Contains(blockedAggregatePayload.Errors[0].Message, testsSlot.Slot) {
		t.Fatalf("expected missing tests-slot error, got %#v", blockedAggregatePayload.Errors)
	}
	support.RequireFileMissing(t, startPayload.Artifacts.AggregatePath)

	stillInReviewStatus := runStatus(t, workspace.Root)
	assertNode(t, stillInReviewStatus, "execution/finalize/review")
	if stillInReviewStatus.Summary != "Plan is in finalize review and waiting for the active review round to be aggregated." {
		t.Fatalf("expected failed aggregate to preserve active finalize-review summary, got %#v", stillInReviewStatus)
	}
	if stillInReviewStatus.Facts.ReviewStatus != "in_progress" {
		t.Fatalf("expected failed aggregate to preserve active review status, got %#v", stillInReviewStatus)
	}
	if stillInReviewStatus.Artifacts.ReviewRoundID != startPayload.Artifacts.RoundID {
		t.Fatalf("expected failed aggregate to preserve active review round %q, got %#v", startPayload.Artifacts.RoundID, stillInReviewStatus)
	}
	if len(stillInReviewStatus.NextAction) == 0 || stillInReviewStatus.NextAction[0].Command == nil || *stillInReviewStatus.NextAction[0].Command != "harness review aggregate --round "+startPayload.Artifacts.RoundID {
		t.Fatalf("expected failed aggregate to keep aggregate guidance for the active round, got %#v", stillInReviewStatus)
	}

	submitReviewSlot(t, workspace, startPayload.Artifacts.RoundID, testsSlot, "Aggregate gating waited for every reviewer slot.", []map[string]any{
		{
			"severity": "minor",
			"title":    "Review path exercised across multiple slots",
			"details":  "This E2E intentionally records one non-blocking finding so the full aggregate preserves reviewer output while still passing.",
		},
	})

	postSubmitLedger := support.ReadJSONFile[reviewLedger](t, startPayload.Artifacts.LedgerPath)
	assertLedgerStatuses(t, postSubmitLedger, map[string]string{
		correctnessSlot.Slot: "submitted",
		testsSlot.Slot:       "submitted",
	})

	aggregate := support.Run(t, workspace.Root, "review", "aggregate", "--round", startPayload.Artifacts.RoundID)
	support.RequireSuccess(t, aggregate)
	support.RequireNoStderr(t, aggregate)
	aggregatePayload := support.RequireJSONResult[aggregateResult](t, aggregate)
	if !aggregatePayload.OK || aggregatePayload.Command != "review aggregate" {
		t.Fatalf("unexpected review-aggregate payload: %#v", aggregatePayload)
	}
	if aggregatePayload.Review.Decision != "pass" {
		t.Fatalf("expected passing aggregate decision, got %#v", aggregatePayload)
	}
	if len(aggregatePayload.Review.NonBlockingFindings) != 1 || aggregatePayload.Review.NonBlockingFindings[0].Severity != "minor" {
		t.Fatalf("expected one non-blocking finding in aggregate result, got %#v", aggregatePayload.Review)
	}
	support.RequireFileExists(t, aggregatePayload.Artifacts.AggregatePath)
	if aggregatePayload.Artifacts.AggregatePath != startPayload.Artifacts.AggregatePath {
		t.Fatalf("expected aggregate to reuse review-start aggregate path, got start=%q aggregate=%q", startPayload.Artifacts.AggregatePath, aggregatePayload.Artifacts.AggregatePath)
	}

	manifest := support.ReadJSONFile[reviewManifest](t, startPayload.Artifacts.ManifestPath)
	if manifest.RoundID != startPayload.Artifacts.RoundID || manifest.PlanPath != planRelPath {
		t.Fatalf("unexpected manifest: %#v", manifest)
	}
	if len(manifest.Dimensions) != 2 {
		t.Fatalf("expected two persisted dimensions, got %#v", manifest)
	}
	manifestInstructions := map[string]string{}
	for _, dimension := range manifest.Dimensions {
		manifestInstructions[dimension.Name] = dimension.Instructions
	}
	if manifestInstructions["correctness"] != "Check that the repo-level binary workflow is wired correctly." ||
		manifestInstructions["tests"] != "Check that aggregate waits for every expected reviewer submission." {
		t.Fatalf("expected persisted manifest instructions, got %#v", manifest)
	}

	correctnessSubmission := support.ReadJSONFile[reviewSubmission](t, correctnessSlot.SubmissionPath)
	if correctnessSubmission.RoundID != startPayload.Artifacts.RoundID || correctnessSubmission.Slot != correctnessSlot.Slot || correctnessSubmission.Dimension != correctnessSlot.Name {
		t.Fatalf("unexpected correctness submission: %#v", correctnessSubmission)
	}
	if correctnessSubmission.Summary != "Core workflow artifacts look correct." {
		t.Fatalf("expected persisted correctness summary, got %#v", correctnessSubmission)
	}
	if len(correctnessSubmission.Findings) != 0 {
		t.Fatalf("expected correctness submission without findings, got %#v", correctnessSubmission)
	}

	testsSubmission := support.ReadJSONFile[reviewSubmission](t, testsSlot.SubmissionPath)
	if testsSubmission.RoundID != startPayload.Artifacts.RoundID || testsSubmission.Slot != testsSlot.Slot || testsSubmission.Dimension != testsSlot.Name {
		t.Fatalf("unexpected tests submission: %#v", testsSubmission)
	}
	if testsSubmission.Summary != "Aggregate gating waited for every reviewer slot." {
		t.Fatalf("expected persisted tests summary, got %#v", testsSubmission)
	}
	if len(testsSubmission.Findings) != 1 || testsSubmission.Findings[0].Title != "Review path exercised across multiple slots" {
		t.Fatalf("expected persisted multi-slot finding in tests submission, got %#v", testsSubmission)
	}

	aggregateArtifact := support.ReadJSONFile[aggregateArtifact](t, aggregatePayload.Artifacts.AggregatePath)
	if aggregateArtifact.RoundID != startPayload.Artifacts.RoundID || aggregateArtifact.Kind != "full" {
		t.Fatalf("unexpected aggregate artifact: %#v", aggregateArtifact)
	}
	if aggregateArtifact.ReviewTitle != "Full branch candidate before archive" || aggregateArtifact.Decision != "pass" || aggregateArtifact.AggregatedAt == "" {
		t.Fatalf("unexpected aggregate artifact contents: %#v", aggregateArtifact)
	}
	if len(aggregateArtifact.NonBlockingFindings) != 1 || aggregateArtifact.NonBlockingFindings[0].Title != "Review path exercised across multiple slots" {
		t.Fatalf("expected persisted non-blocking finding in aggregate artifact, got %#v", aggregateArtifact)
	}

	postAggregateLedger := support.ReadJSONFile[reviewLedger](t, startPayload.Artifacts.LedgerPath)
	assertLedgerStatuses(t, postAggregateLedger, map[string]string{
		correctnessSlot.Slot: "submitted",
		testsSlot.Slot:       "submitted",
	})

	state := support.ReadJSONFile[runState](t, aggregatePayload.Artifacts.LocalStatePath)
	if state.ExecutionStartedAt == "" || state.PlanPath != planRelPath {
		t.Fatalf("unexpected runstate: %#v", state)
	}
	if state.ActiveReviewRound.RoundID != startPayload.Artifacts.RoundID {
		t.Fatalf("expected active review round %q, got %#v", startPayload.Artifacts.RoundID, state)
	}
	if !state.ActiveReviewRound.Aggregated || state.ActiveReviewRound.Decision != "pass" {
		t.Fatalf("unexpected aggregated review state: %#v", state)
	}

	postAggregateStatus := runStatus(t, workspace.Root)
	assertNode(t, postAggregateStatus, "execution/finalize/archive")
	if len(postAggregateStatus.NextAction) == 0 || postAggregateStatus.NextAction[0].Description == "" {
		t.Fatalf("expected archive-stage resume guidance after clean review, got %#v", postAggregateStatus)
	}
}

func reviewWorkflowPlanBody() string {
	return strings.TrimSpace(fmt.Sprintf(`
## Goal

Exercise repo-level review orchestration with deterministic tracked-plan
content so the workflow assertions follow the state-model contract rather than
the packaged template copy.

## Scope

### In Scope

- Drive the built harness binary through step review and finalize review.
- Assert durable review artifacts, state transitions, and aggregate gating.

### Out of Scope

- Archive, publish, and land follow-up.

## Acceptance Criteria

- [ ] Step review passes before the tracked plan advances to the next step.
- [ ] Finalize review waits for every expected reviewer submission before it can pass.

## Deferred Items

- None.

## Work Breakdown

### Step 1: %s

- Done: [ ]

#### Objective

Prepare repo-level helper coverage so the workflow can use the real built
binary in a temporary workspace.

#### Details

Keep the fixture deterministic and scoped to repo-level test support.

#### Expected Files

- tests/support/*

#### Validation

- Run a delta review before advancing beyond step 1.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: %s

- Done: [ ]

#### Objective

Exercise a multi-slot finalize review that proves aggregate gating and durable
artifacts.

#### Details

Use structured tracked-plan updates and review artifacts rather than brittle
template-string rewrites.

#### Expected Files

- tests/e2e/review_workflow_test.go

#### Validation

- Run a delta review before entering finalize review.
- Prove a full review refuses to aggregate while a slot is still missing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level smoke and E2E coverage with the built binary.

## Risks

- Risk: Workflow assertions could accidentally depend on incidental template wording.
  - Mitigation: Rewrite the generated file into a deterministic fixture before driving the workflow.

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
`, stepOneTitle, stepTwoTitle))
}
