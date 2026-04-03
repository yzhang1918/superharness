package reviewui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

type Service struct {
	Workdir string
}

type Result = contracts.ReviewResult
type Artifacts = contracts.ReviewArtifacts
type Round = contracts.ReviewRoundView
type Reviewer = contracts.ReviewSlotView
type Artifact = contracts.ReviewArtifactView
type ErrorDetail = contracts.ErrorDetail
type Manifest = contracts.ReviewManifest
type ManifestSlot = contracts.ReviewManifestSlot
type Ledger = contracts.ReviewLedger
type LedgerSlot = contracts.ReviewLedgerSlot
type Submission = contracts.ReviewSubmission
type Aggregate = contracts.ReviewAggregate
type ReviewFinding = contracts.ReviewFinding
type AggregateFinding = contracts.ReviewAggregateFinding

func (s Service) Read() Result {
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			return Result{
				OK:       true,
				Resource: "review",
				Summary:  "No current plan review data is available in this worktree.",
				Rounds:   []Round{},
			}
		}
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to determine the current plan for review loading.",
			Errors:   []ErrorDetail{{Path: "plan", Message: err.Error()}},
			Rounds:   []Round{},
		}
	}

	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to determine the current plan path for review loading.",
			Errors:   []ErrorDetail{{Path: "plan", Message: err.Error()}},
			Rounds:   []Round{},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	if !isSupportedReviewPlanPath(relPlanPath) {
		return Result{
			OK:       true,
			Resource: "review",
			Summary:  "Review data is only shown for the current tracked plan.",
			Artifacts: &Artifacts{
				PlanPath: relPlanPath,
			},
			Rounds: []Round{},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(relPlanPath), filepath.Ext(relPlanPath))
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	warnings := make([]string, 0, 2)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read local review state for %s; active-round hints may be incomplete.", planStem))
	}
	if isArchivedReviewPlanPath(relPlanPath) && archivedReviewHiddenDuringLand(state) {
		return Result{
			OK:       true,
			Resource: "review",
			Summary:  "Review data is hidden once land cleanup begins.",
			Artifacts: &Artifacts{
				PlanPath:       relPlanPath,
				LocalStatePath: statePath,
			},
			Rounds:   []Round{},
			Warnings: warnings,
		}
	}

	reviewsDir := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, "reviews")
	roundIDs, discoverWarnings, discoverErr := discoverRoundIDs(reviewsDir, state)
	warnings = append(warnings, discoverWarnings...)
	if discoverErr != nil {
		return Result{
			OK:       false,
			Resource: "review",
			Summary:  "Unable to enumerate review rounds for the current plan.",
			Artifacts: &Artifacts{
				PlanPath:       relPlanPath,
				LocalStatePath: statePath,
				ReviewsDir:     filepath.ToSlash(reviewsDir),
			},
			Errors: []ErrorDetail{{Path: "reviews", Message: discoverErr.Error()}},
			Rounds: []Round{},
		}
	}

	rounds := make([]Round, 0, len(roundIDs))
	activeRoundID := ""
	if state != nil && state.ActiveReviewRound != nil {
		activeRoundID = strings.TrimSpace(state.ActiveReviewRound.RoundID)
	}
	for _, roundID := range roundIDs {
		rounds = append(rounds, s.readRound(planStem, roundID, activeRoundID))
	}
	sortRounds(rounds)

	summary := "No review rounds recorded yet for the current plan."
	if len(rounds) > 0 {
		summary = fmt.Sprintf("Loaded %d review round(s) for %s.", len(rounds), filepath.Base(relPlanPath))
	}

	return Result{
		OK:       true,
		Resource: "review",
		Summary:  summary,
		Artifacts: &Artifacts{
			PlanPath:       relPlanPath,
			LocalStatePath: statePath,
			ReviewsDir:     filepath.ToSlash(reviewsDir),
			ActiveRoundID:  activeRoundID,
		},
		Rounds:   rounds,
		Warnings: warnings,
	}
}

func archivedReviewHiddenDuringLand(state *runstate.State) bool {
	if state == nil {
		return false
	}
	if strings.TrimSpace(state.CurrentNode) == "land" {
		return true
	}
	return state.Land != nil &&
		strings.TrimSpace(state.Land.LandedAt) != "" &&
		strings.TrimSpace(state.Land.CompletedAt) == ""
}

