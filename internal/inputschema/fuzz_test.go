package inputschema

import (
	"strings"
	"testing"

	"github.com/catu-ai/easyharness/internal/contracts"
)

func TestRenderInstanceLocationDecodesPointerTokensAndArrays(t *testing.T) {
	got := renderInstanceLocation("/findings/0/locations/1/file~1path/~0anchor")
	want := ".findings[0].locations[1].file/path.~anchor"
	if got != want {
		t.Fatalf("renderInstanceLocation mismatch: got %q want %q", got, want)
	}
}

func TestRenderIssueDetailsSplitsQuotedPropertiesWithoutDuplicates(t *testing.T) {
	issues := renderIssueDetails("input", "/findings/0", "missing properties 'severity', 'details', 'severity'")
	if len(issues) != 2 {
		t.Fatalf("expected two issues, got %#v", issues)
	}
	if issues[0].Path != "input.findings[0].severity" || issues[1].Path != "input.findings[0].details" {
		t.Fatalf("unexpected issue paths: %#v", issues)
	}
}

func TestPruneParentIssuesDropsOnlyStrictParents(t *testing.T) {
	issues := []contracts.ErrorDetail{
		{Path: "input.findings[0]", Message: "parent"},
		{Path: "input.findings[0].severity", Message: "child"},
		{Path: "input.summary", Message: "summary"},
		{Path: "input.summary_text", Message: "sibling"},
	}

	filtered := pruneParentIssues(issues)
	if len(filtered) != 3 {
		t.Fatalf("expected three filtered issues, got %#v", filtered)
	}
	if filtered[0].Path != "input.findings[0].severity" {
		t.Fatalf("expected child issue to survive, got %#v", filtered)
	}
	if filtered[1].Path != "input.summary" || filtered[2].Path != "input.summary_text" {
		t.Fatalf("expected non-parent prefixes to survive, got %#v", filtered)
	}
}

func FuzzValidateNormalizesIssuePaths(f *testing.F) {
	for _, seed := range []struct {
		schemaKey string
		rootLabel string
		payload   string
	}{
		{
			schemaKey: SchemaReviewSpec,
			rootLabel: "spec",
			payload:   `{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check correctness."}]}`,
		},
		{
			schemaKey: SchemaReviewSpec,
			rootLabel: "spec",
			payload:   `{"kind":"delta","dimensions":[{"name":"correctness","instructions":"Check correctness."}],"unexpected":true}`,
		},
		{
			schemaKey: SchemaReviewSubmission,
			rootLabel: "submission",
			payload:   `{"summary":"Missing fields.","findings":[{"title":"Missing metadata"}]}`,
		},
		{
			schemaKey: SchemaEvidenceCI,
			rootLabel: "input",
			payload:   `{}`,
		},
		{
			schemaKey: SchemaEvidencePublish,
			rootLabel: "input",
			payload:   `{"status":"recorded","pr_url":"https://example.invalid/pr/1","unexpected":true}`,
		},
		{
			schemaKey: SchemaEvidenceSync,
			rootLabel: "input",
			payload:   `{"status":"fresh","head_ref":true}`,
		},
		{
			schemaKey: SchemaEvidenceCI,
			rootLabel: "input",
			payload:   `{not-json`,
		},
	} {
		f.Add(seed.schemaKey, seed.rootLabel, seed.payload)
	}

	f.Fuzz(func(t *testing.T, schemaKey, rootLabel, payload string) {
		if len(payload) > 1<<14 {
			t.Skip()
		}
		if schemaKey != SchemaReviewSpec &&
			schemaKey != SchemaReviewSubmission &&
			schemaKey != SchemaEvidenceCI &&
			schemaKey != SchemaEvidencePublish &&
			schemaKey != SchemaEvidenceSync {
			t.Skip()
		}
		rootLabel = strings.TrimSpace(rootLabel)
		if rootLabel == "" {
			t.Skip()
		}

		issues := Validate(schemaKey, rootLabel, []byte(payload))
		assertNormalizedIssueSet(t, rootLabel, issues)
	})
}

func assertNormalizedIssueSet(t *testing.T, rootLabel string, issues []contracts.ErrorDetail) {
	t.Helper()

	for i, issue := range issues {
		if strings.TrimSpace(issue.Path) == "" {
			t.Fatalf("issue %d had empty path: %#v", i, issues)
		}
		if issue.Path != rootLabel && !strings.HasPrefix(issue.Path, rootLabel+".") && !strings.HasPrefix(issue.Path, rootLabel+"[") {
			t.Fatalf("issue path %q did not stay under root %q: %#v", issue.Path, rootLabel, issues)
		}
	}

	for i, issue := range issues {
		for j, other := range issues {
			if i == j {
				continue
			}
			if strings.HasPrefix(other.Path, issue.Path+".") || strings.HasPrefix(other.Path, issue.Path+"[") {
				t.Fatalf("issue set still contained parent-child pair %#v", issues)
			}
		}
	}
}
