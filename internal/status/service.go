package status

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/lifecycle"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/stepcloseout"
)

type Service struct {
	Workdir string
}

type Result = contracts.StatusResult
type State = contracts.StatusState
type Facts = contracts.StatusFacts
type Artifacts = contracts.StatusArtifacts
type NextAction = contracts.NextAction
type StatusError = contracts.ErrorDetail

type reviewContext struct {
	RoundID         string
	Kind            string
	Revision        int
	Trigger         string
	ReviewTitle     string
	Aggregated      bool
	InFlight        bool
	Decision        string
	DecisionKnown   bool
	TargetStepIndex int
	UnsafeFallback  bool
}

type evidenceContext struct {
	Publish *evidence.PublishRecord
	CI      *evidence.CIRecord
	Sync    *evidence.SyncRecord
}

type missingStepCloseoutReminder struct {
	MissingTitles   []string
	UnscopedRoundID string
}

type latestStepCloseoutRound = stepcloseout.RoundRecord

func (s Service) Read() Result {
	return s.read(true)
}

func (s Service) ReadUnlocked() Result {
	return s.read(false)
}

func (s Service) read(acquireLock bool) Result {
	currentPlan, err := runstate.LoadCurrentPlan(s.Workdir)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read current worktree state.",
			Errors:  []StatusError{{Path: "state", Message: err.Error()}},
		}
	}

	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		if errors.Is(err, plan.ErrNoCurrentPlan) {
			return idleResult(currentPlan)
		}
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to determine the current plan.",
			Errors:  []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	release := func() {}
	if acquireLock {
		release, err = runstate.AcquireStateMutationLock(s.Workdir, planStem)
		if err != nil {
			return Result{
				OK:      false,
				Command: "status",
				Summary: "Another local state mutation is already in progress.",
				Artifacts: &Artifacts{
					PlanPath: planPath,
				},
				Errors: []StatusError{{Path: "state", Message: err.Error()}},
			}
		}
		defer release()

		planPath, err = plan.DetectCurrentPathLocked(s.Workdir, planStem)
		if err != nil {
			return Result{
				OK:      false,
				Command: "status",
				Summary: "Unable to determine the current plan.",
				Errors:  []StatusError{{Path: "plan", Message: err.Error()}},
			}
		}
	}

	doc, err := plan.LoadFile(planPath)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read the current plan.",
			Artifacts: &Artifacts{
				PlanPath: planPath,
			},
			Errors: []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to read local harness state.",
			Artifacts: &Artifacts{
				PlanPath:       planPath,
				LocalStatePath: statePath,
			},
			Errors: []StatusError{{Path: "state", Message: err.Error()}},
		}
	}

	result := Result{
		OK:      true,
		Command: "status",
		Artifacts: &Artifacts{
			PlanPath:       planPath,
			LocalStatePath: statePath,
		},
	}
	supplementsPath := plan.SupplementsDirForPlanPath(planPath)
	if info, err := os.Stat(supplementsPath); err == nil && info.IsDir() {
		result.Artifacts.SupplementsPath = supplementsPath
	} else if err != nil && !os.IsNotExist(err) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unable to inspect supplements path %s: %v", supplementsPath, err))
	} else if err == nil && !info.IsDir() {
		result.Warnings = append(result.Warnings, fmt.Sprintf("supplements path is not a directory: %s", supplementsPath))
	}

	reviewCtx, reviewWarnings := loadReviewContext(s.Workdir, planStem, doc, state)
	result.Warnings = append(result.Warnings, reviewWarnings...)
	if reviewCtx != nil && isStructuralReviewTrigger(reviewCtx.Trigger) && strings.TrimSpace(reviewCtx.RoundID) != "" {
		result.Artifacts.ReviewRoundID = reviewCtx.RoundID
	}

	facts := &Facts{}
	if state != nil && state.Revision > 0 {
		facts.Revision = state.Revision
	}
	if reopenMode := effectiveReopenMode(doc, state); reopenMode != "" {
		facts.ReopenMode = reopenMode
	}
	if reviewCtx != nil && isStructuralReviewTrigger(reviewCtx.Trigger) && !reviewCtx.UnsafeFallback {
		facts.ReviewKind = reviewCtx.Kind
		facts.ReviewTrigger = reviewCtx.Trigger
		facts.ReviewTitle = reviewCtx.ReviewTitle
		switch {
		case reviewCtx.InFlight:
			facts.ReviewStatus = "in_progress"
		case reviewCtx.DecisionKnown:
			facts.ReviewStatus = reviewCtx.Decision
		case reviewCtx.Aggregated:
			facts.ReviewStatus = "unknown"
		}
	}

	var blockers []StatusError
	switch {
	case landInProgress(state):
		result.State.CurrentNode = "land"
		if state != nil && state.Land != nil {
			facts.LandPRURL = state.Land.PRURL
			facts.LandCommit = state.Land.Commit
		}
	case doc.DerivedPlanStatus() == "active" && !doc.ExecutionStarted(state):
		result.State.CurrentNode = "plan"
	case doc.DerivedPlanStatus() == "active":
		stepIdx, stepNode := resolveStepNode(doc, reviewCtx)
		if stepNode != "" {
			result.State.CurrentNode = stepNode
			facts.CurrentStep = doc.Steps[stepIdx].Title
		} else {
			result.State.CurrentNode, blockers = resolveFinalizeNode(s.Workdir, planStem, doc, state, reviewCtx)
			if len(blockers) > 0 {
				facts.ArchiveBlockerCount = len(blockers)
			}
		}
	case doc.DerivedPlanStatus() == "archived":
		evidenceCtx, evidenceWarnings := loadEvidenceContext(s.Workdir, planStem, runstate.CurrentRevision(state))
		result.Warnings = append(result.Warnings, evidenceWarnings...)
		applyEvidenceFacts(facts, result.Artifacts, evidenceCtx)
		if archivedCandidateReadyForMerge(evidenceCtx) {
			result.State.CurrentNode = "execution/finalize/await_merge"
		} else {
			result.State.CurrentNode = "execution/finalize/publish"
		}
	default:
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to classify the current plan path.",
			Artifacts: &Artifacts{
				PlanPath:       planPath,
				LocalStatePath: statePath,
			},
			Errors: []StatusError{{Path: "plan", Message: fmt.Sprintf("unsupported plan path kind for %s", planPath)}},
		}
	}

	result.Blockers = blockers
	if strings.HasPrefix(result.State.CurrentNode, "execution/finalize/") && reviewCtx != nil && reviewCtx.Trigger == "step_closeout" {
		clearStepCloseoutReviewMetadata(facts, result.Artifacts)
	}
	missingStepReminder, reminderWarnings := loadMissingStepCloseoutReminder(s.Workdir, planStem, doc, reviewCtx, result.State.CurrentNode)
	result.Warnings = append(result.Warnings, reminderWarnings...)
	result.Summary = buildSummary(result.State.CurrentNode, facts, reviewCtx, blockers, missingStepReminder, currentPlan)
	result.NextAction = buildNextActions(result.State.CurrentNode, facts, reviewCtx, blockers)
	if missingStepReminder != nil {
		result.Warnings = append(result.Warnings, buildMissingStepCloseoutWarnings(result.State.CurrentNode, missingStepReminder)...)
		result.NextAction = prependMissingStepCloseoutActions(result.State.CurrentNode, result.NextAction, facts, reviewCtx, missingStepReminder)
	}
	if doc.UsesLightweightProfile() &&
		(result.State.CurrentNode == "execution/finalize/publish" || result.State.CurrentNode == "execution/finalize/await_merge") &&
		(missingStepReminder == nil || !missingStepReminder.hasDebt()) {
		action := NextAction{
			Command:     nil,
			Description: "Leave or verify the agreed repo-visible breadcrumb, such as a PR body note explaining why the lightweight path was used, before waiting for merge approval.",
		}
		result.NextAction = append([]NextAction{action}, result.NextAction...)
		if !strings.Contains(result.Summary, "lightweight path") {
			result.Summary += " The lightweight path still needs its repo-visible breadcrumb."
		}
	}
	if factsEmpty(facts) {
		result.Facts = nil
	} else {
		result.Facts = facts
	}

	if result.Artifacts != nil && result.Artifacts.PlanPath == "" && result.Artifacts.LocalStatePath == "" &&
		result.Artifacts.ReviewRoundID == "" && result.Artifacts.CIRecordID == "" &&
		result.Artifacts.PublishRecordID == "" && result.Artifacts.SyncRecordID == "" &&
		result.Artifacts.LastLandedPlanPath == "" && result.Artifacts.LastLandedAt == "" {
		result.Artifacts = nil
	}

	return result
}

