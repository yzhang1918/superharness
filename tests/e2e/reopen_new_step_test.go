package e2e_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	reopenNewStepPlanTitle = "Reopen New Step Plan"
	reopenStepOneTitle     = "Finish the first tracked slice"
	reopenStepTwoTitle     = "Finish the second tracked slice"
	reopenStepThreeTitle   = "Finish the original branch candidate"
	reopenStepFourTitle    = "Handle reopened follow-up work"
)

func TestReopenNewStepWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-reopen-new-step.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", reopenNewStepPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, reopenNewStepPlanTitle, reopenNewStepPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	runPassingDeltaReviewAndComplete(t, workspace, planPath, reopenStepOneTitle, 1)
	runPassingDeltaReviewAndComplete(t, workspace, planPath, reopenStepTwoTitle, 2)
	runPassingDeltaReviewAndComplete(t, workspace, planPath, reopenStepThreeTitle, 3)

	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	passingFinalizeRound := runPassingFinalizeReview(t, workspace)
	postFinalizeReview := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeReview, "execution/finalize/archive")
	if len(postFinalizeReview.NextAction) == 0 || postFinalizeReview.NextAction[0].Description == "" {
		t.Fatalf("expected archive guidance after %s, got %#v", passingFinalizeRound, postFinalizeReview)
	}

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := support.RequireJSONResult[lifecycleCommandResult](t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", archivePayload)
	}
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-reopen-new-step.md" {
		t.Fatalf("expected archived plan path in archive payload, got %#v", archivePayload)
	}

	archivedStatus := runStatus(t, workspace.Root)
	assertNode(t, archivedStatus, "execution/finalize/publish")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "new-step")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := support.RequireJSONResult[lifecycleCommandResult](t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.Revision != 2 || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	pendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, pendingNewStepStatus, "execution/finalize/fix")
	if pendingNewStepStatus.Facts.ReopenMode != "new-step" {
		t.Fatalf("expected new-step reopen cue before the new step exists, got %#v", pendingNewStepStatus)
	}
	if !strings.Contains(pendingNewStepStatus.Summary, "needs a new unfinished step") {
		t.Fatalf("expected pending new-step summary, got %#v", pendingNewStepStatus)
	}
	if len(pendingNewStepStatus.NextAction) == 0 || !strings.Contains(pendingNewStepStatus.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected explicit add-step guidance after reopen, got %#v", pendingNewStepStatus)
	}
	stillPendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, stillPendingNewStepStatus, "execution/finalize/fix")

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q after reopen, got %#v", planRelPath, current)
	}

	support.AppendStepBeforeValidationStrategy(t, planPath, reopenedStepFourBody())

	postAppendStatus := runStatus(t, workspace.Root)
	assertNode(t, postAppendStatus, "execution/step-4/implement")
	if postAppendStatus.Facts.CurrentStep != trackedStepTitle(4, reopenStepFourTitle) {
		t.Fatalf("expected newly added step 4 to become current, got %#v", postAppendStatus)
	}
	if postAppendStatus.Facts.ReopenMode != "" {
		t.Fatalf("expected consumed new-step cue to disappear once step 4 exists, got %#v", postAppendStatus)
	}

	stillImplementing := runStatus(t, workspace.Root)
	assertNode(t, stillImplementing, "execution/step-4/implement")

	passingFourthStepRound := runPassingDeltaReview(t, workspace, reopenStepFourTitle, 4)
	support.CompleteStep(
		t,
		planPath,
		4,
		"Added and completed the reopened follow-up scope as a new tracked step.",
		fmt.Sprintf("Clean delta review %s passed for the reopened step before returning to finalize review.", passingFourthStepRound),
	)

	postFourthStepStatus := runStatus(t, workspace.Root)
	assertNode(t, postFourthStepStatus, "execution/finalize/review")
}

func reopenNewStepPlanBody() string {
	return strings.TrimSpace(fmt.Sprintf(`
## Goal

Exercise the real-binary `+"`reopen --mode new-step`"+` path so the workflow
first stays in a pending finalize/new-step state and only resumes ordinary
step execution after a new unfinished step is added to the tracked plan.

## Scope

### In Scope

- Archive a clean three-step candidate.
- Reopen with `+"`new-step`"+` and assert the pending status cue.
- Add step 4 after reopen and prove it behaves like an ordinary current step.

### Out of Scope

- Publish evidence, merge approval, and land cleanup.

## Acceptance Criteria

- [ ] Reopen with `+"`new-step`"+` first resolves to finalize repair and explicitly asks for a new unfinished step.
- [ ] Once step 4 is added, status resumes at `+"`execution/step-4/implement`"+` and the reopened step can follow ordinary step review and finalize progression.

## Deferred Items

- None.

## Work Breakdown

### Step 1: %s

- Done: [ ]

#### Objective

Close out the first original tracked step.

#### Details

Keep the first three steps simple so the reopen behavior is the focus.

#### Expected Files

- tests/e2e/reopen_new_step_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: %s

- Done: [ ]

#### Objective

Close out the second original tracked step.

#### Details

NONE

#### Expected Files

- tests/e2e/reopen_new_step_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 3: %s

- Done: [ ]

#### Objective

Close out the original branch candidate before archive and reopen.

#### Details

The initial candidate should archive cleanly before the reopen step begins.

#### Expected Files

- tests/e2e/reopen_new_step_test.go

#### Validation

- Run a clean delta review before entering finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: A reopened new step could accidentally skip the pending finalize state and jump straight to implementation.
  - Mitigation: Assert both the pre-step and post-step-added nodes explicitly.

## Validation Summary

Validated the three-step candidate and the reopened step flow through the built binary.

## Review Summary

No unresolved blocking review findings remain in the archived candidate used for reopen.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the initial three-step candidate and the tracked reopen follow-up scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`, reopenStepOneTitle, reopenStepTwoTitle, reopenStepThreeTitle))
}

func reopenedStepFourBody() string {
	return strings.TrimSpace(fmt.Sprintf(`
### Step 4: %s

- Done: [ ]

#### Objective

Represent the reopened follow-up as a new unfinished tracked step.

#### Details

This step is added only after `+"`harness reopen --mode new-step`"+` has returned
the archived candidate to active execution.

#### Expected Files

- tests/e2e/reopen_new_step_test.go

#### Validation

- Run a clean delta review before marking the reopened step complete.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW
`, reopenStepFourTitle))
}
