package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/inputschema"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/stepcloseout"
)

var slotNamePattern = regexp.MustCompile(`[^a-z0-9]+`)
var compactRoundIDPattern = regexp.MustCompile(`^review-([0-9]+)-([a-z0-9-]+)$`)
var saveState = runstate.SaveState
var writeJSON = writeJSONFile

type Service struct {
	Workdir        string
	Now            func() time.Time
	AfterStart     func(StartResult) error
	AfterSubmit    func(SubmitResult) error
	AfterAggregate func(AggregateResult) error
}

type Spec = contracts.ReviewSpec
type Dimension = contracts.ReviewDimension
type Manifest = contracts.ReviewManifest
type ManifestSlot = contracts.ReviewManifestSlot
type Ledger = contracts.ReviewLedger
type LedgerSlot = contracts.ReviewLedgerSlot
type SubmissionInput = contracts.ReviewSubmissionInput
type Submission = contracts.ReviewSubmission
type Finding = contracts.ReviewFinding
type Aggregate = contracts.ReviewAggregate
type AggregateFinding = contracts.ReviewAggregateFinding
type CommandError = contracts.ErrorDetail
type NextAction = contracts.NextAction
type StartResult = contracts.ReviewStartResult
type StartArtifacts = contracts.ReviewStartArtifacts
type SubmitResult = contracts.ReviewSubmitResult
type SubmitArtifacts = contracts.ReviewSubmitArtifacts
type AggregateResult = contracts.ReviewAggregateResult
type AggregateArtifacts = contracts.ReviewAggregateArtifacts

