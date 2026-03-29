package plan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/runstate"
)

func TestDetectCurrentPathPrefersSingleActivePlanOverArchivedPointer(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs", "plans", "active", "2026-03-18-new-work.md")
	archivedPath := filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md")
	writeTestFile(t, activePath)
	writeTestFile(t, archivedPath)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "archived", "2026-03-17-old-work.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected active plan %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathUsesCurrentPointerToDisambiguateMultipleActivePlans(t *testing.T) {
	root := t.TempDir()
	activePathA := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	activePathB := filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md")
	writeTestFile(t, activePathA)
	writeTestFile(t, activePathB)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-03-18-second.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	got, err := DetectCurrentPath(root)
	if err != nil {
		t.Fatalf("detect current path: %v", err)
	}
	if got != activePathB {
		t.Fatalf("expected current pointer %s, got %s", activePathB, got)
	}
}

func TestDetectCurrentPathErrorsWhenArchivedPointerCannotDisambiguateMultipleActivePlans(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md"))
	writeTestFile(t, filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md"))
	writeTestFile(t, filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md"))

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "archived", "2026-03-17-old-work.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	if _, err := DetectCurrentPath(root); err == nil {
		t.Fatal("expected error when archived pointer cannot disambiguate multiple active plans")
	}
}

func TestDetectCurrentPathDoesNotFallBackToArchivedPlanWithoutPointer(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "docs", "plans", "archived", "2026-03-17-old-work.md"))

	_, err := DetectCurrentPath(root)
	if !errors.Is(err, ErrNoCurrentPlan) {
		t.Fatalf("expected ErrNoCurrentPlan, got %v", err)
	}
}

func TestDetectCurrentPathLockedAllowsSameStem(t *testing.T) {
	root := t.TempDir()
	activePath := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	writeTestFile(t, activePath)

	got, err := DetectCurrentPathLocked(root, "2026-03-18-first")
	if err != nil {
		t.Fatalf("DetectCurrentPathLocked: %v", err)
	}
	if got != activePath {
		t.Fatalf("expected %s, got %s", activePath, got)
	}
}

func TestDetectCurrentPathLockedRejectsStemChange(t *testing.T) {
	root := t.TempDir()
	activePathA := filepath.Join(root, "docs", "plans", "active", "2026-03-18-first.md")
	activePathB := filepath.Join(root, "docs", "plans", "active", "2026-03-18-second.md")
	writeTestFile(t, activePathA)
	writeTestFile(t, activePathB)

	if _, err := runstate.SaveCurrentPlan(root, filepath.ToSlash(filepath.Join("docs", "plans", "active", "2026-03-18-second.md"))); err != nil {
		t.Fatalf("save current plan: %v", err)
	}

	if _, err := DetectCurrentPathLocked(root, "2026-03-18-first"); err == nil {
		t.Fatal("expected DetectCurrentPathLocked to reject stem change")
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
