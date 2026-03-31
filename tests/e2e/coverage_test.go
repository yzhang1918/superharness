package e2e_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type transitionFamily struct {
	ID             string
	From           string
	To             string
	Driver         string
	RequiredInputs string
}

type scenarioCoverage struct {
	ID            string
	TestName      string
	TransitionIDs []string
}

var canonicalTransitionFamilies = []transitionFamily{
	{ID: "idle_to_plan", From: "idle", To: "plan", Driver: "Derived from current plan presence", RequiredInputs: "Execution-start is absent, and exactly one active tracked plan exists under `docs/plans/active/`"},
	{ID: "plan_self", From: "plan", To: "plan", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "plan_to_step_implement", From: "plan", To: "execution/step-<n>/implement", Driver: "harness execute start", RequiredInputs: "Current plan is approved for execution and has at least one unfinished step"},
	{ID: "step_implement_self", From: "execution/step-<n>/implement", To: "execution/step-<n>/implement", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "step_implement_to_review", From: "execution/step-<n>/implement", To: "execution/step-<n>/review", Driver: "harness review start", RequiredInputs: "The command binds the new round to the current step"},
	{ID: "step_implement_to_next_step_implement", From: "execution/step-<n>/implement", To: "execution/step-<m>/implement", Driver: "Derived from current plan edits", RequiredInputs: "Step `<n>` becomes durably complete, any required step review is clean, and another unfinished step exists"},
	{ID: "step_implement_to_finalize_review", From: "execution/step-<n>/implement", To: "execution/finalize/review", Driver: "Derived from current plan edits", RequiredInputs: "Step `<n>` becomes durably complete, any required step review is clean, and no unfinished steps remain"},
	{ID: "step_review_self", From: "execution/step-<n>/review", To: "execution/step-<n>/review", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "step_review_to_step_implement_clean", From: "execution/step-<n>/review", To: "execution/step-<n>/implement", Driver: "harness review aggregate", RequiredInputs: "Latest aggregate is clean"},
	{ID: "step_review_to_step_implement_repair", From: "execution/step-<n>/review", To: "execution/step-<n>/implement", Driver: "harness review aggregate", RequiredInputs: "Latest aggregate has actionable findings or an unrecoverable conservative outcome"},
	{ID: "finalize_review_self", From: "execution/finalize/review", To: "execution/finalize/review", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "finalize_review_to_finalize_fix", From: "execution/finalize/review", To: "execution/finalize/fix", Driver: "harness review aggregate", RequiredInputs: "Latest finalize review aggregate has actionable findings or an unrecoverable conservative outcome"},
	{ID: "finalize_review_to_finalize_archive", From: "execution/finalize/review", To: "execution/finalize/archive", Driver: "Derived from clean finalize review", RequiredInputs: "Finalize review is satisfied and archive closeout work remains"},
	{ID: "finalize_fix_self", From: "execution/finalize/fix", To: "execution/finalize/fix", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "finalize_fix_to_finalize_review", From: "execution/finalize/fix", To: "execution/finalize/review", Driver: "harness review start", RequiredInputs: "A new finalize review round is started after repair"},
	{ID: "finalize_fix_to_new_step_implement", From: "execution/finalize/fix", To: "execution/step-<m>/implement", Driver: "Derived from current plan edits", RequiredInputs: "Reopen mode is `new-step`, the first new unfinished step has been added, and that new step is now current"},
	{ID: "finalize_archive_self", From: "execution/finalize/archive", To: "execution/finalize/archive", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "finalize_archive_to_publish", From: "execution/finalize/archive", To: "execution/finalize/publish", Driver: "harness archive", RequiredInputs: "Finalize review is satisfied and archive closeout is ready"},
	{ID: "publish_self", From: "execution/finalize/publish", To: "execution/finalize/publish", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "publish_to_await_merge", From: "execution/finalize/publish", To: "execution/finalize/await_merge", Driver: "Derived from latest publish, CI, and sync evidence", RequiredInputs: "Publish evidence identifies the candidate, CI is good enough or explicit `not_applied`, sync is acceptable or explicit `not_applied`, and no unresolved fix condition remains"},
	{ID: "publish_to_finalize_fix", From: "execution/finalize/publish", To: "execution/finalize/fix", Driver: "harness reopen --mode finalize-fix", RequiredInputs: "Archived candidate has been invalidated but does not justify a new step"},
	{ID: "publish_to_finalize_fix_new_step", From: "execution/finalize/publish", To: "execution/finalize/fix", Driver: "harness reopen --mode new-step", RequiredInputs: "Archived candidate has been invalidated and the change deserves a new unfinished step"},
	{ID: "await_merge_to_land", From: "execution/finalize/await_merge", To: "land", Driver: "harness land --pr <url> [--commit <sha>]", RequiredInputs: "Human approval exists, merge happened outside harness, and land entry records the PR URL"},
	{ID: "await_merge_to_finalize_fix", From: "execution/finalize/await_merge", To: "execution/finalize/fix", Driver: "harness reopen --mode finalize-fix", RequiredInputs: "Merge-ready archived candidate has been invalidated without justifying a new step"},
	{ID: "await_merge_to_finalize_fix_new_step", From: "execution/finalize/await_merge", To: "execution/finalize/fix", Driver: "harness reopen --mode new-step", RequiredInputs: "Merge-ready archived candidate has been invalidated and the change deserves a new unfinished step"},
	{ID: "land_self", From: "land", To: "land", Driver: "state-preserving", RequiredInputs: "state-preserving update"},
	{ID: "land_to_idle", From: "land", To: "idle", Driver: "harness land complete", RequiredInputs: "Merge cleanup is done and land completion is intentionally recorded"},
}

