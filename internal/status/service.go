package status

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/runstate"
)

type Service struct {
	Workdir string
}

type Result struct {
	OK         bool          `json:"ok"`
	Command    string        `json:"command"`
	Summary    string        `json:"summary"`
	State      State         `json:"state"`
	Artifacts  *Artifacts    `json:"artifacts,omitempty"`
	NextAction []NextAction  `json:"next_actions"`
	Warnings   []string      `json:"warnings,omitempty"`
	Errors     []StatusError `json:"errors,omitempty"`
}

type State struct {
	PlanStatus string `json:"plan_status"`
	Lifecycle  string `json:"lifecycle"`
	Step       string `json:"step,omitempty"`
	StepState  string `json:"step_state,omitempty"`
}

type Artifacts struct {
	PlanPath       string `json:"plan_path"`
	LocalStatePath string `json:"local_state_path,omitempty"`
	ReviewRoundID  string `json:"review_round_id,omitempty"`
	CISnapshotID   string `json:"ci_snapshot_id,omitempty"`
}

type NextAction struct {
	Command     *string `json:"command"`
	Description string  `json:"description"`
}

type StatusError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (s Service) Read() Result {
	planPath, err := s.detectPlanPath()
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to determine the current plan.",
			Errors:  []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}

	doc, err := plan.LoadFile(planPath)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read the current plan.",
			Artifacts: &Artifacts{
				PlanPath: planPath,
			},
			Errors: []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read local harness state.",
			Artifacts: &Artifacts{
				PlanPath:       planPath,
				LocalStatePath: statePath,
			},
			Errors: []StatusError{{Path: "state", Message: err.Error()}},
		}
	}

	result := Result{
		OK:      true,
		Command: "status",
		State: State{
			PlanStatus: doc.Frontmatter.Status,
			Lifecycle:  doc.Frontmatter.Lifecycle,
		},
		Artifacts: &Artifacts{
			PlanPath:       planPath,
			LocalStatePath: statePath,
		},
	}

	if current := doc.CurrentStep(); current != nil {
		result.State.Step = current.Title
	}
	if doc.Frontmatter.Lifecycle == "executing" {
		result.State.StepState = inferStepState(doc, state)
	}
	if state != nil {
		if state.ActiveReviewRound != nil {
			result.Artifacts.ReviewRoundID = state.ActiveReviewRound.RoundID
		}
		if state.LatestCI != nil {
			result.Artifacts.CISnapshotID = state.LatestCI.SnapshotID
		}
	}

	result.Warnings = buildWarnings(state)
	result.NextAction = buildNextActions(doc, state, result.State.StepState)
	result.Summary = buildSummary(doc, result.State.StepState)

	return result
}

func (s Service) detectPlanPath() (string, error) {
	if current, err := runstate.LoadCurrentPlan(s.Workdir); err != nil {
		return "", err
	} else if current != nil && strings.TrimSpace(current.PlanPath) != "" {
		return filepath.Join(s.Workdir, current.PlanPath), nil
	}

	activeMatches, err := filepath.Glob(filepath.Join(s.Workdir, "docs", "plans", "active", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(activeMatches)
	if len(activeMatches) == 1 {
		return activeMatches[0], nil
	}
	if len(activeMatches) > 1 {
		return "", fmt.Errorf("multiple active plans found; add .local/harness/current-plan.json to disambiguate")
	}

	archivedMatches, err := filepath.Glob(filepath.Join(s.Workdir, "docs", "plans", "archived", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(archivedMatches)
	if len(archivedMatches) == 1 {
		return archivedMatches[0], nil
	}

	return "", fmt.Errorf("no current plan found")
}

func inferStepState(doc *plan.Document, state *runstate.State) string {
	if state != nil {
		if state.Sync != nil && state.Sync.Conflicts {
			return "resolving_conflicts"
		}
		if state.ActiveReviewRound != nil && !state.ActiveReviewRound.Aggregated {
			return "reviewing"
		}
		if state.LatestCI != nil && state.LatestCI.Status == "pending" {
			return "waiting_ci"
		}
	}
	if doc.AllStepsCompleted() && doc.AllAcceptanceChecked() && !doc.HasPendingArchivePlaceholders() && !doc.CompletedStepsHavePendingPlaceholders() {
		return "ready_for_archive"
	}
	return "implementing"
}

func buildWarnings(state *runstate.State) []string {
	if state == nil || state.Sync == nil {
		return nil
	}
	switch state.Sync.Freshness {
	case "stale":
		return []string{"Remote freshness is stale; refresh remote state before making merge-sensitive decisions."}
	case "unknown":
		return []string{"Remote freshness is unknown; consider refreshing remote state before archive or merge-sensitive work."}
	default:
		return nil
	}
}

func buildNextActions(doc *plan.Document, state *runstate.State, stepState string) []NextAction {
	next := make([]NextAction, 0)
	if state != nil && state.Sync != nil && (state.Sync.Freshness == "stale" || state.Sync.Freshness == "unknown") {
		next = append(next, NextAction{
			Command:     nil,
			Description: "Refresh remote state before archive or merge-sensitive decisions.",
		})
	}

	switch doc.Frontmatter.Lifecycle {
	case "awaiting_plan_approval":
		next = append(next, NextAction{Command: nil, Description: "Wait for plan approval or update the plan if scope changed."})
	case "blocked":
		next = append(next, NextAction{Command: nil, Description: "Resolve the blocking dependency or get human input before continuing."})
	case "awaiting_merge_approval":
		next = append(next,
			NextAction{Command: nil, Description: "Wait for merge approval or merge manually from the PR once checks are green."},
			NextAction{Command: strPtr("harness reopen"), Description: "Reopen the plan if new feedback or remote changes mean the archived candidate is no longer ready."},
		)
	default:
		switch stepState {
		case "reviewing":
			next = append(next, NextAction{Command: strPtr("harness review aggregate --round <round-id>"), Description: "Aggregate the active review round once reviewer submissions are ready."})
		case "waiting_ci":
			next = append(next, NextAction{Command: nil, Description: "Wait for required CI to finish, then address any failures before continuing."})
		case "resolving_conflicts":
			next = append(next, NextAction{Command: nil, Description: "Finish conflict resolution, rerun focused validation, and consider a delta review before continuing."})
		case "ready_for_archive":
			next = append(next,
				NextAction{Command: nil, Description: "Ensure deferred items have follow-up issue references and write final archive summaries."},
				NextAction{Command: strPtr("harness archive"), Description: "Archive the current plan once the closeout notes and follow-up links are ready."},
			)
		default:
			next = append(next, NextAction{Command: nil, Description: "Continue the current step and keep step-local Execution Notes and Review Notes up to date."})
		}
	}

	return next
}

func buildSummary(doc *plan.Document, stepState string) string {
	switch doc.Frontmatter.Lifecycle {
	case "awaiting_plan_approval":
		return "Plan is awaiting approval before execution begins."
	case "blocked":
		return "Plan is currently blocked and needs external input before continuing."
	case "awaiting_merge_approval":
		return "Plan is archived and waiting for merge approval."
	}

	if step := doc.CurrentStep(); step != nil {
		if stepState != "" {
			return fmt.Sprintf("Plan is %s %s and the current step state is %s.", doc.Frontmatter.Lifecycle, step.Title, strings.ReplaceAll(stepState, "_", " "))
		}
		return fmt.Sprintf("Plan is %s %s.", doc.Frontmatter.Lifecycle, step.Title)
	}
	return fmt.Sprintf("Plan is %s.", doc.Frontmatter.Lifecycle)
}

func strPtr(value string) *string {
	return &value
}