func clearStepCloseoutReviewMetadata(facts *Facts, artifacts *Artifacts) {
	if facts != nil {
		facts.ReviewKind = ""
		facts.ReviewTrigger = ""
		facts.ReviewTitle = ""
		facts.ReviewStatus = ""
	}
	if artifacts != nil {
		artifacts.ReviewRoundID = ""
	}
}

func resolveStepNode(doc *plan.Document, reviewCtx *reviewContext) (int, string) {
	if reviewCtx != nil && reviewCtx.TargetStepIndex >= 0 && reviewCtx.UnsafeFallback {
		return reviewCtx.TargetStepIndex, stepNode(reviewCtx.TargetStepIndex, "implement")
	}
	if reviewCtx != nil && reviewCtx.Trigger == "step_closeout" &&
		(reviewCtx.InFlight || !reviewCtx.DecisionKnown || reviewCtx.Decision != "pass") &&
		reviewCtx.TargetStepIndex >= 0 {
		if reviewCtx.InFlight {
			return reviewCtx.TargetStepIndex, stepNode(reviewCtx.TargetStepIndex, "review")
		}
		return reviewCtx.TargetStepIndex, stepNode(reviewCtx.TargetStepIndex, "implement")
	}

	currentStepIndex := currentStepIndex(doc)
	if currentStepIndex < 0 {
		return -1, ""
	}
	if reviewCtx != nil && reviewCtx.Trigger == "step_closeout" && reviewCtx.InFlight && reviewCtx.TargetStepIndex == currentStepIndex {
		return currentStepIndex, stepNode(currentStepIndex, "review")
	}
	return currentStepIndex, stepNode(currentStepIndex, "implement")
}

func resolveFinalizeNode(workdir, planStem string, doc *plan.Document, state *runstate.State, reviewCtx *reviewContext) (string, []StatusError) {
	reopenedNewStepPending := state != nil &&
		state.Reopen != nil &&
		state.Reopen.Mode == "new-step" &&
		state.Reopen.BaseStepCount > 0 &&
		len(doc.Steps) <= state.Reopen.BaseStepCount &&
		doc.CurrentStep() == nil &&
		doc.AllStepsCompleted()

	if reviewCtx != nil && reviewCtx.Trigger == "pre_archive" && reviewCtx.InFlight {
		return "execution/finalize/review", nil
	}
	if reopenedNewStepPending {
		return "execution/finalize/fix", nil
	}
	if state != nil && state.Reopen != nil && state.Reopen.Mode == "finalize-fix" {
		if !finalizeReviewSatisfied(reviewCtx, runstate.CurrentRevision(state)) {
			return "execution/finalize/fix", nil
		}
	}
	if reviewCtx != nil && reviewCtx.Trigger == "pre_archive" && reviewCtx.Aggregated &&
		(!reviewCtx.DecisionKnown || reviewCtx.Decision != "pass") {
		return "execution/finalize/fix", nil
	}
	if finalizeReviewSatisfied(reviewCtx, runstate.CurrentRevision(state)) {
		return "execution/finalize/archive", commandErrorsToStatusErrors(lifecycle.EvaluateArchiveReadiness(workdir, planStem, doc, state))
	}
	return "execution/finalize/review", nil
}

