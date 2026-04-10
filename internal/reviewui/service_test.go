package reviewui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestServiceReadReturnsActivePlanRoundsWithConservativeStatuses(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-active.md", "Review UI Active")
	saveReviewState(t, workdir, planStem, relPlanPath, 3, "review-003-full")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-delta", map[string]any{
		"round_id":        "review-001-delta",
		"kind":            "delta",
		"step":            1,
		"revision":        1,
		"review_title":    "Step 1 closeout",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T10:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-delta", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-delta", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-delta", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check correctness.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-delta", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-delta",
		"kind":       "delta",
		"updated_at": "2026-04-02T10:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T10:08:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-delta", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":              "review-001-delta",
		"kind":                  "delta",
		"step":                  1,
		"revision":              1,
		"review_title":          "Step 1 closeout",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T10:12:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-delta",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T10:08:00Z",
			"summary":      "Looks good.",
			"findings":     []any{},
		},
	})

	writeReviewRoundFixture(t, workdir, planStem, "review-002-delta", map[string]any{
		"round_id":        "review-002-delta",
		"kind":            "delta",
		"step":            2,
		"revision":        2,
		"review_title":    "Step 2 closeout",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T11:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-002-delta", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-002-delta", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-002-delta", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Risk",
				"slot":            "risk",
				"instructions":    "Check risk.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-002-delta", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-002-delta",
		"kind":       "delta",
		"updated_at": "2026-04-02T11:15:00Z",
		"slots": []map[string]any{
			{
				"name":            "Risk",
				"slot":            "risk",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T11:10:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-002-delta", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, nil, map[string]map[string]any{
		"risk": {
			"round_id":     "review-002-delta",
			"slot":         "risk",
			"dimension":    "Risk",
			"submitted_at": "2026-04-02T11:10:00Z",
			"summary":      "One concern.",
			"findings": []map[string]any{
				{
					"severity":  "important",
					"title":     "Edge case",
					"details":   "An edge case still needs attention.",
					"locations": []string{"internal/ui/server.go#L1-L5"},
				},
			},
		},
	})
	if err := os.WriteFile(roundArtifactPath(workdir, planStem, "review-002-delta", "aggregate.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write invalid aggregate: %v", err)
	}

	writeReviewRoundFixture(t, workdir, planStem, "review-003-full", map[string]any{
		"round_id":        "review-003-full",
		"kind":            "full",
		"revision":        3,
		"review_title":    "Finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-003-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-003-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-003-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check the final result.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-003-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
			{
				"name":            "UX",
				"slot":            "ux",
				"instructions":    "Check UI clarity.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-003-full", filepath.Join("submissions", "ux", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-003-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T12:07:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-003-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
			{
				"name":            "UX",
				"slot":            "ux",
				"status":          "pending",
				"submission_path": roundArtifactPath(workdir, planStem, "review-003-full", filepath.Join("submissions", "ux", "submission.json")),
			},
		},
	}, nil, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-003-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Core flow looks solid.",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if result.Resource != "review" {
		t.Fatalf("expected resource=review, got %#v", result)
	}
	if len(result.Rounds) != 3 {
		t.Fatalf("expected three rounds, got %#v", result.Rounds)
	}

	if result.Rounds[0].RoundID != "review-003-full" || !result.Rounds[0].IsActive {
		t.Fatalf("expected active round first, got %#v", result.Rounds[0])
	}
	if result.Rounds[0].Status != "waiting_for_submissions" || result.Rounds[0].SubmittedSlots != 1 || result.Rounds[0].PendingSlots != 1 {
		t.Fatalf("expected in-progress waiting round details, got %#v", result.Rounds[0])
	}
	if len(result.Rounds[0].Reviewers) != 2 || result.Rounds[0].Reviewers[0].Instructions == "" {
		t.Fatalf("expected reviewer instructions to be available, got %#v", result.Rounds[0].Reviewers)
	}

	if result.Rounds[1].RoundID != "review-002-delta" || result.Rounds[1].Status != "degraded" {
		t.Fatalf("expected damaged round to be degraded, got %#v", result.Rounds[1])
	}
	if len(result.Rounds[1].Warnings) == 0 {
		t.Fatalf("expected degraded round warnings, got %#v", result.Rounds[1])
	}

	if result.Rounds[2].RoundID != "review-001-delta" || result.Rounds[2].Status != "pass" || result.Rounds[2].Decision != "pass" {
		t.Fatalf("expected clean round to stay readable, got %#v", result.Rounds[2])
	}
}

func TestServiceReadReturnsArchivedPlanRounds(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedArchivedPlan(t, workdir, "2026-04-02-review-ui-archived.md", "Review UI Archived")
	saveReviewStateWithNode(t, workdir, planStem, relPlanPath, 1, "", "execution/finalize/await_merge")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Archived finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check archived candidate correctness.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T12:07:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Archived finalize review",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T12:12:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Archived candidate still looks good.",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected archived plan to keep review rounds visible, got %#v", result.Rounds)
	}
	if result.Rounds[0].RoundID != "review-001-full" || result.Rounds[0].Status != "pass" {
		t.Fatalf("expected archived review round to stay readable, got %#v", result.Rounds[0])
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.PlanPath, "docs/plans/archived/2026-04-02-review-ui-archived.md") {
		t.Fatalf("expected archived plan artifacts to point at the archived plan, got %#v", result.Artifacts)
	}
}

func TestServiceReadHidesArchivedRoundsDuringLandCleanup(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedArchivedPlan(t, workdir, "2026-04-02-review-ui-landed.md", "Review UI Landed")
	saveReviewStateWithLand(t, workdir, planStem, relPlanPath, 1, "2026-04-03T01:00:00Z", "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Landed finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions":      []map[string]any{},
	}, nil, nil, nil)

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 0 {
		t.Fatalf("expected landed archived plan to hide review rounds, got %#v", result.Rounds)
	}
	if !strings.Contains(result.Summary, "hidden once required post-merge bookkeeping begins") {
		t.Fatalf("unexpected summary for landed archived plan: %#v", result)
	}
}