func (s Service) Start(specBytes []byte) StartResult {
	lockedPlanPath, release, err := s.acquireReviewMutationLock()
	if err == nil {
		defer release()
	} else {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Another review state mutation is already in progress.",
			Errors:  []CommandError{{Path: "review", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(lockedPlanPath), filepath.Ext(lockedPlanPath))
	releaseState, err := runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Another local state mutation is already in progress.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	defer releaseState()

	now := s.now()
	planPath, doc, planStem, relPlanPath, state, statePath, errResult := s.loadCurrentExecutingPlan(lockedPlanPath)
	if errResult != nil {
		return *errResult
	}

	var spec Spec
	if issues := inputschema.DecodeAndValidate("inputs.review.spec", "spec", specBytes, &spec); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateSpec(spec); len(issues) > 0 {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec is invalid.",
			Errors:  issues,
		}
	}
	inferredStep, revision, reviewTitle, err := inferReviewBinding(s.Workdir, planStem, doc, state, spec)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review spec does not match the current workflow state.",
			Errors:  []CommandError{{Path: "spec", Message: err.Error()}},
		}
	}

	roundID, err := nextRoundID(s.Workdir, planStem, spec.Kind)
	if err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to determine the next review round identifier.",
			Errors:  []CommandError{{Path: "round", Message: err.Error()}},
		}
	}
	roundDir := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, "reviews", roundID)
	submissionsDir := filepath.Join(roundDir, "submissions")
	manifestPath := filepath.Join(roundDir, "manifest.json")
	ledgerPath := filepath.Join(roundDir, "ledger.json")
	aggregatePath := filepath.Join(roundDir, "aggregate.json")
	if err := os.MkdirAll(submissionsDir, 0o755); err != nil {
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to initialize review artifacts.",
			Errors:  []CommandError{{Path: "round", Message: err.Error()}},
		}
	}

	slots := make([]ManifestSlot, 0, len(spec.Dimensions))
	ledger := Ledger{
		RoundID:   roundID,
		Kind:      spec.Kind,
		UpdatedAt: now.Format(time.RFC3339),
		Slots:     make([]LedgerSlot, 0, len(spec.Dimensions)),
	}
	for _, dimension := range spec.Dimensions {
		slot := normalizeSlot(dimension.Name)
		submissionPath := filepath.Join(submissionsDir, slot+".json")
		slots = append(slots, ManifestSlot{
			Name:           dimension.Name,
			Slot:           slot,
			Instructions:   dimension.Instructions,
			SubmissionPath: submissionPath,
		})
		ledger.Slots = append(ledger.Slots, LedgerSlot{
			Name:           dimension.Name,
			Slot:           slot,
			Status:         "pending",
			SubmissionPath: submissionPath,
		})
	}

	manifest := Manifest{
		RoundID:     roundID,
		Kind:        spec.Kind,
		Step:        inferredStep,
		Revision:    revision,
		ReviewTitle: reviewTitle,
		PlanPath:    relPlanPath,
		PlanStem:    planStem,
		CreatedAt:   now.Format(time.RFC3339),
		Dimensions:  slots,
		LedgerPath:  ledgerPath,
		Aggregate:   aggregatePath,
		Submissions: submissionsDir,
	}
	if err := writeJSON(manifestPath, manifest); err != nil {
		_ = os.RemoveAll(roundDir)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist the review manifest.",
			Errors:  []CommandError{{Path: "manifest", Message: err.Error()}},
		}
	}
	if err := writeJSON(ledgerPath, ledger); err != nil {
		_ = os.RemoveAll(roundDir)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist the review ledger.",
			Errors:  []CommandError{{Path: "ledger", Message: err.Error()}},
		}
	}

	originalState := cloneState(state)
	if state == nil {
		state = &runstate.State{}
	}
	state.ActiveReviewRound = &runstate.ReviewRound{
		RoundID:    roundID,
		Kind:       spec.Kind,
		Step:       inferredStep,
		Revision:   revision,
		Aggregated: false,
		Decision:   "",
	}
	statePath, err = saveState(s.Workdir, planStem, state)
	if err != nil {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		if removeErr := os.RemoveAll(roundDir); removeErr != nil && !os.IsNotExist(removeErr) {
			issues = append(issues, CommandError{Path: "round", Message: fmt.Sprintf("rollback review round: %v", removeErr)})
		}
		issues = append([]CommandError{{Path: "state", Message: err.Error()}}, issues...)
		return StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to persist local harness state.",
			Errors:  issues,
		}
	}

	_ = doc

	return s.finalizeStart(StartResult{
		OK:      true,
		Command: "review start",
		Summary: fmt.Sprintf("Created %s review round %q.", spec.Kind, roundID),
		Artifacts: &StartArtifacts{
			PlanPath:       planPath,
			LocalStatePath: statePath,
			RoundID:        roundID,
			ManifestPath:   manifestPath,
			LedgerPath:     ledgerPath,
			AggregatePath:  aggregatePath,
			Slots:          slots,
		},
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Launch reviewer subagents for the returned slots and have each reviewer submit structured results for its assigned slot.",
			},
			{
				Command:     strPtr(fmt.Sprintf("harness review aggregate --round %s", roundID)),
				Description: "Aggregate the round once every expected reviewer submission has landed.",
			},
		},
	}, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		if err := os.RemoveAll(roundDir); err != nil && !os.IsNotExist(err) {
			issues = append(issues, CommandError{Path: "round", Message: fmt.Sprintf("rollback review round: %v", err)})
		}
		return issues
	})
}

