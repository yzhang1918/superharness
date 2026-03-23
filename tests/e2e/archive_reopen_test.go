package e2e_test

import (
	"strings"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

const (
	archiveReopenPlanTitle = "Archive Reopen Finalize Fix Plan"
	archiveReopenStepOne   = "Prepare the archived candidate"
	archiveReopenStepTwo   = "Finish the original branch candidate"
)

func TestArchiveReopenFinalizeFixWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-archive-reopen-finalize-fix.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", archiveReopenPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, archiveReopenPlanTitle, archiveReopenPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	archivePayload := drivePlanToArchivedPublishNode(t, workspace, planPath, archiveReopenStepOne, archiveReopenStepTwo)
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-archive-reopen-finalize-fix.md" {
		t.Fatalf("expected archived path in archive payload, got %#v", archivePayload)
	}

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := support.RequireJSONResult[lifecycleCommandResult](t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.Revision != 2 || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected finalize-fix reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	postReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, postReopenStatus, "execution/finalize/fix")
	if postReopenStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen cue after reopen, got %#v", postReopenStatus)
	}
	if !strings.Contains(postReopenStatus.Summary, "finalize-scope repair") {
		t.Fatalf("expected finalize-fix summary after reopen, got %#v", postReopenStatus)
	}

	current := support.ReadJSONFile[currentPlan](t, workspace.Path(".local/harness/current-plan.json"))
	if current.PlanPath != planRelPath {
		t.Fatalf("expected current plan pointer %q after finalize-fix reopen, got %#v", planRelPath, current)
	}
}

func archiveReopenPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise archive plus ` + "`" + `reopen --mode finalize-fix` + "`" + ` through the
real built binary so the archived publish handoff can return to active
finalize repair without adding a new step.

## Scope

### In Scope

- Archive a clean candidate.
- Reopen that archived candidate with ` + "`" + `finalize-fix` + "`" + `.
- Assert the active path, revision bump, and finalize-fix status cue.

### Out of Scope

- Publish evidence, merge approval, and land cleanup.

## Acceptance Criteria

- [ ] Archive moves the candidate to the archived plan path and status resolves to execution/finalize/publish.
- [ ] ` + "`" + `reopen --mode finalize-fix` + "`" + ` restores the active tracked plan path and bumps the revision.
- [ ] Status resolves to execution/finalize/fix with finalize-scope repair guidance after reopen.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare the archived candidate

- Done: [ ]

#### Objective

Close out the first tracked step before archive.

#### Details

NONE

#### Expected Files

- tests/e2e/archive_reopen_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Finish the original branch candidate

- Done: [ ]

#### Objective

Close out the final pre-archive step and reach publish handoff.

#### Details

Archive-ready top-level summaries are prefilled so the scenario can reach archive without extra tracked-plan editing.

#### Expected Files

- tests/e2e/archive_reopen_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The scenario could prove archive success without proving the reopen rollback path.
  - Mitigation: Assert the archived path first, then the active path and finalize-fix status after reopen.

## Validation Summary

Validated archive and finalize-fix reopen through the built binary.

## Review Summary

No unresolved blocking review findings remain in the candidate used for archive and reopen.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the archive and finalize-fix reopen scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}
