package status

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/yzhang1918/superharness/internal/evidence"
	"github.com/yzhang1918/superharness/internal/lifecycle"
	"github.com/yzhang1918/superharness/internal/plan"
	"github.com/yzhang1918/superharness/internal/runstate"
)

type Service struct {
	Workdir string
}

type Result struct {
	OK         bool          `json:"ok"`
	Command    string        `json:"command"`
	Summary    string        `json:"summary"`
	State      State         `json:"state"`
	Facts      *Facts        `json:"facts,omitempty"`
	Artifacts  *Artifacts    `json:"artifacts,omitempty"`
	NextAction []NextAction  `json:"next_actions"`
	Blockers   []StatusError `json:"blockers,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	Errors     []StatusError `json:"errors,omitempty"`
}

type State struct {
	CurrentNode string `json:"current_node"`
}

type Facts struct {
	CurrentStep         string `json:"current_step,omitempty"`
	Revision            int    `json:"revision,omitempty"`
	ReopenMode          string `json:"reopen_mode,omitempty"`
	ReviewKind          string `json:"review_kind,omitempty"`
	ReviewTrigger       string `json:"review_trigger,omitempty"`
	ReviewTarget        string `json:"review_target,omitempty"`
	ReviewStatus        string `json:"review_status,omitempty"`
	ArchiveBlockerCount int    `json:"archive_blocker_count,omitempty"`
	PublishStatus       string `json:"publish_status,omitempty"`
	PRURL               string `json:"pr_url,omitempty"`
	CIStatus            string `json:"ci_status,omitempty"`
	SyncStatus          string `json:"sync_status,omitempty"`
	LandPRURL           string `json:"land_pr_url,omitempty"`
	LandCommit          string `json:"land_commit,omitempty"`
}

type Artifacts struct {
	PlanPath           string `json:"plan_path,omitempty"`
	LocalStatePath     string `json:"local_state_path,omitempty"`
	ReviewRoundID      string `json:"review_round_id,omitempty"`
	CIRecordID         string `json:"ci_record_id,omitempty"`
	PublishRecordID    string `json:"publish_record_id,omitempty"`
	SyncRecordID       string `json:"sync_record_id,omitempty"`
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`
	LastLandedAt       string `json:"last_landed_at,omitempty"`
}

type NextAction struct {
	Command     *string `json:"command"`
	Description string  `json:"description"`
}

type StatusError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type reviewContext struct {
	RoundID         string
	Kind            string
	Trigger         string
	Target          string
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

func (s Service) Read() Result {
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

	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
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

	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		return Result{
			OK:      false,
			Command: "status",
			Summary: "Unable to determine the current plan path.",
			Artifacts: &Artifacts{
				PlanPath:       planPath,
				LocalStatePath: statePath,
			},
			Errors: []StatusError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)

	result := Result{
		OK:      true,
		Command: "status",
		Artifacts: &Artifacts{
			PlanPath:       planPath,
			LocalStatePath: statePath,
		},
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
	if state != nil && state.Reopen != nil {
		facts.ReopenMode = state.Reopen.Mode
	}
	if reviewCtx != nil && isStructuralReviewTrigger(reviewCtx.Trigger) && !reviewCtx.UnsafeFallback {
		facts.ReviewKind = reviewCtx.Kind
		facts.ReviewTrigger = reviewCtx.Trigger
		facts.ReviewTarget = reviewCtx.Target
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
		evidenceCtx, evidenceWarnings := loadEvidenceContext(s.Workdir, state)
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
	result.Summary = buildSummary(result.State.CurrentNode, facts, reviewCtx, blockers, currentPlan)
	result.NextAction = buildNextActions(result.State.CurrentNode, facts, reviewCtx, blockers)
	if facts.empty() {
		result.Facts = nil
	} else {
		result.Facts = facts
	}

	cacheSafe := reviewCtx == nil || !reviewCtx.UnsafeFallback
	if !cacheSafe {
		result.Warnings = append(result.Warnings, "Skipping current_node cache refresh because the reviewed step could not be recovered safely from local review metadata.")
	} else if savedStatePath, err := s.cacheResolvedNode(planStem, relPlanPath, state, result.State.CurrentNode); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Unable to refresh the current_node cache: %v", err))
	} else if strings.TrimSpace(savedStatePath) != "" {
		result.Artifacts.LocalStatePath = savedStatePath
	}

	if result.Artifacts != nil && result.Artifacts.PlanPath == "" && result.Artifacts.LocalStatePath == "" &&
		result.Artifacts.ReviewRoundID == "" && result.Artifacts.CIRecordID == "" &&
		result.Artifacts.PublishRecordID == "" && result.Artifacts.SyncRecordID == "" &&
		result.Artifacts.LastLandedPlanPath == "" && result.Artifacts.LastLandedAt == "" {
		result.Artifacts = nil
	}

	return result
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

	if trigger, known, err := runstate.EffectiveReviewTrigger(workdir, planStem, round); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review trigger for %s; status may be conservative.", round.RoundID))
	} else if known {
		ctx.Trigger = trigger
	}

	if target, known, err := runstate.EffectiveReviewTarget(workdir, planStem, round); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read the review target for %s; status may fall back to a conservative step match.", round.RoundID))
	} else if known {
		ctx.Target = target
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

	if ctx.Trigger == "step_closeout" {
		if index, matched := resolveReviewTargetStep(doc, ctx.Target); matched {
			ctx.TargetStepIndex = index
		} else {
			ctx.TargetStepIndex, ctx.UnsafeFallback = fallbackReviewTargetStep(doc, state)
			if strings.TrimSpace(ctx.Target) != "" {
				warnings = append(warnings, fmt.Sprintf("Review target %q did not match a tracked step title; status fell back to the most likely step.", ctx.Target))
			}
		}
	} else if ctx.Trigger == "" {
		if state != nil {
			if index, ok := stepIndexFromNode(state.CurrentNode); ok {
				ctx.TargetStepIndex = index
				ctx.UnsafeFallback = true
				warnings = append(warnings, "Review trigger metadata was missing; status is conservatively pinning the active round to the cached step node without treating it as structural review state.")
			} else if index, _ := fallbackReviewTargetStep(doc, state); index >= 0 {
				ctx.TargetStepIndex = index
				ctx.UnsafeFallback = true
				warnings = append(warnings, "Review trigger metadata was missing; status is conservatively pinning the active round to the most likely reviewed step without treating it as structural review state.")
			}
		} else if index, _ := fallbackReviewTargetStep(doc, state); index >= 0 {
			ctx.TargetStepIndex = index
			ctx.UnsafeFallback = true
			warnings = append(warnings, "Review trigger metadata was missing; status is conservatively pinning the active round to the most likely reviewed step without treating it as structural review state.")
		}
	}

	return ctx, warnings
}

func loadEvidenceContext(workdir string, state *runstate.State) (*evidenceContext, []string) {
	ctx := &evidenceContext{}
	warnings := make([]string, 0)

	if publish, err := evidence.LoadLatestPublish(workdir, state); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read publish evidence: %v", err))
	} else {
		ctx.Publish = publish
	}
	if ci, err := evidence.LoadLatestCI(workdir, state); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read CI evidence: %v", err))
	} else {
		ctx.CI = ci
	}
	if sync, err := evidence.LoadLatestSync(workdir, state); err != nil {
		warnings = append(warnings, fmt.Sprintf("Unable to read sync evidence: %v", err))
	} else {
		ctx.Sync = sync
	}

	return ctx, warnings
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

