package contracts

// LifecycleResult is the JSON result shape currently shared by
// `harness execute start`, `harness archive`, `harness reopen`, `harness land`,
// and `harness land complete`.
type LifecycleResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// State carries the post-command workflow node in the shared v0.2 envelope.
	State LifecycleState `json:"state"`

	// Facts carries selected high-signal lifecycle details when they help
	// explain the post-command state.
	Facts *LifecycleFacts `json:"facts,omitempty"`

	// Artifacts points to relevant plan and local-state paths for the command.
	Artifacts *LifecycleArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// LifecycleState carries the post-command workflow node for lifecycle commands.
type LifecycleState struct {
	// CurrentNode is the canonical v0.2 workflow node after the command runs.
	CurrentNode string `json:"current_node"`
}

// LifecycleFacts carries selected lifecycle-specific details that remain useful
// after the contract converges on the shared v0.2 envelope.
type LifecycleFacts struct {
	// Revision is the current plan-local revision number.
	Revision int `json:"revision,omitempty"`

	// ReopenMode is the active reopen mode when a reopen repair path is in
	// effect.
	ReopenMode string `json:"reopen_mode,omitempty"`

	// LandPRURL is the pull request URL recorded for the land phase.
	LandPRURL string `json:"land_pr_url,omitempty"`

	// LandCommit is the merge commit or landed commit recorded for the land
	// phase.
	LandCommit string `json:"land_commit,omitempty"`
}

// LifecycleArtifacts lists the relevant plan and local-state paths for a
// lifecycle command result.
type LifecycleArtifacts struct {
	// FromPlanPath is the source plan path for transitions that move or archive a
	// plan.
	FromPlanPath string `json:"from_plan_path"`

	// FromSupplementsPath is the source supplements directory for transitions
	// that move a whole plan package.
	FromSupplementsPath string `json:"from_supplements_path,omitempty"`

	// ToPlanPath is the destination plan path for transitions that create or move
	// a plan artifact.
	ToPlanPath string `json:"to_plan_path"`

	// ToSupplementsPath is the destination supplements directory for transitions
	// that move a whole plan package.
	ToSupplementsPath string `json:"to_supplements_path,omitempty"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty"`

	// CurrentPlanPath is the worktree-level current-plan pointer path when the
	// command updated it.
	CurrentPlanPath string `json:"current_plan_path,omitempty"`
}
