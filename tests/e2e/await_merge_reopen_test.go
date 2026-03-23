package e2e_test

import (
	"strings"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

const (
	awaitMergeFinalizeFixPlanTitle = "Await Merge Reopen Finalize Fix Plan"
	awaitMergeFinalizeFixStepOne   = "Prepare the merge-ready candidate"
	awaitMergeFinalizeFixStepTwo   = "Finish the archived handoff before reopen"

	awaitMergeNewStepPlanTitle = "Await Merge Reopen New Step Plan"
	awaitMergeNewStepStepOne   = "Prepare the merge-ready candidate"
	awaitMergeNewStepStepTwo   = "Finish the archived handoff before reopen"
	awaitMergeNewStepStepThree = "Handle merge-ready follow-up as a new step"
)

func TestAwaitMergeReopenFinalizeFixWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-await-merge-reopen-finalize-fix.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", awaitMergeFinalizeFixPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, awaitMergeFinalizeFixPlanTitle, awaitMergeReopenFinalizeFixPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	preExecuteStatus := runStatus(t, workspace.Root)
	assertNode(t, preExecuteStatus, "plan")

	drivePlanToAwaitMergeNode(t, workspace, planPath, awaitMergeFinalizeFixStepOne, awaitMergeFinalizeFixStepTwo)

	preReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, preReopenStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := support.RequireJSONResult[lifecycleCommandResult](t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.Revision != 2 || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected await-merge finalize-fix reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	postReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, postReopenStatus, "execution/finalize/fix")
	if postReopenStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen cue after await-merge reopen, got %#v", postReopenStatus)
	}
	if !strings.Contains(postReopenStatus.Summary, "finalize-scope repair") {
		t.Fatalf("expected finalize-fix summary after await-merge reopen, got %#v", postReopenStatus)
	}
}

func TestAwaitMergeReopenNewStepWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-await-merge-reopen-new-step.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", awaitMergeNewStepPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, awaitMergeNewStepPlanTitle, awaitMergeReopenNewStepPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToAwaitMergeNode(t, workspace, planPath, awaitMergeNewStepStepOne, awaitMergeNewStepStepTwo)

	preReopenStatus := runStatus(t, workspace.Root)
	assertNode(t, preReopenStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "new-step")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := support.RequireJSONResult[lifecycleCommandResult](t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.Revision != 2 || reopenPayload.Artifacts.ToPlanPath != planRelPath {
		t.Fatalf("expected await-merge new-step reopen to restore %q as revision 2, got %#v", planRelPath, reopenPayload)
	}

	pendingNewStepStatus := runStatus(t, workspace.Root)
	assertNode(t, pendingNewStepStatus, "execution/finalize/fix")
	if pendingNewStepStatus.Facts.ReopenMode != "new-step" {
		t.Fatalf("expected new-step reopen cue after await-merge reopen, got %#v", pendingNewStepStatus)
	}
	if !strings.Contains(pendingNewStepStatus.Summary, "needs a new unfinished step") {
		t.Fatalf("expected pending new-step summary after await-merge reopen, got %#v", pendingNewStepStatus)
	}

	support.AppendStepBeforeValidationStrategy(t, planPath, awaitMergeNewStepStepThreeBody())

	postAppendStatus := runStatus(t, workspace.Root)
	assertNode(t, postAppendStatus, "execution/step-3/implement")
	if postAppendStatus.Facts.CurrentStep != trackedStepTitle(3, awaitMergeNewStepStepThree) {
		t.Fatalf("expected newly added step 3 to become current after await-merge reopen, got %#v", postAppendStatus)
	}
	if postAppendStatus.Facts.ReopenMode != "" {
		t.Fatalf("expected consumed new-step cue to disappear once step 3 exists, got %#v", postAppendStatus)
	}
}

func awaitMergeReopenFinalizeFixPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise ` + "`" + `reopen --mode finalize-fix` + "`" + ` from the merge-ready
` + "`" + `execution/finalize/await_merge` + "`" + ` node so the origin-specific
rollback is covered explicitly.

## Scope

### In Scope

- Reach merge-ready handoff through real archive and evidence commands.
- Reopen from ` + "`" + `await_merge` + "`" + ` with ` + "`" + `finalize-fix` + "`" + `.
- Assert the active path, revision bump, and finalize-fix status cue.

