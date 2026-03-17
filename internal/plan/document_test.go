package plan_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
)

func TestLoadFileParsesCurrentStepAndDeferredItems(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-status-smoke-plan.md")
	content := mustRenderTemplate(t, "Status Smoke Plan")
	content = strings.Replace(content, "- Status: pending", "- Status: in_progress", 1)
	content = strings.Replace(content, "- None.", "- `harness ui` remains deferred.", 1)
	writeFile(t, path, content)

	doc, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}
	if doc.CurrentStep() == nil || doc.CurrentStep().Title != "Step 1: Replace with first step title" {
		t.Fatalf("unexpected current step: %#v", doc.CurrentStep())
	}
	if !doc.DeferredItems {
		t.Fatal("expected deferred items to be detected")
	}
}

func TestDocumentReadyForArchiveSignals(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-ready-plan.md")
	content := mustRenderTemplate(t, "Ready Plan")
	content = strings.ReplaceAll(content, "- Status: pending", "- Status: completed")
	content = checkAllBoxes(content)
	content = strings.ReplaceAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = strings.ReplaceAll(content, "PENDING_STEP_REVIEW", "Reviewed.")
	content = strings.ReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Ready.")
	writeFile(t, path, content)

	doc, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}
	if !doc.AllStepsCompleted() || !doc.AllAcceptanceChecked() {
		t.Fatal("expected document to be complete")
	}
	if doc.HasPendingArchivePlaceholders() || doc.CompletedStepsHavePendingPlaceholders() {
		t.Fatal("expected document to be archive-ready")
	}
}

func TestLoadFilePreservesFrontmatter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-18-frontmatter-plan.md")
	content, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Frontmatter Plan",
		Timestamp:  time.Date(2026, 3, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
		SourceType: "issue",
		SourceRefs: []string{"#9"},
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	writeFile(t, path, content)

	doc, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}
	if doc.Frontmatter.SourceType != "issue" || len(doc.Frontmatter.SourceRefs) != 1 || doc.Frontmatter.SourceRefs[0] != "#9" {
		t.Fatalf("unexpected frontmatter: %#v", doc.Frontmatter)
	}
}
