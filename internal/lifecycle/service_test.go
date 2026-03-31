package lifecycle_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
)

func TestArchiveMovesPlanAndUpdatesPointers(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	result := svc.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}

	archivedPath := filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("archived path missing: %v", err)
	}
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("expected active path to be removed, got %v", err)
	}
	if lint := plan.LintFile(archivedPath); !lint.OK {
		t.Fatalf("archived plan should lint, got %#v", lint)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current-plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/archived/2026-03-18-archive-smoke.md" {
		t.Fatalf("unexpected current plan: %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.PlanPath != "docs/plans/archived/2026-03-18-archive-smoke.md" {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestArchiveLightweightMovesLocalPlanAndPromptsBreadcrumb(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight.md"
	activePath := writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-lightweight", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-lightweight",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success, got %#v", result)
	}

	archivedRelPath := ".local/harness/plans/archived/2026-03-18-lightweight.md"
	archivedPath := filepath.Join(root, archivedRelPath)
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected local archived path, got %v", err)
	}
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Fatalf("expected tracked active path to be removed, got %v", err)
	}
	if result.Artifacts == nil || result.Artifacts.ToPlanPath != archivedRelPath {
		t.Fatalf("expected archived artifact path %q, got %#v", archivedRelPath, result.Artifacts)
	}
	if len(result.NextAction) == 0 || !strings.Contains(result.NextAction[0].Description, "repo-visible breadcrumb") {
		t.Fatalf("expected breadcrumb guidance first, got %#v", result.NextAction)
	}
	if !containsActionDescription(result.NextAction, "tracked active-plan removal") {
		t.Fatalf("expected lightweight archive guidance to mention the tracked active-plan removal, got %#v", result.NextAction)
	}
}

func TestExecuteStartPersistsMilestoneAndPointer(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-smoke.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Smoke"))

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC)
		},
	}.ExecuteStart()
	if !result.OK {
		t.Fatalf("expected execute start success, got %#v", result)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("unexpected current plan pointer: %#v", current)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:00:00Z" {
		t.Fatalf("expected execution-start milestone, got %#v", state)
	}
	if state.PlanPath != activeRelPath {
		t.Fatalf("unexpected plan path in state: %#v", state)
	}
}

func TestExecuteStartBackfillsLegacyExecutingPlan(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-legacy.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Legacy Execute Start"))
	if _, err := runstate.SaveState(root, "2026-03-18-execute-start-legacy", &runstate.State{
		PlanPath:    activeRelPath,
		PlanStem:    "2026-03-18-execute-start-legacy",
		Revision:    2,
		CurrentNode: "execution/step-1/implement",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-legacy-delta",
			Kind:       "delta",
			Aggregated: false,
		},
	}); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}
	seeded, _, err := runstate.LoadState(root, "2026-03-18-execute-start-legacy")
	if err != nil {
		t.Fatalf("load seeded state: %v", err)
	}
	if seeded == nil || seeded.ExecutionStartedAt != "" {
		t.Fatalf("expected legacy state to start without execution_started_at, got %#v", seeded)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 5, 0, 0, time.UTC)
		},
	}.ExecuteStart()
	if !result.OK {
		t.Fatalf("expected execute start success, got %#v", result)
	}
	if !strings.Contains(result.Summary, "Execution started") {
		t.Fatalf("unexpected summary: %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-legacy")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:05:00Z" {
		t.Fatalf("expected backfilled execution-start milestone, got %#v", state)
	}
	if state.ActiveReviewRound == nil || state.ActiveReviewRound.RoundID != "review-legacy-delta" {
		t.Fatalf("expected legacy executing state to remain otherwise intact, got %#v", state)
	}
}

func TestExecuteStartIsIdempotent(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-idempotent.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Idempotent"))

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 1, 10, 0, 0, time.UTC)
		},
	}
	first := svc.ExecuteStart()
	if !first.OK {
		t.Fatalf("expected first execute start success, got %#v", first)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 1, 20, 0, 0, time.UTC)
	}
	second := svc.ExecuteStart()
	if !second.OK {
		t.Fatalf("expected second execute start success, got %#v", second)
	}
	if !strings.Contains(second.Summary, "already started") {
		t.Fatalf("unexpected second summary: %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-idempotent")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "2026-03-18T01:10:00Z" {
		t.Fatalf("expected original execution-start timestamp to remain, got %#v", state)
	}
}

func TestExecuteStartRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-locked.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Locked"))

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-18-execute-start-locked")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start failure while state lock is held, got %#v", result)
	}
	if result.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Path != "state" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func TestExecuteStartRollsBackWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-execute-start-rollback.md"
	writeFile(t, filepath.Join(root, activeRelPath), buildAwaitingPlan(t, "Execute Start Rollback"))
	currentPlanAsDir := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(currentPlanAsDir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.ExecuteStart()
	if result.OK {
		t.Fatalf("expected execute start failure, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-execute-start-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected state rollback after pointer write failure, got %#v", state)
	}
}

func TestArchiveRejectsMissingArchiveSummaryFields(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")
	content := buildActiveArchiveCandidate(t)
	content = strings.Replace(content, "- PR: NONE\n", "", 1)
	writeFile(t, path, content)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           "docs/plans/active/2026-03-18-archive-smoke.md",
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{Workdir: root}
	result := svc.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "section.Archive Summary")
}

func TestArchivePreflightFailureLeavesPlanAndPointersUntouched(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := filepath.Join(root, activeRelPath)
	content := buildActiveArchiveCandidate(t)
	content = strings.Replace(content, "## Deferred Items\n\n- None.\n", "## Deferred Items\n\n- Deferred follow-up still needs to be written down.\n", 1)
	content = strings.Replace(content, "- PR: NONE\n", "", 1)
	writeFile(t, activePath, content)
	if _, err := runstate.SaveCurrentPlan(root, activeRelPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	assertErrorPath(t, result.Errors, "section.Archive Summary")
	assertErrorPath(t, result.Errors, "section.Outcome Summary.Follow-Up Issues")

	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active plan to remain after failed archive, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected no archived plan to be written, got %v", err)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("expected current plan pointer to remain on active plan, got %#v", current)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.PlanPath != activeRelPath {
		t.Fatalf("expected state pointer to remain on active plan, got %#v", state)
	}
}

func TestArchiveRollsBackWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	activePath := writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	currentPlanAsDir := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(currentPlanAsDir, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := lifecycle.Service{Workdir: root}.Archive()
	if result.OK {
		t.Fatalf("expected archive failure, got %#v", result)
	}
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("expected active plan to remain after rollback, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs/plans/archived/2026-03-18-archive-smoke.md")); !os.IsNotExist(err) {
		t.Fatalf("expected archived target to be removed on rollback, got %v", err)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.PlanPath != activeRelPath {
		t.Fatalf("expected state to roll back to active path, got %#v", state)
	}
}

func TestArchiveRejectsUnresolvedLocalState(t *testing.T) {
	testCases := []struct {
		name       string
		state      *runstate.State
		errorPath  string
		errorMatch string
	}{
		{
			name: "active review round",
			state: &runstate.State{
				ActiveReviewRound: &runstate.ReviewRound{RoundID: "review-001-full", Kind: "full", Aggregated: false},
			},
			errorPath:  "state.active_review_round",
			errorMatch: "aggregate or clear",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
			writeActiveArchiveCandidate(t, root, activeRelPath)
			tc.state.PlanPath = activeRelPath
			tc.state.PlanStem = "2026-03-18-archive-smoke"
			tc.state.ExecutionStartedAt = "2026-03-18T03:30:00Z"
			if tc.state.ActiveReviewRound == nil && tc.errorPath != "state.active_review_round" {
				tc.state.ActiveReviewRound = &runstate.ReviewRound{
					RoundID:    "review-001-full",
					Kind:       "full",
					Revision:   1,
					Aggregated: true,
					Decision:   "pass",
				}
			}
			if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", tc.state); err != nil {
				t.Fatalf("save state: %v", err)
			}

			result := lifecycle.Service{Workdir: root}.Archive()
			if result.OK {
				t.Fatalf("expected archive failure, got %#v", result)
			}
			assertErrorPath(t, result.Errors, tc.errorPath)
			assertErrorContains(t, result.Errors, tc.errorPath, tc.errorMatch)
		})
	}
}

