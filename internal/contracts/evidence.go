package contracts

// EvidenceSubmitResult is the JSON result returned by `harness evidence
// submit`.
type EvidenceSubmitResult struct {
	// OK reports whether the command succeeded.
	OK bool `json:"ok"`

	// Command is the stable command identifier for the result payload.
	Command string `json:"command"`

	// Summary is the concise human-readable outcome description.
	Summary string `json:"summary"`

	// Artifacts points to the evidence record paths created by the command.
	Artifacts *EvidenceArtifacts `json:"artifacts,omitempty"`

	// NextAction lists the most relevant follow-up steps in priority order.
	NextAction []NextAction `json:"next_actions"`

	// Errors lists hard failures that prevented the command from succeeding.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// EvidenceArtifacts lists the evidence-related paths and identifiers touched by
// `harness evidence submit`.
type EvidenceArtifacts struct {
	// PlanPath is the archived plan path associated with the evidence record.
	PlanPath string `json:"plan_path"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty"`

	// RecordID is the stable identifier of the created evidence record.
	RecordID string `json:"record_id"`

	// RecordPath is the path to the created evidence record artifact.
	RecordPath string `json:"record_path"`

	// Kind is the evidence kind such as ci, publish, or sync.
	Kind string `json:"kind"`
}

// EvidenceCIInput is the JSON input consumed by `harness evidence submit --kind
// ci`.
type EvidenceCIInput struct {
	// Status is the CI result status.
	Status string `json:"status"`

	// Provider is the CI provider name when one is available.
	Provider string `json:"provider,omitempty"`

	// URL is the CI run URL when one is available.
	URL string `json:"url,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}

// EvidencePublishInput is the JSON input consumed by `harness evidence submit
// --kind publish`.
type EvidencePublishInput struct {
	// Status is the publish result status.
	Status string `json:"status"`

	// PRURL is the pull request URL when one is available.
	PRURL string `json:"pr_url,omitempty"`

	// Branch is the candidate branch name when one is available.
	Branch string `json:"branch,omitempty"`

	// Base is the base branch name when one is available.
	Base string `json:"base,omitempty"`

	// Commit is the pushed commit SHA when one is available.
	Commit string `json:"commit,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}

// EvidenceSyncInput is the JSON input consumed by `harness evidence submit
// --kind sync`.
type EvidenceSyncInput struct {
	// Status is the remote-sync result status.
	Status string `json:"status"`

	// BaseRef is the compared base ref when one is available.
	BaseRef string `json:"base_ref,omitempty"`

	// HeadRef is the compared head ref when one is available.
	HeadRef string `json:"head_ref,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}

// EvidenceCIRecord is the command-owned CI evidence artifact.
type EvidenceCIRecord struct {
	// RecordID is the stable identifier of the evidence record.
	RecordID string `json:"record_id"`

	// Kind is the evidence kind, always `ci` for this record family.
	Kind string `json:"kind"`

	// PlanPath is the archived plan path associated with the record.
	PlanPath string `json:"plan_path"`

	// PlanStem is the durable plan stem associated with the record.
	PlanStem string `json:"plan_stem"`

	// Revision is the plan-local revision associated with the record.
	Revision int `json:"revision"`

	// RecordedAt is the record creation timestamp.
	RecordedAt string `json:"recorded_at"`

	// Status is the CI result status.
	Status string `json:"status"`

	// Provider is the CI provider name when one is available.
	Provider string `json:"provider,omitempty"`

	// URL is the CI run URL when one is available.
	URL string `json:"url,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}

// EvidencePublishRecord is the command-owned publish evidence artifact.
type EvidencePublishRecord struct {
	// RecordID is the stable identifier of the evidence record.
	RecordID string `json:"record_id"`

	// Kind is the evidence kind, always `publish` for this record family.
	Kind string `json:"kind"`

	// PlanPath is the archived plan path associated with the record.
	PlanPath string `json:"plan_path"`

	// PlanStem is the durable plan stem associated with the record.
	PlanStem string `json:"plan_stem"`

	// Revision is the plan-local revision associated with the record.
	Revision int `json:"revision"`

	// RecordedAt is the record creation timestamp.
	RecordedAt string `json:"recorded_at"`

	// Status is the publish result status.
	Status string `json:"status"`

	// PRURL is the pull request URL when one is available.
	PRURL string `json:"pr_url,omitempty"`

	// Branch is the candidate branch name when one is available.
	Branch string `json:"branch,omitempty"`

	// Base is the base branch name when one is available.
	Base string `json:"base,omitempty"`

	// Commit is the pushed commit SHA when one is available.
	Commit string `json:"commit,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}

// EvidenceSyncRecord is the command-owned sync evidence artifact.
type EvidenceSyncRecord struct {
	// RecordID is the stable identifier of the evidence record.
	RecordID string `json:"record_id"`

	// Kind is the evidence kind, always `sync` for this record family.
	Kind string `json:"kind"`

	// PlanPath is the archived plan path associated with the record.
	PlanPath string `json:"plan_path"`

	// PlanStem is the durable plan stem associated with the record.
	PlanStem string `json:"plan_stem"`

	// Revision is the plan-local revision associated with the record.
	Revision int `json:"revision"`

	// RecordedAt is the record creation timestamp.
	RecordedAt string `json:"recorded_at"`

	// Status is the remote-sync result status.
	Status string `json:"status"`

	// BaseRef is the compared base ref when one is available.
	BaseRef string `json:"base_ref,omitempty"`

	// HeadRef is the compared head ref when one is available.
	HeadRef string `json:"head_ref,omitempty"`

	// Reason is the human-readable explanation attached to the evidence record.
	Reason string `json:"reason,omitempty"`
}