func finalizeReviewSatisfied(reviewCtx *reviewContext, revision int) bool {
	if reviewCtx == nil || reviewCtx.Trigger != "pre_archive" || !reviewCtx.Aggregated {
		return false
	}
	if !reviewCtx.DecisionKnown || reviewCtx.Decision != "pass" {
		return false
	}
	if revision <= 1 && reviewCtx.Kind != "full" {
		return false
	}
	return true
}

func effectiveReopenMode(doc *plan.Document, state *runstate.State) string {
	if state == nil || state.Reopen == nil {
		return ""
	}
	if state.Reopen.Mode != "new-step" {
		return state.Reopen.Mode
	}
	if state.Reopen.BaseStepCount > 0 && doc != nil && len(doc.Steps) > state.Reopen.BaseStepCount {
		return ""
	}
	return state.Reopen.Mode
}

func loadReviewContext(workdir, planStem string, doc *plan.Document, state *runstate.State) (*reviewContext, []string) {
	if state == nil || state.ActiveReviewRound == nil {
		return nil, nil
	}

	round := state.ActiveReviewRound
	ctx := &reviewContext{
		RoundID:         round.RoundID,
		Kind:            round.Kind,
		Aggregated:      round.Aggregated,
		InFlight:        !round.Aggregated,
		TargetStepIndex: -1,
	}
	warnings := make([]string, 0)

	revision, revisionKnown, err := runstate.EffectiveReviewRevision(workdir, planStem, round)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review revision for %s; status may be conservative.", round.RoundID))
	} else if revisionKnown {
		ctx.Revision = revision
	}

	stepIndex, stepKnown, err := runstate.EffectiveReviewStep(workdir, planStem, round)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review step binding for %s; status may be conservative.", round.RoundID))
	}

	if reviewTitle, known, err := runstate.EffectiveReviewTitle(workdir, planStem, round); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review title for %s; status may be conservative.", round.RoundID))
	} else if known {
		ctx.ReviewTitle = reviewTitle
	}

	if round.Aggregated {
		decision, known, err := runstate.EffectiveReviewDecision(workdir, planStem, round)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Unable to read the aggregate artifact for %s; review status may be stale.", round.RoundID))
		} else {
			ctx.Decision = decision
			ctx.DecisionKnown = known
		}
		if !ctx.DecisionKnown {
			warnings = append(warnings, fmt.Sprintf("The latest aggregated review outcome for %s could not be recovered; status is staying conservative.", round.RoundID))
		}
	}

	if revisionKnown {
		if stepKnown {
			ctx.Trigger = "step_closeout"
			ctx.TargetStepIndex = stepIndex - 1
			if ctx.TargetStepIndex >= 0 && ctx.TargetStepIndex < len(doc.Steps) && strings.TrimSpace(ctx.ReviewTitle) == "" {
				ctx.ReviewTitle = doc.Steps[ctx.TargetStepIndex].Title
			}
		} else {
			ctx.Trigger = "pre_archive"
			if strings.TrimSpace(ctx.ReviewTitle) == "" {
				ctx.ReviewTitle = defaultFinalizeReviewTitle(ctx.Kind)
			}
		}
	}

	return ctx, warnings
}

func loadEvidenceContext(workdir, planStem string, revision int) (*evidenceContext, []string) {
	ctx := &evidenceContext{}
	warnings := make([]string, 0)

	if publish, err := evidence.LoadLatestPublish(workdir, planStem, revision); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read publish evidence: %v", err))
	} else {
		ctx.Publish = publish
	}
	if ci, err := evidence.LoadLatestCI(workdir, planStem, revision); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read CI evidence: %v", err))
	} else {
		ctx.CI = ci
	}
	if sync, err := evidence.LoadLatestSync(workdir, planStem, revision); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read sync evidence: %v", err))
	} else {
		ctx.Sync = sync
	}

	return ctx, warnings
}

func loadMissingStepCloseoutReminder(workdir, planStem string, doc *plan.Document, reviewCtx *reviewContext, currentNode string) (*missingStepCloseoutReminder, []string) {
	reminder := stepcloseout.LoadReminder(workdir, planStem, doc, currentNode, activeReviewForStepCloseoutScan(reviewCtx))
	if len(reminder.MissingTitles) == 0 && reminder.UnscopedRoundID == "" {
		return nil, reminder.Warnings
	}
	return &missingStepCloseoutReminder{
		MissingTitles:   reminder.MissingTitles,
		UnscopedRoundID: reminder.UnscopedRoundID,
	}, reminder.Warnings
}

func loadSatisfiedStepCloseoutTargets(workdir, planStem string, doc *plan.Document, reviewCtx *reviewContext) (map[string]bool, []string) {
	latestByTarget, warnings := loadLatestStepCloseoutTargets(workdir, planStem, doc, reviewCtx)
	satisfied := map[string]bool{}
	for target, record := range latestByTarget {
		if record.Decision == "pass" {
			satisfied[target] = true
		}
	}

	return satisfied, warnings
}