func TestServiceReadHidesArchivedRoundsDuringLegacyLandCleanup(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedArchivedPlan(t, workdir, "2026-04-02-review-ui-legacy-land.md", "Review UI Legacy Land")
	saveReviewStateWithLand(t, workdir, planStem, relPlanPath, 1, "2026-04-03T01:00:00Z", "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Legacy landed finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions":      []map[string]any{},
	}, nil, nil, nil)

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 0 {
		t.Fatalf("expected legacy required post-merge bookkeeping state to hide archived review rounds, got %#v", result.Rounds)
	}
	if !strings.Contains(result.Summary, "hidden once required post-merge bookkeeping begins") {
		t.Fatalf("unexpected summary for legacy landed archived plan: %#v", result)
	}
}

func TestServiceReadReturnsLightweightArchivedPlanRounds(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedLocalArchivedPlan(t, workdir, "2026-04-02-review-ui-lightweight.md", "Review UI Lightweight")
	saveReviewStateWithNode(t, workdir, planStem, relPlanPath, 1, "", "execution/finalize/await_merge")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Lightweight finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions":      []map[string]any{},
	}, map[string]any{
		"round_id":   "review-001-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:10:00Z",
		"slots":      []map[string]any{},
	}, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Lightweight finalize review",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T12:12:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, nil)

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 || result.Rounds[0].RoundID != "review-001-full" {
		t.Fatalf("expected lightweight archived plan to keep review rounds visible, got %#v", result.Rounds)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.PlanPath, ".local/harness/plans/archived/2026-04-02-review-ui-lightweight.md") {
		t.Fatalf("expected lightweight archived plan artifacts to point at the local archive, got %#v", result.Artifacts)
	}
}

func TestServiceReadReturnsEmptyForActivePlanWithoutReviewRounds(t *testing.T) {
	workdir := t.TempDir()
	seedActivePlan(t, workdir, "2026-04-02-review-ui-empty.md", "Review UI Empty")

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 0 {
		t.Fatalf("expected active plan with no review rounds to return none, got %#v", result.Rounds)
	}
	if !strings.Contains(result.Summary, "No review rounds recorded yet for the current plan.") {
		t.Fatalf("unexpected summary for empty active plan: %#v", result)
	}
	if result.Artifacts == nil || !strings.Contains(result.Artifacts.PlanPath, "docs/plans/active/2026-04-02-review-ui-empty.md") {
		t.Fatalf("expected active plan artifacts to point at the empty plan, got %#v", result.Artifacts)
	}
}