func (s Service) Submit(roundID, slot string, inputBytes []byte) SubmitResult {
	lockedPlanPath, release, err := s.acquireReviewMutationLock()
	if err == nil {
		defer release()
	} else {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Another review state mutation is already in progress.",
			Errors:  []CommandError{{Path: "review", Message: err.Error()}},
		}
	}
	_, _, planStem, _, _, _, errResult := s.loadCurrentExecutingPlan(lockedPlanPath)
	if errResult != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: errResult.Summary,
			Errors:  errResult.Errors,
		}
	}

	manifestPath := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, "reviews", roundID, "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to load the review manifest.",
			Errors:  []CommandError{{Path: "manifest", Message: err.Error()}},
		}
	}
	slotDef := findSlot(manifest, slot)
	if slotDef == nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Submission does not match an expected reviewer slot.",
			Errors:  []CommandError{{Path: "slot", Message: fmt.Sprintf("unknown slot %q for review round %q", slot, roundID)}},
		}
	}

	var input SubmissionInput
	if issues := inputschema.DecodeAndValidate("inputs.review.submission", "submission", inputBytes, &input); len(issues) > 0 {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer submission is invalid.",
			Errors:  issues,
		}
	}
	if issues := validateSubmission(input); len(issues) > 0 {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Reviewer submission is invalid.",
			Errors:  issues,
		}
	}

	now := s.now().Format(time.RFC3339)
	submission := Submission{
		RoundID:     roundID,
		Slot:        slotDef.Slot,
		Dimension:   slotDef.Name,
		SubmittedAt: now,
		Summary:     strings.TrimSpace(input.Summary),
		Findings:    input.Findings,
	}
	previousSubmission, previousSubmissionExists, err := readFileIfExists(slotDef.SubmissionPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to snapshot the previous reviewer submission.",
			Errors:  []CommandError{{Path: "submission", Message: err.Error()}},
		}
	}
	ledger, err := loadLedger(manifest.LedgerPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to load the review ledger.",
			Errors:  []CommandError{{Path: "ledger", Message: err.Error()}},
		}
	}
	previousLedger, previousLedgerExists, err := readFileIfExists(manifest.LedgerPath)
	if err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to snapshot the review ledger before writing the submission.",
			Errors:  []CommandError{{Path: "ledger", Message: err.Error()}},
		}
	}
	if err := writeJSON(slotDef.SubmissionPath, submission); err != nil {
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to persist the reviewer submission.",
			Errors:  []CommandError{{Path: "submission", Message: err.Error()}},
		}
	}
	for i := range ledger.Slots {
		if ledger.Slots[i].Slot == slotDef.Slot {
			ledger.Slots[i].Status = "submitted"
			ledger.Slots[i].SubmittedAt = now
		}
	}
	ledger.UpdatedAt = now
	if err := writeJSON(manifest.LedgerPath, ledger); err != nil {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "ledger")
		issues = append(issues, restoreJSONFileSnapshot(slotDef.SubmissionPath, previousSubmission, previousSubmissionExists, "submission")...)
		issues = append([]CommandError{{Path: "ledger", Message: err.Error()}}, issues...)
		return SubmitResult{
			OK:      false,
			Command: "review submit",
			Summary: "Unable to persist the review ledger.",
			Errors:  issues,
		}
	}

	return s.finalizeSubmit(SubmitResult{
		OK:      true,
		Command: "review submit",
		Summary: fmt.Sprintf("Recorded submission for slot %q in review round %q.", slotDef.Slot, roundID),
		Artifacts: &SubmitArtifacts{
			RoundID:        roundID,
			Slot:           slotDef.Slot,
			SubmissionPath: slotDef.SubmissionPath,
			LedgerPath:     manifest.LedgerPath,
		},
		NextAction: []NextAction{
			{
				Command:     nil,
				Description: "Report the submission receipt to the controller agent and end the reviewer thread. If the same slot later needs a narrow follow-up for the same tracked step or the same finalize review title in the same revision, the controller may reopen this reviewer through the runtime's native resume mechanism only after this submission is verified and only while the slot instructions still materially match.",
			},
		},
	}, func() []CommandError {
		issues := restoreJSONFileSnapshot(manifest.LedgerPath, previousLedger, previousLedgerExists, "ledger")
		issues = append(issues, restoreJSONFileSnapshot(slotDef.SubmissionPath, previousSubmission, previousSubmissionExists, "submission")...)
		return issues
	})
}

