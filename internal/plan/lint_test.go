package plan_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestLintFileAcceptsValidActivePlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Valid Active Plan")
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileAcceptsDoneMarkers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-done-marker-plan.md")
	content := mustRenderTemplate(t, "Done Marker Plan")
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected lint success, got %#v", result)
	}
}

func TestLintFileRejectsLegacyRuntimeFrontmatter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-legacy-frontmatter-plan.md")
	content := mustRenderTemplate(t, "Legacy Runtime Frontmatter")
	content = strings.Replace(content, "template_version: 0.2.0\n", "status: active\nlifecycle: awaiting_plan_approval\nrevision: 1\nupdated_at: 2026-03-17T14:00:00+08:00\ntemplate_version: 0.2.0\n", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.status")
	assertHasError(t, result, "frontmatter.lifecycle")
	assertHasError(t, result, "frontmatter.revision")
	assertHasError(t, result, "frontmatter.updated_at")
}

func TestLintFileRejectsMissingDeferredItemsSection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
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
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
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
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Placeholder Plan")
	content = strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")
	content = checkAllBoxes(content)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Validation Summary")
	assertHasError(t, result, "step.Step 1: Replace with first step title.Execution Notes")
}

func TestLintFileRejectsArchivedPlanWithUncheckedDoneMarker(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-archived-done-plan.md")
	content := mustRenderTemplate(t, "Archived Done Plan")
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 1)
	content = makeArchiveReady(checkAllBoxes(content))
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "step.Step 2: Replace with second step title.done")
}

func TestLintFileRejectsLegacyStepStatusMarkers(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-mixed-step-encodings.md")
	content := mustRenderTemplate(t, "Mixed Step Encodings")
	content = strings.Replace(content, "- Done: [ ]", "- Status: in_progress", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "step.Step 1: Replace with first step title")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutOutcomeSummaryWithoutPanic(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Missing Outcome Summary")
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = makeArchiveReady(checkAllBoxes(strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")))
	content = strings.Replace(content, "## Outcome Summary\n\n### Delivered\n\nShipped the planned slice.\n\n### Not Delivered\n\nNONE.\n\n### Follow-Up Issues\n\nNONE\n", "", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Outcome Summary")
}

func TestLintFileRejectsArchivedDeferredItemsWithoutFollowUpIssue(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/archived/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Archived Deferred Item Plan")
	content = strings.Replace(content, "- None.", "- `harness ui` is intentionally deferred.", 1)
	content = makeArchiveReady(checkAllBoxes(strings.ReplaceAll(content, "- Done: [ ]", "- Done: [x]")))
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatal("expected lint failure")
	}
	assertHasError(t, result, "section.Outcome Summary.Follow-Up Issues")
}

func TestLintFileAcceptsHistoricalTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Historical Template Version")
	content = strings.Replace(content, "template_version: 0.2.0", "template_version: 0.0.1", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected older template_version to remain valid, got %#v", result)
	}
}

func TestLintFileRejectsFutureTemplateVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
	content := mustRenderTemplate(t, "Future Template Version")
	content = strings.Replace(content, "template_version: 0.2.0", "template_version: 9.9.9", 1)
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

func TestLintFileAcceptsTrackedActiveLightweightPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Lightweight Tracked Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected tracked lightweight lint success, got %#v", result)
	}
}

func TestLintFileAcceptsArchivedLightweightLocalPlan(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local/harness/plans/archived/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Archived Lightweight Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	content = strings.Replace(content, "- Done: [ ]", "- Done: [x]", 3)
	content = strings.ReplaceAll(content, "- [ ]", "- [x]")
	content = strings.ReplaceAll(content, "PENDING_STEP_EXECUTION", "Completed lightweight closeout.")
	content = strings.ReplaceAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: archived fixture.")
	content = strings.ReplaceAll(content, "PENDING_UNTIL_ARCHIVE", "Archived fixture summary.")
	content = strings.Replace(content, "## Archive Summary\n\nArchived fixture summary.", "## Archive Summary\n\n- Archived At: 2026-03-17T12:00:00Z\n- Revision: 1\n- PR: NONE\n- Ready: Archived lightweight fixture is complete.\n- Merge Handoff: None for this lint fixture.", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if !result.OK {
		t.Fatalf("expected archived lightweight lint success, got %#v", result)
	}
}

func TestLintFileRejectsLightweightActivePlanUnderLocalPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local/harness/plans/2026-03-17-lightweight-plan/active/2026-03-17-lightweight-plan.md")
	content := mustRenderTemplate(t, "Bad Local Active Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: lightweight", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "path")
}

func TestLintFileRejectsUnsupportedWorkflowProfile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-bad-profile.md")
	content := mustRenderTemplate(t, "Bad Profile Plan")
	content = strings.Replace(content, "source_refs: []", "source_refs: []\nworkflow_profile: risky", 1)
	writeFile(t, path, content)

	result := plan.LintFile(path)
	if result.OK {
		t.Fatalf("expected lint failure, got %#v", result)
	}
	assertHasError(t, result, "frontmatter.workflow_profile")
}

func TestLintFileRejectsInvalidStepHeading(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs/plans/active/2026-03-17-easyharness-cli-and-plan-foundations.md")
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

func makeArchiveReady(content string) string {
	content = strings.ReplaceAll(content, "PENDING_STEP_EXECUTION", "Finished step execution notes.")
	content = strings.ReplaceAll(content, "PENDING_STEP_REVIEW", "Finished step review notes.")
	content = strings.Replace(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the planned slice.", 1)
	content = strings.Replace(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nNo unresolved blocking review findings remain.", 1)
	content = strings.Replace(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- Archived At: 2026-03-17T15:00:00+08:00\n- Revision: 1\n- PR: NONE\n- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.\n- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.", 1)
	content = strings.Replace(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nShipped the planned slice.", 1)
	content = strings.Replace(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.", 1)
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
