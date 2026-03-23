package e2e_test

import (
	"strings"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

const (
	landWorkflowPlanTitle = "Land Workflow Plan"
	landStepOneTitle      = "Prepare the archived candidate for land"
	landStepTwoTitle      = "Finish merge-ready handoff setup"
)

func TestLandWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-land-workflow.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", landWorkflowPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, landWorkflowPlanTitle, landWorkflowPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToArchivedPublishNode(t, workspace, planPath, landStepOneTitle, landStepTwoTitle)

	submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/yzhang1918/superharness/pull/99",
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

	preLandStatus := runStatus(t, workspace.Root)
	assertNode(t, preLandStatus, "execution/finalize/await_merge")

	land := support.Run(t, workspace.Root, "land", "--pr", "https://github.com/yzhang1918/superharness/pull/99", "--commit", "abc123")
	support.RequireSuccess(t, land)
	support.RequireNoStderr(t, land)
	landPayload := support.RequireJSONResult[lifecycleCommandResult](t, land)
	if !landPayload.OK || landPayload.Command != "land" {
		t.Fatalf("unexpected land payload: %#v", landPayload)
	}

	inLandStatus := runStatus(t, workspace.Root)
	assertNode(t, inLandStatus, "land")
	if inLandStatus.Facts.LandPRURL != "https://github.com/yzhang1918/superharness/pull/99" {
		t.Fatalf("expected land PR URL in status, got %#v", inLandStatus)
	}
	if len(inLandStatus.NextAction) < 2 || inLandStatus.NextAction[1].Command == nil || *inLandStatus.NextAction[1].Command != "harness land complete" {
		t.Fatalf("expected land-complete guidance in land status, got %#v", inLandStatus)
	}

	stillInLandStatus := runStatus(t, workspace.Root)
	assertNode(t, stillInLandStatus, "land")

	landComplete := support.Run(t, workspace.Root, "land", "complete")
	support.RequireSuccess(t, landComplete)
	support.RequireNoStderr(t, landComplete)
	landCompletePayload := support.RequireJSONResult[lifecycleCommandResult](t, landComplete)
	if !landCompletePayload.OK || landCompletePayload.Command != "land complete" {
		t.Fatalf("unexpected land-complete payload: %#v", landCompletePayload)
	}

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != "" || current.LastLandedPlanPath != "docs/plans/archived/2026-03-23-land-workflow.md" || current.LastLandedAt == "" {
		t.Fatalf("expected idle marker with last-landed context, got %#v", current)
	}

	postLandStatus := runStatus(t, workspace.Root)
	assertNode(t, postLandStatus, "idle")
	if postLandStatus.Artifacts.LastLandedPlanPath != "docs/plans/archived/2026-03-23-land-workflow.md" {
		t.Fatalf("expected last-landed path in idle status, got %#v", postLandStatus)
	}
}

func landWorkflowPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise merge confirmation and post-merge cleanup through the real built
binary so the workflow covers ` + "`" + `await_merge -> land -> idle` + "`" + ` as
explicit command-owned transitions.

## Scope

### In Scope

- Archive a clean candidate.
- Record the evidence needed to reach merge-ready handoff.
- Enter land cleanup and then complete it.

### Out of Scope

- Reopen after merge-ready handoff.

## Acceptance Criteria

- [ ] Merge-ready evidence allows ` + "`" + `harness land --pr ...` + "`" + ` to enter land cleanup.
- [ ] Status remains at ` + "`" + `land` + "`" + ` until ` + "`" + `harness land complete` + "`" + ` is recorded.
- [ ] ` + "`" + `harness land complete` + "`" + ` restores idle while preserving last-landed context.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare the archived candidate for land

- Done: [ ]

#### Objective

Close out the first tracked step before archive.

#### Details

NONE

#### Expected Files

- tests/e2e/land_workflow_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Finish merge-ready handoff setup

- Done: [ ]

#### Objective

Close out the final pre-land step and reach merge-ready handoff.

#### Details

Archive-ready top-level summaries are prefilled so the scenario can reach land cleanup through real commands.

#### Expected Files

- tests/e2e/land_workflow_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The scenario could assert only land completion and miss the intermediate land self-loop.
  - Mitigation: Read status during cleanup before recording land completion.

## Validation Summary

Validated merge-ready handoff and post-merge cleanup through the built binary.

## Review Summary

No unresolved blocking review findings remain in the candidate used for land cleanup.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the merge-ready candidate and land cleanup scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}
