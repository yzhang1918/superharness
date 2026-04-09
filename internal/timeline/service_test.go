package timeline_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/timeline"
)

func TestReadLoadsCurrentPlanTimelineEvents(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-04-01-timeline-plan", &runstate.State{
		ExecutionStartedAt: "2026-04-01T10:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "lifecycle",
		Command:    "execute start",
		Summary:    "Execution started for the current active plan.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ToNode:     "execution/step-1/implement",
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected timeline read success, got %#v", result)
	}
	if result.Resource != "timeline" {
		t.Fatalf("expected timeline resource, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus recorded event, got %#v", result.Events)
	}
	if result.Events[0].Command != "plan" {
		t.Fatalf("expected leading plan event, got %#v", result.Events[0])
	}
	if result.Events[1].Command != "execute start" {
		t.Fatalf("unexpected event order: %#v", result.Events)
	}
	if result.Artifacts == nil || !stringsHasSuffix(result.Artifacts.EventIndexPath, "events.jsonl") {
		t.Fatalf("expected event index artifact, got %#v", result.Artifacts)
	}
}

func TestReadReturnsEmptyTimelineWithoutCurrentPlan(t *testing.T) {
	result := timeline.Service{Workdir: t.TempDir()}.Read()
	if !result.OK {
		t.Fatalf("expected empty timeline success, got %#v", result)
	}
	if len(result.Events) != 0 {
		t.Fatalf("expected no events, got %#v", result.Events)
	}
}

func TestReadLoadsEventsWhenStateCacheIsMissing(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "lifecycle",
		Command:    "execute start",
		Summary:    "Execution started for the current active plan.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ToNode:     "execution/step-1/implement",
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected timeline read success without state cache, got %#v", result)
	}
	if len(result.Events) != 2 || result.Events[0].Command != "plan" || result.Events[1].Command != "execute start" {
		t.Fatalf("unexpected timeline events without state cache: %#v", result.Events)
	}
	if result.Artifacts == nil || result.Artifacts.LocalStatePath == "" {
		t.Fatalf("expected local state path artifact even when cache is missing, got %#v", result.Artifacts)
	}
}

func TestReadSynthesizesImplementBootstrapWhenExecutionStartedPredatesEventIndex(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-04-01-timeline-plan", &runstate.State{
		ExecutionStartedAt: "2026-04-01T10:00:00Z",
		Revision:           1,
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected synthesized bootstrap timeline read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected plan and implement bootstrap events, got %#v", result.Events)
	}
	if result.Events[0].Command != "plan" || result.Events[1].Command != "implement" {
		t.Fatalf("expected plan then implement bootstrap events, got %#v", result.Events)
	}
	if len(result.Events[1].Output) == 0 {
		t.Fatalf("expected raw output payload on bootstrap implement event, got %#v", result.Events[1])
	}
}

func TestReadLoadsLargeTimelineEventPayload(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	rawOutput, err := json.Marshal(map[string]string{
		"blob": strings.Repeat("x", 2*1024*1024),
	})
	if err != nil {
		t.Fatalf("marshal large output: %v", err)
	}
	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review submit",
		Summary:    "Recorded large review submission payload.",
		PlanPath:   relPlanPath,
		Revision:   1,
		Output:     rawOutput,
	}); err != nil {
		t.Fatalf("append large timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected large timeline payload read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus large recorded event, got %#v", result.Events)
	}
	if result.Events[1].Command != "review submit" {
		t.Fatalf("unexpected large payload event: %#v", result.Events[1])
	}
	if len(result.Events[1].Output) == 0 {
		t.Fatalf("expected output payload on large event, got %#v", result.Events[1])
	}
	var decoded map[string]string
	if err := json.Unmarshal(result.Events[1].Output, &decoded); err != nil {
		t.Fatalf("unmarshal large output payload: %v", err)
	}
	if decoded["blob"] != strings.Repeat("x", 2*1024*1024) {
		t.Fatalf("expected large payload integrity to survive round-trip, got %d bytes", len(decoded["blob"]))
	}
}

func TestReadResolvesArtifactRefFileContents(t *testing.T) {
	root := t.TempDir()
	relPlanPath := writeActivePlanForTimeline(t, root, "docs/plans/active/2026-04-01-timeline-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	manifestPath := filepath.Join(root, ".local", "harness", "plans", "2026-04-01-timeline-plan", "reviews", "review-001-full", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte("{\"review_title\":\"Timeline artifacts\"}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	if _, _, err := timeline.AppendEvent(root, "2026-04-01-timeline-plan", timeline.Event{
		RecordedAt: "2026-04-01T10:00:00Z",
		Kind:       "review",
		Command:    "review start",
		Summary:    "Created review round.",
		PlanPath:   relPlanPath,
		Revision:   1,
		ArtifactRefs: []timeline.ArtifactRef{
			{Label: "round_id", Value: "review-001-full"},
			{Label: "manifest_path", Value: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/manifest.json", Path: ".local/harness/plans/2026-04-01-timeline-plan/reviews/review-001-full/manifest.json"},
		},
	}); err != nil {
		t.Fatalf("append timeline event: %v", err)
	}

	result := timeline.Service{Workdir: root}.Read()
	if !result.OK {
		t.Fatalf("expected resolved artifact timeline read success, got %#v", result)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus review start event, got %#v", result.Events)
	}
	refs := result.Events[1].ArtifactRefs
	if len(refs) != 2 {
		t.Fatalf("expected artifact refs to survive read, got %#v", refs)
	}
	if len(refs[0].Content) != 0 {
		t.Fatalf("expected value-only ref to remain unresolved, got %#v", refs[0])
	}
	if refs[1].ContentType != "json" {
		t.Fatalf("expected resolved manifest ref content type json, got %#v", refs[1])
	}
	var decoded map[string]string
	if err := json.Unmarshal(refs[1].Content, &decoded); err != nil {
		t.Fatalf("unmarshal resolved manifest content: %v", err)
	}
	if decoded["review_title"] != "Timeline artifacts" {
		t.Fatalf("unexpected manifest content: %#v", decoded)
	}
}

func writeActivePlanForTimeline(t *testing.T, root, relPath string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: "Timeline Test Plan"})
	if err != nil {
		t.Fatalf("render plan template: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	return relPath
}

func stringsHasSuffix(value, suffix string) bool {
	return len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix
}