func isSupportedReviewPlanPath(relPlanPath string) bool {
	return strings.HasPrefix(relPlanPath, "docs/plans/active/") ||
		strings.HasPrefix(relPlanPath, "docs/plans/archived/") ||
		strings.HasPrefix(relPlanPath, ".local/harness/plans/archived/")
}

func isArchivedReviewPlanPath(relPlanPath string) bool {
	return strings.HasPrefix(relPlanPath, "docs/plans/archived/") ||
		strings.HasPrefix(relPlanPath, ".local/harness/plans/archived/")
}

func discoverRoundIDs(reviewsDir string, state *runstate.State) ([]string, []string, error) {
	roundSet := map[string]bool{}
	warnings := []string{}

	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, warnings, err
		}
		if state != nil && state.ActiveReviewRound != nil && strings.TrimSpace(state.ActiveReviewRound.RoundID) != "" {
			roundID := strings.TrimSpace(state.ActiveReviewRound.RoundID)
			roundSet[roundID] = true
			warnings = append(warnings, fmt.Sprintf("Active review round %s is tracked in local state, but the review directory is missing.", roundID))
		}
		return sortedRoundIDs(roundSet), warnings, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		roundSet[name] = true
	}

	if state != nil && state.ActiveReviewRound != nil && strings.TrimSpace(state.ActiveReviewRound.RoundID) != "" {
		roundID := strings.TrimSpace(state.ActiveReviewRound.RoundID)
		if !roundSet[roundID] {
			warnings = append(warnings, fmt.Sprintf("Active review round %s is tracked in local state, but no matching round directory was found.", roundID))
		}
		roundSet[roundID] = true
	}

	return sortedRoundIDs(roundSet), warnings, nil
}

func sortedRoundIDs(values map[string]bool) []string {
	roundIDs := make([]string, 0, len(values))
	for roundID := range values {
		roundIDs = append(roundIDs, roundID)
	}
	sort.Slice(roundIDs, func(i, j int) bool {
		return compareRoundIDs(roundIDs[i], roundIDs[j]) < 0
	})
	return roundIDs
}

func compareRoundIDs(left, right string) int {
	leftSeq, leftOK := parseRoundSequence(left)
	rightSeq, rightOK := parseRoundSequence(right)
	switch {
	case leftOK && rightOK && leftSeq != rightSeq:
		if leftSeq > rightSeq {
			return -1
		}
		return 1
	case leftOK && !rightOK:
		return -1
	case !leftOK && rightOK:
		return 1
	case left == right:
		return 0
	case left > right:
		return -1
	default:
		return 1
	}
}

func parseRoundSequence(roundID string) (int, bool) {
	trimmed := strings.TrimSpace(roundID)
	if !strings.HasPrefix(trimmed, "review-") {
		return 0, false
	}
	remainder := strings.TrimPrefix(trimmed, "review-")
	sequence, _, found := strings.Cut(remainder, "-")
	if !found || sequence == "" {
		return 0, false
	}
	value, err := strconv.Atoi(sequence)
	if err != nil {
		return 0, false
	}
	return value, true
}

func sortRounds(rounds []Round) {
	sort.Slice(rounds, func(i, j int) bool {
		if rounds[i].IsActive != rounds[j].IsActive {
			return rounds[i].IsActive
		}
		return compareRoundIDs(rounds[i].RoundID, rounds[j].RoundID) < 0
	})
}