func (s Service) Aggregate(roundID string) AggregateResult {
	lockedPlanPath, release, err := s.acquireReviewMutationLock()
	if err == nil {
		defer release()
	} else {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Another review state mutation is already in progress.",
			Errors:  []CommandError{{Path: "review", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(lockedPlanPath), filepath.Ext(lockedPlanPath))
	releaseState, err := runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Another local state mutation is already in progress.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	defer releaseState()

	_, _, planStem, _, state, statePath, errResult := s.loadCurrentExecutingPlan(lockedPlanPath)
	if errResult != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: errResult.Summary,
			Errors:  errResult.Errors,
		}
	}
	if guard := activeAggregateRoundError(state, roundID); guard != nil {
		return *guard
	}

	manifestPath := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, "reviews", roundID, "manifest.json")
	manifest, err := loadManifest(manifestPath)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to load the review manifest.",
			Errors:  []CommandError{{Path: "manifest", Message: err.Error()}},
		}
	}

	blocking := make([]AggregateFinding, 0)
	nonBlocking := make([]AggregateFinding, 0)
	missing := make([]string, 0)
	for _, slotDef := range manifest.Dimensions {
		submission, err := loadSubmission(slotDef.SubmissionPath)
		if err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, slotDef.Slot)
				continue
			}
			return AggregateResult{
				OK:      false,
				Command: "review aggregate",
				Summary: "Unable to load reviewer submissions.",
				Errors:  []CommandError{{Path: "submission", Message: err.Error()}},
			}
		}
		for _, finding := range submission.Findings {
			aggregateFinding := AggregateFinding{
				Slot:         submission.Slot,
				Dimension:    submission.Dimension,
				Severity:     finding.Severity,
				Title:        finding.Title,
				Details:      finding.Details,
				Locations:    cloneLocations(finding.Locations, finding.HasLocations),
				HasLocations: finding.HasLocations,
			}
			if isBlockingSeverity(finding.Severity) {
				blocking = append(blocking, aggregateFinding)
			} else {
				nonBlocking = append(nonBlocking, aggregateFinding)
			}
		}
	}
	if len(missing) > 0 {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Review round is missing required submissions.",
			Errors:  []CommandError{{Path: "submissions", Message: fmt.Sprintf("missing submissions for slots: %s", strings.Join(missing, ", "))}},
		}
	}

	decision := "pass"
	if len(blocking) > 0 {
		decision = "changes_requested"
	}

	aggregate := Aggregate{
		RoundID:             roundID,
		Kind:                manifest.Kind,
		Step:                manifest.Step,
		Revision:            manifest.Revision,
		ReviewTitle:         manifest.ReviewTitle,
		Decision:            decision,
		BlockingFindings:    blocking,
		NonBlockingFindings: nonBlocking,
		AggregatedAt:        s.now().Format(time.RFC3339),
	}
	state, _, err = runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to reload local harness state before persisting the aggregate.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	if guard := activeAggregateRoundError(state, roundID); guard != nil {
		return *guard
	}
	originalState := cloneState(state)
	previousAggregate, previousAggregateExists, err := readFileIfExists(manifest.Aggregate)
	if err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to snapshot the previous aggregate artifact.",
			Errors:  []CommandError{{Path: "aggregate", Message: err.Error()}},
		}
	}
	if err := writeJSON(manifest.Aggregate, aggregate); err != nil {
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to persist the aggregate review result.",
			Errors:  []CommandError{{Path: "aggregate", Message: err.Error()}},
		}
	}

	if state == nil {
		state = &runstate.State{}
	}
	state.ActiveReviewRound = &runstate.ReviewRound{
		RoundID:    manifest.RoundID,
		Kind:       manifest.Kind,
		Step:       manifest.Step,
		Revision:   manifest.Revision,
		Aggregated: true,
		Decision:   decision,
	}
	statePath, err = saveState(s.Workdir, planStem, state)
	if err != nil {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")...)
		issues = append([]CommandError{{Path: "state", Message: err.Error()}}, issues...)
		return AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "Unable to persist local harness state.",
			Errors:  issues,
		}
	}

	return s.finalizeAggregate(AggregateResult{
		OK:      true,
		Command: "review aggregate",
		Summary: buildAggregateSummary(manifest.Kind, decision, len(blocking), len(nonBlocking)),
		Artifacts: &AggregateArtifacts{
			RoundID:        roundID,
			AggregatePath:  manifest.Aggregate,
			LocalStatePath: statePath,
		},
		Review:     &aggregate,
		NextAction: buildAggregateNextActions(manifest.Kind, decision),
	}, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		issues = append(issues, restoreJSONFileSnapshot(manifest.Aggregate, previousAggregate, previousAggregateExists, "aggregate")...)
		return issues
	})
}

func activeAggregateRoundError(state *runstate.State, roundID string) *AggregateResult {
	if state == nil || state.ActiveReviewRound == nil {
		return &AggregateResult{
			OK:      false,
			Command: "review aggregate",
			Summary: "No active review round is available to aggregate.",
			Errors:  []CommandError{{Path: "round", Message: "review aggregate only supports the current active review round"}},
		}
	}
	if state.ActiveReviewRound.RoundID == roundID {
		return nil
	}
	return &AggregateResult{
		OK:      false,
		Command: "review aggregate",
		Summary: "Only the current active review round can be aggregated.",
		Errors: []CommandError{{
			Path:    "round",
			Message: fmt.Sprintf("round %q is not the current active review round %q", roundID, state.ActiveReviewRound.RoundID),
		}},
	}
}