func TestArchiveRequiresPassingReviewForRevisionOne(t *testing.T) {
	testCases := []struct {
		name       string
		state      *runstate.State
		errorMatch string
	}{
		{
			name:       "missing review",
			state:      &runstate.State{},
			errorMatch: "passing full finalize review",
		},
		{
			name: "passing delta is not enough",
			state: &runstate.State{
				ActiveReviewRound: &runstate.ReviewRound{
					RoundID:    "review-001-delta",
					Kind:       "delta",
					Revision:   1,
					Aggregated: true,
					Decision:   "pass",
				},
			},
			errorMatch: "passing full finalize review",
		},
		{
			name: "failed full review still blocks",
			state: &runstate.State{
				ActiveReviewRound: &runstate.ReviewRound{
					RoundID:    "review-001-full",
					Kind:       "full",
					Revision:   1,
					Aggregated: true,
					Decision:   "changes_requested",
				},
			},
			errorMatch: "not archive-ready",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
			writeActiveArchiveCandidate(t, root, activeRelPath)
			tc.state.PlanPath = activeRelPath
			tc.state.PlanStem = "2026-03-18-archive-smoke"
			tc.state.ExecutionStartedAt = "2026-03-18T03:30:00Z"
			if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", tc.state); err != nil {
				t.Fatalf("save state: %v", err)
			}

			result := lifecycle.Service{Workdir: root}.Archive()
			if result.OK {
				t.Fatalf("expected archive failure, got %#v", result)
			}
			assertErrorPath(t, result.Errors, "state.active_review_round")
			assertErrorContains(t, result.Errors, "state.active_review_round", tc.errorMatch)
		})
	}
}

func TestArchiveAllowsPassingDeltaReviewForReopenedRevision(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)

	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T03:55:00Z",
		Revision:           2,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-delta",
			Kind:       "delta",
			Revision:   2,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 0, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success for reopened revision, got %#v", result)
	}
}

func TestArchiveIgnoresCIPublishSyncSignalsOnceFinalizeReviewPasses(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)

	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T04:05:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
		LatestCI: &runstate.CIState{SnapshotID: "ci-001", Status: "pending"},
		Sync:     &runstate.SyncState{Freshness: "stale", Conflicts: true},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 10, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success despite CI/sync signals, got %#v", result)
	}
}

func TestArchiveUsesAggregateArtifactForLegacyReviewDecision(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T04:25:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	writeAggregateArtifact(t, root, "2026-03-18-archive-smoke", "review-001-full", map[string]any{
		"decision": "pass",
	})

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 4, 30, 0, 0, time.UTC)
		},
	}.Archive()
	if !result.OK {
		t.Fatalf("expected archive success for legacy review decision, got %#v", result)
	}
}

