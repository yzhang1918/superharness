package plan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yzhang1918/superharness/internal/runstate"
)

func DetectCurrentPath(workdir string) (string, error) {
	activeMatches, err := filepath.Glob(filepath.Join(workdir, "docs", "plans", "active", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(activeMatches)

	if current, err := runstate.LoadCurrentPlan(workdir); err != nil {
		return "", err
	} else if current != nil && strings.TrimSpace(current.PlanPath) != "" {
		currentPath := filepath.Join(workdir, current.PlanPath)
		currentPath = filepath.Clean(currentPath)

		if containsPath(activeMatches, currentPath) {
			return currentPath, nil
		}

		if currentLooksArchived(currentPath) {
			if len(activeMatches) == 1 {
				return activeMatches[0], nil
			}
			if len(activeMatches) > 1 {
				return "", fmt.Errorf("multiple active plans found; current-plan.json points to archived work and cannot disambiguate")
			}
		}

		if _, err := os.Stat(currentPath); err == nil {
			return currentPath, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}

	if len(activeMatches) == 1 {
		return activeMatches[0], nil
	}
	if len(activeMatches) > 1 {
		return "", fmt.Errorf("multiple active plans found; add .local/harness/current-plan.json to disambiguate")
	}

	archivedMatches, err := filepath.Glob(filepath.Join(workdir, "docs", "plans", "archived", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(archivedMatches)
	if len(archivedMatches) == 1 {
		return archivedMatches[0], nil
	}

	return "", fmt.Errorf("no current plan found")
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == target {
			return true
		}
	}
	return false
}

func currentLooksArchived(path string) bool {
	return strings.Contains(path, filepath.Join("docs", "plans", "archived")+string(os.PathSeparator))
}
