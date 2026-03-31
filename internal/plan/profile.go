package plan

import (
	"path/filepath"
	"sort"
	"strings"
)

const (
	WorkflowProfileStandard    = "standard"
	WorkflowProfileLightweight = "lightweight"
)

func normalizeWorkflowProfile(value string) string {
	switch strings.TrimSpace(value) {
	case "", WorkflowProfileStandard:
		return WorkflowProfileStandard
	case WorkflowProfileLightweight:
		return WorkflowProfileLightweight
	default:
		return strings.TrimSpace(value)
	}
}

func inferWorkflowProfileFromPath(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(clean, "/docs/plans/archived/") || strings.HasPrefix(clean, "docs/plans/archived/"):
		return WorkflowProfileStandard
	case strings.Contains(clean, "/.local/harness/plans/archived/") || strings.HasPrefix(clean, ".local/harness/plans/archived/"):
		return WorkflowProfileLightweight
	default:
		return ""
	}
}

func inferPathKind(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	switch {
	case strings.Contains(clean, "/docs/plans/active/") || strings.HasPrefix(clean, "docs/plans/active/"):
		return "active"
	case strings.Contains(clean, "/docs/plans/archived/") || strings.HasPrefix(clean, "docs/plans/archived/"):
		return "archived"
	case strings.Contains(clean, "/.local/harness/plans/archived/") || strings.HasPrefix(clean, ".local/harness/plans/archived/"):
		return "archived"
	}
	return ""
}

func activeCandidatePaths(workdir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(workdir, "docs", "plans", "active", "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func currentLooksArchived(path string) bool {
	return inferPathKind(path) == "archived"
}

func ArchivedPathFor(workdir, planStem, currentPath string, profile string) string {
	switch normalizeWorkflowProfile(profile) {
	case WorkflowProfileLightweight:
		return filepath.Join(workdir, ".local", "harness", "plans", "archived", filepath.Base(currentPath))
	default:
		return filepath.Join(workdir, "docs", "plans", "archived", filepath.Base(currentPath))
	}
}

func ActivePathFor(workdir, planStem, currentPath string, profile string) string {
	return filepath.Join(workdir, "docs", "plans", "active", filepath.Base(currentPath))
}