func loadLatestStepCloseoutTargets(workdir, planStem string, doc *plan.Document, reviewCtx *reviewContext) (map[string]latestStepCloseoutRound, []string) {
	scan := stepcloseout.LoadLatestScan(workdir, planStem, doc, activeReviewForStepCloseoutScan(reviewCtx))
	latestByTarget := map[string]latestStepCloseoutRound{}
	for index, record := range scan.LatestByStepIndex {
		if index >= 0 && index < len(doc.Steps) {
			latestByTarget[normalizeReviewTitle(doc.Steps[index].Title)] = record
		}
	}
	return latestByTarget, scan.Warnings
}

func applyEvidenceFacts(facts *Facts, artifacts *Artifacts, evidenceCtx *evidenceContext) {
	if evidenceCtx == nil {
		return
	}
	if evidenceCtx.Publish != nil {
		facts.PublishStatus = evidenceCtx.Publish.Status
		facts.PRURL = evidenceCtx.Publish.PRURL
		if artifacts != nil {
			artifacts.PublishRecordID = evidenceCtx.Publish.RecordID
		}
	}
	if evidenceCtx.CI != nil {
		facts.CIStatus = evidenceCtx.CI.Status
		if artifacts != nil {
			artifacts.CIRecordID = evidenceCtx.CI.RecordID
		}
	}
	if evidenceCtx.Sync != nil {
		facts.SyncStatus = evidenceCtx.Sync.Status
		if artifacts != nil {
			artifacts.SyncRecordID = evidenceCtx.Sync.RecordID
		}
	}
}

func archivedCandidateReadyForMerge(evidenceCtx *evidenceContext) bool {
	if evidenceCtx == nil || evidenceCtx.Publish == nil || evidenceCtx.CI == nil || evidenceCtx.Sync == nil {
		return false
	}
	if evidenceCtx.Publish.Status != "recorded" || strings.TrimSpace(evidenceCtx.Publish.PRURL) == "" {
		return false
	}
	if evidenceCtx.CI.Status != "success" && evidenceCtx.CI.Status != "not_applied" {
		return false
	}
	if evidenceCtx.Sync.Status != "fresh" && evidenceCtx.Sync.Status != "not_applied" {
		return false
	}
	return true
}

func buildSummary(node string, facts *Facts, reviewCtx *reviewContext, blockers []StatusError, reminder *missingStepCloseoutReminder, currentPlan *runstate.CurrentPlan) string {
	if strings.HasPrefix(node, "execution/finalize/") && reminder != nil && reminder.hasDebt() && !pendingReopenedNewStep(node, facts) {
		return buildMissingStepCloseoutSummary(node, reviewCtx, reminder)
	}

	switch node {
	case "idle":
		if currentPlan != nil && strings.TrimSpace(currentPlan.LastLandedPlanPath) != "" {
			return "No current plan is active in this worktree. The most recent landed candidate is recorded for handoff context."
		}
		return "No current plan is active in this worktree."
	case "plan":
		return "Current plan exists, but execution has not started yet."
	case "execution/finalize/review":
		if reviewCtx != nil && reviewCtx.InFlight {
			return "Plan is in finalize review and waiting for the active review round to be aggregated."
		}
		return "Plan has finished its tracked steps and needs finalize review before archive."
	case "execution/finalize/fix":
		if facts != nil && facts.ReopenMode == "new-step" && facts.CurrentStep == "" {
			return "Plan was reopened for new-scope work and needs a new unfinished step before implementation can continue."
		}
		if facts != nil && facts.ReopenMode == "finalize-fix" {
			return "Plan was reopened into finalize-scope repair and needs follow-up fixes plus a fresh finalize review before archive."
		}
		if facts != nil && facts.ReviewStatus == "unknown" && reviewCtx != nil {
			return fmt.Sprintf("Plan needs finalize follow-up because the latest aggregated review (%s) could not be recovered from local state.", reviewCtx.RoundID)
		}
		if reviewCtx != nil && facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" {
			return fmt.Sprintf("Plan needs finalize-scope repair because the latest finalize review (%s) requested changes.", reviewCtx.RoundID)
		}
		return "Plan needs finalize-scope repair before archive."
	case "execution/finalize/archive":
		if len(blockers) > 0 {
			return fmt.Sprintf("Plan has a clean finalize review and is in archive closeout, but %d archive blocker(s) still need to be fixed before `harness archive`.", len(blockers))
		}
		return "Plan has a clean finalize review and is ready to archive."
	case "execution/finalize/publish":
		return "Plan is archived, but external publish, CI, or sync evidence is still keeping it from merge-ready handoff."
	case "execution/finalize/await_merge":
		return "Plan is archived, published, and merge-ready; waiting for human merge approval."
	case "land":
		return "Merge has been recorded and required post-merge bookkeeping is still in progress."
	}

	if strings.HasSuffix(node, "/review") {
		return fmt.Sprintf("Plan is reviewing %s.", facts.CurrentStep)
	}
	if strings.HasSuffix(node, "/implement") {
		if facts != nil && facts.ReviewStatus == "unknown" && reviewCtx != nil {
			return fmt.Sprintf("Plan is executing %s, but the latest aggregated review (%s) could not be recovered and should be rerun conservatively.", facts.CurrentStep, reviewCtx.RoundID)
		}
		if facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" && facts.ReviewStatus != "in_progress" && reviewCtx != nil {
			return fmt.Sprintf("Plan is executing %s and the latest aggregated review (%s) requested changes.", facts.CurrentStep, reviewCtx.RoundID)
		}
		if facts != nil && facts.ReviewStatus == "pass" {
			return fmt.Sprintf("Plan is executing %s after a clean review and can continue or be marked done.", facts.CurrentStep)
		}
		return fmt.Sprintf("Plan is executing %s.", facts.CurrentStep)
	}

	return fmt.Sprintf("Plan is at %s.", node)
}

