package resilience_test

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestStatusFailsSafelyWhenCurrentPlanPointerIsMalformed(t *testing.T) {
	workspace := support.NewWorkspace(t)
	relPlanPath := "docs/plans/active/2026-04-11-resilience-current-plan.md"
	writePlanFixture(t, workspace, relPlanPath, "Resilience Current Plan", nil)
	workspace.WriteFile(t, ".local/harness/current-plan.json", []byte("{not-json"))

	result := support.Run(t, workspace.Root, "status")
	support.RequireExitCode(t, result, 1)
	support.RequireNoStderr(t, result)

	parsed := support.RequireJSONResult[statusResult](t, result)
	if parsed.OK || parsed.Command != "status" {
		t.Fatalf("expected failing status payload, got %#v", parsed)
	}
	if parsed.Summary != "Unable to read current worktree state." {
		t.Fatalf("unexpected summary: %#v", parsed)
	}
	if !findError(parsed.Errors, "state") {
		t.Fatalf("expected state error, got %#v", parsed.Errors)
	}
	if len(parsed.Errors) == 0 || !strings.Contains(parsed.Errors[0].Message, "parse current-plan.json") {
		t.Fatalf("expected parse-current-plan failure, got %#v", parsed.Errors)
	}
	support.RequireFileMissing(t, workspace.Path(".local/harness/plans/2026-04-11-resilience-current-plan/state.json"))
}
