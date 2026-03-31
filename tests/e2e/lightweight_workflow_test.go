package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

const (
	lightweightWorkflowTitle = "Lightweight Workflow Plan"
	lightweightStepTitle     = "Update the lightweight workflow docs"
)

func TestLightweightWorkflowWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-31-lightweight-workflow.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", lightweightWorkflowTitle,
		"--timestamp", "2026-03-31T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#69",
		"--lightweight",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RequireFileExists(t, planPath)

	support.RewritePlanPreservingFrontmatter(t, planPath, lightweightWorkflowTitle, lightweightWorkflowPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")

	execute := support.Run(t, workspace.Root, "execute", "start")
	support.RequireSuccess(t, execute)
	support.RequireNoStderr(t, execute)

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q, got %#v", planRelPath, current)
	}

	initialStatus := runStatus(t, workspace.Root)
	assertNode(t, initialStatus, "execution/step-1/implement")
	if initialStatus.Facts.CurrentStep != trackedStepTitle(1, lightweightStepTitle) {
		t.Fatalf("expected current step %q, got %#v", trackedStepTitle(1, lightweightStepTitle), initialStatus)
	}

	runPassingDeltaReviewAndComplete(t, workspace, planPath, lightweightStepTitle, 1)
	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/review")

	runPassingFinalizeReview(t, workspace)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := support.RequireJSONResult[lifecycleCommandResult](t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" {
		t.Fatalf("unexpected archive payload: %#v", archivePayload)
	}
	archivedRelPath := ".local/harness/plans/archived/2026-03-31-lightweight-workflow.md"
	if archivePayload.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected archived lightweight path %q, got %#v", archivedRelPath, archivePayload)
	}

	postArchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postArchiveStatus, "execution/finalize/publish")
	if len(postArchiveStatus.NextAction) == 0 || !strings.Contains(postArchiveStatus.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance after lightweight archive, got %#v", postArchiveStatus.NextAction)
	}

	submitEvidence(t, workspace, "publish", "tmp/lightweight-publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/109",
		"branch": "codex/e2e-lightweight-workflow",
		"base":   "main",
	})
	submitEvidence(t, workspace, "ci", "tmp/lightweight-ci.json", map[string]any{
		"status":   "not_applied",
		"provider": "github-actions",
		"reason":   "docs-only lightweight candidate",
	})
	submitEvidence(t, workspace, "sync", "tmp/lightweight-sync.json", map[string]any{
		"status": "not_applied",
		"reason": "no remote sync requirement in the test workspace",
	})

	awaitMergeStatus := runStatus(t, workspace.Root)
	assertNode(t, awaitMergeStatus, "execution/finalize/await_merge")
	if !strings.Contains(awaitMergeStatus.Summary, "lightweight path") {
		t.Fatalf("expected await-merge summary to mention lightweight breadcrumb, got %#v", awaitMergeStatus)
	}
	if len(awaitMergeStatus.NextAction) == 0 || !strings.Contains(awaitMergeStatus.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected await-merge breadcrumb guidance, got %#v", awaitMergeStatus.NextAction)
	}
}

func lightweightWorkflowPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise the lightweight workflow through the built binary so a tracked active
lightweight plan remains steerable, archives to the local lightweight archive
path, and still reminds the controller to leave a repo-visible breadcrumb
before waiting for merge approval.

## Scope

### In Scope

- Create a tracked lightweight plan from harness plan template --lightweight.
- Execute and close out the one lightweight step.
- Run finalize review, archive to the local archived path, and record publish,
  CI, and sync evidence until status reaches execution/finalize/await_merge.

### Out of Scope

- Land entry and cleanup.

## Acceptance Criteria

- [ ] A tracked active lightweight plan under docs/plans/active resolves to plan before execution starts.
- [ ] Archive moves the plan into .local/harness/plans/archived/<plan-stem>.md and status surfaces breadcrumb guidance.
- [ ] Publish, CI, and sync evidence still move the lightweight candidate to execution/finalize/await_merge.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Update the lightweight workflow docs

- Done: [ ]

#### Objective

Close out one bounded lightweight step before finalize review.

#### Details

This fixture uses a tracked active lightweight plan to prove the profile
reuses the standard workflow shape while changing only archive storage and
publish handoff behavior.

#### Expected Files

- tests/e2e/lightweight_workflow_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The scenario could prove only template rendering but miss local archive or breadcrumb behavior.
  - Mitigation: Assert both the archived local path and the publish/await-merge breadcrumb guidance.

## Validation Summary

Validated the tracked-active lightweight flow through local archive and merge-ready handoff.

## Review Summary

No unresolved review findings remain in the lightweight candidate used for the E2E.

## Archive Summary

- PR: NONE
- Ready: The lightweight candidate satisfies the acceptance criteria and is ready for merge approval once the breadcrumb is updated.
- Merge Handoff: Leave the agreed repo-visible breadcrumb before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the tracked-active lightweight E2E scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}