func (s Service) readRound(planStem, roundID, activeRoundID string) Round {
	roundDir := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, "reviews", roundID)
	manifestPath := filepath.Join(roundDir, "manifest.json")
	ledgerPath := filepath.Join(roundDir, "ledger.json")
	aggregatePath := filepath.Join(roundDir, "aggregate.json")

	manifestArtifact, manifest, manifestWarning := readJSONArtifact[Manifest]("Manifest", manifestPath, validateManifestArtifact)
	ledgerArtifact, ledger, ledgerWarning := readJSONArtifact[Ledger]("Ledger", ledgerPath, validateLedgerArtifact)
	aggregateArtifact, aggregate, aggregateWarning := readJSONArtifact[Aggregate]("Aggregate", aggregatePath, validateAggregateArtifact)

	artifacts := []Artifact{manifestArtifact, ledgerArtifact, aggregateArtifact}
	warnings := make([]string, 0, 8)
	appendWarning := func(message string) {
		message = strings.TrimSpace(message)
		if message == "" {
			return
		}
		warnings = append(warnings, message)
	}

	appendWarning(manifestWarning)
	appendWarning(ledgerWarning)
	if aggregateWarning != "" && aggregateArtifact.Status != "missing" {
		appendWarning(aggregateWarning)
	}

	round := Round{
		RoundID:   roundID,
		Status:    "unknown",
		IsActive:  roundID == strings.TrimSpace(activeRoundID),
		Artifacts: artifacts,
	}

	if manifest != nil {
		round.Kind = manifest.Kind
		round.Step = manifest.Step
		round.Revision = manifest.Revision
		round.ReviewTitle = manifest.ReviewTitle
		round.CreatedAt = manifest.CreatedAt
	}
	if ledger != nil {
		if round.Kind == "" {
			round.Kind = ledger.Kind
		}
		round.UpdatedAt = ledger.UpdatedAt
	}
	if aggregate != nil {
		if round.Kind == "" {
			round.Kind = aggregate.Kind
		}
		if round.Step == nil {
			round.Step = aggregate.Step
		}
		if round.Revision == 0 {
			round.Revision = aggregate.Revision
		}
		if round.ReviewTitle == "" {
			round.ReviewTitle = aggregate.ReviewTitle
		}
		round.AggregatedAt = aggregate.AggregatedAt
		round.Decision = aggregate.Decision
		round.BlockingFindings = aggregate.BlockingFindings
		round.NonBlockingFindings = aggregate.NonBlockingFindings
	}

	reviewers, submissionArtifacts, reviewerWarnings := s.readReviewers(roundDir, manifest, ledger)
	round.Reviewers = reviewers
	round.Artifacts = append(round.Artifacts, submissionArtifacts...)
	warnings = append(warnings, reviewerWarnings...)

	round.TotalSlots = len(reviewers)
	for _, reviewer := range reviewers {
		switch normalizeSlotStatus(reviewer.Status) {
		case "submitted":
			round.SubmittedSlots++
		default:
			round.PendingSlots++
		}
	}

	status, summary := resolveRoundStatus(round, manifestArtifact, ledgerArtifact, aggregateArtifact)
	round.Status = status
	round.StatusSummary = summary
	if manifestArtifact.Status == "invalid" || ledgerArtifact.Status == "invalid" || aggregateArtifact.Status == "invalid" {
		appendWarning("One or more review artifacts are malformed; the round is shown conservatively.")
	}
	if manifestArtifact.Status == "missing" {
		appendWarning("Manifest is missing, so reviewer instructions and round metadata may be incomplete.")
	}
	if ledgerArtifact.Status == "missing" && len(reviewers) > 0 {
		appendWarning("Ledger is missing, so submission progress is inferred conservatively from submission artifacts.")
	}
	if aggregateArtifact.Status == "missing" && round.SubmittedSlots == round.TotalSlots && round.TotalSlots > 0 {
		appendWarning("All reviewer submissions are present, but the aggregate artifact is still missing.")
	}
	round.Warnings = dedupeStrings(warnings)
	return round
}

