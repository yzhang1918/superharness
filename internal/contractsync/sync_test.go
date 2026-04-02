package contractsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpectedStatusSchemaAllowsNullableCurrentOutputs(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	files, err := expectedFiles(repoRoot)
	if err != nil {
		t.Fatalf("expectedFiles: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(files["schema/commands/status.result.schema.json"], &schema); err != nil {
		t.Fatalf("unmarshal status schema: %v", err)
	}

	defs := schema["$defs"].(map[string]any)
	nextAction := defs["NextAction"].(map[string]any)
	nextActionProps := nextAction["properties"].(map[string]any)
	commandSchema := nextActionProps["command"].(map[string]any)
	if !schemaAllowsNull(commandSchema) {
		t.Fatalf("expected NextAction.command to allow null, got %#v", commandSchema)
	}

	statusResult := defs["StatusResult"].(map[string]any)
	statusProps := statusResult["properties"].(map[string]any)
	nextActionsSchema := statusProps["next_actions"].(map[string]any)
	if !schemaAllowsNull(nextActionsSchema) {
		t.Fatalf("expected StatusResult.next_actions to allow null, got %#v", nextActionsSchema)
	}
}

func TestReviewInputSchemasMatchValidatorBoundaries(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	files, err := expectedFiles(repoRoot)
	if err != nil {
		t.Fatalf("expectedFiles: %v", err)
	}

	var reviewSpec map[string]any
	if err := json.Unmarshal(files["schema/inputs/review.spec.schema.json"], &reviewSpec); err != nil {
		t.Fatalf("unmarshal review spec schema: %v", err)
	}
	specDefs := reviewSpec["$defs"].(map[string]any)
	specRoot := specDefs["ReviewSpec"].(map[string]any)
	specProps := specRoot["properties"].(map[string]any)
	dimensionsSchema := specProps["dimensions"].(map[string]any)
	if schemaAllowsNull(dimensionsSchema) {
		t.Fatalf("expected review spec dimensions to reject null, got %#v", dimensionsSchema)
	}
	if got := dimensionsSchema["minItems"]; got != float64(1) {
		t.Fatalf("expected review spec dimensions minItems=1, got %#v", got)
	}

	var submission map[string]any
	if err := json.Unmarshal(files["schema/inputs/review.submission.schema.json"], &submission); err != nil {
		t.Fatalf("unmarshal review submission schema: %v", err)
	}
	submissionDefs := submission["$defs"].(map[string]any)
	submissionRoot := submissionDefs["ReviewSubmissionInput"].(map[string]any)
	required := stringSet(submissionRoot["required"])
	if required["findings"] {
		t.Fatalf("expected review submission findings to be optional, got %#v", submissionRoot["required"])
	}
	findingsSchema := submissionRoot["properties"].(map[string]any)["findings"].(map[string]any)
	if !schemaAllowsNull(findingsSchema) {
		t.Fatalf("expected review submission findings to allow null, got %#v", findingsSchema)
	}
	reviewFinding := submissionDefs["ReviewFinding"].(map[string]any)
	assertLocationsSchema(t, reviewFinding)

	var artifactSubmission map[string]any
	if err := json.Unmarshal(files["schema/artifacts/review-submission.schema.json"], &artifactSubmission); err != nil {
		t.Fatalf("unmarshal review submission artifact schema: %v", err)
	}
	artifactSubmissionFinding := artifactSubmission["$defs"].(map[string]any)["ReviewFinding"].(map[string]any)
	assertLocationsSchema(t, artifactSubmissionFinding)

	var aggregate map[string]any
	if err := json.Unmarshal(files["schema/artifacts/review-aggregate.schema.json"], &aggregate); err != nil {
		t.Fatalf("unmarshal review aggregate schema: %v", err)
	}
	aggregateFinding := aggregate["$defs"].(map[string]any)["ReviewAggregateFinding"].(map[string]any)
	assertLocationsSchema(t, aggregateFinding)
}

func TestSchemaIndexNoLongerPointsAtGeneratedMarkdown(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	files, err := expectedFiles(repoRoot)
	if err != nil {
		t.Fatalf("expectedFiles: %v", err)
	}

	indexJSON := string(files["schema/index.json"])
	if strings.Contains(indexJSON, "\"doc_path\"") {
		t.Fatalf("expected schema index to omit doc_path references, got %s", indexJSON)
	}
	if _, ok := files["docs/reference/contracts/README.md"]; ok {
		t.Fatal("expected contract sync to stop generating docs/reference/contracts/README.md")
	}

	var schemaIndex struct {
		Schemas []struct {
			Key     string `json:"key"`
			Group   string `json:"group"`
			Surface string `json:"surface"`
		} `json:"schemas"`
	}
	if err := json.Unmarshal(files["schema/index.json"], &schemaIndex); err != nil {
		t.Fatalf("unmarshal schema index: %v", err)
	}

	surfaces := map[string]string{}
	for _, entry := range schemaIndex.Schemas {
		surfaces[entry.Key] = entry.Surface
	}
	if surfaces["commands.status.result"] != "public" {
		t.Fatalf("expected commands.status.result surface=public, got %q", surfaces["commands.status.result"])
	}
	if surfaces["artifacts.current_plan"] != "cli_owned_runtime" {
		t.Fatalf("expected artifacts.current_plan surface=cli_owned_runtime, got %q", surfaces["artifacts.current_plan"])
	}
}

func TestCheckFilesFailsOnMissingAndUnexpectedGeneratedFiles(t *testing.T) {
	workdir := t.TempDir()
	ownedRoots := []string{
		filepath.Join(workdir, "schema"),
		filepath.Join(workdir, "docs", "reference", "contracts"),
	}

	if err := os.MkdirAll(filepath.Join(workdir, "schema"), 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "schema", "unexpected.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write unexpected file: %v", err)
	}

	err := checkFiles(workdir, ownedRoots, map[string][]byte{
		"schema/index.json": []byte("{\"ok\":true}\n"),
	})
	if err == nil {
		t.Fatal("expected checkFiles to fail")
	}
	message := err.Error()
	if !strings.Contains(message, "missing generated file: schema/index.json") {
		t.Fatalf("expected missing-file error, got %q", message)
	}
	if !strings.Contains(message, "unexpected generated file: schema/unexpected.json") {
		t.Fatalf("expected unexpected-file error, got %q", message)
	}
}

