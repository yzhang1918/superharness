package evidence_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestSubmitCIEvidenceWritesArtifactWithoutStateCache(t *testing.T) {
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
	if state != nil {
		t.Fatalf("expected CI submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1)
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

func TestSubmitPublishRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("publish", []byte(`{
		"status":"recorded",
		"pr_url":"https://github.com/catu-ai/easyharness/pull/99",
		"unexpected":true
	}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.unexpected")
}

func TestSubmitCIRejectsUnknownSchemaProperty(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{
		"status": "success",
		"provider": "buildkite",
		"unexpected": true
	}`))
	if result.OK {
		t.Fatalf("expected schema validation failure, got %#v", result)
	}
	if len(result.Errors) == 0 || result.Errors[0].Path != "input.unexpected" {
		t.Fatalf("expected unknown-property error, got %#v", result.Errors)
	}
}

func TestSubmitCIRejectsUnknownField(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":"success","unexpected":true}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.unexpected")
}

func TestSubmitCIRejectsWrongStatusType(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("ci", []byte(`{"status":1}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.status")
}

func TestSubmitPublishWritesArtifactWithoutStateCache(t *testing.T) {
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
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99","branch":"codex/test","base":"main"}`))
	if !result.OK {
		t.Fatalf("expected success, got %#v", result)
	}

	state, _, err := runstate.LoadState(root, "2026-03-21-evidence-plan")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state != nil {
		t.Fatalf("expected publish submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestPublish(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest publish record: %v", err)
	}
	if record == nil || record.Status != "recorded" || record.PRURL != "https://github.com/catu-ai/easyharness/pull/99" {
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
	if state != nil {
		t.Fatalf("expected sync submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")
	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.Status != "not_applied" {
		t.Fatalf("unexpected sync record: %#v", record)
	}
}

func TestSubmitSyncRejectsWrongHeadRefType(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := evidence.Service{Workdir: root}.Submit("sync", []byte(`{
		"status":"fresh",
		"head_ref":true
	}`))
	if result.OK {
		t.Fatalf("expected validation failure, got %#v", result)
	}
	assertEvidenceError(t, result, "input.head_ref")
}

func TestSubmitSyncFreshWritesArtifactWithoutStateCache(t *testing.T) {
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
	if state != nil {
		t.Fatalf("expected sync submit to avoid state cache writes, got %#v", state)
	}
	assertStateFileAbsent(t, root, "2026-03-21-evidence-plan")

	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
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
		Land: &runstate.LandState{
			PRURL:    "https://github.com/catu-ai/easyharness/pull/99",
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

func TestLoadLatestCIPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"pending","provider":"buildkite","url":"https://ci.example/1"}`))
	if !first.OK {
		t.Fatalf("expected first CI submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/2"}`))
	if !second.OK {
		t.Fatalf("expected second CI submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record == nil || record.RecordID != "ci-002" || record.Status != "success" || record.URL != "https://ci.example/2" {
		t.Fatalf("expected newest CI record to win, got %#v", record)
	}
}

func TestLoadLatestPublishPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99","branch":"codex/test","base":"main","commit":"abc123"}`))
	if !first.OK {
		t.Fatalf("expected first publish submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/100","branch":"codex/test-2","base":"main","commit":"def456"}`))
	if !second.OK {
		t.Fatalf("expected second publish submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestPublish(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest publish record: %v", err)
	}
	if record == nil || record.RecordID != "publish-002" || record.PRURL != "https://github.com/catu-ai/easyharness/pull/100" || record.Commit != "def456" {
		t.Fatalf("expected newest publish record to win, got %#v", record)
	}
}

func TestLoadLatestSyncPrefersNewestRecord(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"stale","base_ref":"main","head_ref":"codex/test"}`))
	if !first.OK {
		t.Fatalf("expected first sync submit success, got %#v", first)
	}
	second := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 3, 0, 0, time.UTC)
		},
	}.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test-2"}`))
	if !second.OK {
		t.Fatalf("expected second sync submit success, got %#v", second)
	}

	record, err := evidence.LoadLatestSync(root, "2026-03-21-evidence-plan", 1)
	if err != nil {
		t.Fatalf("load latest sync record: %v", err)
	}
	if record == nil || record.RecordID != "sync-002" || record.Status != "fresh" || record.HeadRef != "codex/test-2" {
		t.Fatalf("expected newest sync record to win, got %#v", record)
	}
}

func TestLoadLatestRecordIgnoresOlderRevisionEvidence(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeArchivedPlan(t, root, "docs/plans/archived/2026-03-21-evidence-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	first := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
		},
	}.Submit("ci", []byte(`{"status":"success","provider":"buildkite","url":"https://ci.example/1"}`))
	if !first.OK {
		t.Fatalf("expected first CI submit success, got %#v", first)
	}
	if _, err := runstate.SaveState(root, "2026-03-21-evidence-plan", &runstate.State{Revision: 2}); err != nil {
		t.Fatalf("save reopened revision state: %v", err)
	}

	record, err := evidence.LoadLatestCI(root, "2026-03-21-evidence-plan", 2)
	if err != nil {
		t.Fatalf("load latest CI record: %v", err)
	}
	if record != nil {
		t.Fatalf("expected older revision evidence to be ignored, got %#v", record)
	}
}

func assertEvidenceError(t *testing.T, result evidence.Result, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected evidence error for %s, got %#v", path, result.Errors)
}

func assertStateFileAbsent(t *testing.T, root, planStem string) {
	t.Helper()
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected state.json to stay absent, got %v", err)
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