func TestReopenMovesArchivedPlanBackToActiveAndResetsSummaries(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           "docs/plans/active/2026-03-18-archive-smoke.md",
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	activePath := filepath.Join(root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("reopened active path missing: %v", err)
	}
	if lint := plan.LintFile(activePath); !lint.OK {
		t.Fatalf("reopened active plan should lint, got %#v", lint)
	}
	data, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read reopened plan: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "UPDATE_REQUIRED_AFTER_REOPEN") {
		t.Fatalf("expected reopen update-required markers, got:\n%s", text)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected reopen mode to be recorded, got %#v", state)
	}
	if state.Revision != 2 {
		t.Fatalf("expected revision bump in state, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-archive-smoke.md" {
		t.Fatalf("expected reopened current-plan pointer to move back to active path, got %#v", current)
	}
}

func TestReopenLightweightMovesLocalArchiveBackToTrackedActive(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-lightweight-reopen.md"
	writeLightweightActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-lightweight-reopen", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-lightweight-reopen",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	archivedPath := filepath.Join(root, ".local/harness/plans/archived/2026-03-18-lightweight-reopen.md")
	if _, err := os.Stat(archivedPath); err != nil {
		t.Fatalf("expected lightweight archived path to exist before reopen, got %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	reopenedActivePath := filepath.Join(root, activeRelPath)
	if _, err := os.Stat(reopenedActivePath); err != nil {
		t.Fatalf("reopened tracked active path missing: %v", err)
	}
	if _, err := os.Stat(archivedPath); !os.IsNotExist(err) {
		t.Fatalf("expected local lightweight archive to be removed after reopen, got %v", err)
	}
	if lint := plan.LintFile(reopenedActivePath); !lint.OK {
		t.Fatalf("reopened lightweight active plan should lint, got %#v", lint)
	}
	state, _, err := runstate.LoadState(root, "2026-03-18-lightweight-reopen")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" || state.PlanPath != activeRelPath {
		t.Fatalf("expected reopened lightweight state to point back to tracked active path, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != activeRelPath {
		t.Fatalf("expected reopened current-plan pointer to move back to tracked active path, got %#v", current)
	}
}

func TestReopenNewStepRecordsModeAndStatusCue(t *testing.T) {
	root := t.TempDir()
	writeActiveArchiveCandidate(t, root, "docs/plans/active/2026-03-18-archive-smoke.md")
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           "docs/plans/active/2026-03-18-archive-smoke.md",
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("new-step")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}
	if len(reopen.NextAction) == 0 || !strings.Contains(reopen.NextAction[len(reopen.NextAction)-1].Description, "Add a new unfinished step") {
		t.Fatalf("unexpected reopen next actions: %#v", reopen.NextAction)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "new-step" {
		t.Fatalf("expected reopen mode to be recorded, got %#v", state)
	}
	if state.Reopen.BaseStepCount != 2 {
		t.Fatalf("expected reopen to capture original step count, got %#v", state.Reopen)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected current node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected status summary: %q", statusResult.Summary)
	}
}

func TestReopenMarkersMustBeClearedBeforeRearchive(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T03:05:00Z",
		Revision:           2,
		Reopen:             &runstate.ReopenState{Mode: "finalize-fix", ReopenedAt: "2026-03-18T03:00:00Z"},
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-002-delta",
			Kind:       "delta",
			Revision:   2,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save reopened state: %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 4, 0, 0, 0, time.UTC)
	}
	rearchive := svc.Archive()
	if rearchive.OK {
		t.Fatalf("expected rearchive to fail while reopen markers remain, got %#v", rearchive)
	}
	assertErrorPath(t, rearchive.Errors, "section.Validation Summary")

	activePath := filepath.Join(root, activeRelPath)
	data, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatalf("read reopened plan: %v", err)
	}
	cleared := strings.ReplaceAll(string(data), "UPDATE_REQUIRED_AFTER_REOPEN\n\n", "")
	writeFile(t, activePath, cleared)

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 4, 10, 0, 0, time.UTC)
	}
	rearchive = svc.Archive()
	if !rearchive.OK {
		t.Fatalf("expected rearchive success after clearing markers, got %#v", rearchive)
	}
}

func TestReopenClearsStaleCIAndSyncSignals(t *testing.T) {
	root := t.TempDir()
	activeRelPath := "docs/plans/active/2026-03-18-archive-smoke.md"
	writeActiveArchiveCandidate(t, root, activeRelPath)
	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath:           activeRelPath,
		PlanStem:           "2026-03-18-archive-smoke",
		ExecutionStartedAt: "2026-03-18T01:55:00Z",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
		LatestCI: &runstate.CIState{SnapshotID: "ci-1", Status: "success"},
		Sync:     &runstate.SyncState{Freshness: "fresh", Conflicts: false},
	}); err != nil {
		t.Fatalf("save initial state: %v", err)
	}

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC)
		},
	}
	archive := svc.Archive()
	if !archive.OK {
		t.Fatalf("archive failed: %#v", archive)
	}

	if _, err := runstate.SaveState(root, "2026-03-18-archive-smoke", &runstate.State{
		PlanPath: "docs/plans/archived/2026-03-18-archive-smoke.md",
		PlanStem: "2026-03-18-archive-smoke",
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
		LatestEvidence: &runstate.EvidenceSet{
			CI:      &runstate.EvidencePointer{Kind: "ci", RecordID: "ci-2", Path: ".local/harness/plans/2026-03-18-archive-smoke/evidence/ci/ci-002.json"},
			Publish: &runstate.EvidencePointer{Kind: "publish", RecordID: "publish-1", Path: ".local/harness/plans/2026-03-18-archive-smoke/evidence/publish/publish-001.json"},
			Sync:    &runstate.EvidencePointer{Kind: "sync", RecordID: "sync-2", Path: ".local/harness/plans/2026-03-18-archive-smoke/evidence/sync/sync-002.json"},
		},
		CurrentNode:   "execution/finalize/await_merge",
		LatestCI:      &runstate.CIState{SnapshotID: "ci-2", Status: "failed"},
		Sync:          &runstate.SyncState{Freshness: "stale", Conflicts: true},
		LatestPublish: &runstate.Publish{AttemptID: "publish-1", PRURL: "https://github.com/catu-ai/easyharness/pull/99"},
	}); err != nil {
		t.Fatalf("save archived state: %v", err)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 3, 18, 3, 0, 0, 0, time.UTC)
	}
	reopen := svc.Reopen("finalize-fix")
	if !reopen.OK {
		t.Fatalf("reopen failed: %#v", reopen)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-archive-smoke")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected reopened state")
	}
	if state.ActiveReviewRound != nil || state.CurrentNode != "" || state.Land != nil || state.LatestEvidence != nil || state.LatestCI != nil || state.Sync != nil || state.LatestPublish != nil {
		t.Fatalf("expected reopened state to clear stale review/evidence/cache signals, got %#v", state)
	}
}

