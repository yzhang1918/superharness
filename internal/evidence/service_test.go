package evidence_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catu-ai/microharness/internal/evidence"
	"github.com/catu-ai/microharness/internal/plan"
	"github.com/catu-ai/microharness/internal/runstate"
)

func TestSubmitCIEvidenceWritesArtifactAndUpdatesStatePointer(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/1"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}
	if result.Artifacts == nil || result.Artifacts.RecordID != "ci-001" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.LatestEvidence == nil || state.LatestEvidence.CI == nil {
		t.Fatalf("expected ci evidence pointer in state, got %#v", state)
	}
	if state.LatestCI == nil || state.LatestCI.Status != "success" {
		t.Fatalf("expected transitional CI cache, got %#v", state)
	}

	record, err := evidence.LoadLatestCI(root, state)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record == nil || record.Status != "success" || record.Provider != "buildkite" {
		t.Fatalf("unexpected CI record: %#v", record)
	}
}

func TestSubmitPublishRejectsMissingPRURL(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("publish", []byte(`{"status":"recorded"}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
}

func TestSubmitPublishWritesArtifactAndUpdatesStatePointer(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 2, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/microharness/pull/99","branch":"codex/test","base":"main"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.LatestEvidence == nil || state.LatestEvidence.Publish == nil {
		t.Fatalf("expected publish evidence pointer in state, got %#v", state)
	}
	if state.LatestPublish == nil || state.LatestPublish.PRURL == "" {
		t.Fatalf("expected transitional publish cache, got %#v", state)
	}

	record, err := evidence.LoadLatestPublish(root, state)
	if err != nil {
		t.Fatalf("load latest publish record: %v", err)
	}
	if record == nil || record.Status != "recorded" || record.PRURL != "https://github.com/catu-ai/microharness/pull/99" {
		t.Fatalf("unexpected publish record: %#v", record)
	}
}

func TestSubmitSyncSupportsExplicitNotApplied(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 5, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"not_applied","reason":"repository has no merge target in this environment"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	record, err := evidence.LoadLatestSync(root, state)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.Status != "not_applied" {
		t.Fatalf("unexpected sync record: %#v", record)
	}
	if state.Sync != nil {
		t.Fatalf("expected transitional sync cache to stay nil for not_applied, got %#v", state.Sync)
	}
}

func TestSubmitSyncFreshWritesArtifactAndUpdatesStatePointer(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 7, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.LatestEvidence == nil || state.LatestEvidence.Sync == nil {
		t.Fatalf("expected sync evidence pointer in state, got %#v", state)
	}
	if state.Sync == nil || state.Sync.Freshness != "fresh" || state.Sync.Conflicts {
		t.Fatalf("expected transitional sync cache, got %#v", state.Sync)
	}

	record, err := evidence.LoadLatestSync(root, state)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.Status != "fresh" || record.BaseRef != "main" {
		t.Fatalf("unexpected sync record: %#v", record)
	}
}

func TestSubmitEvidenceRequiresArchivedPlan(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlan(t, root, "docs/plans/active/2026-03-21-active-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected archived-plan requirement failure, got %#v", result)
	}
}

func TestSubmitEvidenceRejectsLandInProgress(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-21-evidence-plan", &runstate.State{
		PlanPath:    relPlanPath,
		PlanStem:    "2026-03-21-evidence-plan",
		CurrentNode: "land",
		Land: &runstate.LandState{
			PRURL:    "https://github.com/catu-ai/microharness/pull/99",
			LandedAt: "2026-03-21T11:00:00Z",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected land-in-progress evidence rejection, got %#v", result)
	}
}

func TestSubmitEvidenceRejectsWhenStateMutationLockIsHeld(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	release, err := runstate.AcquireStateMutationLock(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("acquire state lock: %v", err)
	}
	defer release()

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success"}`))
	if result.OK {
		t.Fatalf("expected state-lock contention failure, got %#v", result)
	}
	if result.Summary != "Another local state mutation is already in progress." {
		t.Fatalf("unexpected summary: %#v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Path != "state" {
		t.Fatalf("unexpected errors: %#v", result.Errors)
	}
}

func writeArchivedPlan(t *testing.T, root, relPath string) string {
	t.Helper()
	return writePlan(t, root, relPath, "Archived Evidence Plan")
}

func writeActivePlan(t *testing.T, root, relPath string) string {
	t.Helper()
	return writePlan(t, root, relPath, "Active Evidence Plan")
}

func writePlan(t *testing.T, root, relPath, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath
}