func (s Service) acquireReviewMutationLock() (string, func(), error) {
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", func() {}, nil
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	lockPath := filepath.Join(s.Workdir, ".local", "harness", "plans", planStem, ".review-mutation.lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return "", nil, err
	}
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return "", nil, fmt.Errorf("another review start or aggregate command is already mutating plan %q; retry after it finishes", planStem)
		}
		return "", nil, err
	}
	return planPath, func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

func (s Service) loadCurrentExecutingPlan(lockedPlanPath string) (string, *plan.Document, string, string, *runstate.State, string, *StartResult) {
	planPath := strings.TrimSpace(lockedPlanPath)
	if planPath == "" {
		var err error
		planPath, err = plan.DetectCurrentPath(s.Workdir)
		if err != nil {
			return "", nil, "", "", nil, "", &StartResult{
				OK:      false,
				Command: "review start",
				Summary: "Unable to determine the current plan.",
				Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
			}
		}
	}
	doc, err := plan.LoadFile(planPath)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	if doc.DerivedPlanStatus() != "active" || doc.DerivedLifecycle(state) != "executing" {
		return "", nil, "", "", nil, "", &StartResult{
			OK:      false,
			Command: "review start",
			Summary: "Review commands require an active executing plan.",
			Errors: []CommandError{{
				Path:    "plan.lifecycle",
				Message: fmt.Sprintf("current plan is status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
			}},
		}
	}
	return planPath, doc, planStem, relPlanPath, state, statePath, nil
}

func validateSpec(spec Spec) []CommandError {
	issues := make([]CommandError, 0)
	if !slices.Contains([]string{"delta", "full"}, spec.Kind) {
		issues = append(issues, CommandError{Path: "spec.kind", Message: "must be delta or full"})
	}
	if spec.Step != nil && *spec.Step <= 0 {
		issues = append(issues, CommandError{Path: "spec.step", Message: "must be a positive 1-based step number"})
	}
	if len(spec.Dimensions) == 0 {
		issues = append(issues, CommandError{Path: "spec.dimensions", Message: "must contain at least one dimension"})
	}
	seenSlots := map[string]bool{}
	for i, dimension := range spec.Dimensions {
		pathPrefix := fmt.Sprintf("spec.dimensions[%d]", i)
		if strings.TrimSpace(dimension.Name) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".name", Message: "must not be empty"})
		}
		if strings.TrimSpace(dimension.Instructions) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".instructions", Message: "must not be empty"})
		}
		slot := normalizeSlot(dimension.Name)
		if slot == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".name", Message: "must normalize to a non-empty slot identifier"})
			continue
		}
		if seenSlots[slot] {
			issues = append(issues, CommandError{Path: pathPrefix + ".name", Message: fmt.Sprintf("duplicates slot %q after normalization", slot)})
		}
		seenSlots[slot] = true
	}
	return issues
}

func validateSubmission(input SubmissionInput) []CommandError {
	issues := make([]CommandError, 0)
	if strings.TrimSpace(input.Summary) == "" {
		issues = append(issues, CommandError{Path: "submission.summary", Message: "must not be empty"})
	}
	for i, finding := range input.Findings {
		pathPrefix := fmt.Sprintf("submission.findings[%d]", i)
		if !slices.Contains([]string{"blocker", "important", "minor"}, finding.Severity) {
			issues = append(issues, CommandError{Path: pathPrefix + ".severity", Message: "must be blocker, important, or minor"})
		}
		if strings.TrimSpace(finding.Title) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".title", Message: "must not be empty"})
		}
		if strings.TrimSpace(finding.Details) == "" {
			issues = append(issues, CommandError{Path: pathPrefix + ".details", Message: "must not be empty"})
		}
		if finding.HasLocations && finding.Locations == nil {
			issues = append(issues, CommandError{
				Path:    pathPrefix + ".locations",
				Message: "must be an array of strings when present",
			})
			continue
		}
		for j, location := range finding.Locations {
			if strings.TrimSpace(location) == "" {
				issues = append(issues, CommandError{
					Path:    fmt.Sprintf("%s.locations[%d]", pathPrefix, j),
					Message: "must not be empty",
				})
			}
		}
	}
	return issues
}