func TestReopenRejectsLandCleanupInProgress(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	reopen := svc.Reopen("finalize-fix")
	if reopen.OK {
		t.Fatalf("expected reopen to fail during land cleanup, got %#v", reopen)
	}
	assertErrorPath(t, reopen.Errors, "state.current_node")

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "land" || state.Land == nil || state.Land.CompletedAt != "" {
		t.Fatalf("expected land cleanup state to remain intact, got %#v", state)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/archived/2026-03-18-landed-plan.md" {
		t.Fatalf("expected archived current plan to remain intact, got %#v", current)
	}
}

func TestLandCompleteWritesIdleMarkerForStatus(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	result := svc.LandComplete()
	if !result.OK {
		t.Fatalf("expected land complete success, got %#v", result)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "" || current.LastLandedPlanPath != "docs/plans/archived/2026-03-18-landed-plan.md" {
		t.Fatalf("unexpected current plan marker: %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected idle-after-land status, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "idle" {
		t.Fatalf("unexpected current node: %#v", statusResult.State)
	}
}

func TestLandCompleteRejectsMissingLandEntry(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.LandComplete()
	if result.OK {
		t.Fatalf("expected land complete failure without prior land entry, got %#v", result)
	}
}

func TestLandUsesLegacyEvidenceCachesWhenPointersAreMissing(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-18-landed-plan", &runstate.State{
		PlanPath:       "docs/plans/archived/2026-03-18-landed-plan.md",
		PlanStem:       "2026-03-18-landed-plan",
		Revision:       3,
		LatestCI:       &runstate.CIState{SnapshotID: "ci-legacy", Status: "success"},
		LatestPublish:  &runstate.Publish{AttemptID: "publish-legacy", PRURL: "https://github.com/catu-ai/easyharness/pull/99"},
		Sync:           &runstate.SyncState{Freshness: "fresh", Conflicts: false},
		LatestEvidence: nil,
	}); err != nil {
		t.Fatalf("save legacy state: %v", err)
	}

	result := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}.Land("https://github.com/catu-ai/easyharness/pull/99", "")
	if !result.OK {
		t.Fatalf("expected land success from legacy caches, got %#v", result)
	}
}

