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
	Title      string
	Timestamp  time.Time
	SourceType string
	SourceRefs []string
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

	return rendered, nil
}
