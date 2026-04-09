package evidence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestSubmitIgnoresStateSaveFailuresAndKeepsRecordArtifact(t *testing.T) {
	testCases := []struct {
		name       string
		kind       string
		input      string
		recordPath string
	}{
		{
			name:       "ci",
			kind:       "ci",
			input:      `{"status":"success","provider":"buildkite","url":"https://ci.example/123"}`,
			recordPath: filepath.Join(".local", "harness", "plans", "2026-04-01-evidence-rollback", "evidence", "ci", "ci-001.json"),
		},
		{
			name:       "publish",
			kind:       "publish",
			input:      `{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/93","branch":"codex/issue-93","base":"main","commit":"abc123def456"}`,
			recordPath: filepath.Join(".local", "harness", "plans", "2026-04-01-evidence-rollback", "evidence", "publish", "publish-001.json"),
		},
		{
			name:       "sync",
			kind:       "sync",
			input:      `{"status":"fresh","base_ref":"main","head_ref":"codex/issue-93"}`,
			recordPath: filepath.Join(".local", "harness", "plans", "2026-04-01-evidence-rollback", "evidence", "sync", "sync-001.json"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			relPlanPath := writeArchivedPlanFixture(t, root, "docs/plans/archived/2026-04-01-evidence-rollback.md")
			if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
				t.Fatalf("save current plan: %v", err)
			}

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
					return time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)
				},
			}.Submit(tc.kind, []byte(tc.input))
			if !result.OK {
				t.Fatalf("expected evidence submit success, got %#v", result)
			}

			recordPath := filepath.Join(root, tc.recordPath)
			if _, err := os.Stat(recordPath); err != nil {
				t.Fatalf("expected evidence record to remain on disk, got %v", err)
			}

			state, statePath, err := runstate.LoadState(root, "2026-04-01-evidence-rollback")
			if err != nil {
				t.Fatalf("load state: %v", err)
			}
			if state != nil {
				t.Fatalf("expected no persisted state cache writes, got %#v", state)
			}
			if statePath != "" {
				if _, err := os.Stat(statePath); !os.IsNotExist(err) {
					t.Fatalf("expected no state file to be written, got %v", err)
				}
			}
		})
	}
}

func writeArchivedPlanFixture(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Evidence Rollback Fixture",
		Timestamp:  time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC),
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
