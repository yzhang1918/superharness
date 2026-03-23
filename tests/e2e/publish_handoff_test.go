package e2e_test

import (
	"strings"
	"testing"

	"github.com/yzhang1918/superharness/tests/support"
)

const (
	publishHandoffPlanTitle = "Publish Handoff Plan"
	publishStepOneTitle     = "Prepare the archived candidate"
	publishStepTwoTitle     = "Finish the branch before publish handoff"
)

func TestPublishHandoffWithBuiltBinary(t *testing.T) {
	workspace := support.NewWorkspace(t)
	planRelPath := "docs/plans/active/2026-03-23-publish-handoff.md"
	planPath := workspace.Path(planRelPath)

	template := support.Run(
		t,
		workspace.Root,
		"plan", "template",
		"--title", publishHandoffPlanTitle,
		"--timestamp", "2026-03-23T00:00:00Z",
		"--source-type", "issue",
		"--source-ref", "#6",
		"--output", planRelPath,
	)
	support.RequireSuccess(t, template)
	support.RequireNoStderr(t, template)
	support.RewritePlanPreservingFrontmatter(t, planPath, publishHandoffPlanTitle, publishHandoffPlanBody())

	lint := support.Run(t, workspace.Root, "plan", "lint", planRelPath)
	support.RequireSuccess(t, lint)
	support.RequireNoStderr(t, lint)

	archivePayload := drivePlanToArchivedPublishNode(t, workspace, planPath, publishStepOneTitle, publishStepTwoTitle)
	if archivePayload.Artifacts.ToPlanPath != "docs/plans/archived/2026-03-23-publish-handoff.md" {
		t.Fatalf("expected archived publish-handoff path, got %#v", archivePayload)
	}

	publish := submitEvidence(t, workspace, "publish", "tmp/publish.json", map[string]any{
		"status": "recorded",
		"pr_url": "https://github.com/yzhang1918/superharness/pull/99",
		"branch": "codex/e2e-lifecycle-handoff-coverage",
		"base":   "main",
	})
	if publish.Artifacts.Kind != "publish" {
		t.Fatalf("expected publish evidence artifacts, got %#v", publish)
	}

	postPublishStatus := runStatus(t, workspace.Root)
	assertNode(t, postPublishStatus, "execution/finalize/publish")
	if postPublishStatus.Facts.PublishStatus != "recorded" {
		t.Fatalf("expected publish status after publish evidence, got %#v", postPublishStatus)
	}
	if postPublishStatus.Artifacts.PublishRecordID != publish.Artifacts.RecordID {
		t.Fatalf("expected publish record id %q in status, got %#v", publish.Artifacts.RecordID, postPublishStatus)
	}

	ci := submitEvidence(t, workspace, "ci", "tmp/ci.json", map[string]any{
		"status":   "success",
		"provider": "github-actions",
		"url":      "https://ci.example/build/1",
	})
	if ci.Artifacts.Kind != "ci" {
		t.Fatalf("expected ci evidence artifacts, got %#v", ci)
	}

	postCIStatus := runStatus(t, workspace.Root)
	assertNode(t, postCIStatus, "execution/finalize/publish")
	if postCIStatus.Facts.CIStatus != "success" {
		t.Fatalf("expected CI success to remain in publish until sync exists, got %#v", postCIStatus)
	}
	if postCIStatus.Artifacts.CIRecordID != ci.Artifacts.RecordID {
		t.Fatalf("expected CI record id %q in status, got %#v", ci.Artifacts.RecordID, postCIStatus)
	}

	sync := submitEvidence(t, workspace, "sync", "tmp/sync.json", map[string]any{
		"status":   "fresh",
		"base_ref": "main",
		"head_ref": "codex/e2e-lifecycle-handoff-coverage",
	})
	if sync.Artifacts.Kind != "sync" {
		t.Fatalf("expected sync evidence artifacts, got %#v", sync)
	}

	postSyncStatus := runStatus(t, workspace.Root)
	assertNode(t, postSyncStatus, "execution/finalize/await_merge")
	if postSyncStatus.Facts.PublishStatus != "recorded" || postSyncStatus.Facts.CIStatus != "success" || postSyncStatus.Facts.SyncStatus != "fresh" {
		t.Fatalf("expected merge-ready evidence facts after sync, got %#v", postSyncStatus)
	}
	if postSyncStatus.Artifacts.PublishRecordID != publish.Artifacts.RecordID ||
		postSyncStatus.Artifacts.CIRecordID != ci.Artifacts.RecordID ||
		postSyncStatus.Artifacts.SyncRecordID != sync.Artifacts.RecordID {
		t.Fatalf("expected latest evidence record ids in status, got %#v", postSyncStatus)
	}
}

func publishHandoffPlanBody() string {
	return strings.TrimSpace(`
## Goal

Exercise the archived publish handoff through the real built binary so status
stays at execution/finalize/publish until publish, CI, and sync evidence are
all recorded.

## Scope

### In Scope

- Archive a clean candidate.
- Record publish, CI, and sync evidence one domain at a time.
- Assert the publish self-loop and the later transition to await_merge.

### Out of Scope

- Land entry and cleanup.

## Acceptance Criteria

- [ ] Publish evidence alone keeps the candidate in execution/finalize/publish.
- [ ] CI evidence alone still keeps the candidate in execution/finalize/publish until sync is recorded.
- [ ] The full publish, CI, and sync evidence set moves status to execution/finalize/await_merge.

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

- tests/e2e/publish_handoff_test.go

#### Validation

- Run a clean delta review before advancing.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

### Step 2: Finish the branch before publish handoff

- Done: [ ]

#### Objective

Close out the final pre-archive step and enter publish handoff.

#### Details

Archive-ready top-level summaries are prefilled so the scenario can reach the
evidence phase without extra tracked-plan editing.

#### Expected Files

- tests/e2e/publish_handoff_test.go

#### Validation

- Run a clean delta review before finalize review.

#### Execution Notes

PENDING_STEP_EXECUTION

#### Review Notes

PENDING_STEP_REVIEW

## Validation Strategy

- Run repo-level E2E coverage with the built binary.

## Risks

- Risk: The scenario could accidentally assert only the final merge-ready state and miss the publish self-loop.
  - Mitigation: Check status after each evidence domain separately.

## Validation Summary

Validated the archived candidate and the evidence-driven publish handoff.

## Review Summary

No unresolved blocking review findings remain in the archived candidate used for evidence handoff.

## Archive Summary

- PR: NONE
- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.
- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.

## Outcome Summary

### Delivered

Delivered the archived candidate and evidence handoff scenario.

### Not Delivered

NONE.

### Follow-Up Issues

NONE
`)
}