func (s Service) readReviewers(roundDir string, manifest *Manifest, ledger *Ledger) ([]Reviewer, []Artifact, []string) {
	slotOrder := make([]string, 0)
	slotSeen := map[string]bool{}
	manifestBySlot := map[string]ManifestSlot{}
	ledgerBySlot := map[string]LedgerSlot{}
	submissionPathBySlot := map[string]string{}
	addSlot := func(slot string) {
		slot = strings.TrimSpace(slot)
		if slot == "" || slotSeen[slot] {
			return
		}
		slotOrder = append(slotOrder, slot)
		slotSeen[slot] = true
	}

	if manifest != nil {
		for _, item := range manifest.Dimensions {
			slot := strings.TrimSpace(item.Slot)
			if slot == "" {
				continue
			}
			addSlot(slot)
			manifestBySlot[slot] = item
			if path := strings.TrimSpace(item.SubmissionPath); path != "" {
				submissionPathBySlot[slot] = path
			}
		}
	}
	if ledger != nil {
		for _, item := range ledger.Slots {
			slot := strings.TrimSpace(item.Slot)
			if slot == "" {
				continue
			}
			addSlot(slot)
			ledgerBySlot[slot] = item
			if path := strings.TrimSpace(item.SubmissionPath); path != "" {
				submissionPathBySlot[slot] = path
			}
		}
	}
	discoveredSubmissionPaths, discoveryWarnings := discoverSubmissionPaths(filepath.Join(roundDir, "submissions"))
	discoveredSlots := make([]string, 0, len(discoveredSubmissionPaths))
	for slot := range discoveredSubmissionPaths {
		discoveredSlots = append(discoveredSlots, slot)
	}
	sort.Strings(discoveredSlots)
	for _, slot := range discoveredSlots {
		path := discoveredSubmissionPaths[slot]
		addSlot(slot)
		if _, exists := submissionPathBySlot[slot]; !exists {
			submissionPathBySlot[slot] = path
		}
	}

	reviewers := make([]Reviewer, 0, len(slotOrder))
	artifacts := make([]Artifact, 0, len(slotOrder))
	warnings := make([]string, 0, len(slotOrder)+len(discoveryWarnings))
	warnings = append(warnings, discoveryWarnings...)
	for _, slot := range slotOrder {
		reviewer := Reviewer{Slot: slot}
		ledgerClaimedSubmitted := false
		ledgerStatusWarning := ""
		hasLedgerEntry := false
		artifactPath := filepath.Join(roundDir, "submissions", slot+".json")
		if path, ok := submissionPathBySlot[slot]; ok && strings.TrimSpace(path) != "" {
			artifactPath = path
		}
		if item, ok := manifestBySlot[slot]; ok {
			reviewer.Name = item.Name
			reviewer.Instructions = item.Instructions
		}
		if item, ok := ledgerBySlot[slot]; ok {
			hasLedgerEntry = true
			if reviewer.Name == "" {
				reviewer.Name = item.Name
			}
			reviewer.Status = item.Status
			reviewer.SubmittedAt = item.SubmittedAt
			normalizedLedgerStatus, warning := canonicalSlotStatus(item.Status)
			ledgerClaimedSubmitted = normalizedLedgerStatus == "submitted"
			ledgerStatusWarning = warning
		}
		reviewer.SubmissionPath = artifactPath

		artifactLabel := fmt.Sprintf("Submission: %s", reviewer.Slot)
		if reviewer.Name != "" {
			artifactLabel = fmt.Sprintf("Submission: %s", reviewer.Name)
		}
		submissionArtifact, submission, submissionWarning := readJSONArtifact[Submission](artifactLabel, artifactPath, validateSubmissionArtifact)
		if submissionArtifact.Status != "" {
			artifacts = append(artifacts, submissionArtifact)
		}
		if submission != nil {
			if reviewer.Name == "" {
				reviewer.Name = submission.Dimension
			}
			reviewer.Summary = submission.Summary
			reviewer.Findings = submission.Findings
			if reviewer.SubmittedAt == "" {
				reviewer.SubmittedAt = submission.SubmittedAt
			}
			if reviewer.Status == "" {
				reviewer.Status = "submitted"
			}
		} else if reviewer.Status == "" || ledgerClaimedSubmitted {
			reviewer.Status = "pending"
		}

		reviewerWarnings := make([]string, 0, 2)
		if ledgerStatusWarning != "" {
			reviewerWarnings = append(reviewerWarnings, ledgerStatusWarning)
		}
		if submissionWarning != "" {
			reviewerWarnings = append(reviewerWarnings, submissionWarning)
		}
		if ledgerClaimedSubmitted && submission == nil {
			reviewerWarnings = append(reviewerWarnings, "Ledger marks this reviewer as submitted, but the submission artifact is unavailable.")
		}
		if hasLedgerEntry && !ledgerClaimedSubmitted && submission != nil {
			reviewerWarnings = append(reviewerWarnings, "Submission artifact exists even though the ledger does not mark this reviewer as submitted.")
		}
		reviewer.Warnings = dedupeStrings(reviewerWarnings)
		if len(reviewer.Warnings) > 0 {
			warnings = append(warnings, fmt.Sprintf("%s (%s): %s", reviewerDisplayName(reviewer), reviewer.Slot, strings.Join(reviewer.Warnings, " ")))
		}
		reviewers = append(reviewers, reviewer)
	}
	return reviewers, artifacts, warnings
}

func reviewerDisplayName(reviewer Reviewer) string {
	if strings.TrimSpace(reviewer.Name) != "" {
		return reviewer.Name
	}
	return reviewer.Slot
}

func normalizeSlotStatus(status string) string {
	value, _ := canonicalSlotStatus(status)
	return value
}

func canonicalSlotStatus(status string) (string, string) {
	value := strings.TrimSpace(strings.ToLower(status))
	switch value {
	case "", "pending":
		return "pending", ""
	case "submitted":
		return "submitted", ""
	default:
		return "pending", fmt.Sprintf("Ledger reports unknown slot status %q, so this reviewer is shown conservatively as pending.", strings.TrimSpace(status))
	}
}

