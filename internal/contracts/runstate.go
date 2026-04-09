package contracts

// CurrentPlanFile is the worktree-level pointer file under
// `.local/harness/current-plan.json`.
type CurrentPlanFile struct {
	// PlanPath is the current active or archived plan path when work is in
	// flight.
	PlanPath string `json:"plan_path,omitempty"`

	// LastLandedPlanPath is the most recent landed plan path when the worktree is
	// otherwise idle.
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`

	// LastLandedAt is the timestamp of the most recent landed plan.
	LastLandedAt string `json:"last_landed_at,omitempty"`
}

// LocalStateFile is the plan-local runtime control artifact under
// `.local/harness/plans/<plan-stem>/state.json`.
type LocalStateFile struct {
	// ExecutionStartedAt is the execution-start timestamp for the plan.
	ExecutionStartedAt string `json:"execution_started_at,omitempty"`

	// Revision is the current plan-local revision number.
	Revision int `json:"revision,omitempty"`

	// Reopen records the active reopen repair state when one exists.
	Reopen *ReopenState `json:"reopen,omitempty"`

	// ActiveReviewRound records the current active review round when review is in
	// flight.
	ActiveReviewRound *ReviewRoundState `json:"active_review_round,omitempty"`

	// Land records the current land state when merge cleanup is in flight.
	Land *LandState `json:"land,omitempty"`
}

// ReopenState records the active reopen repair state.
type ReopenState struct {
	// Mode is the active reopen mode.
	Mode string `json:"mode"`

	// ReopenedAt is the reopen timestamp.
	ReopenedAt string `json:"reopened_at,omitempty"`

	// BaseStepCount is the number of plan steps that existed before the reopen.
	BaseStepCount int `json:"base_step_count,omitempty"`
}

// ReviewRoundState records the current active review round in the local state
// file.
type ReviewRoundState struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision,omitempty"`

	// Aggregated reports whether the round has already been aggregated.
	Aggregated bool `json:"aggregated"`

	// Decision is the aggregate review decision when one is known.
	Decision string `json:"decision,omitempty"`
}

// LandState records the current land state in the local state file.
type LandState struct {
	// PRURL is the pull request URL recorded for the land phase.
	PRURL string `json:"pr_url,omitempty"`

	// Commit is the merge commit or landed commit recorded for the land phase.
	Commit string `json:"commit,omitempty"`

	// LandedAt is the timestamp when the land command recorded merge completion.
	LandedAt string `json:"landed_at,omitempty"`

	// CompletedAt is the timestamp when land cleanup completed.
	CompletedAt string `json:"completed_at,omitempty"`
}
