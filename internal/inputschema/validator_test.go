package inputschema_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/catu-ai/easyharness/internal/inputschema"
)

func TestValidateAcceptsValidReviewSpec(t *testing.T) {
	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}]
	}`))
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %#v", issues)
	}
}

func TestValidateReportsUnknownFieldPath(t *testing.T) {
	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}],
		"unexpected":true
	}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "spec.unexpected" {
		t.Fatalf("expected unknown-field path, got %#v", issues)
	}
}

func TestValidateReportsNestedTypePath(t *testing.T) {
	issues := inputschema.Validate("inputs.review.submission", "submission", []byte(`{
		"summary":"Found one issue.",
		"findings":[{"severity":1,"title":"Wrong type","details":"Severity must be a string."}]
	}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "submission.findings[0].severity" {
		t.Fatalf("expected nested type path, got %#v", issues)
	}
}

func TestValidateReportsMissingRequiredFieldPath(t *testing.T) {
	issues := inputschema.Validate("inputs.evidence.ci", "input", []byte(`{}`))
	if len(issues) != 1 {
		t.Fatalf("expected one validation issue, got %#v", issues)
	}
	if issues[0].Path != "input.status" {
		t.Fatalf("expected missing required field path, got %#v", issues)
	}
}

func TestValidateDoesNotDependOnRepositoryWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	temp := t.TempDir()
	if err := os.Chdir(temp); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	issues := inputschema.Validate("inputs.review.spec", "spec", []byte(`{
		"kind":"delta",
		"dimensions":[{"name":"correctness","instructions":"Check correctness."}]
	}`))
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues away from repo root, got %#v (cwd=%s)", issues, filepath.Clean(temp))
	}
}

func TestValidateSplitsMultipleMissingRequiredFields(t *testing.T) {
	issues := inputschema.Validate("inputs.review.submission", "submission", []byte(`{
		"summary":"Missing fields.",
		"findings":[{"title":"Missing metadata"}]
	}`))
	if len(issues) != 2 {
		t.Fatalf("expected two validation issues, got %#v", issues)
	}
	if issues[0].Path != "submission.findings[0].severity" && issues[1].Path != "submission.findings[0].severity" {
		t.Fatalf("expected a severity issue path, got %#v", issues)
	}
	if issues[0].Path != "submission.findings[0].details" && issues[1].Path != "submission.findings[0].details" {
		t.Fatalf("expected a details issue path, got %#v", issues)
	}
}
