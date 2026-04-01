package contracts

import "encoding/json"

// ReviewSpec is the JSON input consumed by `harness review start`.
type ReviewSpec struct {
	// Step is the tracked plan step number when the review is step-scoped.
	Step *int `json:"step,omitempty"`

	// Kind is the review kind, such as delta or full.
	Kind string `json:"kind"`

	// ReviewTitle is the human-readable title for finalize or custom review
	// rounds.
	ReviewTitle string `json:"review_title,omitempty"`

	// Dimensions lists the review dimensions and instructions assigned to
	// reviewers.
	Dimensions []ReviewDimension `json:"dimensions" jsonschema:"minItems=1" easyharness:"no_null"`
}

// ReviewDimension defines one named review dimension and its reviewer
// instructions.
type ReviewDimension struct {
	// Name is the human-readable dimension label.
	Name string `json:"name"`

	// Instructions is the reviewer prompt for this dimension.
	Instructions string `json:"instructions"`
}

// ReviewManifest is the command-owned review manifest artifact for one review
// round.
type ReviewManifest struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// PlanPath is the tracked or archived plan path associated with the round.
	PlanPath string `json:"plan_path"`

	// PlanStem is the durable plan stem associated with the round.
	PlanStem string `json:"plan_stem"`

	// CreatedAt is the round creation timestamp.
	CreatedAt string `json:"created_at"`

	// Dimensions lists the materialized reviewer slots for the round.
	Dimensions []ReviewManifestSlot `json:"dimensions"`

	// LedgerPath is the path to the round ledger artifact.
	LedgerPath string `json:"ledger_path"`

	// Aggregate is the path to the round aggregate artifact.
	Aggregate string `json:"aggregate_path"`

	// Submissions is the path to the round submissions directory.
	Submissions string `json:"submissions_dir"`
}

// ReviewManifestSlot describes one reviewer submission slot in a review
// manifest.
type ReviewManifestSlot struct {
	// Name is the human-readable dimension label.
	Name string `json:"name"`

	// Slot is the stable slot identifier.
	Slot string `json:"slot"`

	// Instructions is the reviewer prompt for this slot.
	Instructions string `json:"instructions"`

	// SubmissionPath is the target path for this slot's submission artifact.
	SubmissionPath string `json:"submission_path"`
}

// ReviewLedger is the command-owned ledger artifact tracking submission status
// for a review round.
type ReviewLedger struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// UpdatedAt is the timestamp of the most recent ledger update.
	UpdatedAt string `json:"updated_at"`

	// Slots lists the current state of every manifest slot.
	Slots []ReviewLedgerSlot `json:"slots"`
}

// ReviewLedgerSlot records the current state for one reviewer slot in the
// ledger.
type ReviewLedgerSlot struct {
	// Name is the human-readable dimension label.
	Name string `json:"name"`

	// Slot is the stable slot identifier.
	Slot string `json:"slot"`

	// Status is the current submission status for the slot.
	Status string `json:"status"`

	// SubmissionPath is the path where the slot submission should exist.
	SubmissionPath string `json:"submission_path"`

	// SubmittedAt is the submission timestamp when the slot has been submitted.
	SubmittedAt string `json:"submitted_at,omitempty"`
}

// ReviewSubmissionInput is the JSON input consumed by `harness review submit`.
type ReviewSubmissionInput struct {
	// Summary is the reviewer's concise overall assessment.
	Summary string `json:"summary"`

	// Findings lists the review findings for the slot.
	Findings []ReviewFinding `json:"findings,omitempty" easyharness:"allow_null"`
}

// ReviewSubmission is the command-owned submission artifact for one reviewer
// slot.
type ReviewSubmission struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Slot is the stable slot identifier.
	Slot string `json:"slot"`

	// Dimension is the human-readable review dimension label.
	Dimension string `json:"dimension"`

	// SubmittedAt is the submission timestamp.
	SubmittedAt string `json:"submitted_at"`

	// Summary is the reviewer's concise overall assessment.
	Summary string `json:"summary"`

	// Findings lists the review findings for the slot.
	Findings []ReviewFinding `json:"findings"`
}