### Out of Scope

- Adding a new step after reopen.
- Land cleanup.

## Acceptance Criteria

- [ ] Full publish, CI, and sync evidence reaches ` + "`" + `execution/finalize/await_merge` + "`" + `.
- [ ] ` + "`" + `reopen --mode finalize-fix` + "`" + ` from ` + "`" + `await_merge` + "`" + ` restores the active tracked plan path and bumps the revision.
- [ ] Status resolves to ` + "`" + `execution/finalize/fix` + "`" + ` with finalize-scope repair guidance after reopen.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare the merge-ready candidate

- Done: [ ]

#### Objective

Close out the first tracked step before merge-ready handoff.

#### Details

NONE

#### Expected Files

- tests/e2e/await_merge_reopen_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Finish the archived handoff before reopen

- Done: [ ]

#### Objective

Close out the final pre-reopen step and reach merge-ready handoff.

#### Details

Archive-ready top-level summaries are prefilled so the scenario can reach
` + "`" + `await_merge` + "`" + ` before the reopen command runs.

#### Expected Files

- tests/e2e/await_merge_reopen_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The suite could cover await-merge happy path only and miss its rollback branch.
  - Mitigation: Reopen directly from ` + "`" + `await_merge` + "`" + ` and assert the restored active state.

## Validation Summary

Validated finalize-fix reopen from merge-ready handoff through the built binary.

## Review Summary

No unresolved blocking review findings remain in the candidate used for merge-ready reopen.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the await-merge finalize-fix reopen scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}

func awaitMergeReopenNewStepPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise ` + "`" + `reopen --mode new-step` + "`" + ` from the merge-ready
` + "`" + `execution/finalize/await_merge` + "`" + ` node so the origin-specific
rollback and later new-step consumption are both explicit.

## Scope

### In Scope

- Reach merge-ready handoff through real archive and evidence commands.
- Reopen from ` + "`" + `await_merge` + "`" + ` with ` + "`" + `new-step` + "`" + `.
- Assert the pending finalize-fix state before the new step exists.
- Add the new step and prove it becomes the current ` + "`" + `implement` + "`" + ` step.

### Out of Scope

- Running the reopened step through a full review loop.
- Land cleanup.

## Acceptance Criteria

- [ ] Full publish, CI, and sync evidence reaches ` + "`" + `execution/finalize/await_merge` + "`" + `.
- [ ] ` + "`" + `reopen --mode new-step` + "`" + ` from ` + "`" + `await_merge` + "`" + ` first resolves to finalize repair and asks for a new unfinished step.
- [ ] Once the new step is added, status resumes at ` + "`" + `execution/step-3/implement` + "`" + ` and the reopen cue is consumed.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare the merge-ready candidate

- Done: [ ]

#### Objective

Close out the first tracked step before merge-ready handoff.

#### Details

NONE

#### Expected Files

- tests/e2e/await_merge_reopen_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Finish the archived handoff before reopen

- Done: [ ]

#### Objective

Close out the final pre-reopen step and reach merge-ready handoff.

#### Details

Archive-ready top-level summaries are prefilled so the scenario can reach
` + "`" + `await_merge` + "`" + ` before the reopen command runs.

#### Expected Files

- tests/e2e/await_merge_reopen_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The suite could prove only the merge-ready source node and miss the pending new-step semantics after reopen.
  - Mitigation: Assert the finalize-fix node before appending the new step, then assert the new ` + "`" + `implement` + "`" + ` node after the append.

## Validation Summary

Validated new-step reopen from merge-ready handoff through the built binary.

## Review Summary

No unresolved blocking review findings remain in the candidate used for merge-ready reopen.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the await-merge new-step reopen scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}

func awaitMergeNewStepStepThreeBody() string {
	return `
### Step 3: Handle merge-ready follow-up as a new step

- Done: [ ]

#### Objective

Add the new unfinished step required by the await-merge reopen.

#### Details

This step is appended only after ` + "`" + `harness reopen --mode new-step` + "`" + `
returns the merge-ready candidate to active execution.

#### Expected Files

- tests/e2e/await_merge_reopen_test.go

#### Validation

- Confirm status moves to this new current step after the append.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW
`
}
