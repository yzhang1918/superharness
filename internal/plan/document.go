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
	Status                 string
	SectionOrder           []string
	Sections               map[string]string
	StepAcceptanceCriteria []DocumentCheckbox
}

type DocumentCheckbox struct {
	Checked bool
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
			Title:        parsedStep.title,
			Status:       parsedStep.status,
			SectionOrder: append([]string(nil), parsedStep.sectionOrder...),
			Sections:     map[string]string{},
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
	for i := range d.Steps {
		if d.Steps[i].Status == "in_progress" {
			return &d.Steps[i]
		}
	}
	for i := range d.Steps {
		if d.Steps[i].Status == "pending" {
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
		if step.Status != "completed" {
			return false
		}
	}
	return true
}

func (d *Document) HasPendingArchivePlaceholders() bool {
	for _, sectionName := range []string{"Validation Summary", "Review Summary", "Archive Summary", "Outcome Summary"} {
		section := d.Sections[sectionName]
		if section != nil && strings.Contains(strings.Join(section.lines, "\n"), "PENDING_UNTIL_ARCHIVE") {
			return true
		}
	}
	return false
}

func (d *Document) CompletedStepsHavePendingPlaceholders() bool {
	for _, step := range d.Steps {
		if step.Status != "completed" {
			continue
		}
		if step.Sections["Execution Notes"] == "PENDING_STEP_EXECUTION" {
			return true
		}
		if step.Sections["Review Notes"] == "PENDING_STEP_REVIEW" {
			return true
		}
	}
	return false
}
