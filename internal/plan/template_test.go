package plan_test

import (
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/plan"
)

func TestRenderTemplateSeedsFields(t *testing.T) {
	timestamp := time.Date(2026, 3, 17, 13, 50, 0, 0, time.FixedZone("CST", 8*60*60))
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "Superharness Test Plan",
		Timestamp:  timestamp,
		SourceType: "issue",
		SourceRefs: []string{"#12", "https://example.com/item"},
	})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}

	for _, want := range []string{
		"# Superharness Test Plan",
		"created_at: 2026-03-17T13:50:00+08:00",
		"source_type: issue",
		`source_refs: ["#12","https://example.com/item"]`,
		"template_version: 0.2.0",
		"- Done: [ ]",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered template missing %q\n%s", want, rendered)
		}
	}
}

func TestRenderTemplateRejectsMultilineTitle(t *testing.T) {
	_, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "line one\nline two",
	})
	if err == nil {
		t.Fatal("expected multiline title to fail")
	}
}

func TestRenderTemplateUsesEmptyArrayForMissingSourceRefs(t *testing.T) {
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title: "Nil Refs Plan",
	})
	if err != nil {
		t.Fatalf("RenderTemplate returned error: %v", err)
	}
	if !strings.Contains(rendered, "source_refs: []") {
		t.Fatalf("expected empty source_refs array, got:\n%s", rendered)
	}
}
