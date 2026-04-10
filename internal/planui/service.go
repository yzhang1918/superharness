package planui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

const maxPreviewBytes int64 = 256 * 1024

var richPreviewExtensions = map[string]string{
	"json": "json",
	"md":   "markdown",
	"txt":  "text",
	"yaml": "yaml",
	"yml":  "yaml",
}

var imageExtensions = map[string]bool{
	"apng": true,
	"avif": true,
	"gif":  true,
	"jpeg": true,
	"jpg":  true,
	"png":  true,
	"svg":  true,
	"webp": true,
}

type Service struct {
	Workdir string
}

type Result = contracts.PlanResult
type Artifacts = contracts.PlanArtifacts
type Document = contracts.PlanDocumentView
type Heading = contracts.PlanHeadingView
type Node = contracts.PlanNodeView
type Preview = contracts.PlanPreview
type ErrorDetail = contracts.ErrorDetail

type headingEntry struct {
	Label string
	Level int
	ID    string
}

type headingTree struct {
	Heading
	children []*headingTree
}

func (s Service) Read() Result {
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			return Result{
				OK:       true,
				Resource: "plan",
				Summary:  "No current active plan is available to browse in this worktree.",
			}
		}
		return Result{
			OK:       false,
			Resource: "plan",
			Summary:  "Unable to determine the current plan for plan browsing.",
			Errors:   []ErrorDetail{{Path: "plan", Message: err.Error()}},
		}
	}

	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "plan",
			Summary:  "Unable to determine the current plan path for plan browsing.",
			Errors:   []ErrorDetail{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	if !strings.HasPrefix(relPlanPath, "docs/plans/active/") {
		return Result{
			OK:       true,
			Resource: "plan",
			Summary:  "Plan browsing only shows the current active tracked plan.",
			Artifacts: &Artifacts{
				PlanPath: relPlanPath,
			},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
	state, statePath, stateErr := runstate.LoadState(s.Workdir, planStem)
	warnings := make([]string, 0, 2)
	if stateErr != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read local plan state for %s; some artifact hints may be incomplete.", planStem))
	}

	document, err := loadDocument(planPath, relPlanPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "plan",
			Summary:  "Unable to read the current plan document.",
			Artifacts: &Artifacts{
				PlanPath:       relPlanPath,
				LocalStatePath: statePath,
			},
			Warnings: warnings,
			Errors:   []ErrorDetail{{Path: "plan", Message: err.Error()}},
		}
	}

	artifacts := &Artifacts{
		PlanPath: relPlanPath,
	}
	if state != nil || statePath != "" {
		artifacts.LocalStatePath = statePath
	}

	var supplements *Node
	supplementsPath := plan.SupplementsDirForPlanPath(planPath)
	if info, err := os.Stat(supplementsPath); err == nil && info.IsDir() {
		artifacts.SupplementsPath = filepath.ToSlash(mustRel(s.Workdir, supplementsPath))
		rootNode, nodeWarnings, buildErr := buildSupplementsRoot(s.Workdir, supplementsPath)
		warnings = append(warnings, nodeWarnings...)
		if buildErr != nil {
			return Result{
				OK:        false,
				Resource:  "plan",
				Summary:   "Unable to read the current plan supplements.",
				Artifacts: artifacts,
				Document:  document,
				Warnings:  warnings,
				Errors:    []ErrorDetail{{Path: "supplements", Message: buildErr.Error()}},
			}
		}
		supplements = rootNode
	} else if err != nil && !os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Unable to inspect supplements path %s: %v", filepath.ToSlash(supplementsPath), err))
	} else if err == nil && !info.IsDir() {
		warnings = append(warnings, fmt.Sprintf("Supplements path is not a directory: %s", filepath.ToSlash(supplementsPath)))
	}

	return Result{
		OK:       true,
		Resource: "plan",
		Summary:  fmt.Sprintf("Loaded the active plan package for %s.", filepath.Base(relPlanPath)),
		Artifacts: artifacts,
		Document:  document,
		Supplements: supplements,
		Warnings:  warnings,
	}
}

func loadDocument(absPath, relPath string) (*Document, error) {
	doc, err := plan.LoadFile(absPath)
	if err != nil {
		return nil, err
	}
	body, err := extractPlanBody(absPath)
	if err != nil {
		return nil, err
	}
	return &Document{
		Title:    doc.Title,
		Path:     relPath,
		Markdown: body,
		Headings: buildHeadingTree(body, doc.Title),
	}, nil
}

func extractPlanBody(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(content)
	lines := strings.Split(text, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("file must start with YAML frontmatter delimited by ---")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", errors.New("frontmatter is missing a closing --- delimiter")
}

func buildHeadingTree(body, title string) []Heading {
	entries := parseHeadingEntries(body, title)
	if len(entries) == 0 {
		return []Heading{}
	}

	roots := make([]*headingTree, 0)
	stack := make([]*headingTree, 0)
	for _, entry := range entries {
		node := &headingTree{
			Heading: Heading{
				ID:     entry.ID,
				Label:  entry.Label,
				Level:  entry.Level,
				Anchor: entry.ID,
			},
		}
		for len(stack) > 0 && stack[len(stack)-1].Level >= entry.Level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			stack[len(stack)-1].children = append(stack[len(stack)-1].children, node)
		}
		stack = append(stack, node)
	}

	out := make([]Heading, 0, len(roots))
	for _, root := range roots {
		out = append(out, materializeHeading(root))
	}
	return out
}