// ReviewFinding is one review finding in a submission or aggregate.
type ReviewFinding struct {
	// Severity is the finding severity label.
	Severity string `json:"severity"`

	// Title is the short human-readable title of the finding.
	Title string `json:"title"`

	// Details is the full review finding explanation.
	Details string `json:"details"`

	// Locations optionally lists lightweight repo-relative source anchors for
	// the finding, such as "path/to/file.go", "path/to/file.go#L123", or
	// "path/to/file.go#L1-L3".
	Locations []string `json:"locations,omitempty"`

	// HasLocations records whether the payload explicitly included the optional
	// locations field so empty arrays can round-trip without being collapsed
	// into omission.
	HasLocations bool `json:"-"`
}

// ReviewAggregate is the command-owned aggregate artifact for a completed
// review round.
type ReviewAggregate struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Kind is the review kind for the round.
	Kind string `json:"kind"`

	// Step is the tracked plan step number when the round is step-scoped.
	Step *int `json:"step,omitempty"`

	// Revision is the plan-local revision associated with the round.
	Revision int `json:"revision"`

	// ReviewTitle is the human-readable title for the round when one exists.
	ReviewTitle string `json:"review_title,omitempty"`

	// Decision is the aggregate review decision for the round.
	Decision string `json:"decision"`

	// BlockingFindings lists the findings that currently block progression.
	BlockingFindings []ReviewAggregateFinding `json:"blocking_findings"`

	// NonBlockingFindings lists the findings that were recorded without blocking
	// progression.
	NonBlockingFindings []ReviewAggregateFinding `json:"non_blocking_findings"`

	// AggregatedAt is the aggregate timestamp.
	AggregatedAt string `json:"aggregated_at"`
}

// ReviewAggregateFinding is one aggregate finding annotated with its slot and
// dimension context.
type ReviewAggregateFinding struct {
	// Slot is the stable reviewer slot identifier.
	Slot string `json:"slot"`

	// Dimension is the human-readable review dimension label.
	Dimension string `json:"dimension"`

	// Severity is the finding severity label.
	Severity string `json:"severity"`

	// Title is the short human-readable title of the finding.
	Title string `json:"title"`

	// Details is the full review finding explanation.
	Details string `json:"details"`

	// Locations optionally lists lightweight repo-relative source anchors for
	// the finding, such as "path/to/file.go", "path/to/file.go#L123", or
	// "path/to/file.go#L1-L3".
	Locations []string `json:"locations,omitempty"`

	// HasLocations records whether the payload explicitly included the optional
	// locations field so empty arrays can round-trip without being collapsed
	// into omission.
	HasLocations bool `json:"-"`
}

func (f ReviewFinding) MarshalJSON() ([]byte, error) {
	type payload struct {
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations,omitempty"`
	}
	if f.HasLocations {
		type payloadWithLocations struct {
			Severity  string   `json:"severity"`
			Title     string   `json:"title"`
			Details   string   `json:"details"`
			Locations []string `json:"locations"`
		}
		return json.Marshal(payloadWithLocations{
			Severity:  f.Severity,
			Title:     f.Title,
			Details:   f.Details,
			Locations: f.Locations,
		})
	}
	return json.Marshal(payload{
		Severity:  f.Severity,
		Title:     f.Title,
		Details:   f.Details,
		Locations: f.Locations,
	})
}

func (f *ReviewFinding) UnmarshalJSON(data []byte) error {
	type payload struct {
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations"`
	}
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.Severity = decoded.Severity
	f.Title = decoded.Title
	f.Details = decoded.Details
	f.Locations = decoded.Locations
	_, f.HasLocations = raw["locations"]
	return nil
}

