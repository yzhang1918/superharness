package plan

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Path          string
	Frontmatter   Frontmatter
	Title         string
	Sections      map[string]*section
	SectionOrder  []string
	Steps         []DocumentStep
	PathKind      string
	DeferredItems bool
}

type DocumentStep struct {
	Title                  string
	Done                   bool
	UsesDoneMarker         bool
	Status                 string
	SectionOrder           []string
	Sections               map[string]string
	StepAcceptanceCriteria []DocumentCheckbox
}

type DocumentCheckbox struct {
	Checked bool
}

type DocumentIssue struct {
	Path    string
	Message string
}

func LoadFile(path string) (*Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}

	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &fm); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}

	title, sections, order := parseTopSections(body)
	ctx := &lintContext{
		frontmatter:  fm,
		title:        title,
		sections:     sections,
		sectionOrder: order,
		pathKind:     inferPathKind(path),
	}
	steps, issues := parseSteps(ctx)
	if len(issues) > 0 {
		return nil, fmt.Errorf("parse steps: %s", issues[0].Message)
	}

	doc := &Document{
		Path:         path,
		Frontmatter:  fm,
		Title:        title,
		Sections:     sections,
		SectionOrder: order,
		PathKind:     ctx.pathKind,
	}

	for _, parsedStep := range steps {
		stepDoc := DocumentStep{
			Title:          parsedStep.title,
			Done:           parsedStep.done,
			UsesDoneMarker: parsedStep.usesDoneMarker,
			Status:         parsedStep.status,
			SectionOrder:   append([]string(nil), parsedStep.sectionOrder...),
			Sections:       map[string]string{},
		}
		for name, section := range parsedStep.sections {
			stepDoc.Sections[name] = strings.TrimSpace(strings.Join(section.lines, "\n"))
		}
		for _, item := range parsedStep.stepAcceptanceCriteria {
			stepDoc.StepAcceptanceCriteria = append(stepDoc.StepAcceptanceCriteria, DocumentCheckbox{Checked: item.Checked})
		}
		doc.Steps = append(doc.Steps, stepDoc)
	}

	if deferredSection := doc.Sections["Deferred Items"]; deferredSection != nil {
		doc.DeferredItems = hasRealDeferredItems(strings.Join(deferredSection.lines, "\n"))
	}

	return doc, nil
}

func (d *Document) CurrentStep() *DocumentStep {
	hasDoneMarkers := false
	for i := range d.Steps {
		if d.Steps[i].UsesDoneMarker {
			hasDoneMarkers = true
			break
		}
	}
	if hasDoneMarkers {
		for i := range d.Steps {
			if !d.Steps[i].Done {
				return &d.Steps[i]
			}
		}
		return nil
	}
	for i := range d.Steps {
		if d.Steps[i].Status == "in_progress" {
			return &d.Steps[i]
		}
	}
	for i := range d.Steps {
		if !d.Steps[i].Done {
			return &d.Steps[i]
		}
	}
	return nil
}

func (d *Document) AllAcceptanceChecked() bool {
	section := d.Sections["Acceptance Criteria"]
	if section == nil {
		return false
	}
	items, err := parseCheckboxList(section.lines)
	if err != nil || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.Checked {
			return false
		}
	}
	return true
}

func (d *Document) AllStepsCompleted() bool {
	if len(d.Steps) == 0 {
		return false
	}
	for _, step := range d.Steps {
		if !step.Done {
			return false
		}
	}
	return true
}

func (d *Document) HasPendingArchivePlaceholders() bool {
	for _, sectionName := range []string{"Validation Summary", "Review Summary", "Archive Summary", "Outcome Summary"} {
		section := d.Sections[sectionName]
		if section != nil && containsArchivePlaceholderToken(strings.Join(section.lines, "\n")) {
			return true
		}
	}
	return false
}

func (d *Document) CompletedStepsHavePendingPlaceholders() bool {
	for _, step := range d.Steps {
		if !step.Done {
			continue
		}
		if step.Sections["Execution Notes"] == PlaceholderPendingStepExecution {
			return true
		}
		if step.Sections["Review Notes"] == PlaceholderPendingStepReview {
			return true
		}
	}
	return false
}

func (d *Document) ArchiveReadinessIssues() []DocumentIssue {
	issues := make([]DocumentIssue, 0)
	if !d.AllAcceptanceChecked() {
		issues = append(issues, DocumentIssue{
			Path:    "section.Acceptance Criteria",
			Message: "all acceptance criteria must be checked before archive",
		})
	}
	if !d.AllStepsCompleted() {
		issues = append(issues, DocumentIssue{
			Path:    "section.Work Breakdown",
			Message: "all steps must be completed before archive",
		})
	}

	for _, sectionName := range []string{"Validation Summary", "Review Summary", "Archive Summary", "Outcome Summary"} {
		section := d.Sections[sectionName]
		if section != nil && containsArchivePlaceholderToken(strings.Join(section.lines, "\n")) {
			issues = append(issues, DocumentIssue{
				Path:    "section." + sectionName,
				Message: "replace archive-time placeholder tokens before archive",
			})
		}
	}

	for _, step := range d.Steps {
		if !step.Done {
			continue
		}
		if step.Sections["Execution Notes"] == PlaceholderPendingStepExecution {
			issues = append(issues, DocumentIssue{
				Path:    "step." + step.Title + ".Execution Notes",
				Message: "replace PENDING_STEP_EXECUTION before archive",
			})
		}
		if step.Sections["Review Notes"] == PlaceholderPendingStepReview {
			issues = append(issues, DocumentIssue{
				Path:    "step." + step.Title + ".Review Notes",
				Message: "replace PENDING_STEP_REVIEW before archive",
			})
		}
	}

	archiveSummary := d.SectionText("Archive Summary")
	for _, label := range []string{"PR", "Ready", "Merge Handoff"} {
		if !strings.Contains(archiveSummary, "- "+label+":") {
			issues = append(issues, DocumentIssue{
				Path:    "section.Archive Summary",
				Message: fmt.Sprintf("add archive summary line for: %s", label),
			})
		}
	}

	if d.DeferredItems && d.followUpIssuesUnset() {
		issues = append(issues, DocumentIssue{
			Path:    "section.Outcome Summary.Follow-Up Issues",
			Message: "replace NONE with follow-up information before archive when deferred items remain",
		})
	}

	return issues
}

func (d *Document) SectionText(name string) string {
	section := d.Sections[name]
	if section == nil {
		return ""
	}
	return strings.TrimSpace(strings.Join(section.lines, "\n"))
}

func (d *Document) followUpIssuesUnset() bool {
	section := d.Sections["Outcome Summary"]
	if section == nil {
		return true
	}
	subsections, _ := parseLevelThreeSections(section.lines)
	followUp := subsections["Follow-Up Issues"]
	if followUp == nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Join(followUp.lines, "\n")), "NONE")
}
