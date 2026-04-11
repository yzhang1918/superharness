package support

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type Workspace struct {
	Root string
}

func NewWorkspace(t *testing.T) *Workspace {
	t.Helper()
	return &Workspace{Root: t.TempDir()}
}

func (w *Workspace) Path(rel string) string {
	if rel == "" {
		return w.Root
	}
	return filepath.Join(w.Root, filepath.FromSlash(rel))
}

func (w *Workspace) WriteJSON(t *testing.T, rel string, value any) string {
	t.Helper()

	path := w.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", rel, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func (w *Workspace) WriteFile(t *testing.T, rel string, data []byte) string {
	t.Helper()

	path := w.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	return path
}