func pendingReopenedNewStep(node string, facts *Facts) bool {
	return node == "execution/finalize/fix" &&
		facts != nil &&
		facts.ReopenMode == "new-step" &&
		strings.TrimSpace(facts.CurrentStep) == ""
}

func (r *missingStepCloseoutReminder) hasDebt() bool {
	return r != nil && len(r.MissingTitles) > 0
}

func (r *missingStepCloseoutReminder) hasWarning() bool {
	return r != nil && (len(r.MissingTitles) > 0 || strings.TrimSpace(r.UnscopedRoundID) != "")
}

func buildMissingStepCloseoutSummary(node string, reviewCtx *reviewContext, reminder *missingStepCloseoutReminder) string {
	if len(reminder.MissingTitles) == 0 {
		roundID := strings.TrimSpace(reminder.UnscopedRoundID)
		switch node {
		case "execution/finalize/review":
			if reviewCtx != nil && reviewCtx.InFlight {
				return fmt.Sprintf("Finalize review is in flight, but unreadable historical review metadata (%s) could still hide earlier step-closeout debt; inspect or rerun the relevant closeout before treating the candidate as finalize-ready.", roundID)
			}
			return fmt.Sprintf("Unreadable historical review metadata (%s) could still hide earlier step-closeout debt; inspect or rerun the relevant closeout before relying on finalize progression.", roundID)
		case "execution/finalize/fix":
			return fmt.Sprintf("Unreadable historical review metadata (%s) could still hide earlier step-closeout debt; inspect or rerun the relevant closeout before treating finalize repair as complete.", roundID)
		case "execution/finalize/archive":
			return fmt.Sprintf("Plan has a clean finalize review, but unreadable historical review metadata (%s) could still hide earlier step-closeout debt; inspect or rerun the relevant closeout before archive.", roundID)
		case "execution/finalize/publish":
			return fmt.Sprintf("Plan is archived, but unreadable historical review metadata (%s) could still hide earlier step-closeout debt; reopen the candidate and resolve the relevant closeout before merge-ready handoff.", roundID)
		case "execution/finalize/await_merge":
			return fmt.Sprintf("Plan is archived, but unreadable historical review metadata (%s) could still hide earlier step-closeout debt; reopen the candidate and resolve the relevant closeout before treating it as merge-ready.", roundID)
		default:
			return ""
		}
	}

	earliestTitle := reminder.MissingTitles[0]

	switch node {
	case "execution/finalize/review":
		if reviewCtx != nil && reviewCtx.InFlight {
			return fmt.Sprintf("Finalize review is in flight, but earlier completed steps still need review-complete closeout; resolve %s before treating the candidate as finalize-ready.", earliestTitle)
		}
		return fmt.Sprintf("Earlier completed steps still need review-complete closeout; resolve %s before relying on finalize progression.", earliestTitle)
	case "execution/finalize/fix":
		return fmt.Sprintf("Earlier completed steps still need review-complete closeout; resolve %s before treating finalize repair as complete.", earliestTitle)
	case "execution/finalize/archive":
		return fmt.Sprintf("Plan has a clean finalize review, but earlier completed steps still need review-complete closeout; resolve %s before archive.", earliestTitle)
	case "execution/finalize/publish":
		return fmt.Sprintf("Plan is archived, but earlier completed steps still need review-complete closeout; reopen the candidate and resolve %s before merge-ready handoff.", earliestTitle)
	case "execution/finalize/await_merge":
		return fmt.Sprintf("Plan is archived, but earlier completed steps still need review-complete closeout; reopen the candidate and resolve %s before treating it as merge-ready.", earliestTitle)
	default:
		return ""
	}
}

