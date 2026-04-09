package contracts

import "encoding/json"

// TimelineEvent is one append-only line in
// `.local/harness/plans/<plan-stem>/events.jsonl`.
type TimelineEvent struct {
	// EventID is the stable identifier for this event within one plan timeline.
	EventID string `json:"event_id"`

	// Sequence is the monotonically increasing event sequence for the plan.
	Sequence int `json:"sequence"`

	// RecordedAt is the event creation timestamp.
	RecordedAt string `json:"recorded_at"`

	// Kind groups related event families such as lifecycle, review, or evidence.
	Kind string `json:"kind"`

	// Command is the harness command that produced the event.
	Command string `json:"command"`

	// Summary is the concise human-readable command outcome.
	Summary string `json:"summary"`

	// PlanPath is the tracked or archived plan path associated with this event.
	PlanPath string `json:"plan_path,omitempty"`

	// PlanStem is the durable plan stem associated with this event.
	PlanStem string `json:"plan_stem"`

	// Revision is the plan-local revision associated with this event.
	Revision int `json:"revision,omitempty"`

	// FromNode is the canonical workflow node before the command when known.
	FromNode string `json:"from_node,omitempty"`

	// ToNode is the canonical workflow node after the command when known.
	ToNode string `json:"to_node,omitempty"`

	// Synthetic reports that this entry is a read-model bootstrap row derived
	// from durable plan or state data rather than a line from `events.jsonl`.
	Synthetic bool `json:"synthetic,omitempty"`

	// Details lists small, display-ready event facts without duplicating full
	// underlying artifact payloads.
	Details []TimelineDetail `json:"details,omitempty"`

	// ArtifactRefs points at high-signal existing harness artifacts related to
	// the event.
	ArtifactRefs []TimelineArtifactRef `json:"artifact_refs,omitempty"`

	// Input is the raw JSON input payload when the command accepted one.
	Input json.RawMessage `json:"input,omitempty"`

	// Output is the raw JSON command result payload.
	Output json.RawMessage `json:"output,omitempty"`

	// Artifacts is the raw JSON artifact payload associated with the event.
	Artifacts json.RawMessage `json:"artifacts,omitempty"`
}

// TimelineDetail is one small display-ready fact carried on a timeline event.
type TimelineDetail struct {
	// Key is the stable label for the detail.
	Key string `json:"key"`

	// Value is the rendered detail value.
	Value string `json:"value"`
}

// TimelineArtifactRef points at one existing harness-owned artifact or
// identifier related to a timeline event.
type TimelineArtifactRef struct {
	// Label is the stable label for the referenced artifact.
	Label string `json:"label"`

	// Value is the rendered identifier or path value.
	Value string `json:"value"`

	// Path is the artifact path when the reference points to a file.
	Path string `json:"path,omitempty"`

	// ContentType reports how Content should be rendered in the UI resource.
	// It is omitted from append-only event lines and only populated by the
	// timeline read model when it resolves a referenced file.
	ContentType string `json:"content_type,omitempty"`

	// Content is the resolved artifact file payload for UI tabs when the
	// reference points at a readable file. It is omitted from append-only event
	// lines and only populated by the timeline read model.
	Content json.RawMessage `json:"content,omitempty"`
}

// TimelineResult is the read-only UI resource returned by `/api/timeline`.
type TimelineResult struct {
	// OK reports whether timeline loading succeeded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded timeline.
	Summary string `json:"summary"`

	// Artifacts points to the plan and local timeline artifact paths used to
	// build this response.
	Artifacts *TimelineArtifacts `json:"artifacts,omitempty"`

	// Events lists the ordered timeline entries for the selected plan.
	Events []TimelineEvent `json:"events"`

	// Errors lists hard failures that prevented timeline loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// TimelineArtifacts points to the current plan-local timeline sources.
type TimelineArtifacts struct {
	// PlanPath is the current or last-landed plan path associated with the
	// timeline.
	PlanPath string `json:"plan_path,omitempty"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty"`

	// EventIndexPath is the append-only event index path.
	EventIndexPath string `json:"event_index_path,omitempty"`
}
