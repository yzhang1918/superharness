package plan

import "github.com/catu-ai/easyharness/internal/runstate"

func (d *Document) DerivedPlanStatus() string {
	switch d.PathKind {
	case "active":
		return "active"
	case "archived":
		return "archived"
	default:
		return ""
	}
}

func (d *Document) WorkflowProfile() string {
	if d == nil {
		return WorkflowProfileStandard
	}
	return normalizeWorkflowProfile(d.Frontmatter.WorkflowProfile)
}

func (d *Document) UsesLightweightProfile() bool {
	return d.WorkflowProfile() == WorkflowProfileLightweight
}

func (d *Document) ExecutionStarted(state *runstate.State) bool {
	if state == nil {
		return false
	}
	return state.ExecutionStartedAt != "" ||
		state.ActiveReviewRound != nil ||
		state.Land != nil ||
		state.Reopen != nil
}

func (d *Document) DerivedLifecycle(state *runstate.State) string {
	switch d.DerivedPlanStatus() {
	case "archived":
		return "awaiting_merge_approval"
	case "active":
		if d.ExecutionStarted(state) {
			return "executing"
		}
		return "awaiting_plan_approval"
	default:
		return ""
	}
}