func materializeHeading(node *headingTree) Heading {
	view := node.Heading
	if len(node.children) == 0 {
		return view
	}
	view.Children = make([]Heading, 0, len(node.children))
	for _, child := range node.children {
		view.Children = append(view.Children, materializeHeading(child))
	}
	return view
}

func parseHeadingEntries(body, title string) []headingEntry {
	lines := strings.Split(body, "\n")
	entries := make([]headingEntry, 0, 16)
	inFence := false
	titleDropped := false
	usedIDs := map[string]int{}

	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		level, label, ok := markdownHeading(trimmed)
		if !ok {
			continue
		}
		if !titleDropped && level == 1 && normalizeSpace(label) == normalizeSpace(title) {
			titleDropped = true
			continue
		}
		id := uniqueSlug(label, usedIDs)
		entries = append(entries, headingEntry{
			Label: label,
			Level: level,
			ID:    id,
		})
	}

	return entries
}

func markdownHeading(line string) (int, string, bool) {
	if line == "" || !strings.HasPrefix(line, "#") {
		return 0, "", false
	}
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 6 {
		return 0, "", false
	}
	if len(line) <= level || !unicode.IsSpace(rune(line[level])) {
		return 0, "", false
	}
	label := strings.TrimSpace(line[level:])
	label = strings.TrimSpace(strings.TrimRight(label, "#"))
	if label == "" {
		return 0, "", false
	}
	return level, label, true
}

func uniqueSlug(label string, used map[string]int) string {
	base := slugify(label)
	used[base]++
	if used[base] == 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, used[base])
}

func slugify(value string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "section"
	}
	return out
}

func normalizeSpace(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func buildSupplementsRoot(workdir, absDir string) (*Node, []string, error) {
	children, warnings, err := buildSupplementNodes(workdir, absDir)
	if err != nil {
		return nil, warnings, err
	}
	relPath := filepath.ToSlash(mustRel(workdir, absDir))
	return &Node{
		ID:       relPath,
		Kind:     "directory",
		Label:    filepath.Base(absDir),
		Path:     relPath,
		Children: children,
	}, warnings, nil
}

func buildSupplementNodes(workdir, absDir string) ([]Node, []string, error) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	nodes := make([]Node, 0, len(entries))
	warnings := make([]string, 0)
	for _, entry := range entries {
		childPath := filepath.Join(absDir, entry.Name())
		relPath := filepath.ToSlash(mustRel(workdir, childPath))
		if entry.IsDir() {
			children, childWarnings, childErr := buildSupplementNodes(workdir, childPath)
			warnings = append(warnings, childWarnings...)
			if childErr != nil {
				return nil, warnings, childErr
			}
			nodes = append(nodes, Node{
				ID:       relPath,
				Kind:     "directory",
				Label:    entry.Name(),
				Path:     relPath,
				Children: children,
			})
			continue
		}

		preview, warning := buildFilePreview(childPath)
		if warning != "" {
			warnings = append(warnings, fmt.Sprintf("%s: %s", relPath, warning))
		}
		nodes = append(nodes, Node{
			ID:      relPath,
			Kind:    "file",
			Label:   entry.Name(),
			Path:    relPath,
			Preview: preview,
		})
	}

	return nodes, warnings, nil
}

func buildFilePreview(path string) (*Preview, string) {
	info, err := os.Stat(path)
	if err != nil {
		return &Preview{
			Status: "not_supported",
			Reason: "File preview is unavailable because the file could not be read.",
		}, err.Error()
	}

	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	preview := &Preview{
		Status:    "supported",
		ByteSize:  info.Size(),
		Extension: extension,
	}
	if info.Size() > maxPreviewBytes {
		preview.Status = "not_supported"
		preview.Reason = fmt.Sprintf("File is larger than the %d byte preview limit.", maxPreviewBytes)
		return preview, ""
	}
	if imageExtensions[extension] {
		preview.Status = "not_supported"
		preview.Reason = "Image preview is not supported in the Plan page yet."
		return preview, ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		preview.Status = "not_supported"
		preview.Reason = "File preview is unavailable because the file could not be read."
		return preview, err.Error()
	}

	if contentType, ok := richPreviewExtensions[extension]; ok {
		preview.ContentType = contentType
		preview.Content = string(data)
		return preview, ""
	}

	if looksLikeText(data) {
		preview.Status = "fallback"
		preview.ContentType = "text"
		preview.Content = string(data)
		preview.Reason = "Rendered as plain text because this extension has no richer preview yet."
		return preview, ""
	}

	preview.Status = "not_supported"
	preview.Reason = "Binary or unsupported file content cannot be previewed."
	return preview, ""
}

func looksLikeText(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	if !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			continue
		case r < 0x20:
			return false
		}
	}
	return true
}

func mustRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return rel
}
