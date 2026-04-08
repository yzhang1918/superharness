package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/catu-ai/easyharness/internal/cli"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/status"
	"github.com/catu-ai/easyharness/internal/timeline"
	version "github.com/catu-ai/easyharness/internal/version"
)

func TestPlanTemplateWritesOutputFile(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 17, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(t.TempDir(), "docs/plans/active/2026-03-17-test-plan.md")
	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
		"--source-type", "issue",
		"--source-ref", "#42",
	})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%s", exitCode, stderr.String())
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !bytes.Contains(data, []byte("# CLI Generated Plan")) {
		t.Fatalf("generated file missing title:\n%s", data)
	}
}

func TestPlanTemplateDateSeedsCurrentLocalTimeOfDay(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 25, 14, 15, 16, 0, time.FixedZone("CST", 8*60*60))
	}

	exitCode := app.Run([]string{
		"plan", "template",
		"--title", "Date Seeded Plan",
		"--date", "2026-03-20",
	})
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d, stderr=%s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "created_at: 2026-03-20T14:15:16+08:00") {
		t.Fatalf("expected date-seeded template to preserve current local time-of-day, got:\n%s", stdout.String())
	}
}

func TestVersionFlagPrintsHumanReadableDebugInfo(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Version = func() version.Info {
		return version.Info{
			Version: "v0.1.0-alpha.1",
			Commit:  "abc123",
			Mode:    "dev",
			Path:    "/tmp/harness",
		}
	}

	exitCode := app.Run([]string{"--version"})
	if exitCode != 0 {
		t.Fatalf("expected version exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr for version output, got %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "{") {
		t.Fatalf("expected non-JSON version output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "mode: dev") {
		t.Fatalf("expected mode in version output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "commit: abc123") {
		t.Fatalf("expected commit in version output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "path: /tmp/harness") {
		t.Fatalf("expected dev path in version output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "version: v0.1.0-alpha.1") {
		t.Fatalf("expected version in version output, got %q", stdout.String())
	}
}

func TestVersionFlagOmitsPathOutsideDevMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Version = func() version.Info {
		return version.Info{
			Commit: "abc123",
			Mode:   "release",
		}
	}

	exitCode := app.Run([]string{"--version"})
	if exitCode != 0 {
		t.Fatalf("expected version exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if strings.Contains(stdout.String(), "path:") {
		t.Fatalf("expected release version output to omit path, got %q", stdout.String())
	}
}

func TestVersionHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"--version", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected version help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness --version") {
		t.Fatalf("expected version help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for version help, got %q", stdout.String())
	}
}

func TestPlanLintCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 3, 17, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(t.TempDir(), "docs/plans/active/2026-03-17-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"plan", "lint", outputPath})
	if exitCode != 0 {
		t.Fatalf("lint command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON lint output: %v\n%s", err, stdout.String())
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
}

func TestPlanTemplateHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "template", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: harness plan template")) {
		t.Fatalf("expected help text, got %s", stderr.String())
	}
}

func TestPlanTemplateLightweightFlagSeedsLocalVariant(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "template", "--title", "Lightweight Plan", "--lightweight"})
	if exitCode != 0 {
		t.Fatalf("expected lightweight template success, got %d: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "workflow_profile: lightweight") {
		t.Fatalf("expected workflow_profile in template, got %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "### Step 2:") {
		t.Fatalf("expected lightweight template to collapse to one step, got %s", stdout.String())
	}
}

func TestRootHelpMentionsVersionFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"--help"})
	if exitCode != 0 {
		t.Fatalf("expected root help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness <command> [subcommand] [flags]") {
		t.Fatalf("expected root help usage, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--version       Print concise debug information for the running harness binary") {
		t.Fatalf("expected root help to mention --version, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "install         Install or refresh the harness-managed repository bootstrap") {
		t.Fatalf("expected root help to mention install, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "ui              Start the local read-only harness UI workbench") {
		t.Fatalf("expected root help to mention ui, got %q", stderr.String())
	}
}

func TestUIHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"ui", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected ui help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness ui") {
		t.Fatalf("expected ui help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for ui help, got %q", stdout.String())
	}
}

func TestUIRejectsPositionalArguments(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"ui", "extra"})
	if exitCode != 2 {
		t.Fatalf("expected ui positional-arg exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness ui") {
		t.Fatalf("expected ui usage on positional args, got %q", stderr.String())
	}
}

func TestInstallCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	exitCode := app.Run([]string{"install", "--dry-run"})
	if exitCode != 0 {
		t.Fatalf("install dry-run failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON install output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "install" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if payload["mode"] != "dry_run" {
		t.Fatalf("expected dry_run mode, got %#v", payload)
	}
}

func TestInstallCommandWritesManagedAssets(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	exitCode := app.Run([]string{"install", "--scope", "agents"})
	if exitCode != 0 {
		t.Fatalf("install command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON install output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "install" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatalf("expected AGENTS.md to be written: %v", err)
	}
}

func TestInstallCommandRejectsInvalidScope(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	exitCode := app.Run([]string{"install", "--scope", "bogus"})
	if exitCode != 1 {
		t.Fatalf("expected invalid scope exit code 1, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON install output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "install" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected install failure, got %#v", payload)
	}
}

func TestInstallHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"install", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness install") {
		t.Fatalf("expected install help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for install help, got %q", stdout.String())
	}
}

func TestPlanLintHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"plan", "lint", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected help exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage: harness plan lint")) {
		t.Fatalf("expected help text, got %s", stderr.String())
	}
}

func TestStatusCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"status"})
	if exitCode != 0 {
		t.Fatalf("status command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON status output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "status" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestExecuteStartCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Generated Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"execute", "start"})
	if exitCode != 0 {
		t.Fatalf("execute start command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON execute-start output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "execute start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus execute-start event, got %#v", timelineResult)
	}
	if timelineResult.Events[0].Command != "plan" || timelineResult.Events[1].Command != "execute start" {
		t.Fatalf("unexpected execute-start timeline events: %#v", timelineResult.Events)
	}
}

func TestExecuteStartRollsBackWhenTimelineAppendFails(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Generated Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if err := os.MkdirAll(filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/events.jsonl"), 0o755); err != nil {
		t.Fatalf("seed broken event index path: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	exitCode := app.Run([]string{"execute", "start"})
	if exitCode != 1 {
		t.Fatalf("expected execute start failure when timeline append fails, got %d", exitCode)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-test-plan")
	if err != nil {
		t.Fatalf("load state after rollback: %v", err)
	}
	if state == nil || state.ExecutionStartedAt != "" || state.CurrentNode != "plan" {
		t.Fatalf("expected execute start rollback to restore pre-start state, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan after rollback: %v", err)
	}
	if current != nil {
		t.Fatalf("expected execute start rollback to restore nil current-plan pointer, got %#v", current)
	}
}

func TestEvidenceSubmitCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"status":"success","provider":"github-actions"}`)
	exitCode := app.Run([]string{"evidence", "submit", "--kind", "ci"})
	if exitCode != 0 {
		t.Fatalf("evidence submit command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON evidence submit output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "evidence submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 2 {
		t.Fatalf("expected bootstrap plan plus evidence event, got %#v", timelineResult)
	}
	if timelineResult.Events[0].Command != "plan" || timelineResult.Events[1].Command != "evidence submit" {
		t.Fatalf("unexpected evidence timeline events: %#v", timelineResult.Events)
	}
}

func TestEvidenceSubmitCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	app.Stdin = bytes.NewBufferString(`{"status":"success","unexpected":true}`)
	exitCode := app.Run([]string{"evidence", "submit", "--kind", "ci"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON evidence submit output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "evidence submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "input.unexpected")
}

func TestReviewStartCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Review Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	exitCode := app.Run([]string{"review", "start"})
	if exitCode != 0 {
		t.Fatalf("review start command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review start output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "review start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) != 3 {
		t.Fatalf("expected plan, execute-start, and review-start events, got %#v", timelineResult)
	}
	last := timelineResult.Events[len(timelineResult.Events)-1]
	if last.Command != "review start" {
		t.Fatalf("unexpected review-start timeline event: %#v", last)
	}
}

func TestReviewStartCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Review Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{
		"kind":"delta",
		"dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}],
		"unexpected":true
	}`)
	exitCode := app.Run([]string{"review", "start"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review start output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "review start" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "spec.unexpected")
}

func TestReviewSubmitCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}

	assertLastTimelineEventCommand(t, root, "review submit")
}

func TestReviewSubmitCommandReturnsSchemaValidationErrors(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"findings":[]}`)
	exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness"})
	if exitCode != 1 {
		t.Fatalf("expected schema validation failure, got %d: %s", exitCode, stderr.String())
	}

	var payload struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Errors  []struct {
			Path string `json:"path"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON review submit output: %v\n%s", err, stdout.String())
	}
	if payload.OK || payload.Command != "review submit" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	assertCLIErrorPath(t, payload.Errors, "submission.summary")
}

func TestReviewAggregateCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Aggregate Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness"}); exitCode != 0 {
		t.Fatalf("review submit failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if exitCode := app.Run([]string{"review", "aggregate", "--round", "review-001-delta"}); exitCode != 0 {
		t.Fatalf("review aggregate failed with %d: %s", exitCode, stderr.String())
	}

	assertLastTimelineEventCommand(t, root, "review aggregate")
}

func TestReviewSubmitRollsBackWhenTimelineAppendFails(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 15, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-test-plan.md")
	if exitCode := app.Run([]string{"plan", "template", "--title", "CLI Review Submit Rollback Plan", "--output", outputPath}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if exitCode := app.Run([]string{"execute", "start"}); exitCode != 0 {
		t.Fatalf("execute start failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check the status and contracts."}]}`)
	if exitCode := app.Run([]string{"review", "start"}); exitCode != 0 {
		t.Fatalf("review start failed with %d: %s", exitCode, stderr.String())
	}

	eventIndexPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/events.jsonl")
	if err := os.Remove(eventIndexPath); err != nil {
		t.Fatalf("remove seeded event index: %v", err)
	}
	if err := os.MkdirAll(eventIndexPath, 0o755); err != nil {
		t.Fatalf("replace event index with directory: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = bytes.NewBufferString(`{"summary":"Looks good","findings":[]}`)
	if exitCode := app.Run([]string{"review", "submit", "--round", "review-001-delta", "--slot", "correctness"}); exitCode != 1 {
		t.Fatalf("expected review submit failure when timeline append fails, got %d", exitCode)
	}

	submissionPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/reviews/review-001-delta/submissions/correctness.json")
	if _, err := os.Stat(submissionPath); !os.IsNotExist(err) {
		t.Fatalf("expected submission rollback after timeline append failure, got err=%v", err)
	}
	ledgerPath := filepath.Join(root, ".local/harness/plans/2026-03-18-test-plan/reviews/review-001-delta/ledger.json")
	var ledger struct {
		Slots []struct {
			Slot   string `json:"slot"`
			Status string `json:"status"`
		} `json:"slots"`
	}
	ledgerBytes, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read ledger after rollback: %v", err)
	}
	if err := json.Unmarshal(ledgerBytes, &ledger); err != nil {
		t.Fatalf("unmarshal ledger after rollback: %v", err)
	}
	if len(ledger.Slots) != 1 || ledger.Slots[0].Status != "pending" {
		t.Fatalf("expected pending ledger after rollback, got %#v", ledger.Slots)
	}
}

func TestArchiveCommandAppendsTimelineEvent(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 16, 0, 0, 0, time.UTC)
	}

	relPlanPath := "docs/plans/active/2026-03-18-archive-ready.md"
	writeArchiveReadyPlanForCLI(t, root, relPlanPath)
	if _, err := runstate.SaveCurrentPlan(root, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	if _, err := runstate.SaveState(root, "2026-03-18-archive-ready", &runstate.State{
		PlanPath:           relPlanPath,
		PlanStem:           "2026-03-18-archive-ready",
		ExecutionStartedAt: "2026-03-18T15:00:00Z",
		Revision:           1,
		ActiveReviewRound: &runstate.ReviewRound{
			RoundID:    "review-001-full",
			Kind:       "full",
			Revision:   1,
			Aggregated: true,
			Decision:   "pass",
		},
	}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	seedPassingFinalizeReviewForCLI(t, root, "2026-03-18-archive-ready", relPlanPath, "review-001-full")

	if exitCode := app.Run([]string{"archive"}); exitCode != 0 {
		t.Fatalf("archive failed with %d: %s", exitCode, stderr.String())
	}

	assertLastTimelineEventCommand(t, root, "archive")
}

func TestLandCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"})
	if exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	assertLastTimelineEventCommand(t, root, "land")
}

func TestReopenNewStepCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "new-step"})
	if exitCode != 0 {
		t.Fatalf("reopen command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "new-step" {
		t.Fatalf("expected reopen mode to persist, got %#v", state)
	}
	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-landed-plan.md" {
		t.Fatalf("expected reopened current-plan pointer to move back to active path, got %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs a new unfinished step") {
		t.Fatalf("unexpected reopen summary: %q", statusResult.Summary)
	}
	if len(statusResult.NextAction) == 0 || !strings.Contains(statusResult.NextAction[0].Description, "Add a new unfinished step") {
		t.Fatalf("expected new-step guidance after reopen, got %#v", statusResult.NextAction)
	}
}

func TestReopenFinalizeFixCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 15, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 0 {
		t.Fatalf("reopen command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	state, _, err := runstate.LoadState(root, "2026-03-18-landed-plan")
	if err != nil {
		t.Fatalf("load reopened state: %v", err)
	}
	if state == nil || state.Reopen == nil || state.Reopen.Mode != "finalize-fix" {
		t.Fatalf("expected finalize-fix reopen mode to persist, got %#v", state)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected status after reopen, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode != "execution/finalize/fix" {
		t.Fatalf("unexpected node after reopen: %#v", statusResult.State)
	}
	if !strings.Contains(statusResult.Summary, "needs follow-up fixes") {
		t.Fatalf("unexpected reopen summary: %q", statusResult.Summary)
	}
	if len(statusResult.NextAction) == 0 || !strings.Contains(statusResult.NextAction[0].Description, "review-023-full") && !strings.Contains(strings.ToLower(statusResult.NextAction[0].Description), "review") {
		t.Fatalf("expected finalize-fix guidance after reopen, got %#v", statusResult.NextAction)
	}

	assertLastTimelineEventCommand(t, root, "reopen")
}

func TestReopenCommandRequiresMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen"})
	if exitCode != 2 {
		t.Fatalf("expected missing-mode exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for usage error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen usage text, got %q", stderr.String())
	}
}

func TestReopenCommandRejectsInvalidMode(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 30, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	exitCode := app.Run([]string{"reopen", "--mode", "bogus"})
	if exitCode != 1 {
		t.Fatalf("expected invalid-mode exit code 1, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected invalid reopen mode to fail, got %#v", payload)
	}
}

func TestReopenCommandRejectsMalformedModeFlag(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--mode"})
	if exitCode != 2 {
		t.Fatalf("expected malformed-mode exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for parse error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "flag needs an argument: -mode") {
		t.Fatalf("expected parse error for missing mode value, got %q", stderr.String())
	}
}

func TestReopenHelpExitsZero(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--help"})
	if exitCode != 0 {
		t.Fatalf("expected reopen help exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen help text, got %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for help, got %q", stdout.String())
	}
}

func TestReopenCommandRejectsExtraPositionalArgs(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix", "extra"})
	if exitCode != 2 {
		t.Fatalf("expected extra-args exit code 2, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout for usage error, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Usage: harness reopen --mode <finalize-fix|new-step>") {
		t.Fatalf("expected reopen usage text, got %q", stderr.String())
	}
}

func TestReopenCommandReportsGetwdFailure(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	app.Getwd = func() (string, error) {
		return "", errors.New("boom")
	}

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 1 {
		t.Fatalf("expected getwd failure exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout on getwd failure, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "resolve working directory: boom") {
		t.Fatalf("expected getwd failure in stderr, got %q", stderr.String())
	}
}

func TestReopenCommandRejectsLandCleanupInProgress(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 7, 45, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)
	if exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"}); exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"reopen", "--mode", "finalize-fix"})
	if exitCode != 1 {
		t.Fatalf("expected reopen failure during land cleanup, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON reopen output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "reopen" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected reopen failure during land cleanup, got %#v", payload)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK || statusResult.State.CurrentNode != "land" {
		t.Fatalf("expected land status to remain after failed reopen, got %#v", statusResult)
	}
}

func TestLandCompleteCommandReturnsJSON(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	writeArchivedPlanForCLI(t, root, "docs/plans/archived/2026-03-18-landed-plan.md")
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/archived/2026-03-18-landed-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	seedMergeReadyEvidenceForCLI(t, root)
	if exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"}); exitCode != 0 {
		t.Fatalf("land command failed with %d: %s", exitCode, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "complete"})
	if exitCode != 0 {
		t.Fatalf("land complete command failed with %d: %s", exitCode, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land complete output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land complete" {
		t.Fatalf("unexpected payload: %#v", payload)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK || statusResult.State.CurrentNode != "idle" {
		t.Fatalf("expected idle status after land complete, got %#v", statusResult)
	}

	assertLastTimelineEventCommand(t, root, "land complete")
}

func TestLandCommandRejectsActivePlanWithoutWritingLandedMarker(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	app := cli.New(stdout, stderr)
	root := t.TempDir()
	app.Getwd = func() (string, error) { return root, nil }
	app.Now = func() time.Time {
		return time.Date(2026, 3, 18, 6, 0, 0, 0, time.UTC)
	}

	outputPath := filepath.Join(root, "docs/plans/active/2026-03-18-active-plan.md")
	if exitCode := app.Run([]string{
		"plan", "template",
		"--title", "CLI Active Plan",
		"--output", outputPath,
	}); exitCode != 0 {
		t.Fatalf("template command failed with %d: %s", exitCode, stderr.String())
	}
	if _, err := runstate.SaveCurrentPlan(root, "docs/plans/active/2026-03-18-active-plan.md"); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	exitCode := app.Run([]string{"land", "--pr", "https://github.com/catu-ai/easyharness/pull/99"})
	if exitCode != 1 {
		t.Fatalf("expected land failure exit code, got %d", exitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON land output: %v\n%s", err, stdout.String())
	}
	if payload["command"] != "land" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected ok=false, got %#v", payload)
	}

	current, err := runstate.LoadCurrentPlan(root)
	if err != nil {
		t.Fatalf("load current plan: %v", err)
	}
	if current == nil || current.PlanPath != "docs/plans/active/2026-03-18-active-plan.md" {
		t.Fatalf("expected active current plan to remain, got %#v", current)
	}
	if current.LastLandedPlanPath != "" || current.LastLandedAt != "" {
		t.Fatalf("expected no landed marker on failed command, got %#v", current)
	}

	statusResult := status.Service{Workdir: root}.Read()
	if !statusResult.OK {
		t.Fatalf("expected active-plan status after failed land, got %#v", statusResult)
	}
	if statusResult.State.CurrentNode == "idle" {
		t.Fatalf("failed land should not switch status to idle: %#v", statusResult)
	}
}

func seedMergeReadyEvidenceForCLI(t *testing.T, root string) {
	t.Helper()
	svc := evidence.Service{
		Workdir: root,
		Now: func() time.Time {
			return time.Date(2026, 3, 18, 5, 55, 0, 0, time.UTC)
		},
	}
	if result := svc.Submit("publish", []byte(`{"status":"recorded","pr_url":"https://github.com/catu-ai/easyharness/pull/99"}`)); !result.OK {
		t.Fatalf("seed publish evidence: %#v", result)
	}
	if result := svc.Submit("ci", []byte(`{"status":"success","provider":"github-actions"}`)); !result.OK {
		t.Fatalf("seed ci evidence: %#v", result)
	}
	if result := svc.Submit("sync", []byte(`{"status":"fresh","base_ref":"main","head_ref":"codex/test"}`)); !result.OK {
		t.Fatalf("seed sync evidence: %#v", result)
	}
}

func assertLastTimelineEventCommand(t *testing.T, root, command string) {
	t.Helper()
	timelineResult := timeline.Service{Workdir: root}.Read()
	if !timelineResult.OK || len(timelineResult.Events) == 0 {
		t.Fatalf("expected timeline events for %q, got %#v", command, timelineResult)
	}
	last := timelineResult.Events[len(timelineResult.Events)-1]
	if last.Command != command {
		t.Fatalf("expected last timeline event %q, got %#v", command, last)
	}
}

func writeArchiveReadyPlanForCLI(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "CLI Archive Ready Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	content := rendered
	content = replaceCLIAll(content, "- Done: [ ]", "- Done: [x]")
	content = replaceCLIAll(content, "- Status: pending", "- Status: completed")
	content = replaceCLIAll(content, "- [ ]", "- [x]")
	content = replaceCLIAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceCLIAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: Archive-ready CLI fixture uses finalize review artifacts.")
	content = replaceCLI(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the slice before archive.")
	content = replaceCLI(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nFull review passed before archive.")
	content = replaceCLI(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- PR: NONE\n- Ready: The candidate satisfies the acceptance criteria and is ready for merge approval.\n- Merge Handoff: Commit and push the archive move before treating this candidate as awaiting merge approval.")
	content = replaceCLI(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the slice.")
	content = replaceCLI(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.")
	content = replaceCLI(content, "### Follow-Up Issues\n\nPENDING_UNTIL_ARCHIVE", "### Follow-Up Issues\n\nNONE")
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write archive-ready plan: %v", err)
	}
	return path
}

func seedPassingFinalizeReviewForCLI(t *testing.T, root, planStem, relPlanPath, roundID string) {
	t.Helper()
	reviewDir := filepath.Join(root, ".local/harness/plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatalf("mkdir review dir: %v", err)
	}
	manifest := `{
  "round_id": "` + roundID + `",
  "kind": "full",
  "revision": 1,
  "review_title": "Full branch candidate before archive",
  "plan_path": "` + relPlanPath + `",
  "plan_stem": "` + planStem + `"
}`
	if err := os.WriteFile(filepath.Join(reviewDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	aggregate := `{
  "round_id": "` + roundID + `",
  "kind": "full",
  "revision": 1,
  "review_title": "Full branch candidate before archive",
  "decision": "pass",
  "blocking_findings": [],
  "non_blocking_findings": [],
  "aggregated_at": "2026-03-18T15:30:00Z"
}`
	if err := os.WriteFile(filepath.Join(reviewDir, "aggregate.json"), []byte(aggregate), 0o644); err != nil {
		t.Fatalf("write aggregate: %v", err)
	}
}

func assertCLIErrorPath(t *testing.T, errors []struct {
	Path string `json:"path"`
}, path string) {
	t.Helper()
	for _, issue := range errors {
		if issue.Path == path {
			return
		}
	}
	t.Fatalf("expected CLI error path %q, got %#v", path, errors)
}

func writeArchivedPlanForCLI(t *testing.T, root, relPath string) string {
	t.Helper()
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{
		Title:      "CLI Landed Plan",
		Timestamp:  time.Date(2026, 3, 18, 2, 0, 0, 0, time.UTC),
		SourceType: "direct_request",
	})
	if err != nil {
		t.Fatalf("RenderTemplate: %v", err)
	}
	content := rendered
	content = bytes.NewBufferString(content).String()
	content = replaceCLI(content, "status: active", "status: archived")
	content = replaceCLI(content, "lifecycle: awaiting_plan_approval", "lifecycle: awaiting_merge_approval")
	content = replaceCLIAll(content, "- Done: [ ]", "- Done: [x]")
	content = replaceCLIAll(content, "- Status: pending", "- Status: completed")
	content = replaceCLIAll(content, "- [ ]", "- [x]")
	content = replaceCLIAll(content, "PENDING_STEP_EXECUTION", "Done.")
	content = replaceCLIAll(content, "PENDING_STEP_REVIEW", "NO_STEP_REVIEW_NEEDED: Archived CLI fixture uses explicit review-complete closeout.")
	content = replaceCLI(content, "## Validation Summary\n\nPENDING_UNTIL_ARCHIVE", "## Validation Summary\n\nValidated the slice.")
	content = replaceCLI(content, "## Review Summary\n\nPENDING_UNTIL_ARCHIVE", "## Review Summary\n\nFull review passed.")
	content = replaceCLI(content, "## Archive Summary\n\nPENDING_UNTIL_ARCHIVE", "## Archive Summary\n\n- Archived At: 2026-03-18T02:00:00Z\n- Revision: 1\n- PR: NONE\n- Ready: Ready for merge approval.\n- Merge Handoff: Commit and push before merge approval.")
	content = replaceCLI(content, "### Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Delivered\n\nDelivered the slice.")
	content = replaceCLI(content, "### Not Delivered\n\nPENDING_UNTIL_ARCHIVE", "### Not Delivered\n\nNONE.")
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write archived plan: %v", err)
	}
	return path
}

func replaceCLI(content, old, new string) string {
	tuned := bytes.Replace([]byte(content), []byte(old), []byte(new), 1)
	return string(tuned)
}

func replaceCLIAll(content, old, new string) string {
	tuned := bytes.ReplaceAll([]byte(content), []byte(old), []byte(new))
	return string(tuned)
}
