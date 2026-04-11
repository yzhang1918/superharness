package resilience_test

import (
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

func TestStatusWarnsAndStaysConservativeWhenHistoricalReviewArtifactsAreMalformed(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/active/2026-04-11-resilience-review-artifacts.md"
	writePlanFixture(t, workspace, relPlanPath, "Resilience Review Artifacts", func(content string) string {
		return completeFirstStep(content)
	})
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-review-artifacts", &runstate.State{
		ExecutionStartedAt: "2026-04-11T13:10:00Z",
	})
	writeReviewManifest(t, workspace, "2026-04-11-resilience-review-artifacts", "review-001-delta", map[string]any{
		"review_title": "Step 1: Replace with first step title",
		"step":         1,
		"revision":     1,
	})
	writeReviewAggregate(t, workspace, "2026-04-11-resilience-review-artifacts", "review-001-delta", map[string]any{
		"decision": "pass",
	})
	workspace.WriteFile(t, ".local/harness/plans/2026-04-11-resilience-review-artifacts/reviews/review-002-delta/manifest.json", []byte("{not-json"))
	writeReviewAggregate(t, workspace, "2026-04-11-resilience-review-artifacts", "review-002-delta", map[string]any{
		"review_title": "mystery historical target",
		"revision":     1,
		"decision":     "changes_requested",
	})

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[statusResult](t, result)
	if parsed.State.CurrentNode != "execution/step-2/implement" {
		t.Fatalf("expected step 2 node to remain stable, got %#v", parsed)
	}
	if !findWarning(parsed.Warnings, "Unable to read historical review manifest") {
		t.Fatalf("expected unreadable review-manifest warning, got %#v", parsed.Warnings)
	}
	if !findWarning(parsed.Warnings, "review-002-delta") {
		t.Fatalf("expected conservative unmapped-round warning, got %#v", parsed.Warnings)
	}
	for _, action := range parsed.NextAction {
		if action.Description != "" && action.Command == nil && findWarning([]string{action.Description}, "review-002-delta") {
			t.Fatalf("did not expect malformed historical review to inject repair guidance, got %#v", parsed.NextAction)
		}
	}
}

func TestStatusDoesNotTreatMalformedEvidenceAsMergeReady(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/archived/2026-04-11-resilience-evidence-artifacts.md"
	writeArchivedArchiveCandidate(t, workspace, relPlanPath)
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-evidence-artifacts", &runstate.State{
		Revision: 1,
	})
	writePublishRecord(t, workspace, "2026-04-11-resilience-evidence-artifacts", relPlanPath, "publish-001", "https://github.com/catu-ai/easyharness/pull/201", 1)
	writeCIRecord(t, workspace, "2026-04-11-resilience-evidence-artifacts", relPlanPath, "ci-001", "success", 1)
	workspace.WriteFile(t, ".local/harness/plans/2026-04-11-resilience-evidence-artifacts/evidence/sync/sync-001.json", []byte("{not-json"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireSuccess(t, result)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[statusResult](t, result)
	if parsed.State.CurrentNode != "execution/finalize/publish" {
		t.Fatalf("expected malformed evidence to keep publish node, got %#v", parsed)
	}
	if !findWarning(parsed.Warnings, "Unable to read sync evidence") {
		t.Fatalf("expected sync-evidence warning, got %#v", parsed.Warnings)
	}
}