func normalizeSlot(name string) string {
	slot := strings.ToLower(strings.TrimSpace(name))
	slot = slotNamePattern.ReplaceAllString(slot, "-")
	slot = strings.Trim(slot, "-")
	return slot
}

func cloneLocations(locations []string, present bool) []string {
	if !present {
		return nil
	}
	if len(locations) == 0 {
		return []string{}
	}
	return append([]string(nil), locations...)
}

func nextRoundID(workdir, planStem, kind string) (string, error) {
	sequence, err := nextRoundSequence(workdir, planStem)
	if err != nil {
		return "", err
	}
	return formatRoundID(sequence, kind), nil
}

func nextRoundSequence(workdir, planStem string) (int, error) {
	reviewsDir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews")
	entries, err := os.ReadDir(reviewsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, err
	}

	maxSequence := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		matches := compactRoundIDPattern.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		sequence, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, fmt.Errorf("parse compact review round sequence from %q: %w", entry.Name(), err)
		}
		if sequence > maxSequence {
			maxSequence = sequence
		}
	}
	return maxSequence + 1, nil
}

func formatRoundID(sequence int, kind string) string {
	return fmt.Sprintf("review-%03d-%s", sequence, kind)
}

func inferReviewBinding(workdir, planStem string, doc *plan.Document, state *runstate.State, spec Spec) (*int, int, string, error) {
	revision := runstate.CurrentRevision(state)
	if stepIndex, ok, err := inferReviewStepIndex(doc, state, spec); err != nil {
		return nil, 0, "", err
	} else if ok {
		reviewTitle := strings.TrimSpace(spec.ReviewTitle)
		if reviewTitle == "" {
			reviewTitle = doc.Steps[stepIndex].Title
		}
		stepNumber := stepIndex + 1
		return &stepNumber, revision, reviewTitle, nil
	}

	if pendingNewStepReopen(doc, state) {
		return nil, 0, "", fmt.Errorf("reopen mode new-step still requires a new unfinished step before review can start")
	}
	if !doc.AllStepsCompleted() {
		return nil, 0, "", fmt.Errorf("no reviewable tracked step could be inferred; set spec.step to select a tracked step explicitly")
	}
	reminder := stepcloseout.LoadReminder(workdir, planStem, doc, "execution/finalize/review", nil)
	if len(reminder.MissingTitles) > 0 {
		earliestTitle := reminder.MissingTitles[0]
		earliestStepNumber := reminder.MissingIndexes[0] + 1
		return nil, 0, "", fmt.Errorf("earlier completed steps still need review-complete closeout; repair %s first with spec.step=%d or record NO_STEP_REVIEW_NEEDED: <reason> in Review Notes before starting default finalize review", earliestTitle, earliestStepNumber)
	}

	reviewTitle := strings.TrimSpace(spec.ReviewTitle)
	if reviewTitle == "" {
		if spec.Kind == "full" {
			reviewTitle = "Full branch candidate before archive"
		} else {
			reviewTitle = "Branch candidate before archive"
		}
	}
	return nil, revision, reviewTitle, nil
}

func inferReviewStepIndex(doc *plan.Document, state *runstate.State, spec Spec) (int, bool, error) {
	if doc == nil {
		return 0, false, fmt.Errorf("current plan is unavailable")
	}
	if spec.Step != nil {
		index := *spec.Step - 1
		if index < 0 || index >= len(doc.Steps) {
			return 0, false, fmt.Errorf("spec.step=%d does not match a tracked step", *spec.Step)
		}
		return index, true, nil
	}
	if current := currentStepIndex(doc); current >= 0 {
		return current, true, nil
	}
	return 0, false, nil
}

func currentStepIndex(doc *plan.Document) int {
	if doc == nil {
		return -1
	}
	for index, step := range doc.Steps {
		if !step.Done {
			return index
		}
	}
	return -1
}

func pendingNewStepReopen(doc *plan.Document, state *runstate.State) bool {
	return state != nil &&
		state.Reopen != nil &&
		state.Reopen.Mode == "new-step" &&
		state.Reopen.BaseStepCount > 0 &&
		doc != nil &&
		len(doc.Steps) <= state.Reopen.BaseStepCount &&
		doc.CurrentStep() == nil &&
		doc.AllStepsCompleted()
}

