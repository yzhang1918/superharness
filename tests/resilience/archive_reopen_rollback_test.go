package resilience_test

import (
	"os"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/tests/support"
)

func TestArchiveRollsBackWhenActivePlanCannotBeRemoved(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/active/2026-04-11-resilience-archive-rollback.md"
	writeActiveArchiveCandidate(t, workspace, relPlanPath)
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-archive-rollback", &runstate.State{
		ExecutionStartedAt: "2026-04-11T13:15:00Z",
		Revision:           1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	})

	activeDir := workspace.Path("docs/plans/active")
	if err := os.Chmod(activeDir, 0o555); err != nil {
		t.Fatalf("chmod active dir: %v", err)
	}

	result := support.Run(t, workspace.Root, "archive")

	if err := os.Chmod(activeDir, 0o755); err != nil {
		t.Fatalf("restore active dir perms: %v", err)
	}

	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[lifecycleResult](t, result)
	if parsed.OK || parsed.Command != "archive" {
		t.Fatalf("expected archive failure payload, got %#v", parsed)
	}
	if parsed.Summary != "Unable to remove the active plan after archiving." {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
	support.RequireFileExists(t, workspace.Path(relPlanPath))
	support.RequireFileMissing(t, workspace.Path("docs/plans/archived/2026-04-11-resilience-archive-rollback.md"))
	current := readCurrentPlan(t, workspace)
	if current["plan_path"] != relPlanPath {
		t.Fatalf("expected current plan pointer to roll back to active path, got %#v", current)
	}
	state, _, err := runstate.LoadState(workspace.Root, "2026-04-11-resilience-archive-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Reopen != nil || state.ActiveReviewRound == nil {
		t.Fatalf("expected archived state mutation to roll back cleanly, got %#v", state)
	}
}

func TestReopenRollsBackWhenArchivedPlanCannotBeRemoved(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/archived/2026-04-11-resilience-reopen-rollback.md"
	writeArchivedArchiveCandidate(t, workspace, relPlanPath)
	writeCurrentPlan(t, workspace, relPlanPath)
	writeState(t, workspace, "2026-04-11-resilience-reopen-rollback", &runstate.State{
		Revision: 1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	})

	archivedDir := workspace.Path("docs/plans/archived")
	if err := os.Chmod(archivedDir, 0o555); err != nil {
		t.Fatalf("chmod archived dir: %v", err)
	}

	result := support.Run(t, workspace.Root, "reopen", "--mode", "finalize-fix")

	if err := os.Chmod(archivedDir, 0o755); err != nil {
		t.Fatalf("restore archived dir perms: %v", err)
	}

	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[lifecycleResult](t, result)
	if parsed.OK || parsed.Command != "reopen" {
		t.Fatalf("expected reopen failure payload, got %#v", parsed)
	}
	if parsed.Summary != "Unable to remove the archived plan after reopening." {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
	support.RequireFileExists(t, workspace.Path(relPlanPath))
	support.RequireFileMissing(t, workspace.Path("docs/plans/active/2026-04-11-resilience-reopen-rollback.md"))
	current := readCurrentPlan(t, workspace)
	if current["plan_path"] != relPlanPath {
		t.Fatalf("expected current plan pointer to roll back to archived path, got %#v", current)
	}
	state, _, err := runstate.LoadState(workspace.Root, "2026-04-11-resilience-reopen-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Revision != 1 || state.Reopen != nil {
		t.Fatalf("expected reopened state mutation to roll back cleanly, got %#v", state)
	}
}