func buildSummary(node string, facts *Facts, reviewCtx *reviewContext, blockers []StatusError, currentPlan *runstate.CurrentPlan) string {
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
		return "Merge has been recorded and post-merge cleanup is still in progress."
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
				Description: "After the PR is merged outside harness and the worktree is synced, record merge confirmation and enter post-merge cleanup.",
			})
		}
		actions = append(actions, NextAction{
			Command:     nil,
			Description: "If new feedback or remote changes invalidate the archived candidate, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step.",
		})
		return actions
	case "land":
		return []NextAction{
			{Command: nil, Description: "Finish post-merge cleanup such as comments, issue updates, and other closeout tasks while the plan is in land."},
			{Command: strPtr("harness land complete"), Description: "Record cleanup completion and restore the worktree to idle."},
		}
	}

	if strings.HasSuffix(node, "/review") {
		return []NextAction{
			{Command: aggregateCommand(reviewCtx.RoundID), Description: "Aggregate the active review round once the expected reviewer submissions are ready."},
		}
	}
	if strings.HasSuffix(node, "/implement") {
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

func buildPublishNextActions(facts *Facts) []NextAction {
	actions := make([]NextAction, 0)

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

func (s Service) cacheResolvedNode(planStem, relPlanPath string, state *runstate.State, currentNode string) (string, error) {
	if strings.TrimSpace(planStem) == "" {
		return "", nil
	}
	if state == nil {
		state = &runstate.State{}
	}
	state.PlanPath = relPlanPath
	state.PlanStem = planStem
	state.CurrentNode = currentNode
	return runstate.SaveState(s.Workdir, planStem, state)
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

func resolveReviewTargetStep(doc *plan.Document, target string) (int, bool) {
	target = normalizeReviewTarget(target)
	if target == "" {
		return -1, false
	}
	for index, step := range doc.Steps {
		if normalizeReviewTarget(step.Title) == target {
			return index, true
		}
	}
	return -1, false
}

func normalizeReviewTarget(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func fallbackReviewTargetStep(doc *plan.Document, state *runstate.State) (int, bool) {
	if state != nil {
		if index, ok := stepIndexFromNode(state.CurrentNode); ok {
			return index, false
		}
	}
	if index := currentStepIndex(doc); index >= 0 {
		if index > 0 {
			return index - 1, true
		}
		return index, true
	}
	if len(doc.Steps) == 0 {
		return -1, false
	}
	return len(doc.Steps) - 1, true
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

func stepIndexFromNode(node string) (int, bool) {
	node = strings.TrimSpace(node)
	if !strings.HasPrefix(node, "execution/step-") {
		return -1, false
	}
	node = strings.TrimPrefix(node, "execution/step-")
	parts := strings.SplitN(node, "/", 2)
	if len(parts) != 2 {
		return -1, false
	}
	var stepNumber int
	if _, err := fmt.Sscanf(parts[0], "%d", &stepNumber); err != nil || stepNumber <= 0 {
		return -1, false
	}
	return stepNumber - 1, true
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

func (f *Facts) empty() bool {
	if f == nil {
		return true
	}
	return strings.TrimSpace(f.CurrentStep) == "" &&
		f.Revision == 0 &&
		strings.TrimSpace(f.ReopenMode) == "" &&
		strings.TrimSpace(f.ReviewKind) == "" &&
		strings.TrimSpace(f.ReviewTrigger) == "" &&
		strings.TrimSpace(f.ReviewTarget) == "" &&
		strings.TrimSpace(f.ReviewStatus) == "" &&
		f.ArchiveBlockerCount == 0 &&
		strings.TrimSpace(f.PublishStatus) == "" &&
		strings.TrimSpace(f.PRURL) == "" &&
		strings.TrimSpace(f.CIStatus) == "" &&
		strings.TrimSpace(f.SyncStatus) == "" &&
		strings.TrimSpace(f.LandPRURL) == "" &&
		strings.TrimSpace(f.LandCommit) == ""
}
