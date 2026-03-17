package plan_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yzhang1918/superharness/internal/plan"
)

func TestLintFileAcceptsValidActivePlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Valid Active Plan")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileRejectsMissingDeferredItemsSection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Invalid Active Plan")
	content = strings.Replace(content, "## Deferred Items\n\n- None.\n\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "sections")
}

func TestLintFileRejectsMissingAcceptanceCriteriaSectionWithoutPanic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Missing Acceptance Criteria")
	content = strings.Replace(content, "## Acceptance Criteria\n\n- [ ] Criterion 1\n- [ ] Criterion 2\n\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Acceptance Criteria")
}

func TestLintFileRejectsArchivedPlanWithPlaceholders(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Placeholder Plan")
	content = strings.Replace(content, "status: active", "status: archived", 1)
	content = strings.Replace(content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval", 1)
	content = strings.ReplaceAll(content, "- Status: pending", "- Status: completed")
	content = checkAllBoxes(content)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Validation Summary")
	assertHasError(t, result, "step.Step 1: Replace with first step title.Execution Notes")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutOutcomeSummaryWithoutPanic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Missing Outcome Summary")
	content = strings.Replace(content, "status: active", "status: archived", 1)
	content = strings.Replace(content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval", 1)
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = checkAllBoxes(strings.ReplaceAll(content, "- Status: pending", "- Status: completed"))
	content = strings.ReplaceAll(content, "PENDING_STEP_EXECUTION", "Finished step execution notes.")
	content = strings.ReplaceAll(content, "PENDING_STEP_REVIEW", "Finished step review notes.")
	content = strings.ReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Archive-ready summary.")
	content = strings.Replace(content, "## Outcome Summary\n\n### Delivered\n\nArchive-ready summary.\n\n### Not Delivered\n\nArchive-ready summary.\n\n### Follow-Up Issues\n\nNONE\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Outcome Summary")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutFollowUpIssue(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Deferred Item Plan")
	content = strings.Replace(content, "status: active", "status: archived", 1)
	content = strings.Replace(content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval", 1)
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = checkAllBoxes(strings.ReplaceAll(content, "- Status: pending", "- Status: completed"))
	content = strings.Replace(content, "PENDING_STEP_EXECUTION", "Finished step execution notes.", -1)
	content = strings.Replace(content, "PENDING_STEP_REVIEW", "Finished step review notes.", -1)
	content = strings.Replace(content, "PENDING_UNTIL_ARCHIVE", "Archive-ready summary.", -1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Outcome Summary.Follow-Up Issues")
}

func TestLintFileAcceptsHistoricalTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Historical Template Version")
	content = strings.Replace(content, "template_version: 0.1.0", "template_version: 0.0.1", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected older template_version to remain valid, got %#v", result)
	}
}

func TestLintFileRejectsFutureTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Future Template Version")
	content = strings.Replace(content, "template_version: 0.1.0", "template_version: 9.9.9", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "frontmatter.template_version")
}

func TestLintFileRejectsInvalidFilename(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/not-a-valid-name.md")
	content := mustRenderTemplate(t, "Bad Filename")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "path")
}

func TestLintFileRejectsInvalidStepHeading(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-superharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Bad Step Heading")
	content = strings.Replace(content, "### Step 1: Replace with first step title", "### Step banana", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Work Breakdown")
}

func TestLintResultJSONRoundTrip(t *testing.T) {
	result := plan.LintFile(filepath.Join(t.TempDir(), "missing.md"))
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal lint result: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected JSON output")
	}
}

func mustRenderTemplate(t *testing.T, title string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      title,
		Timestamp:  time.Date(2026, 3, 17, 14, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
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

func checkAllBoxes(content string) string {
	content = strings.ReplaceAll(content, "- [ ]", "- [x]")
	return content
}

func assertHasError(t *testing.T, result plan.LintResult, path string) {
	t.Helper()
	for _, issue := range result.Errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected error for %s, got %#v", path, result.Errors)
}
