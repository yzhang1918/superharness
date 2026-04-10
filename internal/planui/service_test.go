package planui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestServiceReadReturnsIdleEmptyStateWithoutCurrentPlan(t *testing.T) {
	result := Service{Workdir: t.TempDir()}.Read()

	if !result.OK || result.Resource != "plan" {
		t.Fatalf("unexpected idle result: %#v", result)
	}
	if result.Document != nil {
		t.Fatalf("expected no document for idle worktree, got %#v", result.Document)
	}
	if result.Supplements != nil {
		t.Fatalf("expected no supplements for idle worktree, got %#v", result.Supplements)
	}
	if !strings.Contains(result.Summary, "No current active plan") {
		t.Fatalf("unexpected idle summary: %q", result.Summary)
	}
}

func TestServiceReadLoadsActivePlanPackageAndPreviewStates(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/active/2026-04-10-plan-page.md"
	planPath := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	content, err := plan.RenderTemplate(plan.TemplateOptions{Title: "Plan Browser Demo"})
	if err != nil {
		t.Fatalf("render plan template: %v", err)
	}
	content = strings.Replace(
		content,
		"Describe the intended outcome in one or two short paragraphs.",
		"Read the plan package comfortably.\n\n```md\n## Not a real heading\n```\n",
		1,
	)
	if err := os.WriteFile(planPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(workdir, "2026-04-10-plan-page", &runstate.State{Revision: 1}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	supplementsDir := filepath.Join(workdir, "docs", "plans", "active", "supplements", "2026-04-10-plan-page")
	if err := os.MkdirAll(filepath.Join(supplementsDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir supplements: %v", err)
	}
	mustWriteFile(t, filepath.Join(supplementsDir, "design.md"), []byte("# Design\n\nReader notes.\n"))
	mustWriteFile(t, filepath.Join(supplementsDir, "notes.log"), []byte("line one\nline two\n"))
	mustWriteFile(t, filepath.Join(supplementsDir, "data.json"), []byte("{\"ok\":true}\n"))
	mustWriteFile(t, filepath.Join(supplementsDir, "nested", "config.yaml"), []byte("theme: slate\n"))
	mustWriteFile(t, filepath.Join(supplementsDir, "image.png"), []byte{})
	mustWriteFile(t, filepath.Join(supplementsDir, "binary.bin"), []byte{0xff, 0x00, 0x01})
	mustWriteFile(t, filepath.Join(supplementsDir, "large.txt"), []byte(strings.Repeat("x", int(maxPreviewBytes)+1)))

	result := Service{Workdir: workdir}.Read()

	if !result.OK || result.Resource != "plan" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Document == nil {
		t.Fatal("expected document payload")
	}
	if result.Document.Title != "Plan Browser Demo" {
		t.Fatalf("unexpected document title: %#v", result.Document)
	}
	if result.Document.Path != relPlanPath {
		t.Fatalf("unexpected document path: %#v", result.Document)
	}
	if strings.Contains(result.Document.Markdown, "template_version") {
		t.Fatalf("expected frontmatter-free markdown body, got %q", result.Document.Markdown)
	}
	if len(result.Document.Headings) < 5 {
		t.Fatalf("expected multiple top-level headings, got %#v", result.Document.Headings)
	}
	if result.Document.Headings[0].Label != "Goal" {
		t.Fatalf("unexpected goal heading tree: %#v", result.Document.Headings[0])
	}
	if result.Document.Headings[1].Label != "Scope" || len(result.Document.Headings[1].Children) != 2 {
		t.Fatalf("unexpected scope heading tree: %#v", result.Document.Headings[1])
	}
	if result.Document.Headings[1].Children[0].Label != "In Scope" || result.Document.Headings[1].Children[1].Label != "Out of Scope" {
		t.Fatalf("unexpected nested scope tree: %#v", result.Document.Headings[1].Children)
	}
	workBreakdown := findHeading(result.Document.Headings, "Work Breakdown")
	if workBreakdown == nil || len(workBreakdown.Children) == 0 || !strings.HasPrefix(workBreakdown.Children[0].Label, "Step 1:") {
		t.Fatalf("unexpected work breakdown tree: %#v", workBreakdown)
	}
	if result.Supplements == nil || result.Supplements.Label != "2026-04-10-plan-page" {
		t.Fatalf("expected supplements root, got %#v", result.Supplements)
	}
	if result.Artifacts == nil || !strings.HasSuffix(result.Artifacts.SupplementsPath, "/2026-04-10-plan-page") {
		t.Fatalf("expected supplements artifact path, got %#v", result.Artifacts)
	}

	design := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/design.md")
	if design.Preview == nil || design.Preview.Status != "supported" || design.Preview.ContentType != "markdown" {
		t.Fatalf("unexpected markdown preview: %#v", design)
	}
	notes := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/notes.log")
	if notes.Preview == nil || notes.Preview.Status != "fallback" || notes.Preview.ContentType != "text" {
		t.Fatalf("unexpected fallback preview: %#v", notes)
	}
	data := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/data.json")
	if data.Preview == nil || data.Preview.ContentType != "json" {
		t.Fatalf("unexpected json preview: %#v", data)
	}
	config := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/nested/config.yaml")
	if config.Preview == nil || config.Preview.ContentType != "yaml" {
		t.Fatalf("unexpected yaml preview: %#v", config)
	}
	image := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/image.png")
	if image.Preview == nil || image.Preview.Status != "not_supported" || !strings.Contains(image.Preview.Reason, "Image preview") {
		t.Fatalf("unexpected image preview: %#v", image)
	}
	binary := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/binary.bin")
	if binary.Preview == nil || binary.Preview.Status != "not_supported" || !strings.Contains(binary.Preview.Reason, "Binary or unsupported") {
		t.Fatalf("unexpected binary preview: %#v", binary)
	}
	large := findNode(t, result.Supplements, "docs/plans/active/supplements/2026-04-10-plan-page/large.txt")
	if large.Preview == nil || large.Preview.Status != "not_supported" || !strings.Contains(large.Preview.Reason, "preview limit") {
		t.Fatalf("unexpected oversize preview: %#v", large)
	}
}

func TestServiceReadHidesArchivedCurrentPlanFromBrowser(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath := "docs/plans/archived/2026-04-10-archived.md"
	planPath := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(planPath), 0o755); err != nil {
		t.Fatalf("mkdir archived dir: %v", err)
	}
	content, err := plan.RenderTemplate(plan.TemplateOptions{Title: "Archived"})
	if err != nil {
		t.Fatalf("render archived template: %v", err)
	}
	mustWriteFile(t, planPath, []byte(content))
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	result := Service{Workdir: workdir}.Read()
	if !result.OK || result.Document != nil {
		t.Fatalf("expected archived plan to return empty browser state, got %#v", result)
	}
	if !strings.Contains(result.Summary, "only shows the current active tracked plan") {
		t.Fatalf("unexpected archived summary: %q", result.Summary)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func findNode(t *testing.T, root *Node, path string) Node {
	t.Helper()
	if root == nil {
		t.Fatal("root is nil")
	}
	if root.Path == path {
		return *root
	}
	for _, child := range root.Children {
		if child.Path == path {
			return child
		}
		if child.Kind == "directory" {
			found := findNodeOrZero(&child, path)
			if found != nil {
				return *found
			}
		}
	}
	t.Fatalf("could not find node %s", path)
	return Node{}
}

func findNodeOrZero(root *Node, path string) *Node {
	if root.Path == path {
		return root
	}
	for index := range root.Children {
		child := &root.Children[index]
		if child.Path == path {
			return child
		}
		if child.Kind == "directory" {
			if found := findNodeOrZero(child, path); found != nil {
				return found
			}
		}
	}
	return nil
}

func findHeading(headings []Heading, label string) *Heading {
	for index := range headings {
		if headings[index].Label == label {
			return &headings[index]
		}
		if found := findHeading(headings[index].Children, label); found != nil {
			return found
		}
	}
	return nil
}
