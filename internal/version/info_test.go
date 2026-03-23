package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestCurrentUsesBuildInfoCommitInReleaseMode(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
	})

	info := current(
		func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abc123"},
				},
			}, true
		},
		func() (string, error) {
			t.Fatal("release mode should not need executable path")
			return "", nil
		},
	)

	if info.Commit != "abc123" {
		t.Fatalf("expected build-info commit, got %#v", info)
	}
	if info.Mode != "release" {
		t.Fatalf("expected release mode by default, got %#v", info)
	}
	if info.Path != "" {
		t.Fatalf("expected release mode to omit path, got %#v", info)
	}
}

func TestCurrentUsesExplicitDevMetadata(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
	})

	BuildCommit = "deadbeef"
	BuildMode = "dev"

	info := current(
		func() (*debug.BuildInfo, bool) {
			return nil, false
		},
		func() (string, error) {
			return "/tmp/dev-harness", nil
		},
	)

	if info.Commit != "deadbeef" {
		t.Fatalf("expected explicit build commit, got %#v", info)
	}
	if info.Mode != "dev" {
		t.Fatalf("expected dev mode, got %#v", info)
	}
	if info.Path != "/tmp/dev-harness" {
		t.Fatalf("expected dev path, got %#v", info)
	}
	if !strings.Contains(info.String(), "path: /tmp/dev-harness") {
		t.Fatalf("expected formatted version output to include dev path, got %q", info.String())
	}
}

func TestCurrentFallsBackToUnknownCommit(t *testing.T) {
	t.Cleanup(func() {
		BuildCommit = ""
		BuildMode = ""
	})

	info := current(
		func() (*debug.BuildInfo, bool) {
			return nil, false
		},
		func() (string, error) {
			return "", nil
		},
	)

	if info.Commit != "unknown" {
		t.Fatalf("expected unknown commit fallback, got %#v", info)
	}
}
