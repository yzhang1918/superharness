package runstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveCurrentPlanWritesExactJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current plan dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"plan_path":"old","last_landed_plan_path":"stale-trailing-bytes-should-disappear"}`), 0o644); err != nil {
		t.Fatalf("seed current-plan.json: %v", err)
	}

	savedPath, err := SaveCurrentPlan(root, "docs/plans/active/example.md")
	if err != nil {
		t.Fatalf("SaveCurrentPlan: %v", err)
	}
	if savedPath != path {
		t.Fatalf("saved path = %q, want %q", savedPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current-plan.json: %v", err)
	}
	want, err := json.MarshalIndent(CurrentPlan{PlanPath: "docs/plans/active/example.md"}, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected current plan: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("current-plan.json mismatch:\n got: %s\nwant: %s", data, want)
	}
}

func TestSaveStateWritesExactJSON(t *testing.T) {
	root := t.TempDir()
	planStem := "2026-03-26-atomic-save"
	path := filepath.Join(root, ".local", "harness", "plans", planStem, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"current_node":"execution/step-999/review","plan_path":"stale","plan_stem":"old","revision":999}`), 0o644); err != nil {
		t.Fatalf("seed state.json: %v", err)
	}

	state := &State{
		CurrentNode: "execution/step-1/implement",
		PlanPath:    "docs/plans/active/example.md",
		PlanStem:    planStem,
		Revision:    1,
	}
	savedPath, err := SaveState(root, planStem, state)
	if err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if savedPath != path {
		t.Fatalf("saved path = %q, want %q", savedPath, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state.json: %v", err)
	}
	want, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal expected state: %v", err)
	}
	if string(data) != string(want) {
		t.Fatalf("state.json mismatch:\n got: %s\nwant: %s", data, want)
	}

	loaded, _, err := LoadState(root, planStem)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if loaded == nil {
		t.Fatalf("LoadState returned nil state")
	}
	if loaded.CurrentNode != state.CurrentNode || loaded.PlanPath != state.PlanPath || loaded.PlanStem != state.PlanStem || loaded.Revision != state.Revision {
		t.Fatalf("loaded state = %#v, want %#v", loaded, state)
	}
}

func TestWriteJSONAtomicPreservesOriginalFileWhenRenameFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir current plan dir: %v", err)
	}
	original := []byte(`{"plan_path":"docs/plans/active/original.md"}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("seed current-plan.json: %v", err)
	}

	restoreRename := renameFile
	renameFile = func(_, _ string) error {
		return errors.New("rename failed")
	}
	defer func() {
		renameFile = restoreRename
	}()

	if err := writeJSONAtomic(path, []byte(`{"plan_path":"docs/plans/active/new.md"}`), 0o644); err == nil {
		t.Fatal("expected writeJSONAtomic to fail when rename fails")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read current-plan.json: %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("expected original file to remain intact, got %s", data)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read current plan dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "current-plan.json" {
		t.Fatalf("expected failed atomic write to clean up temp files, got %#v", entries)
	}
}