var currentScenarioCoverage = []scenarioCoverage{
	{
		ID:       "review_workflow",
		TestName: "TestReviewWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_self",
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
		},
	},
	{
		ID:       "review_repair_loop",
		TestName: "TestReviewRepairLoopsWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_repair",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_fix",
			"finalize_fix_to_finalize_review",
			"finalize_review_to_finalize_archive",
		},
	},
	{
		ID:       "archive_reopen_finalize_fix",
		TestName: "TestArchiveReopenFinalizeFixWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_self",
			"finalize_archive_to_publish",
			"publish_to_finalize_fix",
		},
	},
	{
		ID:       "reopen_new_step",
		TestName: "TestReopenNewStepWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_finalize_fix_new_step",
			"finalize_fix_self",
			"finalize_fix_to_new_step_implement",
			"step_implement_self",
		},
	},
	{
		ID:       "publish_handoff",
		TestName: "TestPublishHandoffWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_self",
			"finalize_archive_to_publish",
			"publish_self",
			"publish_to_await_merge",
		},
	},
	{
		ID:       "lightweight_workflow",
		TestName: "TestLightweightWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_self",
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_self",
			"publish_to_await_merge",
		},
	},
	{
		ID:       "land_workflow",
		TestName: "TestLandWorkflowWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"await_merge_to_land",
			"land_self",
			"land_to_idle",
		},
	},
	{
		ID:       "await_merge_reopen_finalize_fix",
		TestName: "TestAwaitMergeReopenFinalizeFixWithBuiltBinary",
		TransitionIDs: []string{
			"idle_to_plan",
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"await_merge_to_finalize_fix",
		},
	},
	{
		ID:       "await_merge_reopen_new_step",
		TestName: "TestAwaitMergeReopenNewStepWithBuiltBinary",
		TransitionIDs: []string{
			"plan_to_step_implement",
			"step_implement_to_review",
			"step_review_self",
			"step_review_to_step_implement_clean",
			"step_implement_to_next_step_implement",
			"step_implement_to_finalize_review",
			"finalize_review_self",
			"finalize_review_to_finalize_archive",
			"finalize_archive_to_publish",
			"publish_to_await_merge",
			"await_merge_to_finalize_fix_new_step",
			"finalize_fix_to_new_step_implement",
		},
	},
}

func TestCanonicalTransitionCoverageCatalogIsWellFormed(t *testing.T) {
	transitionIDs := map[string]bool{}
	transitionKeys := map[string]bool{}
	for _, family := range canonicalTransitionFamilies {
		if family.ID == "" {
			t.Fatal("transition family id must be non-empty")
		}
		if family.From == "" || family.To == "" || family.Driver == "" || family.RequiredInputs == "" {
			t.Fatalf("canonical transition family must include from, to, driver, and required inputs: %#v", family)
		}
		if transitionIDs[family.ID] {
			t.Fatalf("duplicate transition family id %q", family.ID)
		}
		transitionIDs[family.ID] = true
		key := transitionFamilyKey(family.From, family.To, family.Driver, family.RequiredInputs)
		if transitionKeys[key] {
			t.Fatalf("duplicate canonical transition family triple %q", key)
		}
		transitionKeys[key] = true
	}

	scenarioIDs := map[string]bool{}
	for _, scenario := range currentScenarioCoverage {
		if scenario.ID == "" || scenario.TestName == "" {
			t.Fatalf("scenario coverage entry must have id and test name: %#v", scenario)
		}
		if scenarioIDs[scenario.ID] {
			t.Fatalf("duplicate scenario coverage id %q", scenario.ID)
		}
		scenarioIDs[scenario.ID] = true

		seenInScenario := map[string]bool{}
		for _, transitionID := range scenario.TransitionIDs {
			if !transitionIDs[transitionID] {
				t.Fatalf("scenario %q references unknown transition family %q", scenario.ID, transitionID)
			}
			if seenInScenario[transitionID] {
				t.Fatalf("scenario %q references transition family %q more than once", scenario.ID, transitionID)
			}
			seenInScenario[transitionID] = true
		}
	}
}