func resolveRoundStatus(round Round, manifestArtifact, ledgerArtifact, aggregateArtifact Artifact) (string, string) {
	if manifestArtifact.Status == "invalid" || ledgerArtifact.Status == "invalid" || aggregateArtifact.Status == "invalid" {
		return "degraded", "One or more artifacts are malformed; review state is shown conservatively."
	}
	if manifestArtifact.Status == "missing" && ledgerArtifact.Status == "missing" {
		return "degraded", "Core review artifacts are missing."
	}
	if round.TotalSlots == 0 {
		if round.IsActive {
			return "in_progress", "Review round is active, but reviewer slots could not be recovered yet."
		}
		return "incomplete", "Review round metadata is incomplete."
	}
	if round.PendingSlots > 0 {
		return "waiting_for_submissions", fmt.Sprintf("Waiting for %d of %d reviewer submissions.", round.PendingSlots, round.TotalSlots)
	}
	if manifestArtifact.Status != "available" || ledgerArtifact.Status != "available" {
		return "degraded", "Core review artifacts are incomplete, so aggregate state is shown conservatively."
	}
	if aggregateArtifact.Status == "available" && strings.TrimSpace(round.Decision) != "" {
		switch round.Decision {
		case "pass":
			return "pass", "Aggregate review passed cleanly."
		case "changes_requested":
			return "changes_requested", "Aggregate review requested changes."
		default:
			return "aggregated", fmt.Sprintf("Aggregate review decision: %s.", round.Decision)
		}
	}
	if aggregateArtifact.Status == "missing" {
		return "waiting_for_aggregation", "All reviewer submissions are present; waiting for aggregation."
	}
	if round.IsActive {
		return "in_progress", "Review round is still active."
	}
	return "complete", "Review round artifacts are present."
}

func discoverSubmissionPaths(submissionsDir string) (map[string]string, []string) {
	paths := map[string]string{}
	warnings := []string{}

	entries, err := os.ReadDir(submissionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return paths, warnings
		}
		return paths, []string{fmt.Sprintf("Unable to inspect submissions directory %s: %v", filepath.ToSlash(submissionsDir), err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if filepath.Ext(name) != ".json" {
			continue
		}
		slot := strings.TrimSuffix(name, ".json")
		slot = strings.TrimSpace(slot)
		if slot == "" {
			continue
		}
		paths[slot] = filepath.Join(submissionsDir, name)
	}

	return paths, warnings
}

type artifactValidator[T any] func(*T) []string

func readJSONArtifact[T any](label, path string, validator artifactValidator[T]) (Artifact, *T, string) {
	artifact := Artifact{
		Label: label,
		Path:  path,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			artifact.Status = "missing"
			artifact.Summary = "Artifact file is missing."
			return artifact, nil, fmt.Sprintf("%s is missing.", label)
		}
		artifact.Status = "invalid"
		artifact.Summary = err.Error()
		return artifact, nil, fmt.Sprintf("Unable to read %s: %v", strings.ToLower(label), err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		artifact.Status = "invalid"
		artifact.Summary = "Artifact file is empty."
		return artifact, nil, fmt.Sprintf("%s is empty.", label)
	}
	if !json.Valid([]byte(trimmed)) {
		artifact.Status = "invalid"
		artifact.Summary = "Artifact file is not valid JSON."
		artifact.ContentType = "text"
		artifact.Content = mustMarshalString(trimmed)
		return artifact, nil, fmt.Sprintf("%s is not valid JSON.", label)
	}

	var parsed T
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		artifact.Status = "invalid"
		artifact.Summary = "Artifact JSON could not be parsed."
		artifact.ContentType = "json"
		artifact.Content = json.RawMessage([]byte(trimmed))
		return artifact, nil, fmt.Sprintf("%s could not be parsed cleanly.", label)
	}
	if validator != nil {
		if missing := dedupeStrings(validator(&parsed)); len(missing) > 0 {
			artifact.Status = "invalid"
			artifact.Summary = "Artifact JSON is missing required fields."
			artifact.ContentType = "json"
			artifact.Content = json.RawMessage([]byte(trimmed))
			return artifact, nil, fmt.Sprintf("%s is missing required fields: %s.", label, strings.Join(missing, ", "))
		}
	}

	artifact.Status = "available"
	artifact.Summary = "Artifact is available."
	artifact.ContentType = "json"
	artifact.Content = json.RawMessage([]byte(trimmed))
	return artifact, &parsed, ""
}

