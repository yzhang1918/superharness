package review

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestStartRemovesRoundArtifactsWhenStateSaveFails(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-04-01-review-rollback.md"
	writeExecutingPlanFixture(t, root, relPath)

	originalSaveState := saveState
	saveState = func(string, string, *runstate.State) (string, error) {
		return "", errors.New("boom")
	}
	t.Cleanup(func() {
		saveState = originalSaveState
	})

	result := Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		},
	}.Start(mustJSONBytes(t, Spec{
		Kind: "delta",
		Dimensions: []Dimension{
			{Name: "correctness", Instructions: "Check rollback behavior."},
		},
	}))
	if result.OK {
		t.Fatalf("expected review start failure, got %#v", result)
	}
	assertCommandErrorPath(t, result.Errors, "state")

	roundDir := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-review-rollback", "reviews", "review-001-delta")
	if _, err := os.Stat(roundDir); !os.IsNotExist(err) {
		t.Fatalf("expected round directory to be removed on rollback, got %v", err)
	}

	state, _, err := runstate.LoadState(root, "2026-04-01-review-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound != nil {
		t.Fatalf("expected original state to be restored, got %#v", state)
	}
}

func TestStartRemovesRoundArtifactsWhenLedgerWriteFails(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-04-01-review-ledger-rollback.md"
	writeExecutingPlanFixture(t, root, relPath)

	originalWriteJSON := writeJSON
	writeJSON = func(path string, value any) error {
		if filepath.Base(path) == "ledger.json" {
			return errors.New("boom")
		}
		return writeJSONFile(path, value)
	}
	t.Cleanup(func() {
		writeJSON = originalWriteJSON
	})

	result := Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		},
	}.Start(mustJSONBytes(t, Spec{
		Kind: "delta",
		Dimensions: []Dimension{
			{Name: "correctness", Instructions: "Check rollback behavior."},
		},
	}))
	if result.OK {
		t.Fatalf("expected review start failure, got %#v", result)
	}
	assertCommandErrorPath(t, result.Errors, "ledger")

	roundDir := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-review-ledger-rollback", "reviews", "review-001-delta")
	if _, err := os.Stat(roundDir); !os.IsNotExist(err) {
		t.Fatalf("expected round directory to be removed on rollback, got %v", err)
	}
}

func TestAggregateRestoresPreviousAggregateWhenStateSaveFails(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-04-01-review-aggregate-rollback.md"
	writeExecutingPlanFixture(t, root, relPath)

	svc := Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSONBytes(t, Spec{
		Kind: "delta",
		Dimensions: []Dimension{
			{Name: "correctness", Instructions: "Check aggregate rollback behavior."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	svc.Now = func() time.Time {
		return time.Date(2026, 4, 1, 10, 1, 0, 0, time.UTC)
	}
	submit := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSONBytes(t, SubmissionInput{
		Summary: "Looks good.",
	}))
	if !submit.OK {
		t.Fatalf("submit failed: %#v", submit)
	}

	aggregatePath := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-review-aggregate-rollback", "reviews", start.Artifacts.RoundID, "aggregate.json")
	previousAggregate := []byte("{\n  \"decision\": \"stale\"\n}\n")
	if err := os.WriteFile(aggregatePath, previousAggregate, 0o644); err != nil {
		t.Fatalf("seed aggregate snapshot: %v", err)
	}

	originalSaveState := saveState
	saveState = func(string, string, *runstate.State) (string, error) {
		return "", errors.New("boom")
	}
	t.Cleanup(func() {
		saveState = originalSaveState
	})

	svc.Now = func() time.Time {
		return time.Date(2026, 4, 1, 10, 2, 0, 0, time.UTC)
	}
	result := svc.Aggregate(start.Artifacts.RoundID)
	if result.OK {
		t.Fatalf("expected aggregate failure, got %#v", result)
	}
	assertCommandErrorPath(t, result.Errors, "state")

	restoredAggregate, err := os.ReadFile(aggregatePath)
	if err != nil {
		t.Fatalf("read restored aggregate: %v", err)
	}
	if string(restoredAggregate) != string(previousAggregate) {
		t.Fatalf("expected previous aggregate to be restored, got %q", string(restoredAggregate))
	}

	state, _, err := runstate.LoadState(root, "2026-04-01-review-aggregate-rollback")
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if state == nil || state.ActiveReviewRound == nil || state.ActiveReviewRound.Aggregated {
		t.Fatalf("expected original active review state to be restored, got %#v", state)
	}
}

func TestSubmitRestoresSubmissionWhenLedgerWriteFails(t *testing.T) {
	root := t.TempDir()
	relPath := "docs/plans/active/2026-04-01-review-submit-rollback.md"
	writeExecutingPlanFixture(t, root, relPath)

	svc := Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
		},
	}
	start := svc.Start(mustJSONBytes(t, Spec{
		Kind: "delta",
		Dimensions: []Dimension{
			{Name: "correctness", Instructions: "Check aggregate rollback behavior."},
		},
	}))
	if !start.OK {
		t.Fatalf("start failed: %#v", start)
	}

	manifestPath := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-review-submit-rollback", "reviews", start.Artifacts.RoundID, "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}

	originalWriteJSON := writeJSON
	writeJSON = func(path string, value any) error {
		if path == manifest.LedgerPath {
			return errors.New("boom")
		}
		return writeJSONFile(path, value)
	}
	t.Cleanup(func() {
		writeJSON = originalWriteJSON
	})

	svc.Now = func() time.Time {
		return time.Date(2026, 4, 1, 10, 1, 0, 0, time.UTC)
	}
	result := svc.Submit(start.Artifacts.RoundID, "correctness", mustJSONBytes(t, SubmissionInput{
		Summary: "Looks good.",
	}))
	if result.OK {
		t.Fatalf("expected submit failure, got %#v", result)
	}
	assertCommandErrorPath(t, result.Errors, "ledger")

	if _, err := os.Stat(manifest.Dimensions[0].SubmissionPath); !os.IsNotExist(err) {
		t.Fatalf("expected submission artifact to be removed on rollback, got %v", err)
	}
}

func writeExecutingPlanFixture(t *testing.T, root, relPath string) {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Review Rollback Fixture",
		Timestamp:  time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC),
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

	planStem := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	if _, err := runstate.SaveState(root, planStem, &runstate.State{
		ExecutionStartedAt: "2026-04-01T09:30:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save execute-start state: %v", err)
	}
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return data
}

func assertCommandErrorPath(t *testing.T, issues []CommandError, path string) {
	t.Helper()
	for _, issue := range issues {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected error for %s, got %#v", path, issues)
}
