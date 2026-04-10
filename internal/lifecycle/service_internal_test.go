package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveSupplementsDirIfPresentRejectsExistingTarget(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "docs/plans/active/supplements/example")
	target := filepath.Join(root, "docs/plans/archived/supplements/example")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "spec.md"), []byte("# source\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "stale.md"), []byte("# stale\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	moved, err := moveSupplementsDirIfPresent(source, target)
	if err == nil {
		t.Fatal("expected move to fail when target already exists")
	}
	if moved {
		t.Fatal("expected moved=false when target already exists")
	}
	if _, err := os.Stat(filepath.Join(source, "spec.md")); err != nil {
		t.Fatalf("expected source supplements to remain, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "stale.md")); err != nil {
		t.Fatalf("expected target supplements to remain, got %v", err)
	}
}