func loadManifest(path string) (*Manifest, error) {
	var manifest Manifest
	if err := readJSONFile(path, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func loadLedger(path string) (*Ledger, error) {
	var ledger Ledger
	if err := readJSONFile(path, &ledger); err != nil {
		return nil, err
	}
	return &ledger, nil
}

func loadSubmission(path string) (*Submission, error) {
	var submission Submission
	if err := readJSONFile(path, &submission); err != nil {
		return nil, err
	}
	return &submission, nil
}

func readJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func findSlot(manifest *Manifest, slot string) *ManifestSlot {
	for i := range manifest.Dimensions {
		if manifest.Dimensions[i].Slot == slot {
			return &manifest.Dimensions[i]
		}
	}
	return nil
}

func isBlockingSeverity(severity string) bool {
	return severity == "blocker" || severity == "important"
}

func buildAggregateSummary(kind, decision string, blocking, nonBlocking int) string {
	if decision == "pass" {
		return fmt.Sprintf("%s review passed with %d non-blocking finding(s).", kind, nonBlocking)
	}
	return fmt.Sprintf("%s review found %d blocking and %d non-blocking finding(s).", kind, blocking, nonBlocking)
}

func buildAggregateNextActions(kind, decision string) []NextAction {
	if decision == "pass" {
		if kind == "delta" {
			return []NextAction{{
				Command:     nil,
				Description: "Continue the current step or mark it complete, then update the step's Execution Notes and Review Notes.",
			}}
		}
		return []NextAction{{
			Command:     nil,
			Description: "Move toward final CI and archive readiness for the current candidate.",
		}}
	}
	if kind == "delta" {
		return []NextAction{{
			Command:     nil,
			Description: "Fix the current slice and rerun a delta review once the blocking findings are addressed.",
		}}
	}
	return []NextAction{{
		Command:     nil,
		Description: "Fix the blocking findings before archive and rerun full review if the candidate scope changed materially.",
	}}
}

func (s Service) finalizeStart(result StartResult, rollback func() []CommandError) StartResult {
	if !result.OK || s.AfterStart == nil {
		return result
	}
	if err := s.AfterStart(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return StartResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	return result
}

func (s Service) finalizeSubmit(result SubmitResult, rollback func() []CommandError) SubmitResult {
	if !result.OK || s.AfterSubmit == nil {
		return result
	}
	if err := s.AfterSubmit(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return SubmitResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	return result
}

func (s Service) finalizeAggregate(result AggregateResult, rollback func() []CommandError) AggregateResult {
	if !result.OK || s.AfterAggregate == nil {
		return result
	}
	if err := s.AfterAggregate(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return AggregateResult{
			OK:      false,
			Command: result.Command,
			Summary: "Unable to record the timeline event for the successful command result.",
			Errors:  issues,
		}
	}
	return result
}

func cloneState(state *runstate.State) *runstate.State {
	if state == nil {
		return nil
	}
	cloned := *state
	if state.ActiveReviewRound != nil {
		round := *state.ActiveReviewRound
		cloned.ActiveReviewRound = &round
	}
	if state.Reopen != nil {
		reopen := *state.Reopen
		cloned.Reopen = &reopen
	}
	if state.Land != nil {
		land := *state.Land
		cloned.Land = &land
	}
	return &cloned
}

func restoreStateSnapshot(workdir, planStem string, originalState *runstate.State, statePath string) []CommandError {
	issues := make([]CommandError, 0)
	if originalState != nil {
		if _, err := runstate.SaveState(workdir, planStem, originalState); err != nil {
			issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback local state: %v", err)})
		}
		return issues
	}
	if statePath == "" {
		return issues
	}
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback local state: %v", err)})
	}
	return issues
}

func readFileIfExists(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}

func restoreJSONFileSnapshot(path string, data []byte, existed bool, pathLabel string) []CommandError {
	issues := make([]CommandError, 0, 1)
	if existed {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return append(issues, CommandError{Path: pathLabel, Message: fmt.Sprintf("rollback %s: %v", pathLabel, err)})
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return append(issues, CommandError{Path: pathLabel, Message: fmt.Sprintf("rollback %s: %v", pathLabel, err)})
		}
		return issues
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: pathLabel, Message: fmt.Sprintf("rollback %s: %v", pathLabel, err)})
	}
	return issues
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func strPtr(value string) *string {
	return &value
}