func buildNextActions(node string, facts *Facts, reviewCtx *reviewContext, blockers []StatusError) []NextAction {
	switch node {
	case "idle":
		return []NextAction{
			{Command: nil, Description: "Start discovery or create a new tracked plan when the next slice is ready."},
		}
	case "plan":
		return []NextAction{
			{Command: strPtr("harness execute start"), Description: "Start execution once the plan is approved for implementation."},
			{Command: nil, Description: "If scope changed before implementation begins, update the tracked plan first."},
		}
	case "execution/finalize/review":
		if reviewCtx != nil && reviewCtx.InFlight {
			return []NextAction{
				{Command: aggregateCommand(reviewCtx.RoundID), Description: "Aggregate the active finalize review round once the expected reviewer submissions are ready."},
			}
		}
		return []NextAction{
			{Command: strPtr("harness review start --spec <path>"), Description: "Start a fresh finalize review for the full candidate before archive."},
		}
	case "execution/finalize/fix":
		if facts != nil && facts.ReopenMode == "new-step" && facts.CurrentStep == "" {
			return []NextAction{
				{Command: nil, Description: "Add a new unfinished step for the reopened scope before continuing implementation; do not fold the new work into already completed steps."},
			}
		}
		description := "Repair the finalize-scope issues, refresh durable summaries as needed, rerun focused validation, and start a fresh finalize review before archive."
		if reviewCtx != nil && facts != nil && facts.ReviewStatus == "unknown" {
			description = fmt.Sprintf("Recover or rerun %s before continuing. The latest aggregated finalize review outcome could not be recovered from local state, so archive-sensitive guidance is intentionally blocked.", reviewCtx.RoundID)
		}
		if reviewCtx != nil && facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" && facts.ReviewStatus != "unknown" {
			description = fmt.Sprintf("Address the findings from %s, refresh durable summaries as needed, rerun focused validation, and start a fresh finalize review once the repair is ready.", reviewCtx.RoundID)
		}
		return []NextAction{
			{Command: nil, Description: description},
			{Command: strPtr("harness review start --spec <path>"), Description: "Start a fresh finalize review once the repaired candidate is ready."},
		}
	case "execution/finalize/archive":
		if len(blockers) > 0 {
			return []NextAction{
				{Command: nil, Description: "Fix the archive blockers surfaced below, refresh the durable summaries, and rerun `harness status` before archiving."},
			}
		}
		return []NextAction{
			{Command: nil, Description: "Archive-ready closeout is complete; archive the plan and then commit and push the tracked move."},
			{Command: strPtr("harness archive"), Description: "Archive the current plan now that the closeout notes and follow-up links are ready."},
		}
	case "execution/finalize/publish":
		return buildPublishNextActions(facts)
	case "execution/finalize/await_merge":
		actions := []NextAction{
			{Command: nil, Description: "Wait for explicit human approval before merging the PR."},
		}
		if facts != nil && strings.TrimSpace(facts.PRURL) != "" {
			actions = append(actions, NextAction{
				Command:     strPtr(fmt.Sprintf("harness land --pr %s [--commit <sha>]", facts.PRURL)),
				Description: "After the PR is merged outside harness and the worktree is synced, record merge confirmation and enter required post-merge bookkeeping.",
			})
		}
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "If new feedback or remote changes invalidate the archived candidate, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step.",
		})
		return actions
	case "land":
		return []NextAction{
			{Command: nil, Description: "Finish required post-merge bookkeeping and cleanup while the plan is in land: add the final PR comment when the permanent record still needs one, close resolved linked issues or add follow-up references for unresolved ones, and complete the remaining closeout tasks."},
			{Command: strPtr("harness land complete"), Description: "Record required post-merge bookkeeping completion only after the required PR and issue bookkeeping is done, then restore the worktree to idle."},
		}
	}

	if strings.HasSuffix(node, "/review") {
		return []NextAction{
			{Command: aggregateCommand(reviewCtx.RoundID), Description: "Aggregate the active review round once the expected reviewer submissions are ready."},
		}
	}
	if strings.HasSuffix(node, "/implement") {
		if reviewCtx != nil && reviewCtx.InFlight {
			return []NextAction{
				{Command: aggregateCommand(reviewCtx.RoundID), Description: "Aggregate the active review round once the expected reviewer submissions are ready."},
			}
		}
		if facts != nil && facts.ReviewStatus == "unknown" {
			description := "Recover the latest aggregated review result or rerun review conservatively before advancing this step."
			if reviewCtx != nil && strings.TrimSpace(reviewCtx.RoundID) != "" {
				description = fmt.Sprintf("Recover or rerun %s before continuing. The latest aggregated review outcome could not be recovered from local state, so advancement is intentionally blocked.", reviewCtx.RoundID)
			}
			return []NextAction{
				{Command: nil, Description: description},
				{Command: strPtr("harness review start --spec <path>"), Description: "Start a fresh review round once the repair is ready."},
			}
		}
		if facts != nil && facts.ReviewStatus != "" && facts.ReviewStatus != "pass" && facts.ReviewStatus != "in_progress" {
			description := "Address the latest review findings, update the step-local notes, rerun focused validation, and start a fresh review round once the slice is ready."
			if reviewCtx != nil && strings.TrimSpace(reviewCtx.RoundID) != "" {
				description = fmt.Sprintf("Address the findings from %s, update step-local notes, rerun focused validation, and start a fresh review round once the slice is ready.", reviewCtx.RoundID)
			}
			return []NextAction{
				{Command: nil, Description: description},
				{Command: strPtr("harness review start --spec <path>"), Description: "Start a fresh delta or full review after the fixes are in place."},
			}
		}
		if facts != nil && facts.ReviewStatus == "pass" {
			return []NextAction{
				{Command: nil, Description: "Continue the current step or mark it done, then keep the step's Execution Notes and Review Notes up to date."},
			}
		}
		return []NextAction{
			{Command: nil, Description: "Continue the current step and keep step-local Execution Notes and Review Notes up to date."},
		}
	}

	return nil
}

func buildMissingStepCloseoutWarnings(node string, reminder *missingStepCloseoutReminder) []string {
	if reminder == nil || !reminder.hasWarning() {
		return nil
	}

	unscopedWarning := ""
	if strings.TrimSpace(reminder.UnscopedRoundID) != "" {
		unscopedWarning = fmt.Sprintf("Historical review round %s is invalid and cannot be mapped to a tracked step; it is being ignored and you do not need to do anything.", reminder.UnscopedRoundID)
	}

	if len(reminder.MissingTitles) == 0 {
		return []string{unscopedWarning}
	}

	if len(reminder.MissingTitles) == 1 {
		title := reminder.MissingTitles[0]
		if strings.HasPrefix(node, "execution/finalize/") {
			warnings := []string{fmt.Sprintf("Finalize progression is continuing while %s is marked done but still lacks review-complete closeout.", title)}
			if unscopedWarning != "" {
				warnings = append(warnings, unscopedWarning)
			}
			return warnings
		}
		warnings := []string{fmt.Sprintf("%s is marked done, but no clean step-closeout review was found and Review Notes do not record NO_STEP_REVIEW_NEEDED.", title)}
		if unscopedWarning != "" {
			warnings = append(warnings, unscopedWarning)
		}
		return warnings
	}

	context := "Later-step progression is continuing"
	if strings.HasPrefix(node, "execution/finalize/") {
		context = "Finalize progression is continuing"
	}
	warnings := []string{fmt.Sprintf("%s while %d completed steps still lack review-complete closeout: %s.", context, len(reminder.MissingTitles), strings.Join(reminder.MissingTitles, "; "))}
	for _, title := range reminder.MissingTitles {
		warnings = append(warnings, fmt.Sprintf("%s is marked done, but no clean step-closeout review was found and Review Notes do not record NO_STEP_REVIEW_NEEDED.", title))
	}
	if unscopedWarning != "" {
		warnings = append(warnings, unscopedWarning)
	}
	return warnings
}