func TestWriteFilesReplacesOwnedRoots(t *testing.T) {
	workdir := t.TempDir()
	ownedRoots := []string{
		filepath.Join(workdir, "schema"),
		filepath.Join(workdir, "docs", "reference", "contracts"),
	}
	if err := os.MkdirAll(filepath.Join(workdir, "schema"), 0o755); err != nil {
		t.Fatalf("mkdir schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "schema", "stale.json"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	expected := map[string][]byte{
		"schema/index.json": []byte("{\"title\":\"ok\"}\n"),
	}
	if err := writeFiles(workdir, ownedRoots, expected); err != nil {
		t.Fatalf("writeFiles: %v", err)
	}

	if _, err := os.Stat(filepath.Join(workdir, "schema", "stale.json")); !os.IsNotExist(err) {
		t.Fatalf("expected stale generated file to be removed, got err=%v", err)
	}
	if data, err := os.ReadFile(filepath.Join(workdir, "schema", "index.json")); err != nil || string(data) != "{\"title\":\"ok\"}\n" {
		t.Fatalf("unexpected schema/index.json contents: err=%v data=%q", err, data)
	}
	if _, err := os.Stat(filepath.Join(workdir, "docs", "reference", "contracts")); !os.IsNotExist(err) {
		t.Fatalf("expected deprecated generated docs root to be removed, got err=%v", err)
	}
}

func schemaAllowsNull(schema map[string]any) bool {
	oneOf, ok := schema["oneOf"].([]any)
	if !ok {
		return false
	}
	for _, branch := range oneOf {
		mapped, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		if mapped["type"] == "null" {
			return true
		}
	}
	return false
}

func assertLocationsSchema(t *testing.T, findingSchema map[string]any) {
	t.Helper()
	locationsSchema := findingSchema["properties"].(map[string]any)["locations"].(map[string]any)
	if schemaAllowsNull(locationsSchema) {
		t.Fatalf("expected review finding locations to be optional rather than nullable, got %#v", locationsSchema)
	}
	if locationsSchema["type"] != "array" {
		t.Fatalf("expected review finding locations to be an array, got %#v", locationsSchema)
	}
	locationItem := locationsSchema["items"].(map[string]any)
	if locationItem["type"] != "string" {
		t.Fatalf("expected review finding location items to be strings, got %#v", locationItem)
	}
}
