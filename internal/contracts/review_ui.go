package contracts

import "encoding/json"

// ReviewResult is the read-only UI resource returned by `/api/review`.
type ReviewResult struct {
	// OK reports whether review loading succeeded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded review
	// rounds.
	Summary string `json:"summary"`

	// Artifacts points to the current-plan review artifact paths used to build
	// this response.
	Artifacts *ReviewArtifacts `json:"artifacts,omitempty"`

	// Rounds lists the discovered review rounds for the current plan.
	Rounds []ReviewRoundView `json:"rounds"`

	// Warnings lists non-fatal degraded-state notes for the overall review
	// resource.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented review loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewArtifacts points to the current plan review sources.
type ReviewArtifacts struct {
	// PlanPath is the current plan path associated with the review data.
	PlanPath string `json:"plan_path,omitempty"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty"`

	// ReviewsDir is the current-plan review rounds directory.
	ReviewsDir string `json:"reviews_dir,omitempty"`

	// ActiveRoundID is the currently active review round when one exists in
	// local state.
	ActiveRoundID string `json:"active_round_id,omitempty"`
}

// ReviewRoundView is one read-only UI representation of a review round.
type ReviewRoundView struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind,omitempty"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision,omitempty"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// Status is the stable UI status label for the round.
	Status string `json:"status,omitempty"`

	// StatusSummary is the concise explanation of the current round status.
	StatusSummary string `json:"status_summary,omitempty"`

	// Decision is the aggregate decision when one is known.
	Decision string `json:"decision,omitempty"`

	// CreatedAt is the round creation timestamp when known.
	CreatedAt string `json:"created_at,omitempty"`

	// UpdatedAt is the ledger update timestamp when known.
	UpdatedAt string `json:"updated_at,omitempty"`

	// AggregatedAt is the aggregate timestamp when known.
	AggregatedAt string `json:"aggregated_at,omitempty"`

	// IsActive reports whether local state still points at this round as active.
	IsActive bool `json:"is_active,omitempty"`

	// TotalSlots is the number of reviewer slots represented by the round.
	TotalSlots int `json:"total_slots,omitempty"`

	// SubmittedSlots is the number of reviewer slots with a submission.
	SubmittedSlots int `json:"submitted_slots,omitempty"`

	// PendingSlots is the number of reviewer slots still waiting on a
	// submission.
	PendingSlots int `json:"pending_slots,omitempty"`

	// Reviewers lists the reviewer-centric slot views for the round.
	Reviewers []ReviewSlotView `json:"reviewers,omitempty"`

	// BlockingFindings lists aggregate blocking findings when they exist.
	BlockingFindings []ReviewAggregateFinding `json:"blocking_findings,omitempty"`

	// NonBlockingFindings lists aggregate non-blocking findings when they
	// exist.
	NonBlockingFindings []ReviewAggregateFinding `json:"non_blocking_findings,omitempty"`

	// Artifacts lists supporting raw review artifacts for the round.
	Artifacts []ReviewArtifactView `json:"artifacts,omitempty"`

	// Warnings lists non-fatal degraded-state notes for the round.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewSlotView is one reviewer-centric view of a manifest slot plus any
// submission that was returned for that slot.
type ReviewSlotView struct {
	// Name is the human-readable dimension label.
	Name string `json:"name,omitempty"`

	// Slot is the stable reviewer slot identifier.
	Slot string `json:"slot"`

	// Instructions is the reviewer prompt for this slot when the manifest is
	// readable.
	Instructions string `json:"instructions,omitempty"`

	// Status is the current submission status label for the slot.
	Status string `json:"status,omitempty"`

	// SubmissionPath is the path to the submission artifact for this slot.
	SubmissionPath string `json:"submission_path,omitempty"`

	// SubmittedAt is the submission timestamp when one is known.
	SubmittedAt string `json:"submitted_at,omitempty"`

	// Summary is the reviewer's concise overall assessment when one was
	// submitted.
	Summary string `json:"summary,omitempty"`

	// Findings lists the slot-level review findings when one was submitted.
	Findings []ReviewFinding `json:"findings,omitempty"`

	// Warnings lists non-fatal degraded-state notes for this slot.
	Warnings []string `json:"warnings,omitempty"`
}

// ReviewArtifactView is one raw supporting artifact view for a review round.
type ReviewArtifactView struct {
	// Label is the stable display label for the artifact.
	Label string `json:"label"`

	// Path is the artifact path when one exists.
	Path string `json:"path,omitempty"`

	// Status reports whether the artifact is available, missing, or invalid.
	Status string `json:"status,omitempty"`

	// Summary is the concise artifact-state explanation.
	Summary string `json:"summary,omitempty"`

	// ContentType reports how Content should be rendered in the UI resource.
	ContentType string `json:"content_type,omitempty"`

	// Content is the resolved artifact file payload for UI tabs when readable.
	Content json.RawMessage `json:"content,omitempty"`
}