func mustMarshalString(value string) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`""`)
	}
	return data
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		result = append(result, trimmed)
	}
	return result
}

func validateManifestArtifact(manifest *Manifest) []string {
	missing := []string{}
	if strings.TrimSpace(manifest.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(manifest.Kind) == "" {
		missing = append(missing, "kind")
	}
	if manifest.Revision <= 0 {
		missing = append(missing, "revision")
	}
	if strings.TrimSpace(manifest.PlanPath) == "" {
		missing = append(missing, "plan_path")
	}
	if strings.TrimSpace(manifest.PlanStem) == "" {
		missing = append(missing, "plan_stem")
	}
	if strings.TrimSpace(manifest.CreatedAt) == "" {
		missing = append(missing, "created_at")
	}
	if strings.TrimSpace(manifest.LedgerPath) == "" {
		missing = append(missing, "ledger_path")
	}
	if strings.TrimSpace(manifest.Aggregate) == "" {
		missing = append(missing, "aggregate_path")
	}
	if strings.TrimSpace(manifest.Submissions) == "" {
		missing = append(missing, "submissions_dir")
	}
	if len(manifest.Dimensions) == 0 {
		missing = append(missing, "dimensions")
	}
	for index, slot := range manifest.Dimensions {
		prefix := fmt.Sprintf("dimensions[%d]", index)
		if strings.TrimSpace(slot.Name) == "" {
			missing = append(missing, prefix+".name")
		}
		if strings.TrimSpace(slot.Slot) == "" {
			missing = append(missing, prefix+".slot")
		}
		if strings.TrimSpace(slot.Instructions) == "" {
			missing = append(missing, prefix+".instructions")
		}
		if strings.TrimSpace(slot.SubmissionPath) == "" {
			missing = append(missing, prefix+".submission_path")
		}
	}
	return missing
}

func validateLedgerArtifact(ledger *Ledger) []string {
	missing := []string{}
	if strings.TrimSpace(ledger.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(ledger.Kind) == "" {
		missing = append(missing, "kind")
	}
	if strings.TrimSpace(ledger.UpdatedAt) == "" {
		missing = append(missing, "updated_at")
	}
	if len(ledger.Slots) == 0 {
		missing = append(missing, "slots")
	}
	for index, slot := range ledger.Slots {
		prefix := fmt.Sprintf("slots[%d]", index)
		if strings.TrimSpace(slot.Name) == "" {
			missing = append(missing, prefix+".name")
		}
		if strings.TrimSpace(slot.Slot) == "" {
			missing = append(missing, prefix+".slot")
		}
		if strings.TrimSpace(slot.Status) == "" {
			missing = append(missing, prefix+".status")
		}
		if strings.TrimSpace(slot.SubmissionPath) == "" {
			missing = append(missing, prefix+".submission_path")
		}
		if normalizedStatus, _ := canonicalSlotStatus(slot.Status); normalizedStatus == "submitted" && strings.TrimSpace(slot.SubmittedAt) == "" {
			missing = append(missing, prefix+".submitted_at")
		}
	}
	return missing
}

func validateSubmissionArtifact(submission *Submission) []string {
	missing := []string{}
	if strings.TrimSpace(submission.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(submission.Slot) == "" {
		missing = append(missing, "slot")
	}
	if strings.TrimSpace(submission.Dimension) == "" {
		missing = append(missing, "dimension")
	}
	if strings.TrimSpace(submission.SubmittedAt) == "" {
		missing = append(missing, "submitted_at")
	}
	if strings.TrimSpace(submission.Summary) == "" {
		missing = append(missing, "summary")
	}
	return missing
}

func validateAggregateArtifact(aggregate *Aggregate) []string {
	missing := []string{}
	if strings.TrimSpace(aggregate.RoundID) == "" {
		missing = append(missing, "round_id")
	}
	if strings.TrimSpace(aggregate.Kind) == "" {
		missing = append(missing, "kind")
	}
	if aggregate.Revision <= 0 {
		missing = append(missing, "revision")
	}
	if strings.TrimSpace(aggregate.Decision) == "" {
		missing = append(missing, "decision")
	}
	if strings.TrimSpace(aggregate.AggregatedAt) == "" {
		missing = append(missing, "aggregated_at")
	}
	return missing
}
