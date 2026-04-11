package plan_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

type canonicalPlanSeed struct {
	name                 string
	relPath              string
	content              string
	wantCurrentStepTitle string
	wantAllChecked       bool
	wantAllCompleted     bool
	wantArchiveReady     bool
}

func TestCanonicalPlanSeedsKeepLintAndLoadAligned(t *testing.T) {
	for _, seed := range canonicalPlanSeeds(t) {
		t.Run(seed.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(seed.relPath))
			writeFile(t, path, seed.content)

			result := plan.LintFile(path)
			if !result.OK {
				t.Fatalf("expected canonical seed to lint clean, got %#v", result)
			}

			doc, err := plan.LoadFile(path)
			if err != nil {
				t.Fatalf("expected canonical seed to load, got %v", err)
			}
			if doc == nil {
				t.Fatal("expected document")
			}

			current := doc.CurrentStep()
			if seed.wantCurrentStepTitle == "" {
				if current != nil {
					t.Fatalf("expected no current step, got %#v", current)
				}
			} else {
				if current == nil || current.Title != seed.wantCurrentStepTitle {
					t.Fatalf("unexpected current step: %#v", current)
				}
			}

			if got := doc.AllAcceptanceChecked(); got != seed.wantAllChecked {
				t.Fatalf("AllAcceptanceChecked mismatch: got %v want %v", got, seed.wantAllChecked)
			}
			if got := doc.AllStepsCompleted(); got != seed.wantAllCompleted {
				t.Fatalf("AllStepsCompleted mismatch: got %v want %v", got, seed.wantAllCompleted)
			}

			archiveReady := !doc.HasPendingArchivePlaceholders() && !doc.CompletedStepsHavePendingPlaceholders()
			if archiveReady != seed.wantArchiveReady {
				t.Fatalf("archive readiness mismatch: got %v want %v", archiveReady, seed.wantArchiveReady)
			}
		})
	}
}

func TestTrackedPlanCorpusKeepsLintAndLoadAligned(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "docs", "plans", "*", "*.md"))
	if err != nil {
		t.Fatalf("glob tracked plans: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("expected tracked plans in repository")
	}

	for _, path := range paths {
		result := plan.LintFile(path)
		if !result.OK {
			t.Fatalf("expected tracked plan %s to lint clean, got %#v", path, result)
		}
		if _, err := plan.LoadFile(path); err != nil {
			t.Fatalf("expected tracked plan %s to load cleanly after lint success, got %v", path, err)
		}
	}
}

func FuzzLintFileAndLoadFileAgreement(f *testing.F) {
	canonicalByKey := map[canonicalFuzzKey]canonicalPlanSeed{}
	for _, seed := range canonicalPlanSeeds(f) {
		key := canonicalFuzzKey{
			mode:    uint8(modeFromPlanPath(seed.relPath)),
			content: seed.content,
		}
		canonicalByKey[key] = seed
		f.Add(uint8(modeFromPlanPath(seed.relPath)), seed.content)
	}
	f.Add(uint8(0), "not a plan")
	f.Add(uint8(1), "---\ncreated_at: nope\n")
	f.Add(uint8(2), strings.Repeat("### Step 1: fuzz\n", 32))

	f.Fuzz(func(t *testing.T, mode uint8, content string) {
		if len(content) > 1<<16 {
			t.Skip()
		}

		root := t.TempDir()
		path := filepath.Join(root, filepath.FromSlash(relPathForMode(mode)))
		writeFile(t, path, content)

		result := plan.LintFile(path)
		doc, err := plan.LoadFile(path)
		if seed, ok := canonicalByKey[canonicalFuzzKey{mode: mode % 3, content: content}]; ok {
			if !result.OK {
				t.Fatalf("canonical seed %q stopped linting cleanly: %#v", seed.name, result)
			}
			if err != nil {
				t.Fatalf("canonical seed %q stopped loading cleanly: %v", seed.name, err)
			}
		}

		if result.OK && err != nil {
			t.Fatalf("lint succeeded but load failed: %v (result=%#v)", err, result)
		}
		if err != nil {
			return
		}
		if doc == nil {
			t.Fatal("expected non-nil document when load succeeds")
		}

		current := doc.CurrentStep()
		if current != nil && !documentContainsStep(doc, current.Title) {
			t.Fatalf("current step %q was not found in document steps %#v", current.Title, doc.Steps)
		}
		if doc.AllStepsCompleted() && current != nil {
			t.Fatalf("expected no current step once all steps are completed, got %#v", current)
		}

		_ = doc.AllAcceptanceChecked()
		_ = doc.HasPendingArchivePlaceholders()
		_ = doc.CompletedStepsHavePendingPlaceholders()
	})
}