func TestServiceReadPreservesAggregateFindings(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-positive-findings.md", "Review UI Positive Findings")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "review-001-full")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Positive findings round",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
			{
				"name":            "Risk",
				"slot":            "risk",
				"instructions":    "Check for residual workflow risk.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T12:07:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
			{
				"name":            "Risk",
				"slot":            "risk",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T12:08:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":      "review-001-full",
		"kind":          "full",
		"revision":      1,
		"review_title":  "Positive findings round",
		"decision":      "changes_requested",
		"aggregated_at": "2026-04-02T12:12:00Z",
		"blocking_findings": []map[string]any{
			{
				"slot":      "correctness",
				"dimension": "Correctness",
				"severity":  "important",
				"title":     "Missing provenance",
				"details":   "This should stay visible in the aggregate render path.",
				"locations": []string{"web/src/main.tsx#L1450-L1487"},
			},
		},
		"non_blocking_findings": []map[string]any{
			{
				"slot":      "risk",
				"dimension": "Risk",
				"severity":  "minor",
				"title":     "Minor polish",
				"details":   "This should also stay visible in the aggregate render path.",
			},
		},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "One blocking finding remains.",
			"findings":     []any{},
		},
		"risk": {
			"round_id":     "review-001-full",
			"slot":         "risk",
			"dimension":    "Risk",
			"submitted_at": "2026-04-02T12:08:00Z",
			"summary":      "No blocking risk.",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "changes_requested" {
		t.Fatalf("expected aggregate decision to drive requested changes, got %#v", round)
	}
	if round.Decision != "changes_requested" {
		t.Fatalf("expected aggregate decision to remain visible, got %#v", round)
	}
	if len(round.BlockingFindings) != 1 || len(round.NonBlockingFindings) != 1 {
		t.Fatalf("expected both finding groups to survive, got %#v", round)
	}
	if round.BlockingFindings[0].Slot != "correctness" || round.BlockingFindings[0].Dimension != "Correctness" {
		t.Fatalf("expected blocking finding provenance to survive, got %#v", round.BlockingFindings[0])
	}
	if round.NonBlockingFindings[0].Slot != "risk" || round.NonBlockingFindings[0].Dimension != "Risk" {
		t.Fatalf("expected non-blocking finding provenance to survive, got %#v", round.NonBlockingFindings[0])
	}
}

func TestServiceReadNormalizesReviewerWorklogAndRawSubmission(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-10-review-ui-progressive-worklog.md", "Review UI Progressive Worklog")
	saveReviewState(t, workdir, planStem, relPlanPath, 2, "review-002-delta")

	writeReviewRoundFixture(t, workdir, planStem, "review-002-delta", map[string]any{
		"round_id":        "review-002-delta",
		"kind":            "delta",
		"anchor_sha":      "abc123def",
		"step":            2,
		"revision":        2,
		"review_title":    "Step 2 closeout",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-10T01:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-002-delta", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-002-delta", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-002-delta", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check delta correctness and related risks.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-002-delta", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-002-delta",
		"kind":       "delta",
		"updated_at": "2026-04-10T01:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-10T01:09:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-002-delta", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, nil, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-002-delta",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-10T01:09:00Z",
			"summary":      "One possible follow-up remains.",
			"findings": []map[string]any{
				{
					"severity": "minor",
					"title":    "Polish follow-up",
					"details":  "UI wording could still be tightened.",
				},
			},
			"worklog": map[string]any{
				"full_plan_read":     true,
				"checked_areas":      []string{"docs/plans/active/2026-04-10-reviewer-worklog-ui-and-aggregate-provenance.md", "web/src/pages.tsx"},
				"open_questions":     []string{"Should the summary page stay unchanged?"},
				"candidate_findings": []string{"Polish follow-up"},
			},
			"coverage": map[string]any{
				"review_kind": "delta",
				"anchor_sha":  "abc123def",
			},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.AnchorSHA != "abc123def" {
		t.Fatalf("expected round anchor SHA to survive, got %#v", round)
	}
	if len(round.Reviewers) != 1 {
		t.Fatalf("expected one reviewer, got %#v", round.Reviewers)
	}

	reviewer := round.Reviewers[0]
	if reviewer.Worklog == nil {
		t.Fatalf("expected normalized reviewer worklog, got %#v", reviewer)
	}
	if reviewer.Worklog.FullPlanRead == nil || !*reviewer.Worklog.FullPlanRead {
		t.Fatalf("expected full_plan_read=true, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.CheckedAreas) != 2 || reviewer.Worklog.CheckedAreas[0] != "docs/plans/active/2026-04-10-reviewer-worklog-ui-and-aggregate-provenance.md" {
		t.Fatalf("expected checked areas to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.OpenQuestions) != 1 || reviewer.Worklog.OpenQuestions[0] != "Should the summary page stay unchanged?" {
		t.Fatalf("expected open questions to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.CandidateFindings) != 1 || reviewer.Worklog.CandidateFindings[0] != "Polish follow-up" {
		t.Fatalf("expected candidate findings to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.RawSubmission) == 0 || !strings.Contains(string(reviewer.RawSubmission), `"anchor_sha":"abc123def"`) {
		t.Fatalf("expected raw submission payload to remain available, got %#v", reviewer.RawSubmission)
	}
}

func TestServiceReadDegradesMalformedReviewerWorklogFieldsConservatively(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-10-review-ui-malformed-worklog.md", "Review UI Malformed Worklog")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "review-001-delta")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-delta", map[string]any{
		"round_id":        "review-001-delta",
		"kind":            "delta",
		"anchor_sha":      "anchor-sha",
		"step":            1,
		"revision":        1,
		"review_title":    "Malformed worklog round",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-10T01:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-delta", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-delta", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-delta", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Risk",
				"slot":            "risk",
				"instructions":    "Check degraded worklog handling.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-delta", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-delta",
		"kind":       "delta",
		"updated_at": "2026-04-10T01:10:00Z",
		"slots": []map[string]any{
			{
				"name":            "Risk",
				"slot":            "risk",
				"status":          "submitted",
				"submitted_at":    "2026-04-10T01:09:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-delta", filepath.Join("submissions", "risk", "submission.json")),
			},
		},
	}, nil, map[string]map[string]any{
		"risk": {
			"round_id":     "review-001-delta",
			"slot":         "risk",
			"dimension":    "Risk",
			"submitted_at": "2026-04-10T01:09:00Z",
			"summary":      "Malformed worklog fields should degrade conservatively.",
			"findings":     []any{},
			"worklog": map[string]any{
				"full_plan_read":     "yes",
				"checked_areas":      []string{"web/src/pages.tsx"},
				"open_questions":     "still investigating",
				"candidate_findings": []string{"Candidate trail"},
			},
			"coverage": map[string]any{
				"review_kind": 7,
				"anchor_sha":  "worklog-anchor",
			},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 || len(result.Rounds[0].Reviewers) != 1 {
		t.Fatalf("expected one round with one reviewer, got %#v", result.Rounds)
	}

	reviewer := result.Rounds[0].Reviewers[0]
	if reviewer.Worklog == nil {
		t.Fatalf("expected partially recovered worklog, got %#v", reviewer)
	}
	if reviewer.Worklog.FullPlanRead != nil {
		t.Fatalf("expected malformed boolean field to be omitted, got %#v", reviewer.Worklog)
	}
	if reviewer.Worklog.ReviewKind != "" {
		t.Fatalf("expected malformed review_kind to be omitted, got %#v", reviewer.Worklog)
	}
	if reviewer.Worklog.AnchorSHA != "worklog-anchor" {
		t.Fatalf("expected valid anchor_sha to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.CheckedAreas) != 1 || reviewer.Worklog.CheckedAreas[0] != "web/src/pages.tsx" {
		t.Fatalf("expected valid checked_areas to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Worklog.CandidateFindings) != 1 || reviewer.Worklog.CandidateFindings[0] != "Candidate trail" {
		t.Fatalf("expected valid candidate_findings to survive, got %#v", reviewer.Worklog)
	}
	if len(reviewer.Warnings) == 0 || !strings.Contains(strings.Join(reviewer.Warnings, " "), "malformed") {
		t.Fatalf("expected malformed worklog warnings, got %#v", reviewer.Warnings)
	}
}

func TestServiceReadRecoversSubmissionOnlyDamagedRounds(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-submissions-only.md", "Review UI Submission Recovery")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "review-001-full")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", nil, nil, nil, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Recovered from submission only.",
			"findings": []map[string]any{
				{
					"severity": "important",
					"title":    "Recovered finding",
					"details":  "This round only has a submission artifact left.",
				},
			},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.RoundID != "review-001-full" {
		t.Fatalf("expected recovered round, got %#v", round)
	}
	if len(round.Reviewers) != 1 {
		t.Fatalf("expected reviewer recovered from submission artifact, got %#v", round.Reviewers)
	}
	reviewer := round.Reviewers[0]
	if reviewer.Slot != "correctness" || reviewer.Name != "Correctness" {
		t.Fatalf("expected recovered reviewer metadata, got %#v", reviewer)
	}
	if reviewer.Summary != "Recovered from submission only." || len(reviewer.Findings) != 1 {
		t.Fatalf("expected recovered reviewer content, got %#v", reviewer)
	}
	if reviewer.Status != "submitted" {
		t.Fatalf("expected submission-only reviewer to count as submitted, got %#v", reviewer)
	}
	if round.TotalSlots != 1 || round.SubmittedSlots != 1 || round.PendingSlots != 0 {
		t.Fatalf("expected recovered slot counts, got %#v", round)
	}
	if round.Status != "degraded" {
		t.Fatalf("expected submissions-only round to stay degraded without manifest/ledger, got %#v", round)
	}
}

func TestServiceReadKeepsUnknownLedgerStatusConservative(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-unknown-ledger-status.md", "Review UI Unknown Ledger Status")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "review-001-full")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:05:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "mystery_state",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, nil, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Submission exists despite odd ledger status.",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "waiting_for_submissions" {
		t.Fatalf("expected conservative round status, got %#v", round)
	}
	if round.SubmittedSlots != 0 || round.PendingSlots != 1 {
		t.Fatalf("expected unknown ledger state to stay pending, got %#v", round)
	}
	if len(round.Reviewers) != 1 {
		t.Fatalf("expected one reviewer, got %#v", round.Reviewers)
	}
	reviewer := round.Reviewers[0]
	if reviewer.Status != "mystery_state" {
		t.Fatalf("expected raw reviewer status to stay visible, got %#v", reviewer)
	}
	if len(reviewer.Warnings) == 0 || !strings.Contains(strings.Join(reviewer.Warnings, " "), "unknown slot status") {
		t.Fatalf("expected warning for unknown slot status, got %#v", reviewer)
	}
}

func TestServiceReadKeepsMalformedLedgerRoundsDegradedEvenWithAggregateDecision(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-malformed-ledger.md", "Review UI Malformed Ledger")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, nil, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Finalize review",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T12:10:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Submission exists, but ledger is malformed.",
			"findings":     []any{},
		},
	})
	if err := os.WriteFile(roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"), []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write invalid ledger: %v", err)
	}

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "degraded" {
		t.Fatalf("expected malformed ledger to force degraded status, got %#v", round)
	}
	if round.Decision != "pass" {
		t.Fatalf("expected aggregate decision to remain visible, got %#v", round)
	}
	if len(round.Warnings) == 0 || !strings.Contains(strings.Join(round.Warnings, " "), "malformed") {
		t.Fatalf("expected malformed warning, got %#v", round)
	}
}

func TestServiceReadKeepsUnreadableAggregateRoundsDegraded(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-unreadable-aggregate.md", "Review UI Unreadable Aggregate")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":   "review-001-full",
		"kind":       "full",
		"updated_at": "2026-04-02T12:08:00Z",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submitted_at":    "2026-04-02T12:07:00Z",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Finalize review",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T12:10:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Submission exists, but aggregate is unreadable.",
			"findings":     []any{},
		},
	})

	aggregatePath := roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json")
	if err := os.Remove(aggregatePath); err != nil {
		t.Fatalf("remove aggregate file: %v", err)
	}
	if err := os.Mkdir(aggregatePath, 0o755); err != nil {
		t.Fatalf("replace aggregate with directory: %v", err)
	}

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "degraded" {
		t.Fatalf("expected unreadable aggregate to force degraded status, got %#v", round)
	}
	if len(round.Warnings) == 0 || !strings.Contains(strings.Join(round.Warnings, " "), "Unable to read aggregate") {
		t.Fatalf("expected unreadable aggregate warning, got %#v", round)
	}
	if len(round.Artifacts) < 3 || round.Artifacts[2].Status != "invalid" {
		t.Fatalf("expected aggregate artifact to be invalid, got %#v", round.Artifacts)
	}
}

func TestServiceReadKeepsSemanticallyBrokenArtifactsConservative(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-semantic-damage.md", "Review UI Semantic Damage")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"kind":            "full",
		"revision":        1,
		"review_title":    "Finalize review",
		"plan_path":       relPlanPath,
		"plan_stem":       planStem,
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id": "review-001-full",
		"kind":     "full",
		"slots": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"status":          "submitted",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Finalize review",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
		"aggregated_at":         "2026-04-02T12:10:00Z",
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "degraded" {
		t.Fatalf("expected semantically broken artifacts to degrade the round, got %#v", round)
	}
	if round.Decision != "" {
		t.Fatalf("expected incomplete aggregate not to render as a clean decision, got %#v", round)
	}
	if len(round.Artifacts) < 4 {
		t.Fatalf("expected core and submission artifacts, got %#v", round.Artifacts)
	}
	if round.Artifacts[0].Status != "invalid" || round.Artifacts[1].Status != "invalid" || round.Artifacts[2].Status != "invalid" || round.Artifacts[3].Status != "available" {
		t.Fatalf("expected semantically incomplete artifacts to be marked invalid, got %#v", round.Artifacts)
	}
	if len(round.Warnings) == 0 || !strings.Contains(strings.Join(round.Warnings, " "), "missing required fields") {
		t.Fatalf("expected missing-required-fields warning, got %#v", round)
	}
	if len(round.Reviewers) != 1 {
		t.Fatalf("expected one reviewer, got %#v", round.Reviewers)
	}
	reviewer := round.Reviewers[0]
	if reviewer.Status != "submitted" {
		t.Fatalf("expected ledger-submitted reviewer to stay submitted even when summary stays hidden, got %#v", reviewer)
	}
	if reviewer.Summary != "" {
		t.Fatalf("expected invalid submission summary to stay hidden, got %#v", reviewer)
	}
}

func TestServiceReadKeepsMissingLedgerRoundsDegradedEvenWithAggregateDecision(t *testing.T) {
	workdir := t.TempDir()
	relPlanPath, planStem := seedActivePlan(t, workdir, "2026-04-02-review-ui-missing-ledger.md", "Review UI Missing Ledger")
	saveReviewState(t, workdir, planStem, relPlanPath, 1, "")

	writeReviewRoundFixture(t, workdir, planStem, "review-001-full", map[string]any{
		"round_id":        "review-001-full",
		"kind":            "full",
		"revision":        1,
		"review_title":    "Finalize review",
		"created_at":      "2026-04-02T12:00:00Z",
		"ledger_path":     roundArtifactPath(workdir, planStem, "review-001-full", "ledger.json"),
		"aggregate_path":  roundArtifactPath(workdir, planStem, "review-001-full", "aggregate.json"),
		"submissions_dir": filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", "review-001-full", "submissions"),
		"dimensions": []map[string]any{
			{
				"name":            "Correctness",
				"slot":            "correctness",
				"instructions":    "Check final behavior.",
				"submission_path": roundArtifactPath(workdir, planStem, "review-001-full", filepath.Join("submissions", "correctness", "submission.json")),
			},
		},
	}, nil, map[string]any{
		"round_id":              "review-001-full",
		"kind":                  "full",
		"revision":              1,
		"review_title":          "Finalize review",
		"decision":              "pass",
		"aggregated_at":         "2026-04-02T12:10:00Z",
		"blocking_findings":     []any{},
		"non_blocking_findings": []any{},
	}, map[string]map[string]any{
		"correctness": {
			"round_id":     "review-001-full",
			"slot":         "correctness",
			"dimension":    "Correctness",
			"submitted_at": "2026-04-02T12:07:00Z",
			"summary":      "Submission exists, but ledger is missing.",
			"findings":     []any{},
		},
	})

	result := Service{Workdir: workdir}.Read()
	if !result.OK {
		t.Fatalf("expected review read to succeed, got %#v", result)
	}
	if len(result.Rounds) != 1 {
		t.Fatalf("expected one round, got %#v", result.Rounds)
	}

	round := result.Rounds[0]
	if round.Status != "degraded" {
		t.Fatalf("expected missing ledger to force degraded status, got %#v", round)
	}
	if round.Decision != "pass" {
		t.Fatalf("expected aggregate decision to remain visible, got %#v", round)
	}
	if len(round.Warnings) == 0 || !strings.Contains(strings.Join(round.Warnings, " "), "Ledger is missing") {
		t.Fatalf("expected missing-ledger warning, got %#v", round)
	}
}

func seedActivePlan(t *testing.T, workdir, filename, title string) (string, string) {
	t.Helper()
	return seedPlan(t, workdir, filepath.Join("docs/plans/active", filename), title)
}

func seedArchivedPlan(t *testing.T, workdir, filename, title string) (string, string) {
	t.Helper()
	return seedPlan(t, workdir, filepath.Join("docs/plans/archived", filename), title)
}

func seedLocalArchivedPlan(t *testing.T, workdir, filename, title string) (string, string) {
	t.Helper()
	return seedPlan(t, workdir, filepath.Join(".local/harness/plans/archived", filename), title)
}

func seedPlan(t *testing.T, workdir, relPlanPath, title string) (string, string) {
	t.Helper()
	path := filepath.Join(workdir, filepath.FromSlash(relPlanPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir plan dir: %v", err)
	}
	rendered, err := plan.RenderTemplate(plan.TemplateOptions{Title: title})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relPlanPath); err != nil {
		t.Fatalf("save current plan: %v", err)
	}
	return relPlanPath, strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
}

func saveReviewState(t *testing.T, workdir, planStem, relPlanPath string, revision int, activeRoundID string) {
	t.Helper()
	saveReviewStateWithNode(t, workdir, planStem, relPlanPath, revision, activeRoundID, "")
}

func saveReviewStateWithNode(t *testing.T, workdir, planStem, relPlanPath string, revision int, activeRoundID, currentNode string) {
	t.Helper()
	state := &runstate.State{
		Revision: revision,
	}
	_ = relPlanPath
	_ = currentNode
	if activeRoundID != "" {
		state.ActiveReviewRound = &runstate.ReviewRound{
			RoundID:    activeRoundID,
			Kind:       "full",
			Revision:   revision,
			Aggregated: false,
		}
	}
	if _, err := runstate.SaveState(workdir, planStem, state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

func saveReviewStateWithLand(t *testing.T, workdir, planStem, relPlanPath string, revision int, landedAt, completedAt string) {
	t.Helper()
	state := &runstate.State{
		Revision: revision,
		Land: &runstate.LandState{
			LandedAt:    landedAt,
			CompletedAt: completedAt,
		},
	}
	_ = relPlanPath
	if _, err := runstate.SaveState(workdir, planStem, state); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

func writeReviewRoundFixture(t *testing.T, workdir, planStem, roundID string, manifest, ledger, aggregate map[string]any, submissions map[string]map[string]any) {
	t.Helper()
	roundDir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", roundID)
	if err := os.MkdirAll(filepath.Join(roundDir, "submissions"), 0o755); err != nil {
		t.Fatalf("mkdir round dir: %v", err)
	}
	if manifest != nil {
		writeJSONFileFixture(t, filepath.Join(roundDir, "manifest.json"), manifest)
	}
	if ledger != nil {
		writeJSONFileFixture(t, filepath.Join(roundDir, "ledger.json"), ledger)
	}
	if aggregate != nil {
		writeJSONFileFixture(t, filepath.Join(roundDir, "aggregate.json"), aggregate)
	}
	for slot, payload := range submissions {
		writeJSONFileFixture(t, filepath.Join(roundDir, "submissions", slot, "submission.json"), payload)
	}
}

func writeJSONFileFixture(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func roundArtifactPath(workdir, planStem, roundID, suffix string) string {
	return filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", roundID, suffix)
}