func (f ReviewAggregateFinding) MarshalJSON() ([]byte, error) {
	type payload struct {
		Slot      string   `json:"slot"`
		Dimension string   `json:"dimension"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations,omitempty"`
	}
	if f.HasLocations {
		type payloadWithLocations struct {
			Slot      string   `json:"slot"`
			Dimension string   `json:"dimension"`
			Severity  string   `json:"severity"`
			Title     string   `json:"title"`
			Details   string   `json:"details"`
			Locations []string `json:"locations"`
		}
		return json.Marshal(payloadWithLocations{
			Slot:      f.Slot,
			Dimension: f.Dimension,
			Severity:  f.Severity,
			Title:     f.Title,
			Details:   f.Details,
			Locations: f.Locations,
		})
	}
	return json.Marshal(payload{
		Slot:      f.Slot,
		Dimension: f.Dimension,
		Severity:  f.Severity,
		Title:     f.Title,
		Details:   f.Details,
		Locations: f.Locations,
	})
}

func (f *ReviewAggregateFinding) UnmarshalJSON(data []byte) error {
	type payload struct {
		Slot      string   `json:"slot"`
		Dimension string   `json:"dimension"`
		Severity  string   `json:"severity"`
		Title     string   `json:"title"`
		Details   string   `json:"details"`
		Locations []string `json:"locations"`
	}
	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	f.Slot = decoded.Slot
	f.Dimension = decoded.Dimension
	f.Severity = decoded.Severity
	f.Title = decoded.Title
	f.Details = decoded.Details
	f.Locations = decoded.Locations
	_, f.HasLocations = raw["locations"]
	return nil
}

// ReviewStartResult is the JSON result returned by `harness review start`.
type ReviewStartResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the created review artifacts for the round.
	Artifacts *ReviewStartArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewStartArtifacts lists the review artifacts created by
// `harness review start`.
type ReviewStartArtifacts struct {
	// PlanPath is the current plan path associated with the review round.
	PlanPath string `json:"plan_path"`

	// LocalStatePath is the plan-local state cache path.
	LocalStatePath string `json:"local_state_path"`

	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// ManifestPath is the path to the review manifest artifact.
	ManifestPath string `json:"manifest_path"`

	// LedgerPath is the path to the review ledger artifact.
	LedgerPath string `json:"ledger_path"`

	// AggregatePath is the path to the review aggregate artifact.
	AggregatePath string `json:"aggregate_path"`

	// Slots lists the materialized review slots created for the round.
	Slots []ReviewManifestSlot `json:"slots"`
}

// ReviewSubmitResult is the JSON result returned by `harness review submit`.
type ReviewSubmitResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the created submission artifacts.
	Artifacts *ReviewSubmitArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewSubmitArtifacts lists the artifacts touched by `harness review submit`.
type ReviewSubmitArtifacts struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// Slot is the stable reviewer slot identifier.
	Slot string `json:"slot"`

	// SubmissionPath is the path to the created submission artifact.
	SubmissionPath string `json:"submission_path"`

	// LedgerPath is the path to the updated review ledger artifact.
	LedgerPath string `json:"ledger_path"`
}

// ReviewAggregateResult is the JSON result returned by
// `harness review aggregate`.
type ReviewAggregateResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the updated aggregate artifacts.
	Artifacts *ReviewAggregateArtifacts `json:"artifacts,omitempty"`

	// Review is the aggregate decision payload when aggregation succeeded.
	Review *ReviewAggregate `json:"review,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// ReviewAggregateArtifacts lists the artifacts touched by
// `harness review aggregate`.
type ReviewAggregateArtifacts struct {
	// RoundID is the stable identifier for the review round.
	RoundID string `json:"round_id"`

	// AggregatePath is the path to the updated aggregate artifact.
	AggregatePath string `json:"aggregate_path"`

	// LocalStatePath is the plan-local state cache path.
	LocalStatePath string `json:"local_state_path"`
}
