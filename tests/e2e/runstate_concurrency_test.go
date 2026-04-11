package e2e_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

const (
	runstateConcurrencyPlanTitle = "Runstate Concurrency Coverage Plan"
	runstateConcurrencyStepOne   = "Prepare the first archived candidate"
	runstateConcurrencyStepTwo   = "Reach merge-ready handoff before reopen"
)

func TestArchivedRunstateInterleavingsIgnoreStaleEvidenceAndFailClearlyUnderLock(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-04-11-runstate-concurrency-coverage.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", runstateConcurrencyPlanTitle,
		"--timestamp", "2026-04-11T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#56",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, runstateConcurrencyPlanTitle, runstateConcurrencyPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	drivePlanToAwaitMergeNode(t, workspace, planPath, runstateConcurrencyStepOne, runstateConcurrencyStepTwo)

	mergeReadyStatus := runStatus(t, workspace.Root)
	assertNode(t, mergeReadyStatus, "execution/finalize/await_merge")

	reopen := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")
	support.RequireSuccess(t, reopen)
	support.RequireNoStderr(t, reopen)
	reopenPayload := requireLifecycleResult(t, reopen)
	if !reopenPayload.OK || reopenPayload.Command != "reopen" {
		t.Fatalf("unexpected reopen payload: %#v", reopenPayload)
	}
	if reopenPayload.State.CurrentNode != "execution/finalize/fix" || reopenPayload.Facts.Revision != 2 || reopenPayload.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen to bump revision to 2, got %#v", reopenPayload)
	}

	support.RewritePlanPreservingFrontmatter(t, planPath, runstateConcurrencyPlanTitle, runstateConcurrencyPlanBody())
	support.CompleteStep(
		t,
		planPath,
		1,
		"Re-established the first completed step after reopen.",
		"NO_STEP_REVIEW_NEEDED: Rehydrated reopen fixture keeps the original closeout intent.",
	)
	support.CompleteStep(
		t,
		planPath,
		2,
		"Re-established the second completed step after reopen.",
		"NO_STEP_REVIEW_NEEDED: Rehydrated reopen fixture keeps the original closeout intent.",
	)
	support.CheckAllAcceptanceCriteria(t, planPath)

	preFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, preFinalizeStatus, "execution/finalize/fix")
	if preFinalizeStatus.Facts.ReopenMode != "finalize-fix" {
		t.Fatalf("expected finalize-fix repair cue before the fresh finalize review, got %#v", preFinalizeStatus)
	}

	runPassingFinalizeReview(t, workspace)

	postFinalizeStatus := runStatus(t, workspace.Root)
	assertNode(t, postFinalizeStatus, "execution/finalize/archive")

	archive := support.Run(t, workspace.Root, "archive")
	support.RequireSuccess(t, archive)
	support.RequireNoStderr(t, archive)
	archivePayload := requireLifecycleResult(t, archive)
	if !archivePayload.OK || archivePayload.Command != "archive" || archivePayload.Facts.Revision != 2 {
		t.Fatalf("expected second archive to preserve revision 2, got %#v", archivePayload)
	}

	postRearchiveStatus := runStatus(t, workspace.Root)
	assertNode(t, postRearchiveStatus, "execution/finalize/publish")
	if postRearchiveStatus.Artifacts.PublishRecordID != "" || postRearchiveStatus.Artifacts.CIRecordID != "" || postRearchiveStatus.Artifacts.SyncRecordID != "" {
		t.Fatalf("expected revision-1 evidence to stay ignored after reopen, got %#v", postRearchiveStatus.Artifacts)
	}

	release, err := runstate.AcquireStateMutationLock(workspace.Root, "2026-04-11-runstate-concurrency-coverage")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}

	lockedStatus := support.Run(t, workspace.Root, "status")
	support.RequireExitCode(t, lockedStatus, 1)
	support.RequireNoStderr(t, lockedStatus)
	lockedStatusPayload := support.RequireJSONResult[statusResult](t, lockedStatus)
	if lockedStatusPayload.OK || lockedStatusPayload.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("expected locked status failure, got %#v", lockedStatusPayload)
	}

	lockedCIInput := workspace.WriteJSON(t, "tmp/locked-ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/rev2-locked",
	})
	lockedEvidence := support.Run(t, workspace.Root, "evidence", "submit", "--kind", "ci", "--input", lockedCIInput)
	support.RequireExitCode(t, lockedEvidence, 1)
	support.RequireNoStderr(t, lockedEvidence)
	lockedEvidencePayload := support.RequireJSONResult[evidenceSubmitResult](t, lockedEvidence)
	if lockedEvidencePayload.OK || lockedEvidencePayload.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("expected locked evidence-submit failure, got %#v", lockedEvidencePayload)
	}
	support.RequireFileMissing(t, workspace.Path(".local/harness/plans/2026-04-11-runstate-concurrency-coverage/evidence/ci/ci-002.json"))

	release()

	submitEvidence(t, workspace, "publish", "tmp/rev2-publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/catu-ai/easyharness/pull/156",
		"branch": "codex/runstate-concurrency-coverage",
		"base":   "main",
		"commit": "def456abc789",
	})
	submitEvidence(t, workspace, "ci", "tmp/rev2-ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/rev2",
	})
	submitEvidence(t, workspace, "sync", "tmp/rev2-sync.json", map[string]any{
		"status":   "fresh",
		"base_ref": "main",
		"head_ref": "codex/runstate-concurrency-coverage",
	})

	finalStatus := runStatus(t, workspace.Root)
	assertNode(t, finalStatus, "execution/finalize/await_merge")
	if finalStatus.Artifacts.PublishRecordID != "publish-002" || finalStatus.Artifacts.CIRecordID != "ci-002" || finalStatus.Artifacts.SyncRecordID != "sync-002" {
		t.Fatalf("expected revision-2 evidence to drive merge-ready status, got %#v", finalStatus.Artifacts)
	}
}

func runstateConcurrencyPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise deterministic archive, reopen, evidence, and status interleavings so
stale archived evidence never masquerades as the current candidate and lock
contention fails clearly.

## Scope

### In Scope

- Reach ` + "`" + `execution/finalize/await_merge` + "`" + ` with real evidence records.
- Reopen the archived candidate and archive it again at a new revision.
- Prove older revision evidence stays ignored until the new revision records its
  own publish, CI, and sync evidence.
- Hold the shared state lock while ` + "`" + `status` + "`" + ` and
  ` + "`" + `evidence submit` + "`" + ` run so the user-facing failure remains
  clear and deterministic.

### Out of Scope

- Nondeterministic race harnesses.
- New-step reopen semantics.

## Acceptance Criteria

- [ ] Revision 1 reaches ` + "`" + `execution/finalize/await_merge` + "`" + ` through real publish, CI, and sync evidence.
- [ ] After ` + "`" + `reopen --mode finalize-fix` + "`" + ` and re-archive, stale revision-1 evidence does not advance revision 2 past ` + "`" + `execution/finalize/publish` + "`" + `.
- [ ] While the shared state lock is held, ` + "`" + `status` + "`" + ` and
  ` + "`" + `evidence submit` + "`" + ` fail clearly without creating new evidence records.
- [ ] Fresh revision-2 evidence advances the candidate back to ` + "`" + `execution/finalize/await_merge` + "`" + `.

## Deferred Items

- None.

## Work Breakdown

### Step 1: Prepare the first archived candidate

- Done: [ ]

#### Objective

Close out the initial tracked step before the first archive.

#### Details

NONE

#### Expected Files

- tests/e2e/runstate_concurrency_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Reach merge-ready handoff before reopen

- Done: [ ]

#### Objective

Archive the initial candidate, record merge-ready evidence, then reopen for a
new revision of finalize-scope repair.

#### Details

The scenario should use the same plan stem across both archived revisions so the
test can prove stale evidence stays revision-scoped rather than bleeding into
the reopened candidate.

#### Expected Files

- tests/e2e/runstate_concurrency_test.go

#### Validation

- Run a clean finalize review before each archive point.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run ` + "`" + `go test ./tests/e2e -count=1` + "`" + `.

## Risks

- Risk: The test could accidentally re-prove only helper-level lock behavior.
  - Mitigation: Assert node progression and revision-sensitive evidence
    selection through the real binary before and after the lock-contention
    checks.

## Validation Summary

Validated the deterministic archive, reopen, evidence, and status interleaving.

## Review Summary

No unresolved blocking review findings remain in the candidate used for
concurrency coverage.

## Archive Summary

- PR: NONE
- Ready: The candidate is ready for archive and later merge handoff.
- Merge Handoff: Commit and push the archive move before treating the candidate
  as waiting for merge approval.

## Outcome Summary

### Delivered

Delivered deterministic runstate concurrency coverage.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}
