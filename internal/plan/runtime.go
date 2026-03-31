package plan

import (
	"strings"

	"github.com/catu-ai/easyharness/internal/runstate"
)

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
		currentNodeImpliesExecution(state.CurrentNode) ||
		state.ActiveReviewRound != nil ||
		state.LatestEvidence != nil ||
		state.Land != nil ||
		state.LatestCI != nil ||
		state.Sync != nil ||
		state.LatestPublish != nil ||
		state.Reopen != nil
}

func currentNodeImpliesExecution(node string) bool {
	node = strings.TrimSpace(node)
	return strings.HasPrefix(node, "execution/") || node == "land"
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
