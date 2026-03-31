package plan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/catu-ai/easyharness/internal/runstate"
)

var ErrNoCurrentPlan = errors.New("no current plan found")

func DetectCurrentPath(workdir string) (string, error) {
	activeMatches, err := activeCandidatePaths(workdir)
	if err != nil {
		return "", err
	}
	if len(activeMatches) > 1 {
		return "", fmt.Errorf("multiple active plans found; state resolution must fail rather than guess")
	}

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

	return "", ErrNoCurrentPlan
}

func DetectCurrentPathLocked(workdir, lockedPlanStem string) (string, error) {
	currentPath, err := DetectCurrentPath(workdir)
	if err != nil {
		return "", err
	}
	currentStem := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))
	if currentStem != strings.TrimSpace(lockedPlanStem) {
		return "", fmt.Errorf("current plan changed from %q to %q while acquiring the local state lock; retry", lockedPlanStem, currentStem)
	}
	return currentPath, nil
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if filepath.Clean(path) == target {
			return true
		}
	}
	return false
}
