package smoke_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/tests/support"
)

func TestReleaseDocsPresentStableOnboardingSurface(t *testing.T) {
	repoRoot := support.RepoRoot(t)

	readmeData, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	readme := string(readmeData)
	normalizedReadme := strings.Join(strings.Fields(readme), " ")
	support.RequireContains(t, normalizedReadme, "Harnesses matter. Building one shouldn't be the project.")
	support.RequireContains(t, normalizedReadme, "brew install easyharness")
	support.RequireContains(t, normalizedReadme, "harness init")
	support.RequireContains(t, normalizedReadme, "breaking changes may happen between releases")
	support.RequireContains(t, readme, "./docs/development.md")
	if strings.Contains(strings.ToLower(readme), "public alpha") {
		t.Fatalf("expected README to avoid public alpha wording, got:\n%s", readme)
	}

	developmentData, err := os.ReadFile(filepath.Join(repoRoot, "docs", "development.md"))
	if err != nil {
		t.Fatalf("read development doc: %v", err)
	}
	development := string(developmentData)
	normalizedDevelopment := strings.Join(strings.Fields(development), " ")
	support.RequireContains(t, normalizedDevelopment, "stable `harness` installation to already be available on `PATH`")
	support.RequireContains(t, normalizedDevelopment, "Homebrew install shown in the root")
	support.RequireContains(t, development, "bundled Playwright wrapper")
	if strings.Contains(development, "--global") {
		t.Fatalf("expected development doc to avoid retired --global guidance, got:\n%s", development)
	}
	if strings.Contains(development, "/Users/yaozhang/") {
		t.Fatalf("expected development doc to avoid workstation-specific absolute paths, got:\n%s", development)
	}

	releasingData, err := os.ReadFile(filepath.Join(repoRoot, "docs", "releasing.md"))
	if err != nil {
		t.Fatalf("read releasing doc: %v", err)
	}
	releasing := string(releasingData)
	normalizedReleasing := strings.Join(strings.Fields(releasing), " ")
	support.RequireContains(t, normalizedReleasing, "`0.2.0`")
	support.RequireContains(t, normalizedReleasing, "`v0.2.0`")
	support.RequireContains(t, normalizedReleasing, "including prerelease tags")
	support.RequireContains(t, normalizedReleasing, "default Homebrew formula `easyharness`")
	if strings.Contains(strings.ToLower(releasing), "first public alpha") {
		t.Fatalf("expected releasing doc to avoid first-public-alpha wording, got:\n%s", releasing)
	}
}
