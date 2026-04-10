package plan

import (
	"path/filepath"
	"sort"
	"strings"
)

const (
	WorkflowProfileStandard    = "standard"
	WorkflowProfileLightweight = "lightweight"
	SupplementsDirName         = "supplements"
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

func SupplementsDirForPlanPath(path string) string {
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	stem := strings.TrimSuffix(filepath.Base(clean), filepath.Ext(clean))
	return filepath.Join(dir, SupplementsDirName, stem)
}

func AlternateSupplementsDirsForPlanPath(path string) []string {
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	clean := filepath.ToSlash(filepath.Clean(path))
	prefix := ""
	for _, marker := range []string{"/docs/plans/active/", "/docs/plans/archived/", "/.local/harness/plans/archived/"} {
		if idx := strings.Index(clean, marker); idx >= 0 {
			prefix = clean[:idx]
			break
		}
	}
	base := filepath.FromSlash(prefix)
	candidates := []string{
		filepath.Join(base, "docs", "plans", "active", SupplementsDirName, stem),
		filepath.Join(base, "docs", "plans", "archived", SupplementsDirName, stem),
		filepath.Join(base, ".local", "harness", "plans", "archived", SupplementsDirName, stem),
	}

	expected := filepath.Clean(SupplementsDirForPlanPath(path))
	filtered := make([]string, 0, len(candidates)-1)
	for _, candidate := range candidates {
		if filepath.Clean(candidate) == expected {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func relativePathWithinPlanRoot(path string) string {
	clean := filepath.ToSlash(filepath.Clean(path))
	for _, marker := range []string{"/docs/plans/active/", "/docs/plans/archived/", "/.local/harness/plans/archived/"} {
		if idx := strings.Index(clean, marker); idx >= 0 {
			return strings.TrimPrefix(clean[idx+len(marker):], "/")
		}
	}
	for _, marker := range []string{"docs/plans/active/", "docs/plans/archived/", ".local/harness/plans/archived/"} {
		if strings.HasPrefix(clean, marker) {
			return strings.TrimPrefix(clean, marker)
		}
	}
	return ""
}

func ArchivedSupplementsDirFor(workdir, planStem, currentPath string, profile string) string {
	return SupplementsDirForPlanPath(ArchivedPathFor(workdir, planStem, currentPath, profile))
}

func ActiveSupplementsDirFor(workdir, planStem, currentPath string, profile string) string {
	return SupplementsDirForPlanPath(ActivePathFor(workdir, planStem, currentPath, profile))
}
