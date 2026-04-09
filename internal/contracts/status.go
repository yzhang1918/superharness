package contracts

// StatusResult is the JSON result returned by `harness status`.
type StatusResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable explanation of the current state.
	Summary string `json:"summary"`

	// State is the canonical workflow position resolved for the current worktree.
	State StatusState `json:"state"`

	// Facts carries selected high-signal derived details that help explain the
	// current node.
	Facts *StatusFacts `json:"facts,omitempty"`

	// Artifacts points to stable paths or identifiers that help the caller locate
	// related contract artifacts.
	Artifacts *StatusArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Blockers lists state issues that block ordinary execution progression.
	Blockers []ErrorDetail `json:"blockers,omitempty"`

	// Warnings lists non-blocking workflow reminders or ambiguity notes.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented full state resolution.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// StatusState is the canonical workflow node returned by `harness status`.
type StatusState struct {
	// CurrentNode is the resolved v0.2 workflow node for the current worktree.
	CurrentNode string `json:"current_node" jsonschema:"example=execution/step-1/implement"`
}

// StatusFacts carries selected derived details that help interpret a status
// node without dumping the full local state model.
type StatusFacts struct {
	// CurrentStep is the first unfinished plan step when one exists.
	CurrentStep string `json:"current_step,omitempty"`

	// Revision is the current plan-local revision number.
	Revision int `json:"revision,omitempty"`

	// ReopenMode is the active reopen mode when a reopen repair path is in
	// effect.
	ReopenMode string `json:"reopen_mode,omitempty"`

	// ReviewKind is the review kind for the active review round.
	ReviewKind string `json:"review_kind,omitempty"`

	// ReviewTrigger is the derived reason label for the active review round.
	ReviewTrigger string `json:"review_trigger,omitempty"`

	// ReviewTitle is the human-readable title for the active review round when
	// one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// ReviewStatus summarizes the aggregate review state for the current node.
	ReviewStatus string `json:"review_status,omitempty"`

	// ArchiveBlockerCount reports how many archive-readiness blockers are still
	// present.
	ArchiveBlockerCount int `json:"archive_blocker_count,omitempty"`

	// PublishStatus summarizes the latest publish evidence state.
	PublishStatus string `json:"publish_status,omitempty"`

	// PRURL is the currently known pull request URL for the candidate branch.
	PRURL string `json:"pr_url,omitempty"`

	// CIStatus summarizes the latest CI evidence state.
	CIStatus string `json:"ci_status,omitempty"`

	// SyncStatus summarizes the latest remote-sync evidence state.
	SyncStatus string `json:"sync_status,omitempty"`

	// LandPRURL is the pull request URL recorded for the land phase.
	LandPRURL string `json:"land_pr_url,omitempty"`

	// LandCommit is the merge commit or landed commit recorded for the land
	// phase.
	LandCommit string `json:"land_commit,omitempty"`
}

// StatusArtifacts lists stable paths or identifiers related to the current
// status result.
type StatusArtifacts struct {
	// PlanPath is the active, archived, or last-landed plan path relevant to the
	// current status resolution.
	PlanPath string `json:"plan_path,omitempty" jsonschema:"example=docs/plans/active/2026-03-31-centralize-contract-schemas-and-generated-reference-docs.md"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty" jsonschema:"example=.local/harness/plans/2026-03-31-centralize-contract-schemas-and-generated-reference-docs/state.json"`

	// ReviewRoundID is the active review round identifier when review is in
	// flight.
	ReviewRoundID string `json:"review_round_id,omitempty"`

	// CIRecordID is the latest CI evidence record identifier when known.
	CIRecordID string `json:"ci_record_id,omitempty"`

	// PublishRecordID is the latest publish evidence record identifier when
	// known.
	PublishRecordID string `json:"publish_record_id,omitempty"`

	// SyncRecordID is the latest sync evidence record identifier when known.
	SyncRecordID string `json:"sync_record_id,omitempty"`

	// LastLandedPlanPath is the most recent landed plan path when the repository
	// is otherwise idle.
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`

	// LastLandedAt is the timestamp of the most recent landed plan.
	LastLandedAt string `json:"last_landed_at,omitempty"`
}