func prependMissingStepCloseoutActions(node string, actions []NextAction, facts *Facts, reviewCtx *reviewContext, reminder *missingStepCloseoutReminder) []NextAction {
	if reminder == nil || !reminder.hasDebt() {
		return actions
	}
	if pendingReopenedNewStep(node, facts) {
		return actions
	}

	earliestTitle := ""
	if len(reminder.MissingTitles) > 0 {
		earliestTitle = reminder.MissingTitles[0]
	}
	inFlight := reviewRoundAlreadyInFlight(node, reviewCtx)
	if earliestTitle == "" {
		description := fmt.Sprintf("Historical review round %s could not be mapped back to a tracked step; inspect or repair the local review artifacts, then rerun the relevant step-closeout review conservatively before relying on further progression.", reminder.UnscopedRoundID)
		if inFlight {
			description = fmt.Sprintf("Historical review round %s could not be mapped back to a tracked step; aggregate the active review round first, then inspect or repair the local review artifacts and rerun the relevant step-closeout review conservatively before relying on further progression.", reminder.UnscopedRoundID)
		}
		prefixed := []NextAction{{Command: nil, Description: description}}
		if node == "execution/finalize/publish" || node == "execution/finalize/await_merge" {
			prefixed = append(prefixed, NextAction{
				Command:     strPtr("harness reopen --mode finalize-fix"),
				Description: fmt.Sprintf("Reopen the archived candidate before repairing the ambiguous historical closeout evidence from %s.", reminder.UnscopedRoundID),
			})
			return append(prefixed, actions...)
		}
		if node == "execution/finalize/archive" && containsNextActionCommand(actions, "harness archive") {
			return prefixed
		}
		return append(prefixed, actions...)
	}
	if node == "execution/finalize/publish" || node == "execution/finalize/await_merge" {
		prefixed := []NextAction{
			{
				Command:     nil,
				Description: fmt.Sprintf("Earlier completed steps still need review-complete closeout before this archived candidate can be treated as merge-ready; reopen first, then resolve %s.", earliestTitle),
			},
			{
				Command:     strPtr("harness reopen --mode finalize-fix"),
				Description: fmt.Sprintf("Reopen the archived candidate before repairing missing closeout for %s or any other earlier completed step.", earliestTitle),
			},
		}
		return append(prefixed, actions...)
	}

	description := fmt.Sprintf("%s is already marked done but still needs review-complete closeout; resolve it first by starting step-closeout review or recording NO_STEP_REVIEW_NEEDED: <reason> in Review Notes.", earliestTitle)
	if inFlight {
		description = fmt.Sprintf("%s is already marked done but still needs review-complete closeout; aggregate the active review round first, then resolve the closeout gap by recording NO_STEP_REVIEW_NEEDED: <reason> in Review Notes or starting step-closeout review once no other review round is active.", earliestTitle)
	}
	if strings.HasPrefix(node, "execution/finalize/") {
		description = fmt.Sprintf("Earlier completed steps still need review-complete closeout before relying on finalize progression; resolve %s first by starting step-closeout review or recording NO_STEP_REVIEW_NEEDED: <reason> in Review Notes.", earliestTitle)
		if inFlight {
			description = fmt.Sprintf("Earlier completed steps still need review-complete closeout before relying on finalize progression; aggregate the active review round first, then resolve %s by recording NO_STEP_REVIEW_NEEDED: <reason> in Review Notes or starting step-closeout review once no other review round is active.", earliestTitle)
		}
	}

	prefixed := []NextAction{{Command: nil, Description: description}}
	if !inFlight {
		prefixed = append(prefixed, NextAction{
			Command:     strPtr("harness review start --spec <path>"),
			Description: fmt.Sprintf("Start a fresh step-closeout review for %s once the closeout slice is ready.", earliestTitle),
		})
	}
	if strings.HasPrefix(node, "execution/finalize/") {
		if node == "execution/finalize/archive" && containsNextActionCommand(actions, "harness archive") {
			return prefixed
		}
		return append(prefixed, actions...)
	}
	return append(prefixed, actions...)
}

func reviewRoundAlreadyInFlight(node string, reviewCtx *reviewContext) bool {
	return reviewCtx != nil && reviewCtx.InFlight
}

func containsNextActionCommand(actions []NextAction, command string) bool {
	for _, action := range actions {
		if action.Command != nil && *action.Command == command {
			return true
		}
	}
	return false
}