type canonicalFuzzKey struct {
	mode    uint8
	content string
}

func canonicalPlanSeeds(tb testing.TB) []canonicalPlanSeed {
	tb.Helper()

	active := renderCanonicalTemplate(tb, "Canonical Active Plan")
	archived := makeArchiveReady(checkAllBoxes(strings.ReplaceAll(renderCanonicalTemplate(tb, "Canonical Archived Plan"), "- Done: [ ]", "- Done: [x]")))
	lightweight := renderCanonicalTemplate(tb, "Canonical Lightweight Archived Plan")
	lightweight = strings.Replace(lightweight, "size: M", "size: XXS", 1)
	lightweight = strings.Replace(lightweight, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	lightweight = strings.ReplaceAll(lightweight, "- Done: [ ]", "- Done: [x]")
	lightweight = checkAllBoxes(lightweight)
	lightweight = strings.ReplaceAll(lightweight, "PENDING_STEP_EXECUTION", "Completed lightweight execution notes.")
	lightweight = strings.ReplaceAll(lightweight, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: lightweight canonical seed.")
	lightweight = strings.ReplaceAll(lightweight, "PENDING_UNTIL_ARCHIVE", "Archived lightweight seed summary.")
	lightweight = strings.Replace(lightweight, "## Archive Summary\n\nArchived lightweight seed summary.", "## Archive Summary\n\n- Archived At: 2026-03-17T12:00:00Z\n- Revision: 1\n- PR: NONE\n- Ready: Archived lightweight canonical seed is complete.\n- Merge Handoff: None for this lightweight canonical seed.", 1)

	return []canonicalPlanSeed{
		{
			name:                 "active",
			relPath:              "docs/plans/active/2026-03-17-canonical-plan.md",
			content:              active,
			wantCurrentStepTitle: "Step 1: Replace with first step title",
			wantAllChecked:       false,
			wantAllCompleted:     false,
			wantArchiveReady:     false,
		},
		{
			name:                 "archived",
			relPath:              "docs/plans/archived/2026-03-17-canonical-plan.md",
			content:              archived,
			wantCurrentStepTitle: "",
			wantAllChecked:       true,
			wantAllCompleted:     true,
			wantArchiveReady:     true,
		},
		{
			name:                 "archived_lightweight",
			relPath:              ".local/harness/plans/archived/2026-03-17-canonical-lightweight-plan.md",
			content:              lightweight,
			wantCurrentStepTitle: "",
			wantAllChecked:       true,
			wantAllCompleted:     true,
			wantArchiveReady:     true,
		},
	}
}

func renderCanonicalTemplate(tb testing.TB, title string) string {
	tb.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 17, 14, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		SourceType: "direct_request",
		Size:       "M",
	})
	if err != nil {
		tb.Fatalf("render template: %v", err)
	}
	return rendered
}

func modeFromPlanPath(relPath string) int {
	switch {
	case strings.HasPrefix(relPath, ".local/harness/plans/archived/"):
		return 2
	case strings.Contains(relPath, "/archived/"):
		return 1
	default:
		return 0
	}
}

func relPathForMode(mode uint8) string {
	switch mode % 3 {
	case 1:
		return "docs/plans/archived/2026-03-17-fuzz-plan.md"
	case 2:
		return ".local/harness/plans/archived/2026-03-17-fuzz-plan.md"
	default:
		return "docs/plans/active/2026-03-17-fuzz-plan.md"
	}
}

func documentContainsStep(doc *plan.Document, title string) bool {
	for _, step := range doc.Steps {
		if step.Title == title {
			return true
		}
	}
	return false
}