func TestLandRejectsOverwriteDuringCleanup(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	first := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !first.OK {
		t.Fatalf("expected initial land success, got %#v", first)
	}

	second := svc.Land("https://github.com/catu-ai/easyharness/pull/100", "def456")
	if second.OK {
		t.Fatalf("expected second land entry to fail, got %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.PRURL != "https://github.com/catu-ai/easyharness/pull/99" || state.Land.Commit != "abc123" {
		t.Fatalf("expected original land record to remain intact, got %#v", state)
	}
}

func TestLandAllowsCommitEnrichmentDuringCleanup(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	first := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "")
	if !first.OK {
		t.Fatalf("expected initial land success, got %#v", first)
	}

	second := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !second.OK {
		t.Fatalf("expected commit enrichment success, got %#v", second)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.Land == nil || state.Land.PRURL != "https://github.com/catu-ai/easyharness/pull/99" || state.Land.Commit != "abc123" {
		t.Fatalf("expected commit enrichment to update the existing land record, got %#v", state)
	}
}

func TestLandCompleteRollsBackStateWhenCurrentPlanWriteFails(t *testing.T) {
	root := t.TempDir()
	writeArchivedLandedPlan(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForLifecycle(t, root)

	svc := lifecycle.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
		},
	}
	land := svc.Land("https://github.com/catu-ai/easyharness/pull/99", "abc123")
	if !land.OK {
		t.Fatalf("expected land success, got %#v", land)
	}

	currentPlanPath := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.Remove(currentPlanPath); err != nil {
		t.Fatalf("remove current-plan file: %v", err)
	}
	if err := os.MkdirAll(currentPlanPath, 0o755); err != nil {
		t.Fatalf("mkdir current-plan dir: %v", err)
	}

	result := svc.LandComplete()
	if result.OK {
		t.Fatalf("expected land complete failure when current-plan write fails, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.CurrentNode != "land" || state.Land == nil || state.Land.CompletedAt != "" {
		t.Fatalf("expected land state rollback after pointer write failure, got %#v", state)
	}
}

func seedMergeReadyEvidenceForLifecycle(t *testing.T, root string) {
	t.Helper()
	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 5, 50, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("seed ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("seed sync evidence: %#v", result)
	}
}

func writeActiveArchiveCandidate(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	writeFile(t, path, buildActiveArchiveCandidate(t))
	return path
}

func writeLightweightActiveArchiveCandidate(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	content := strings.Replace(buildActiveArchiveCandidate(t), "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)
	return path
}

func writeArchivedLandedPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, relPath)
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Landed Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_EXECUTION", "Done.")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_REVIEW", "Reviewed.")
	rendered = strings.Replace(rendered, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the slice.", 1)
	rendered = strings.Replace(rendered, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nFull review passed.", 1)
	rendered = strings.Replace(rendered, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- Archived At: 2026-03-18T02:00:00Z\n- Revision: 1\n- PR: NONE\n- Ready: Ready for merge approval.\n- Merge Handoff: Commit and push before merge approval.", 1)
	rendered = strings.Replace(rendered, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the slice.", 1)
	rendered = strings.Replace(rendered, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.", 1)
	writeFile(t, path, rendered)
	return path
}

func buildActiveArchiveCandidate(t *testing.T) string {
	t.Helper()
	rendered := buildAwaitingPlan(t, "Archive Smoke")
	rendered = strings.ReplaceAll(rendered, "- Done: [ ]", "- Done: [x]")
	rendered = strings.ReplaceAll(rendered, "- [ ]", "- [x]")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_EXECUTION", "Completed execution notes.")
	rendered = strings.ReplaceAll(rendered, "PENDING_STEP_REVIEW", "Completed review notes.")
	rendered = strings.Replace(rendered, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the implementation and command surfaces.", 1)
	rendered = strings.Replace(rendered, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nNo unresolved blocking review findings remain.", 1)
	rendered = strings.Replace(rendered, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.\n- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.", 1)
	rendered = strings.Replace(rendered, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the planned CLI slice.", 1)
	rendered = strings.Replace(rendered, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.", 1)
	return rendered
}

func buildAwaitingPlan(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	return rendered
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func writeAggregateArtifact(t *testing.T, root, planStem, roundID string, payload map[string]any) {
	t.Helper()
	dir := filepath.Join(root, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir aggregate dir: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal aggregate payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "aggregate.json"), data, 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func containsActionDescription(actions []lifecycle.NextAction, snippet string) bool {
	for _, action := range actions {
		if strings.Contains(action.Description, snippet) {
			return true
		}
	}
	return false
}

func assertErrorPath(t *testing.T, issues []lifecycle.CommandError, path string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected error for %s, got %#v", path, issues)
}

func assertErrorContains(t *testing.T, issues []lifecycle.CommandError, path, fragment string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Path == path && strings.Contains(issue.Message, fragment) {
			return
		}
	}
	t.Fatalf("expected error for %s containing %q, got %#v", path, fragment, issues)
}
