package plan

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	templateassets "github.com/catu-ai/easyharness/assets/templates"
)

const (
	placeholderTitle     = "Replace With Plan Title"
	placeholderTimestamp = "REPLACE_WITH_RFC3339_TIMESTAMP"
)

type TemplateOptions struct {
	Title           string
	Timestamp       time.Time
	SourceType      string
	SourceRefs      []string
	WorkflowProfile string
}

func RenderTemplate(opts TemplateOptions) (string, error) {
	template := templateassets.PlanTemplate()
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = placeholderTitle
	}

	if strings.Contains(title, "\n") {
		return "", fmt.Errorf("title must be a single line")
	}

	if opts.Timestamp.IsZero() {
		opts.Timestamp = time.Now()
	}

	sourceType := strings.TrimSpace(opts.SourceType)
	if sourceType == "" {
		sourceType = "direct_request"
	}
	if opts.SourceRefs == nil {
		opts.SourceRefs = []string{}
	}
	workflowProfile := normalizeWorkflowProfile(opts.WorkflowProfile)
	if workflowProfile != WorkflowProfileStandard && workflowProfile != WorkflowProfileLightweight {
		return "", fmt.Errorf("workflow profile must be %q or %q", WorkflowProfileStandard, WorkflowProfileLightweight)
	}

	sourceRefsJSON, err := json.Marshal(opts.SourceRefs)
	if err != nil {
		return "", fmt.Errorf("marshal source refs: %w", err)
	}

	timestamp := opts.Timestamp.Format(time.RFC3339)

	rendered := template
	rendered = strings.Replace(rendered, "# "+placeholderTitle, "# "+title, 1)
	rendered = strings.ReplaceAll(rendered, placeholderTimestamp, timestamp)
	rendered = strings.Replace(rendered, "source_type: direct_request", "source_type: "+sourceType, 1)
	rendered = strings.Replace(rendered, "source_refs: []", "source_refs: "+string(sourceRefsJSON), 1)
	if workflowProfile == WorkflowProfileLightweight {
		rendered = strings.Replace(rendered, "source_refs: "+string(sourceRefsJSON), "source_refs: "+string(sourceRefsJSON)+"\nworkflow_profile: lightweight", 1)
		rendered = strings.Replace(rendered, "### Step 1: Replace with first step title", "### Step 1: Describe the low-risk change", 1)
		rendered = strings.Replace(rendered, "Describe the concrete outcome for this step.", "Describe the narrow low-risk change to make.", 1)
		rendered = strings.Replace(rendered, "Describe the step-specific details, tradeoffs, or constraints that do not fit\nin the one-line objective. Write `NONE` if the objective is already enough.", "Explain why this change qualifies for the lightweight path and note any constraints. Write `NONE` if the objective already says enough.", 1)
		rendered = strings.Replace(rendered, "Describe the validation approach for the whole plan.", "Describe the focused validation needed for this low-risk change.", 1)
		stepTwoMarker := "\n### Step 2: Replace with second step title"
		if start := strings.Index(rendered, stepTwoMarker); start >= 0 {
			if end := strings.Index(rendered[start:], "\n## Validation Strategy"); end >= 0 {
				rendered = rendered[:start] + rendered[start+end:]
			}
		}
	}

	return rendered, nil
}
