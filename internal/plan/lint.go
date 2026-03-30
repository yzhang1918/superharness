package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	templateassets "github.com/catu-ai/easyharness/assets/templates"
	"gopkg.in/yaml.v3"
)

var (
	stepHeadingPattern     = regexp.MustCompile(`^### Step [1-9][0-9]*: .+$`)
	planFilenamePattern    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-([a-z0-9]+(?:-[a-z0-9]+)*)\.md$`)
	templateVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	checkboxPattern        = regexp.MustCompile(`^- \[( |x|X)\] .+`)
	donePattern            = regexp.MustCompile(`^- Done:\s*\[( |x|X)\]\s*$`)
	statusPattern          = regexp.MustCompile(`^- Status:\s*(\S+)\s*$`)
)

var (
	requiredTopSections = []string{
		"Goal",
		"Scope",
		"Acceptance Criteria",
		"Deferred Items",
		"Work Breakdown",
		"Validation Strategy",
		"Risks",
		"Validation Summary",
		"Review Summary",
		"Archive Summary",
		"Outcome Summary",
	}
	requiredStepSections = []string{
		"Objective",
		"Details",
		"Expected Files",
		"Validation",
		"Execution Notes",
		"Review Notes",
	}
	stepSectionOrder = map[string]int{
		"Objective":                0,
		"Details":                  1,
		"Step Acceptance Criteria": 2,
		"Expected Files":           3,
		"Validation":               4,
		"Execution Notes":          5,
		"Review Notes":             6,
	}
	allowedStepStatuses = []string{"pending", "in_progress", "completed", "blocked"}
)

type Frontmatter struct {
	TemplateVersion string   `yaml:"template_version"`
	CreatedAt       string   `yaml:"created_at"`
	SourceType      string   `yaml:"source_type"`
	SourceRefs      []string `yaml:"source_refs"`
}

type LintIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type LintResult struct {
	OK                       bool          `json:"ok"`
	Command                  string        `json:"command"`
	Summary                  string        `json:"summary"`
	Artifacts                lintArtifacts `json:"artifacts,omitempty"`
	SupportedTemplateVersion string        `json:"supported_template_version,omitempty"`
	Errors                   []LintIssue   `json:"errors,omitempty"`
}

type lintArtifacts struct {
	PlanPath string `json:"plan_path"`
}

type lintContext struct {
	path           string
	frontmatter    Frontmatter
	rawFrontmatter map[string]any
	title          string
	sections       map[string]*section
	sectionOrder   []string
	steps          []step
	pathKind       string
}

type section struct {
	name  string
	lines []string
}

type step struct {
	title                  string
	done                   bool
	usesDoneMarker         bool
	status                 string
	sections               map[string]*section
	sectionOrder           []string
	stepAcceptanceCriteria []checkboxItem
}

type checkboxItem struct {
	Checked bool
}

func LintFile(path string) LintResult {
	result := LintResult{
		Command:   "plan lint",
		Artifacts: lintArtifacts{PlanPath: path},
	}

	ctx, issues := parseAndValidate(path)
	if len(issues) > 0 {
		result.OK = false
		result.Summary = fmt.Sprintf("Plan is invalid with %d issue(s).", len(issues))
		result.Errors = issues
		if version, err := templateassets.PlanTemplateVersion(); err == nil {
			result.SupportedTemplateVersion = version
		}
		return result
	}

	version, err := templateassets.PlanTemplateVersion()
	if err == nil {
		result.SupportedTemplateVersion = version
	}
	result.OK = true
	result.Summary = fmt.Sprintf("Plan %q is valid.", ctx.title)
	return result
}

func parseAndValidate(path string) (*lintContext, []LintIssue) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, []LintIssue{{Path: "file", Message: err.Error()}}
	}

	issues := make([]LintIssue, 0)
	ctx := &lintContext{}
	ctx.path = path
	ctx.pathKind = inferPathKind(path)

	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, []LintIssue{{Path: "frontmatter", Message: err.Error()}}
	}

	var rawMap map[string]any
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &rawMap); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML: %v", err)})
	} else {
		ctx.rawFrontmatter = rawMap
	}

	if err := yaml.Unmarshal([]byte(rawFrontmatter), &ctx.frontmatter); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter", Message: fmt.Sprintf("invalid YAML structure: %v", err)})
	}

	ctx.title, ctx.sections, ctx.sectionOrder = parseTopSections(body)
	if strings.TrimSpace(ctx.title) == "" {
		issues = append(issues, LintIssue{Path: "title", Message: "missing H1 title"})
	}

	issues = append(issues, validateFrontmatter(ctx)...)
	issues = append(issues, validateSectionOrder(ctx)...)
	issues = append(issues, validateScope(ctx)...)
	issues = append(issues, validateAcceptanceCriteria(ctx)...)
	issues = append(issues, validateOutcomeSummary(ctx)...)

	steps, stepIssues := parseSteps(ctx)
	ctx.steps = steps
	issues = append(issues, stepIssues...)
	issues = append(issues, validateStepMarkers(ctx)...)

	issues = append(issues, validatePathRules(ctx)...)
	issues = append(issues, validateArchivedRules(ctx)...)

	return ctx, issues
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", errors.New("file must start with YAML frontmatter delimited by ---")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", errors.New("frontmatter is missing a closing --- delimiter")
}

func parseTopSections(body string) (string, map[string]*section, []string) {
	lines := strings.Split(body, "\n")
	title := ""
	sections := map[string]*section{}
	order := make([]string, 0)
	var current *section

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		switch {
		case strings.HasPrefix(line, "# "):
			if title == "" {
				title = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			} else if current != nil {
				current.lines = append(current.lines, line)
			}
		case strings.HasPrefix(line, "## "):
			name := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			current = &section{name: name}
			sections[name] = current
			order = append(order, name)
		default:
			if current != nil {
				current.lines = append(current.lines, line)
			}
		}
	}

	return title, sections, order
}

func validateFrontmatter(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	requiredKeys := []string{
		"template_version",
		"created_at",
		"source_type",
		"source_refs",
	}
	for _, key := range requiredKeys {
		if _, ok := ctx.rawFrontmatter[key]; !ok {
			issues = append(issues, LintIssue{Path: "frontmatter." + key, Message: "missing required field"})
		}
	}

	if _, err := time.Parse(time.RFC3339, ctx.frontmatter.CreatedAt); err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter.created_at", Message: "must be RFC3339"})
	}
	if strings.TrimSpace(ctx.frontmatter.CreatedAt) == "" {
		issues = append(issues, LintIssue{Path: "frontmatter.created_at", Message: "must not be empty"})
	}
	if strings.TrimSpace(ctx.frontmatter.SourceType) == "" {
		issues = append(issues, LintIssue{Path: "frontmatter.source_type", Message: "must not be empty"})
	}
	for _, legacyKey := range []string{"status", "lifecycle", "revision", "updated_at"} {
		if _, ok := ctx.rawFrontmatter[legacyKey]; ok {
			issues = append(issues, LintIssue{
				Path:    "frontmatter." + legacyKey,
				Message: "legacy runtime field is no longer allowed in v0.2 tracked plans",
			})
		}
	}
	supportedVersion, err := templateassets.PlanTemplateVersion()
	if err != nil {
		issues = append(issues, LintIssue{Path: "frontmatter.template_version", Message: err.Error()})
	} else if err := validateTemplateVersion(ctx.frontmatter.TemplateVersion, supportedVersion); err != nil {
		issues = append(issues, LintIssue{
			Path:    "frontmatter.template_version",
			Message: err.Error(),
		})
	}

	return issues
}

func validateSectionOrder(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	if !slices.Equal(ctx.sectionOrder, requiredTopSections) {
		issues = append(issues, LintIssue{
			Path:    "sections",
			Message: fmt.Sprintf("top-level sections must appear in order: %s", strings.Join(requiredTopSections, " -> ")),
		})
	}
	return issues
}

func validateStepMarkers(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0, len(ctx.steps))
	for _, step := range ctx.steps {
		if step.usesDoneMarker {
			continue
		}
		issues = append(issues, LintIssue{
			Path:    "step." + step.title,
			Message: "step must use '- Done: [ ]' or '- Done: [x]'; legacy '- Status: ...' is no longer allowed",
		})
	}
	return issues
}

func validateScope(ctx *lintContext) []LintIssue {
	scope := ctx.sections["Scope"]
	if scope == nil {
		return []LintIssue{{Path: "section.Scope", Message: "missing Scope section"}}
	}
	content := strings.Join(scope.lines, "\n")
	issues := make([]LintIssue, 0)
	if !strings.Contains(content, "### In Scope") {
		issues = append(issues, LintIssue{Path: "section.Scope", Message: "missing ### In Scope"})
	}
	if !strings.Contains(content, "### Out of Scope") {
		issues = append(issues, LintIssue{Path: "section.Scope", Message: "missing ### Out of Scope"})
	}
	return issues
}

func validateAcceptanceCriteria(ctx *lintContext) []LintIssue {
	section := ctx.sections["Acceptance Criteria"]
	if section == nil {
		return []LintIssue{{Path: "section.Acceptance Criteria", Message: "missing Acceptance Criteria section"}}
	}
	issues := make([]LintIssue, 0)
	items, err := parseCheckboxList(section.lines)
	if err != nil {
		issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: err.Error()})
		return issues
	}
	if len(items) == 0 {
		issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: "must contain at least one checkbox"})
	}
	if ctx.pathKind == "archived" {
		for _, item := range items {
			if !item.Checked {
				issues = append(issues, LintIssue{Path: "section.Acceptance Criteria", Message: "archived plans must have all acceptance criteria checked"})
				break
			}
		}
	}
	return issues
}

func validateOutcomeSummary(ctx *lintContext) []LintIssue {
	section := ctx.sections["Outcome Summary"]
	if section == nil {
		return []LintIssue{{Path: "section.Outcome Summary", Message: "missing Outcome Summary section"}}
	}

	subsections, order := parseLevelThreeSections(section.lines)
	required := []string{"Delivered", "Not Delivered", "Follow-Up Issues"}
	issues := make([]LintIssue, 0)
	if !slices.Equal(order, required) {
		issues = append(issues, LintIssue{
			Path:    "section.Outcome Summary",
			Message: "Outcome Summary must contain Delivered, Not Delivered, and Follow-Up Issues in order",
		})
	}
	for _, name := range required {
		if subsection := subsections[name]; subsection == nil || strings.TrimSpace(strings.Join(subsection.lines, "\n")) == "" {
			issues = append(issues, LintIssue{Path: "section.Outcome Summary." + name, Message: "must not be empty"})
		}
	}
	return issues
}

func parseSteps(ctx *lintContext) ([]step, []LintIssue) {
	workBreakdown := ctx.sections["Work Breakdown"]
	if workBreakdown == nil {
		return nil, []LintIssue{{Path: "section.Work Breakdown", Message: "missing Work Breakdown section"}}
	}

	lines := workBreakdown.lines
	steps := make([]step, 0)
	issues := make([]LintIssue, 0)
	var current *step
	var buffer []string

	flush := func() {
		if current == nil {
			return
		}
		parsed, errs := finalizeStep(*current, buffer)
		steps = append(steps, parsed)
		issues = append(issues, errs...)
	}

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "### ") {
			flush()
			if !stepHeadingPattern.MatchString(line) {
				issues = append(issues, LintIssue{
					Path:    "section.Work Breakdown",
					Message: fmt.Sprintf("invalid step heading %q; use ### Step N: Title", strings.TrimSpace(strings.TrimPrefix(line, "### "))),
				})
			}
			current = &step{title: strings.TrimSpace(strings.TrimPrefix(line, "### "))}
			buffer = nil
			continue
		}
		if current != nil {
			buffer = append(buffer, line)
		}
	}
	flush()

	if len(steps) == 0 {
		issues = append(issues, LintIssue{Path: "section.Work Breakdown", Message: "must contain at least one step"})
	}
	return steps, issues
}

func finalizeStep(base step, lines []string) (step, []LintIssue) {
	issues := make([]LintIssue, 0)
	stepPath := "step." + base.title
	trimmedIndex := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			trimmedIndex = i
			break
		}
	}
	if trimmedIndex == -1 {
		return base, []LintIssue{{Path: stepPath, Message: "step body is empty"}}
	}
	matches := statusPattern.FindStringSubmatch(strings.TrimSpace(lines[trimmedIndex]))
	if doneMatches := donePattern.FindStringSubmatch(strings.TrimSpace(lines[trimmedIndex])); len(doneMatches) == 2 {
		base.done = strings.EqualFold(doneMatches[1], "x")
		base.usesDoneMarker = true
		base.status = stepStatusFromDone(base.done)
	} else if len(matches) == 2 {
		base.status = matches[1]
		base.done = base.status == "completed"
		if !slices.Contains(allowedStepStatuses, base.status) {
			issues = append(issues, LintIssue{Path: stepPath + ".status", Message: "invalid step status"})
		}
	} else {
		return base, []LintIssue{{Path: stepPath, Message: "step must start with '- Done: [ ]' or legacy '- Status: ...' during migration"}}
	}

	base.sections = map[string]*section{}
	base.sectionOrder = make([]string, 0)

	var current *section
	previousOrder := -1
	for _, rawLine := range lines[trimmedIndex+1:] {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "#### ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "#### "))
			order, ok := stepSectionOrder[name]
			if !ok {
				issues = append(issues, LintIssue{Path: stepPath, Message: fmt.Sprintf("unknown step subsection %q", name)})
				current = nil
				continue
			}
			if order < previousOrder {
				issues = append(issues, LintIssue{Path: stepPath, Message: "step subsections are out of order"})
			}
			previousOrder = order
			current = &section{name: name}
			base.sections[name] = current
			base.sectionOrder = append(base.sectionOrder, name)
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		} else if strings.TrimSpace(line) != "" {
			issues = append(issues, LintIssue{Path: stepPath, Message: "content before first required step subsection"})
		}
	}

	for _, name := range requiredStepSections {
		section := base.sections[name]
		if section == nil {
			issues = append(issues, LintIssue{Path: stepPath, Message: fmt.Sprintf("missing #### %s", name)})
			continue
		}
		if strings.TrimSpace(strings.Join(section.lines, "\n")) == "" {
			issues = append(issues, LintIssue{Path: stepPath + "." + name, Message: "must not be empty"})
		}
	}

	if section := base.sections["Step Acceptance Criteria"]; section != nil {
		items, err := parseCheckboxList(section.lines)
		if err != nil {
			issues = append(issues, LintIssue{Path: stepPath + ".Step Acceptance Criteria", Message: err.Error()})
		} else {
			base.stepAcceptanceCriteria = items
		}
	}

	return base, issues
}

func validatePathRules(ctx *lintContext) []LintIssue {
	issues := make([]LintIssue, 0)
	switch ctx.pathKind {
	case "active":
	case "archived":
	default:
		issues = append(issues, LintIssue{Path: "path", Message: "plan must live under docs/plans/active or docs/plans/archived"})
	}

	if filenameErr := validatePlanFilename(filepath.Base(ctx.path)); filenameErr != nil {
		issues = append(issues, LintIssue{Path: "path", Message: filenameErr.Error()})
	}
	return issues
}

func validateArchivedRules(ctx *lintContext) []LintIssue {
	if ctx.pathKind != "archived" {
		return nil
	}

	issues := make([]LintIssue, 0)
	for _, step := range ctx.steps {
		if !step.done {
			issues = append(issues, LintIssue{Path: "step." + step.title + ".done", Message: "archived plans require every step to be done"})
		}
		for _, item := range step.stepAcceptanceCriteria {
			if !item.Checked {
				issues = append(issues, LintIssue{Path: "step." + step.title + ".Step Acceptance Criteria", Message: "archived plans require checked step-local acceptance criteria"})
				break
			}
		}
		if hasPlaceholder(step.sections["Execution Notes"], "PENDING_STEP_EXECUTION") {
			issues = append(issues, LintIssue{Path: "step." + step.title + ".Execution Notes", Message: "archived completed steps must not keep PENDING_STEP_EXECUTION"})
		}
		if hasPlaceholder(step.sections["Review Notes"], "PENDING_STEP_REVIEW") {
			issues = append(issues, LintIssue{Path: "step." + step.title + ".Review Notes", Message: "archived completed steps must not keep PENDING_STEP_REVIEW"})
		}
	}

	for _, sectionName := range []string{"Validation Summary", "Review Summary", "Archive Summary", "Outcome Summary"} {
		section := ctx.sections[sectionName]
		if section != nil && containsArchivePlaceholderToken(strings.Join(section.lines, "\n")) {
			issues = append(issues, LintIssue{Path: "section." + sectionName, Message: "archived plans must not keep archive-time placeholder tokens"})
		}
	}

	archiveSummary := ctx.sections["Archive Summary"]
	if archiveSummary == nil {
		issues = append(issues, LintIssue{Path: "section.Archive Summary", Message: "missing Archive Summary section"})
	} else {
		requiredLabels := []string{"Archived At", "Revision", "PR", "Ready", "Merge Handoff"}
		content := strings.Join(archiveSummary.lines, "\n")
		for _, label := range requiredLabels {
			if !strings.Contains(content, "- "+label+":") {
				issues = append(issues, LintIssue{Path: "section.Archive Summary", Message: fmt.Sprintf("archived plans must include - %s:", label)})
			}
		}
	}

	if deferredItemsSection := ctx.sections["Deferred Items"]; deferredItemsSection != nil {
		if hasRealDeferredItems(strings.Join(deferredItemsSection.lines, "\n")) {
			outcomeSummary := ctx.sections["Outcome Summary"]
			if outcomeSummary == nil {
				issues = append(issues, LintIssue{Path: "section.Outcome Summary", Message: "missing Outcome Summary section"})
				return issues
			}
			outcomeSubsections, _ := parseLevelThreeSections(outcomeSummary.lines)
			followUp := outcomeSubsections["Follow-Up Issues"]
			if followUp == nil || strings.EqualFold(strings.TrimSpace(strings.Join(followUp.lines, "\n")), "NONE") {
				issues = append(issues, LintIssue{Path: "section.Outcome Summary.Follow-Up Issues", Message: "archived plans with deferred items must include follow-up issue references"})
			}
		}
	}

	return issues
}

func inferPathKind(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(clean, "/docs/plans/active/") || strings.HasPrefix(clean, "docs/plans/active/"):
		return "active"
	case strings.Contains(clean, "/docs/plans/archived/") || strings.HasPrefix(clean, "docs/plans/archived/"):
		return "archived"
	default:
		return ""
	}
}

func parseCheckboxList(lines []string) ([]checkboxItem, error) {
	items := make([]checkboxItem, 0)
	hasItem := false
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		if checkboxPattern.MatchString(line) {
			hasItem = true
			items = append(items, checkboxItem{Checked: strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]")})
			continue
		}
		if strings.HasPrefix(line, "- ") {
			return nil, fmt.Errorf("must use markdown checkboxes")
		}
		if !hasItem {
			return nil, fmt.Errorf("must start with a markdown checkbox")
		}
	}
	return items, nil
}

func parseLevelThreeSections(lines []string) (map[string]*section, []string) {
	sections := map[string]*section{}
	order := make([]string, 0)
	var current *section
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "### ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "### "))
			current = &section{name: name}
			sections[name] = current
			order = append(order, name)
			continue
		}
		if current != nil {
			current.lines = append(current.lines, line)
		}
	}
	return sections, order
}

func hasPlaceholder(section *section, token string) bool {
	if section == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(strings.Join(section.lines, "\n")), token)
}

func hasRealDeferredItems(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return false
	}
	candidates := []string{
		"NONE",
		"None.",
		"- None.",
		"- NONE",
	}
	for _, candidate := range candidates {
		if trimmed == candidate {
			return false
		}
	}
	return true
}

func validatePlanFilename(filename string) error {
	matches := planFilenamePattern.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return fmt.Errorf("plan filename must match YYYY-MM-DD-short-topic.md")
	}
	if _, err := time.Parse("2006-01-02", matches[1]); err != nil {
		return fmt.Errorf("plan filename must start with a valid date")
	}
	return nil
}

func validateTemplateVersion(planVersion, supportedVersion string) error {
	planSemver, err := parseTemplateVersion(planVersion)
	if err != nil {
		return fmt.Errorf("template_version must be semver-like (for example 0.1.0)")
	}
	supportedSemver, err := parseTemplateVersion(supportedVersion)
	if err != nil {
		return fmt.Errorf("supported template version is invalid: %v", err)
	}
	if compareTemplateVersions(planSemver, supportedSemver) > 0 {
		return fmt.Errorf("template_version %q is newer than this harness supports (%q)", planVersion, supportedVersion)
	}
	return nil
}

func parseTemplateVersion(version string) ([3]int, error) {
	matches := templateVersionPattern.FindStringSubmatch(strings.TrimSpace(version))
	if len(matches) != 4 {
		return [3]int{}, fmt.Errorf("invalid version %q", version)
	}

	var parsed [3]int
	for i := 1; i < len(matches); i++ {
		value, err := strconv.Atoi(matches[i])
		if err != nil {
			return [3]int{}, err
		}
		parsed[i-1] = value
	}
	return parsed, nil
}

func compareTemplateVersions(left, right [3]int) int {
	for i := 0; i < len(left); i++ {
		switch {
		case left[i] < right[i]:
			return -1
		case left[i] > right[i]:
			return 1
		}
	}
	return 0
}

func stepStatusFromDone(done bool) string {
	if done {
		return "completed"
	}
	return "pending"
}