func TestScenarioCoverageSpansEveryCanonicalTransitionFamily(t *testing.T) {
	covered := map[string]bool{}
	for _, scenario := range currentScenarioCoverage {
		for _, transitionID := range scenario.TransitionIDs {
			covered[transitionID] = true
		}
	}

	var missing []string
	for _, family := range canonicalTransitionFamilies {
		if !covered[family.ID] {
			missing = append(missing, family.ID)
		}
	}

	if len(missing) != 0 {
		t.Fatalf("scenario coverage is missing canonical transition families: %v", missing)
	}
}

func TestCanonicalTransitionCatalogMatchesTrackedSpecMatrix(t *testing.T) {
	specTransitions := loadTrackedSpecTransitions(t)

	specKeys := map[string]bool{}
	for _, transition := range specTransitions {
		key := transitionFamilyKey(transition.From, transition.To, transition.Driver, transition.RequiredInputs)
		if specKeys[key] {
			t.Fatalf("tracked spec transition matrix contains duplicate transition triple %q", key)
		}
		specKeys[key] = true
	}

	canonicalKeys := map[string]bool{}
	for _, family := range canonicalTransitionFamilies {
		key := transitionFamilyKey(family.From, family.To, family.Driver, family.RequiredInputs)
		canonicalKeys[key] = true
	}

	var missingFromCatalog []string
	for key := range specKeys {
		if !canonicalKeys[key] {
			missingFromCatalog = append(missingFromCatalog, key)
		}
	}

	var missingFromSpec []string
	for key := range canonicalKeys {
		if !specKeys[key] {
			missingFromSpec = append(missingFromSpec, key)
		}
	}

	if len(missingFromCatalog) != 0 || len(missingFromSpec) != 0 {
		t.Fatalf("canonical transition catalog drifted from docs/specs/state-transitions.md; missing from catalog: %v; missing from spec: %v", missingFromCatalog, missingFromSpec)
	}
}

func transitionFamilyKey(from, to, driver, requiredInputs string) string {
	return from + " => " + to + " [" + driver + "] {" + requiredInputs + "}"
}

func loadTrackedSpecTransitions(t *testing.T) []transitionFamily {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate coverage test file path")
	}

	specPath := filepath.Join(filepath.Dir(filename), "..", "..", "docs", "specs", "state-transitions.md")
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read transition spec: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	inStatePreserving := false
	var transitions []transitionFamily
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if strings.HasPrefix(line, "## ") {
			inStatePreserving = line == "## State-Preserving Updates"
			continue
		}

		if strings.HasPrefix(line, "|") {
			columns := splitMarkdownRow(line)
			if len(columns) < 4 {
				continue
			}
			if columns[0] == "From" || strings.HasPrefix(columns[0], "---") {
				continue
			}

			transitions = append(transitions, transitionFamily{
				From:           trimInlineCode(columns[0]),
				To:             trimInlineCode(columns[1]),
				Driver:         trimInlineCode(columns[2]),
				RequiredInputs: trimInlineCode(columns[3]),
			})
			continue
		}

		if !inStatePreserving || !strings.HasPrefix(line, "- `") {
			continue
		}

		transitionText, ok := extractInlineCode(line)
		if !ok {
			t.Fatalf("failed to parse state-preserving transition from line %q", line)
		}
		from, to, ok := strings.Cut(transitionText, " -> ")
		if !ok {
			t.Fatalf("state-preserving transition line missing arrow: %q", line)
		}
		transitions = append(transitions, transitionFamily{
			From:           from,
			To:             to,
			Driver:         "state-preserving",
			RequiredInputs: "state-preserving update",
		})
	}

	return transitions
}

func splitMarkdownRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	parts := strings.Split(trimmed, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func trimInlineCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "`") && strings.HasSuffix(trimmed, "`") && strings.Count(trimmed, "`") == 2 {
		return strings.TrimSuffix(strings.TrimPrefix(trimmed, "`"), "`")
	}
	return trimmed
}

func extractInlineCode(line string) (string, bool) {
	start := strings.Index(line, "`")
	if start == -1 {
		return "", false
	}
	end := strings.Index(line[start+1:], "`")
	if end == -1 {
		return "", false
	}
	return line[start+1 : start+1+end], true
}
