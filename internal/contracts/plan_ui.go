package contracts

// PlanResult is the read-only UI resource returned by `/api/plan`.
type PlanResult struct {
	// OK reports whether plan loading succeeded.
	OK bool `json:"ok"`

	// Resource is the stable UI resource identifier.
	Resource string `json:"resource"`

	// Summary is the concise human-readable explanation of the loaded plan
	// state.
	Summary string `json:"summary"`

	// Artifacts points to the active-plan package paths used to build this
	// response.
	Artifacts *PlanArtifacts `json:"artifacts,omitempty"`

	// Document is the current plan markdown document when one is available for
	// browsing.
	Document *PlanDocumentView `json:"document,omitempty"`

	// Supplements is the current plan package supplements tree when one exists.
	Supplements *PlanNodeView `json:"supplements,omitempty"`

	// Warnings lists non-fatal degraded-state notes for the overall plan
	// resource.
	Warnings []string `json:"warnings,omitempty"`

	// Errors lists hard failures that prevented plan loading.
	Errors []ErrorDetail `json:"errors,omitempty"`
}

// PlanArtifacts points to the current plan package sources.
type PlanArtifacts struct {
	// PlanPath is the current plan path relevant to the plan browser.
	PlanPath string `json:"plan_path,omitempty"`

	// SupplementsPath is the current plan's companion supplements directory
	// when one exists.
	SupplementsPath string `json:"supplements_path,omitempty"`

	// LocalStatePath is the plan-local control-plane state path when one exists.
	LocalStatePath string `json:"local_state_path,omitempty"`
}

// PlanDocumentView is the main markdown document for the current plan.
type PlanDocumentView struct {
	// Title is the plan title rendered from the markdown H1.
	Title string `json:"title"`

	// Path is the current plan markdown path.
	Path string `json:"path"`

	// Markdown is the frontmatter-free markdown body for the plan document.
	Markdown string `json:"markdown"`

	// Headings is the hierarchical heading tree used by the Plan explorer.
	Headings []PlanHeadingView `json:"headings"`
}

// PlanHeadingView is one heading node in the current plan markdown document.
type PlanHeadingView struct {
	// ID is the stable node identifier for this heading within the plan tree.
	ID string `json:"id"`

	// Label is the rendered heading text.
	Label string `json:"label"`

	// Level is the markdown heading depth from 1 to 6.
	Level int `json:"level"`

	// Anchor is the stable in-document anchor for reader navigation.
	Anchor string `json:"anchor"`

	// Children lists nested heading nodes.
	Children []PlanHeadingView `json:"children,omitempty"`
}

// PlanNodeView is one supplement tree node in the current plan package.
type PlanNodeView struct {
	// ID is the stable explorer node identifier.
	ID string `json:"id"`

	// Kind reports whether the node is a directory or file.
	Kind string `json:"kind"`

	// Label is the display label for the node.
	Label string `json:"label"`

	// Path is the repository-relative node path.
	Path string `json:"path,omitempty"`

	// Children lists child directory or file nodes.
	Children []PlanNodeView `json:"children,omitempty"`

	// Preview is the file preview payload when Kind is `file`.
	Preview *PlanPreview `json:"preview,omitempty"`
}

// PlanPreview is the preview state for one supplement file.
type PlanPreview struct {
	// Status reports whether the preview is fully supported, plain-text
	// fallback, or unavailable.
	Status string `json:"status"`

	// ContentType reports how the preview should be rendered.
	ContentType string `json:"content_type,omitempty"`

	// Content carries the preview body when a preview is available.
	Content string `json:"content,omitempty"`

	// Reason explains why preview is unavailable or downgraded.
	Reason string `json:"reason,omitempty"`

	// ByteSize is the source file size in bytes.
	ByteSize int64 `json:"byte_size,omitempty"`

	// Extension is the normalized lowercase file extension without the leading
	// dot.
	Extension string `json:"extension,omitempty"`
}