func buildPublishNextActions(facts *Facts) []NextAction {
	actions := []NextAction{
		{
			Command:     nil,
			Description: "Commit and push the tracked plan change created by archiving before treating the candidate as merge-ready.",
		},
	}

	switch {
	case facts == nil || facts.PublishStatus == "":
		actions = append(actions,
			NextAction{Command: nil, Description: "Open or update the PR for the archived candidate, then record publish evidence with the PR URL."},
			NextAction{Command: strPtr("harness evidence submit --kind publish --input <json>"), Description: "Record publish evidence for the archived candidate once the PR or handoff record exists."},
		)
	case facts.PublishStatus == "not_applied":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Publish was marked not_applied, but v0.2 land still requires a PR URL; record publish evidence with a PR URL or reopen if the workflow changed.",
		})
	}

	switch {
	case facts == nil || facts.CIStatus == "":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind ci --input <json>"),
			Description: "Record CI evidence once the relevant post-archive check result is known.",
		})
	case facts.CIStatus == "pending":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Wait for the relevant post-archive CI to finish, then record the updated result if it changes.",
		})
	case facts.CIStatus == "failed":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Fix the CI failures or record an explicit not_applied decision before treating the candidate as merge-ready.",
		})
	}

	switch {
	case facts == nil || facts.SyncStatus == "":
		actions = append(actions, NextAction{
			Command:     strPtr("harness evidence submit --kind sync --input <json>"),
			Description: "Record sync evidence after checking freshness and conflict status against the merge base.",
		})
	case facts.SyncStatus == "stale":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Refresh the branch against the merge base, then record a fresh sync result before merge approval.",
		})
	case facts.SyncStatus == "conflicted":
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "Resolve merge conflicts or otherwise repair the branch, then record a fresh sync result before merge approval.",
		})
	}

	actions = append(actions, NextAction{
		Command:     nil,
		Description: "If the archived candidate is invalidated, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step.",
	})

	return actions
}

func idleResult(currentPlan *runstate.CurrentPlan) Result {
	result := Result{
		OK:      true,
		Command: "status",
		State: State{
			CurrentNode: "idle",
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Start discovery or create a new tracked plan when the next slice is ready."},
		},
	}
	if currentPlan != nil && strings.TrimSpace(currentPlan.LastLandedPlanPath) != "" {
		result.Summary = "No current plan is active in this worktree. The most recent landed candidate is recorded for handoff context."
		result.Artifacts = &Artifacts{
			LastLandedPlanPath: currentPlan.LastLandedPlanPath,
			LastLandedAt:       currentPlan.LastLandedAt,
		}
		return result
	}
	result.Summary = "No current plan is active in this worktree."
	return result
}

func currentStepIndex(doc *plan.Document) int {
	currentStep := doc.CurrentStep()
	if currentStep == nil {
		return -1
	}
	for index, step := range doc.Steps {
		if step.Title == currentStep.Title {
			return index
		}
	}
	return -1
}

func resolveReviewTitleStep(doc *plan.Document, reviewTitle string) (int, bool) {
	reviewTitle = normalizeReviewTitle(reviewTitle)
	if reviewTitle == "" {
		return -1, false
	}
	for index, step := range doc.Steps {
		if normalizeReviewTitle(step.Title) == reviewTitle {
			return index, true
		}
	}
	return -1, false
}

func normalizeReviewTitle(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func defaultFinalizeReviewTitle(kind string) string {
	if kind == "full" {
		return "Full branch candidate before archive"
	}
	return "Branch candidate before archive"
}

func activeReviewForStepCloseoutScan(reviewCtx *reviewContext) *stepcloseout.ActiveReviewContext {
	if reviewCtx == nil {
		return nil
	}
	return &stepcloseout.ActiveReviewContext{
		RoundID:         reviewCtx.RoundID,
		Trigger:         reviewCtx.Trigger,
		TargetStepIndex: reviewCtx.TargetStepIndex,
	}
}

func landInProgress(state *runstate.State) bool {
	return state != nil &&
		state.Land != nil &&
		strings.TrimSpace(state.Land.LandedAt) != "" &&
		strings.TrimSpace(state.Land.CompletedAt) == ""
}

func stepNode(index int, phase string) string {
	return fmt.Sprintf("execution/step-%d/%s", index+1, phase)
}

func commandErrorsToStatusErrors(errors []lifecycle.CommandError) []StatusError {
	out := make([]StatusError, 0, len(errors))
	for _, issue := range errors {
		out = append(out, StatusError{
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	return out
}

func aggregateCommand(roundID string) *string {
	if strings.TrimSpace(roundID) == "" {
		return strPtr("harness review aggregate --round <round-id>")
	}
	return strPtr(fmt.Sprintf("harness review aggregate --round %s", roundID))
}

func strPtr(value string) *string {
	return &value
}

func isStructuralReviewTrigger(trigger string) bool {
	return trigger == "step_closeout" || trigger == "pre_archive"
}

func factsEmpty(f *Facts) bool {
	if f == nil {
		return true
	}
	return strings.TrimSpace(f.CurrentStep) == "" &&
		f.Revision == 0 &&
		strings.TrimSpace(f.ReopenMode) == "" &&
		strings.TrimSpace(f.ReviewKind) == "" &&
		strings.TrimSpace(f.ReviewTrigger) == "" &&
		strings.TrimSpace(f.ReviewTitle) == "" &&
		strings.TrimSpace(f.ReviewStatus) == "" &&
		f.ArchiveBlockerCount == 0 &&
		strings.TrimSpace(f.PublishStatus) == "" &&
		strings.TrimSpace(f.PRURL) == "" &&
		strings.TrimSpace(f.CIStatus) == "" &&
		strings.TrimSpace(f.SyncStatus) == "" &&
		strings.TrimSpace(f.LandPRURL) == "" &&
		strings.TrimSpace(f.LandCommit) == ""
}
