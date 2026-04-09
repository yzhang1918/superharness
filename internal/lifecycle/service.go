package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/evidence"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
	"github.com/catu-ai/easyharness/internal/stepcloseout"
	"gopkg.in/yaml.v3"
)

type Service struct {
	Workdir       string
	Now           func() time.Time
	AfterMutation func(Result) error
}

type Result = contracts.LifecycleResult
type State = contracts.LifecycleState
type Facts = contracts.LifecycleFacts
type Artifacts = contracts.LifecycleArtifacts
type NextAction = contracts.NextAction
type CommandError = contracts.ErrorDetail

type editablePlan struct {
	Frontmatter plan.Frontmatter
	Body        string
}

func (s Service) ExecuteStart() Result {
	now := s.now()
	_, doc, _, planStem, relCurrentPath, state, statePath, release, errResult := s.loadCurrentPlan()
	if errResult != nil {
		errResult.Command = "execute start"
		return *errResult
	}
	defer release()

	if doc.DerivedPlanStatus() != "active" {
		return errorResult("execute start", "Current plan is not active.", []CommandError{{
			Path:    "plan.status",
			Message: fmt.Sprintf("execute start requires an active plan, got status=%q", doc.DerivedPlanStatus()),
		}})
	}

	if state == nil {
		state = &runstate.State{Revision: 1}
	}
	originalState := cloneState(state)
	currentPlanBefore, err := runstate.LoadCurrentPlan(s.Workdir)
	if err != nil {
		return errorResult("execute start", "Unable to read the current-plan pointer before execution start.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	alreadyStarted := strings.TrimSpace(state.ExecutionStartedAt) != ""
	if state.Revision <= 0 {
		state.Revision = 1
	}

	if strings.TrimSpace(state.ExecutionStartedAt) == "" {
		state.ExecutionStartedAt = now.Format(time.RFC3339)
	}

	savedStatePath, err := runstate.SaveState(s.Workdir, planStem, state)
	if err != nil {
		return errorResult("execute start", "Unable to persist execution-start state.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	if statePath == "" {
		statePath = savedStatePath
	}

	currentPlanPath, err := runstate.SaveCurrentPlan(s.Workdir, relCurrentPath)
	if err != nil {
		if originalState != nil {
			_, _ = runstate.SaveState(s.Workdir, planStem, originalState)
		} else {
			_ = os.Remove(savedStatePath)
		}
		return errorResult("execute start", "Unable to update current-plan pointer.", []CommandError{{Path: "state", Message: err.Error()}})
	}

	summary := "Execution started for the current active plan."
	if alreadyStarted {
		summary = "Execution is already started for the current active plan."
	}

	result := Result{
		OK:      true,
		Command: "execute start",
		Summary: summary,
		State: State{
			CurrentNode: "execution/step-1/implement",
		},
		Facts: &Facts{Revision: runstate.CurrentRevision(state)},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Continue the current step and keep step-local Execution Notes and Review Notes up to date."},
		},
	}
	return s.finalizeMutation(result, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, savedStatePath)
		if _, restoreErr := runstate.WriteCurrentPlan(s.Workdir, currentPlanBefore); restoreErr != nil {
			issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback current-plan pointer: %v", restoreErr)})
		}
		return issues
	})
}

func (s Service) Archive() Result {
	now := s.now()
	currentPath, doc, editable, planStem, relCurrentPath, state, statePath, release, errResult := s.loadCurrentPlan()
	if errResult != nil {
		errResult.Command = "archive"
		return *errResult
	}
	defer release()
	if doc.DerivedPlanStatus() != "active" || doc.DerivedLifecycle(state) != "executing" {
		return errorResult("archive", "Current plan is not archive-ready.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("archive requires status=active and lifecycle=executing, got status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
		}})
	}
	if issues := EvaluateArchiveReadiness(s.Workdir, planStem, doc, state); len(issues) > 0 {
		return errorResult("archive", "Current plan is not archive-ready.", issues)
	}

	archiveSummary := doc.SectionText("Archive Summary")
	archiveSummary = stripArchiveSummaryLines(archiveSummary, []string{"Archived At", "Revision"})
	revision := runstate.CurrentRevision(state)
	archiveSummary = strings.TrimSpace(strings.Join([]string{
		fmt.Sprintf("- Archived At: %s", now.Format(time.RFC3339)),
		fmt.Sprintf("- Revision: %d", revision),
		archiveSummary,
	}, "\n"))

	body, err := replaceTopLevelSection(editable.Body, "Archive Summary", archiveSummary)
	if err != nil {
		return errorResult("archive", "Unable to update Archive Summary.", []CommandError{{Path: "section.Archive Summary", Message: err.Error()}})
	}

	targetPath := plan.ArchivedPathFor(s.Workdir, planStem, currentPath, doc.WorkflowProfile())
	if _, err := os.Stat(targetPath); err == nil {
		return errorResult("archive", "Archived target path already exists.", []CommandError{{Path: "path", Message: fmt.Sprintf("target already exists: %s", targetPath)}})
	}

	content, err := renderEditablePlan(editable.Frontmatter, body)
	if err != nil {
		return errorResult("archive", "Unable to render archived plan.", []CommandError{{Path: "frontmatter", Message: err.Error()}})
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errorResult("archive", "Unable to create archived plan directory.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return errorResult("archive", "Unable to write archived plan.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if lint := plan.LintFile(targetPath); !lint.OK {
		_ = os.Remove(targetPath)
		return errorResult("archive", "Archived plan did not pass validation.", lintErrorsToCommandErrors(lint.Errors))
	}

	relTargetPath, err := filepath.Rel(s.Workdir, targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return errorResult("archive", "Unable to relativize archived plan path.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	relTargetPath = filepath.ToSlash(relTargetPath)

	originalState := cloneState(state)
	nextState := cloneState(state)
	if nextState != nil {
		nextState.Reopen = nil
		nextState.Land = nil
		statePath, err = runstate.SaveState(s.Workdir, planStem, nextState)
		if err != nil {
			_ = os.Remove(targetPath)
			return errorResult("archive", "Unable to update local state after archiving.", []CommandError{{Path: "state", Message: err.Error()}})
		}
	}

	currentPlanPath, err := runstate.SaveCurrentPlan(s.Workdir, relTargetPath)
	if err != nil {
		if originalState != nil {
			_, _ = runstate.SaveState(s.Workdir, planStem, originalState)
		}
		_ = os.Remove(targetPath)
		return errorResult("archive", "Unable to update current-plan pointer.", []CommandError{{Path: "state", Message: err.Error()}})
	}

	nextActions := []NextAction{
		{Command: nil, Description: "Commit and push the tracked plan change created by archiving before treating the candidate as truly waiting for merge approval."},
		{Command: nil, Description: "Wait for human merge approval or merge manually from the PR once checks are green."},
		{Command: nil, Description: "If new feedback or remote changes invalidate the archived candidate, reopen with `harness reopen --mode finalize-fix` for narrow repair or `harness reopen --mode new-step` when the change deserves a new unfinished step."},
	}
	if doc.UsesLightweightProfile() {
		nextActions = append([]NextAction{
			{Command: nil, Description: "Update the agreed repo-visible breadcrumb, such as the PR body note that explains why the lightweight path was used, before treating the candidate as truly waiting for merge approval."},
			{Command: nil, Description: "Commit and push the tracked active-plan removal created by lightweight archiving before treating the candidate as truly waiting for merge approval."},
		}, nextActions...)
	}

	result := Result{
		OK:      true,
		Command: "archive",
		Summary: "Plan archived and frozen for merge handoff.",
		State: State{
			CurrentNode: "execution/finalize/publish",
		},
		Facts: &Facts{Revision: revision},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			ToPlanPath:      relTargetPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: nextActions,
	}
	if err := os.Remove(currentPath); err != nil {
		rollbackErrors := rollbackTransition(s.Workdir, relCurrentPath, planStem, originalState, targetPath, statePath)
		rollbackErrors = append([]CommandError{{Path: "path", Message: err.Error()}}, rollbackErrors...)
		return errorResult("archive", "Unable to remove the active plan after archiving.", rollbackErrors)
	}
	result = s.finalizeMutation(result, func() []CommandError {
		return rollbackTransition(s.Workdir, relCurrentPath, planStem, originalState, targetPath, statePath)
	})
	return result
}

func (s Service) Reopen(mode string) Result {
	now := s.now()
	currentPath, doc, editable, planStem, relCurrentPath, state, statePath, release, errResult := s.loadCurrentPlan()
	if errResult != nil {
		errResult.Command = "reopen"
		return *errResult
	}
	defer release()
	if doc.DerivedPlanStatus() != "archived" || doc.DerivedLifecycle(state) != "awaiting_merge_approval" {
		return errorResult("reopen", "Current plan is not archived.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("reopen requires status=archived and lifecycle=awaiting_merge_approval, got status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
		}})
	}
	if state != nil && state.Land != nil && strings.TrimSpace(state.Land.LandedAt) != "" && strings.TrimSpace(state.Land.CompletedAt) == "" {
		return errorResult("reopen", "Archived candidate is already in required post-merge bookkeeping.", []CommandError{{
			Path:    "state.land",
			Message: "required post-merge bookkeeping must finish with `harness land complete` before reopen is allowed",
		}})
	}
	mode = strings.TrimSpace(mode)
	if mode != "finalize-fix" && mode != "new-step" {
		return errorResult("reopen", "Reopen mode is required.", []CommandError{{
			Path:    "mode",
			Message: "mode must be finalize-fix or new-step",
		}})
	}

	body, err := markTopLevelSectionUpdateRequired(editable.Body, "Validation Summary")
	if err != nil {
		return errorResult("reopen", "Unable to refresh Validation Summary.", []CommandError{{Path: "section.Validation Summary", Message: err.Error()}})
	}
	body, err = markTopLevelSectionUpdateRequired(body, "Review Summary")
	if err != nil {
		return errorResult("reopen", "Unable to refresh Review Summary.", []CommandError{{Path: "section.Review Summary", Message: err.Error()}})
	}
	body, err = markTopLevelSectionUpdateRequired(body, "Archive Summary")
	if err != nil {
		return errorResult("reopen", "Unable to refresh Archive Summary.", []CommandError{{Path: "section.Archive Summary", Message: err.Error()}})
	}
	body, err = markOutcomeSummaryUpdateRequired(body)
	if err != nil {
		return errorResult("reopen", "Unable to refresh Outcome Summary.", []CommandError{{Path: "section.Outcome Summary", Message: err.Error()}})
	}

	targetPath := plan.ActivePathFor(s.Workdir, planStem, currentPath, doc.WorkflowProfile())
	if _, err := os.Stat(targetPath); err == nil {
		return errorResult("reopen", "Active target path already exists.", []CommandError{{Path: "path", Message: fmt.Sprintf("target already exists: %s", targetPath)}})
	}

	content, err := renderEditablePlan(editable.Frontmatter, body)
	if err != nil {
		return errorResult("reopen", "Unable to render reopened plan.", []CommandError{{Path: "frontmatter", Message: err.Error()}})
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return errorResult("reopen", "Unable to create active plan directory.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if err := os.WriteFile(targetPath, []byte(content), 0o644); err != nil {
		return errorResult("reopen", "Unable to write reopened plan.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	if lint := plan.LintFile(targetPath); !lint.OK {
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Reopened plan did not pass validation.", lintErrorsToCommandErrors(lint.Errors))
	}

	relTargetPath, err := filepath.Rel(s.Workdir, targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Unable to relativize active plan path.", []CommandError{{Path: "path", Message: err.Error()}})
	}
	relTargetPath = filepath.ToSlash(relTargetPath)

	originalState := cloneState(state)
	nextState := cloneState(state)
	if nextState == nil {
		nextState = &runstate.State{}
	}
	nextState.ExecutionStartedAt = now.Format(time.RFC3339)
	nextState.Revision = runstate.CurrentRevision(state) + 1
	nextState.Reopen = &runstate.ReopenState{
		Mode:          mode,
		ReopenedAt:    now.Format(time.RFC3339),
		BaseStepCount: len(doc.Steps),
	}
	nextState.ActiveReviewRound = nil
	nextState.Land = nil
	statePath, err = runstate.SaveState(s.Workdir, planStem, nextState)
	if err != nil {
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Unable to update local state after reopen.", []CommandError{{Path: "state", Message: err.Error()}})
	}

	currentPlanPath, err := runstate.SaveCurrentPlan(s.Workdir, relTargetPath)
	if err != nil {
		if originalState != nil {
			_, _ = runstate.SaveState(s.Workdir, planStem, originalState)
		}
		_ = os.Remove(targetPath)
		return errorResult("reopen", "Unable to update current-plan pointer.", []CommandError{{Path: "state", Message: err.Error()}})
	}

	result := Result{
		OK:      true,
		Command: "reopen",
		Summary: "Archived plan reopened for active execution.",
		State: State{
			CurrentNode: "execution/finalize/fix",
		},
		Facts: &Facts{Revision: nextState.Revision, ReopenMode: mode},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			ToPlanPath:      relTargetPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Review the feedback or remote change that caused reopen."},
			{Command: nil, Description: "Update the plan content if scope or acceptance criteria changed."},
			{Command: nil, Description: reopenNextActionDescription(mode)},
		},
	}
	if err := os.Remove(currentPath); err != nil {
		rollbackErrors := rollbackTransition(s.Workdir, relCurrentPath, planStem, originalState, targetPath, statePath)
		rollbackErrors = append([]CommandError{{Path: "path", Message: err.Error()}}, rollbackErrors...)
		return errorResult("reopen", "Unable to remove the archived plan after reopening.", rollbackErrors)
	}
	result = s.finalizeMutation(result, func() []CommandError {
		return rollbackTransition(s.Workdir, relCurrentPath, planStem, originalState, targetPath, statePath)
	})
	return result
}

func (s Service) Land(prURL, commit string) Result {
	now := s.now()
	currentPath, doc, _, planStem, relCurrentPath, state, statePath, release, errResult := s.loadCurrentPlan()
	if errResult != nil {
		errResult.Command = "land"
		return *errResult
	}
	defer release()
	if doc.DerivedPlanStatus() != "archived" || doc.DerivedLifecycle(state) != "awaiting_merge_approval" {
		return errorResult("land", "Current plan is not archived.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("land requires status=archived and lifecycle=awaiting_merge_approval, got status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
		}})
	}
	prURL = strings.TrimSpace(prURL)
	if prURL == "" {
		return errorResult("land", "PR URL is required.", []CommandError{{
			Path:    "pr",
			Message: "land requires --pr <url>",
		}})
	}
	if state != nil && state.Land != nil && strings.TrimSpace(state.Land.LandedAt) != "" && strings.TrimSpace(state.Land.CompletedAt) == "" {
		recordedPR := strings.TrimSpace(state.Land.PRURL)
		recordedCommit := strings.TrimSpace(state.Land.Commit)
		requestedCommit := strings.TrimSpace(commit)
		if recordedPR != prURL || (requestedCommit != "" && recordedCommit != "" && requestedCommit != recordedCommit) {
			return errorResult("land", "Required post-merge bookkeeping is already in progress.", []CommandError{{
				Path:    "state.land",
				Message: fmt.Sprintf("land already recorded for pr=%q commit=%q at %s", recordedPR, recordedCommit, state.Land.LandedAt),
			}})
		}
		if requestedCommit != "" && recordedCommit == "" {
			originalState := cloneState(state)
			state.Land.Commit = requestedCommit
			statePath, err := runstate.SaveState(s.Workdir, planStem, state)
			if err != nil {
				return errorResult("land", "Unable to update land entry state.", []CommandError{{Path: "state", Message: err.Error()}})
			}
			result := Result{
				OK:      true,
				Command: "land",
				Summary: fmt.Sprintf("Recorded landed commit for the in-progress required post-merge bookkeeping of %s.", filepath.Base(currentPath)),
				State: State{
					CurrentNode: "land",
				},
				Facts: &Facts{
					Revision:   runstate.CurrentRevision(state),
					LandPRURL:  strings.TrimSpace(state.Land.PRURL),
					LandCommit: strings.TrimSpace(state.Land.Commit),
				},
				Artifacts: &Artifacts{
					FromPlanPath:   relCurrentPath,
					LocalStatePath: statePath,
				},
				NextAction: []NextAction{
					{Command: nil, Description: "Finish required post-merge bookkeeping and cleanup: add the final PR comment when the permanent record still needs one, close resolved linked issues or add follow-up references for unresolved ones, sync local branches, and complete any final remote updates."},
					{Command: strPtr("harness land complete"), Description: "Record required post-merge bookkeeping completion only after the required PR and issue bookkeeping is done."},
				},
			}
			return s.finalizeMutation(result, func() []CommandError {
				return restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
			})
		}
		return s.finalizeMutation(Result{
			OK:      true,
			Command: "land",
			Summary: fmt.Sprintf("Required post-merge bookkeeping is already in progress for %s.", filepath.Base(currentPath)),
			State: State{
				CurrentNode: "land",
			},
			Facts: &Facts{
				Revision:   runstate.CurrentRevision(state),
				LandPRURL:  strings.TrimSpace(state.Land.PRURL),
				LandCommit: strings.TrimSpace(state.Land.Commit),
			},
			Artifacts: &Artifacts{
				FromPlanPath:   relCurrentPath,
				LocalStatePath: statePath,
			},
			NextAction: []NextAction{
				{Command: nil, Description: "Finish required post-merge bookkeeping and cleanup: add the final PR comment when the permanent record still needs one, close resolved linked issues or add follow-up references for unresolved ones, sync local branches, and complete any final remote updates."},
				{Command: strPtr("harness land complete"), Description: "Record required post-merge bookkeeping completion only after the required PR and issue bookkeeping is done."},
			},
		}, nil)
	}
	if issues := s.landReadinessIssues(planStem, state, prURL); len(issues) > 0 {
		return errorResult("land", "Archived candidate is not ready to enter required post-merge bookkeeping.", issues)
	}
	originalState := cloneState(state)
	if state == nil {
		state = &runstate.State{}
	}
	if state.Revision <= 0 {
		state.Revision = 1
	}
	state.Land = &runstate.LandState{
		PRURL:    prURL,
		Commit:   strings.TrimSpace(commit),
		LandedAt: now.Format(time.RFC3339),
	}
	statePath, err := runstate.SaveState(s.Workdir, planStem, state)
	if err != nil {
		return errorResult("land", "Unable to record land entry state.", []CommandError{{Path: "state", Message: err.Error()}})
	}

	result := Result{
		OK:      true,
		Command: "land",
		Summary: fmt.Sprintf("Recorded merge confirmation for %s and entered required post-merge bookkeeping.", filepath.Base(currentPath)),
		State: State{
			CurrentNode: "land",
		},
		Facts: &Facts{
			Revision:   runstate.CurrentRevision(state),
			LandPRURL:  strings.TrimSpace(state.Land.PRURL),
			LandCommit: strings.TrimSpace(state.Land.Commit),
		},
		Artifacts: &Artifacts{
			FromPlanPath:   relCurrentPath,
			LocalStatePath: statePath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Finish required post-merge bookkeeping and cleanup: add the final PR comment when the permanent record still needs one, close resolved linked issues or add follow-up references for unresolved ones, sync local branches, and complete any final remote updates."},
			{Command: strPtr("harness land complete"), Description: "Record required post-merge bookkeeping completion only after the required PR and issue bookkeeping is done."},
		},
	}
	return s.finalizeMutation(result, func() []CommandError {
		return restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
	})
}

func (s Service) LandComplete() Result {
	now := s.now()
	currentPath, doc, _, planStem, relCurrentPath, state, statePath, release, errResult := s.loadCurrentPlan()
	if errResult != nil {
		errResult.Command = "land complete"
		return *errResult
	}
	defer release()
	if doc.DerivedPlanStatus() != "archived" || doc.DerivedLifecycle(state) != "awaiting_merge_approval" {
		return errorResult("land complete", "Current plan is not archived.", []CommandError{{
			Path:    "plan.lifecycle",
			Message: fmt.Sprintf("land complete requires status=archived and lifecycle=awaiting_merge_approval, got status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
		}})
	}
	if state == nil || state.Land == nil || strings.TrimSpace(state.Land.LandedAt) == "" {
		return errorResult("land complete", "Required post-merge bookkeeping cannot complete before land entry.", []CommandError{{
			Path:    "state.land",
			Message: "run `harness land --pr <url>` before `harness land complete`",
		}})
	}
	originalState := cloneState(state)
	currentPlanBefore, err := runstate.LoadCurrentPlan(s.Workdir)
	if err != nil {
		return errorResult("land complete", "Unable to read the current-plan pointer before land completion.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	state.Land.CompletedAt = now.Format(time.RFC3339)
	statePath, err = runstate.SaveState(s.Workdir, planStem, state)
	if err != nil {
		return errorResult("land complete", "Unable to persist land completion state.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	currentPlanPath, err := runstate.SaveLandedPlan(s.Workdir, relCurrentPath, now.Format(time.RFC3339))
	if err != nil {
		if originalState != nil {
			if _, rollbackErr := runstate.SaveState(s.Workdir, planStem, originalState); rollbackErr != nil {
				return errorResult("land complete", "Unable to record landed worktree state.", []CommandError{
					{Path: "state", Message: err.Error()},
					{Path: "state", Message: fmt.Sprintf("rollback local state: %v", rollbackErr)},
				})
			}
		}
		return errorResult("land complete", "Unable to record landed worktree state.", []CommandError{{Path: "state", Message: err.Error()}})
	}
	result := Result{
		OK:      true,
		Command: "land complete",
		Summary: fmt.Sprintf("Recorded required post-merge bookkeeping completion for %s.", filepath.Base(currentPath)),
		State: State{
			CurrentNode: "idle",
		},
		Facts: &Facts{Revision: runstate.CurrentRevision(state)},
		Artifacts: &Artifacts{
			FromPlanPath:    relCurrentPath,
			LocalStatePath:  statePath,
			CurrentPlanPath: currentPlanPath,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Run harness status to confirm the worktree now reports idle-after-land state."},
			{Command: nil, Description: "Start discovery or create a new plan when the next slice is ready."},
		},
	}
	return s.finalizeMutation(result, func() []CommandError {
		issues := restoreStateSnapshot(s.Workdir, planStem, originalState, statePath)
		if _, restoreErr := runstate.WriteCurrentPlan(s.Workdir, currentPlanBefore); restoreErr != nil {
			issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback current-plan pointer: %v", restoreErr)})
		}
		return issues
	})
}

func (s Service) loadCurrentPlan() (string, *plan.Document, *editablePlan, string, string, *runstate.State, string, func(), *Result) {
	release := func() {}
	currentPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", nil, nil, "", "", nil, "", release, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(currentPath), filepath.Ext(currentPath))
	release, err = runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Another local state mutation is already in progress.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	currentPath, err = plan.DetectCurrentPathLocked(s.Workdir, planStem)
	if err != nil {
		release()
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	doc, err := plan.LoadFile(currentPath)
	if err != nil {
		release()
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	editable, err := loadEditablePlan(currentPath)
	if err != nil {
		release()
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to load the editable plan representation.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relCurrentPath, err := filepath.Rel(s.Workdir, currentPath)
	if err != nil {
		release()
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "path", Message: err.Error()}},
		}
	}
	relCurrentPath = filepath.ToSlash(relCurrentPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		release()
		return "", nil, nil, "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	return currentPath, doc, editable, planStem, relCurrentPath, state, statePath, release, nil
}

func loadEditablePlan(path string) (*editablePlan, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	rawFrontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, err
	}
	var frontmatter plan.Frontmatter
	if err := yaml.Unmarshal([]byte(rawFrontmatter), &frontmatter); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return &editablePlan{Frontmatter: frontmatter, Body: strings.TrimLeft(body, "\n")}, nil
}

func splitFrontmatter(content string) (string, string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("file must start with YAML frontmatter delimited by ---")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", fmt.Errorf("frontmatter is missing a closing --- delimiter")
}

func replaceTopLevelSection(body, sectionName, newContent string) (string, error) {
	header := "## " + sectionName + "\n\n"
	start := strings.Index(body, header)
	if start == -1 {
		return "", fmt.Errorf("missing ## %s section", sectionName)
	}

	searchStart := start + len(header)
	nextRelative := strings.Index(body[searchStart:], "\n## ")
	end := len(body)
	if nextRelative != -1 {
		end = searchStart + nextRelative + 1
	}

	replacement := fmt.Sprintf("## %s\n\n%s\n\n", sectionName, strings.TrimSpace(newContent))
	return body[:start] + replacement + strings.TrimLeft(body[end:], "\n"), nil
}

func markTopLevelSectionUpdateRequired(body, sectionName string) (string, error) {
	currentBody, err := topLevelSectionBody(body, sectionName)
	if err != nil {
		return "", err
	}
	content := strings.TrimSpace(currentBody)
	if !strings.Contains(content, plan.PlaceholderUpdateRequiredAfterReopen) {
		content = strings.TrimSpace(strings.Join([]string{
			plan.PlaceholderUpdateRequiredAfterReopen,
			"",
			content,
		}, "\n"))
	}
	return replaceTopLevelSection(body, sectionName, content)
}

func markOutcomeSummaryUpdateRequired(body string) (string, error) {
	content, err := topLevelSectionBody(body, "Outcome Summary")
	if err != nil {
		return "", err
	}
	subsections, order := parseLevelThreeSections(strings.Split(content, "\n"))
	required := []string{"Delivered", "Not Delivered", "Follow-Up Issues"}
	if !slices.Equal(order, required) {
		return "", fmt.Errorf("Outcome Summary must contain Delivered, Not Delivered, and Follow-Up Issues in order")
	}

	rendered := make([]string, 0, 12)
	for _, name := range required {
		subsection := subsections[name]
		if subsection == nil {
			return "", fmt.Errorf("missing Outcome Summary subsection %q", name)
		}
		subcontent := strings.TrimSpace(strings.Join(subsection, "\n"))
		if !strings.Contains(subcontent, plan.PlaceholderUpdateRequiredAfterReopen) {
			subcontent = strings.TrimSpace(strings.Join([]string{
				plan.PlaceholderUpdateRequiredAfterReopen,
				"",
				subcontent,
			}, "\n"))
		}
		rendered = append(rendered, "### "+name, "", subcontent, "")
	}

	return replaceTopLevelSection(body, "Outcome Summary", strings.TrimSpace(strings.Join(rendered, "\n")))
}

func topLevelSectionBody(body, sectionName string) (string, error) {
	header := "## " + sectionName + "\n\n"
	start := strings.Index(body, header)
	if start == -1 {
		return "", fmt.Errorf("missing ## %s section", sectionName)
	}

	searchStart := start + len(header)
	nextRelative := strings.Index(body[searchStart:], "\n## ")
	end := len(body)
	if nextRelative != -1 {
		end = searchStart + nextRelative + 1
	}
	return strings.TrimSpace(body[searchStart:end]), nil
}

func parseLevelThreeSections(lines []string) (map[string][]string, []string) {
	sections := map[string][]string{}
	order := make([]string, 0)
	current := ""
	for _, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r")
		if strings.HasPrefix(line, "### ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			order = append(order, current)
			sections[current] = nil
			continue
		}
		if current != "" {
			sections[current] = append(sections[current], line)
		}
	}
	return sections, order
}

func renderEditablePlan(frontmatter plan.Frontmatter, body string) (string, error) {
	data, err := yaml.Marshal(frontmatter)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("---\n%s---\n\n%s", string(data), strings.TrimLeft(body, "\n")), nil
}

func missingArchiveSummaryLabels(content string, labels []string) []string {
	missing := make([]string, 0)
	for _, label := range labels {
		if !strings.Contains(content, "- "+label+":") {
			missing = append(missing, label)
		}
	}
	return missing
}

func stripArchiveSummaryLines(content string, labels []string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		keep := true
		for _, label := range labels {
			if strings.HasPrefix(strings.TrimSpace(line), "- "+label+":") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, line)
		}
	}
	return strings.TrimSpace(strings.Join(filtered, "\n"))
}

func lintErrorsToCommandErrors(issues []plan.LintIssue) []CommandError {
	errors := make([]CommandError, 0, len(issues))
	for _, issue := range issues {
		errors = append(errors, CommandError{Path: issue.Path, Message: issue.Message})
	}
	return errors
}

func errorResult(command, summary string, errors []CommandError) Result {
	return Result{
		OK:      false,
		Command: command,
		Summary: summary,
		Errors:  errors,
	}
}

func (s Service) finalizeMutation(result Result, rollback func() []CommandError) Result {
	if !result.OK || s.AfterMutation == nil {
		return result
	}
	if err := s.AfterMutation(result); err != nil {
		issues := []CommandError{{Path: "timeline", Message: err.Error()}}
		if rollback != nil {
			issues = append(issues, rollback()...)
		}
		return errorResult(result.Command, "Unable to record the timeline event for the successful command result.", issues)
	}
	return result
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

func reopenNextActionDescription(mode string) string {
	if mode == "new-step" {
		return "Add a new unfinished step for the reopened scope, then continue implementation from that new step."
	}
	return "Repair the reopened finalize-scope issues, refresh durable summaries as needed, and rerun review before archive."
}

func rollbackTransition(workdir, relCurrentPath, planStem string, originalState *runstate.State, targetPath, statePath string) []CommandError {
	issues := make([]CommandError, 0)
	absCurrentPath := filepath.Join(workdir, filepath.FromSlash(relCurrentPath))
	currentExists := false
	if _, err := os.Stat(absCurrentPath); err == nil {
		currentExists = true
	} else if !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "path", Message: fmt.Sprintf("rollback current path stat: %v", err)})
	}
	if _, err := os.Stat(targetPath); err == nil {
		if currentExists {
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				issues = append(issues, CommandError{Path: "path", Message: fmt.Sprintf("rollback target path: %v", err)})
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(absCurrentPath), 0o755); err != nil {
				issues = append(issues, CommandError{Path: "path", Message: fmt.Sprintf("rollback current path parent: %v", err)})
			} else if err := os.Rename(targetPath, absCurrentPath); err != nil {
				issues = append(issues, CommandError{Path: "path", Message: fmt.Sprintf("rollback target restore: %v", err)})
			}
		}
	} else if !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "path", Message: fmt.Sprintf("rollback target path stat: %v", err)})
	}
	if _, err := runstate.SaveCurrentPlan(workdir, relCurrentPath); err != nil {
		issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback current-plan pointer: %v", err)})
	}
	issues = append(issues, restoreStateSnapshot(workdir, planStem, originalState, statePath)...)
	return issues
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

func EvaluateArchiveReadiness(workdir, planStem string, doc *plan.Document, state *runstate.State) []CommandError {
	issues := make([]CommandError, 0)
	for _, issue := range doc.ArchiveReadinessIssues() {
		issues = append(issues, CommandError{Path: issue.Path, Message: issue.Message})
	}
	issues = append(issues, archiveEarlierStepCloseoutIssues(workdir, planStem, doc)...)
	issues = append(issues, archiveStateIssues(workdir, planStem, runstate.CurrentRevision(state), state)...)
	return issues
}

func archiveEarlierStepCloseoutIssues(workdir, planStem string, doc *plan.Document) []CommandError {
	reminder := stepcloseout.LoadReminder(workdir, planStem, doc, "execution/finalize/archive", nil)
	if len(reminder.MissingTitles) == 0 {
		return nil
	}
	earliestIndex := reminder.MissingIndexes[0]
	earliestTitle := reminder.MissingTitles[0]
	return []CommandError{{
		Path:    fmt.Sprintf("plan.steps[%d].review_notes", earliestIndex),
		Message: fmt.Sprintf("%s still needs review-complete closeout before archive; start an explicit repair review for that step or record NO_STEP_REVIEW_NEEDED: <reason> in Review Notes first", earliestTitle),
	}}
}

func archiveStateIssues(workdir, planStem string, revision int, state *runstate.State) []CommandError {
	issues := make([]CommandError, 0)
	if state == nil || state.ActiveReviewRound == nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: requiredReviewMessage(revision),
		})
		return issues
	}

	if !state.ActiveReviewRound.Aggregated {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "aggregate or clear the active review round before archive",
		})
	}
	decision, known, err := runstate.EffectiveReviewDecision(workdir, planStem, state.ActiveReviewRound)
	if err != nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("unable to read the latest aggregate artifact for %s: %v", state.ActiveReviewRound.RoundID, err),
		})
		return issues
	}
	if !known {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "latest review decision is unknown; rerun or re-aggregate the latest review before archive",
		})
	}
	reviewRevision, reviewRevisionKnown, err := runstate.EffectiveReviewRevision(workdir, planStem, state.ActiveReviewRound)
	if err != nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("unable to read the latest manifest artifact for %s: %v", state.ActiveReviewRound.RoundID, err),
		})
		return issues
	}
	if !reviewRevisionKnown || reviewRevision != revision {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: requiredReviewMessage(revision),
		})
	}
	stepNumber, stepKnown, err := runstate.EffectiveReviewStep(workdir, planStem, state.ActiveReviewRound)
	if err != nil {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("unable to read the latest manifest artifact for %s: %v", state.ActiveReviewRound.RoundID, err),
		})
		return issues
	}
	if stepKnown {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("latest review is still bound to step %d; archive requires a finalize review for revision %d", stepNumber, revision),
		})
	}
	if known && decision != "pass" {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: fmt.Sprintf("latest review decision %q is not archive-ready; fix findings or rerun review", decision),
		})
	}
	if revision <= 1 && state.ActiveReviewRound.Kind != "full" {
		issues = append(issues, CommandError{
			Path:    "state.active_review_round",
			Message: "revision 1 requires a passing full finalize review before archive",
		})
	}
	return issues
}

func requiredReviewMessage(revision int) string {
	if revision <= 1 {
		return "revision 1 requires a passing full finalize review before archive"
	}
	return "archive requires a passing aggregated finalize review before archive"
}

func (s Service) landReadinessIssues(planStem string, state *runstate.State, prURL string) []CommandError {
	issues := make([]CommandError, 0)

	publish, err := evidence.LoadLatestPublish(s.Workdir, planStem, runstate.CurrentRevision(state))
	if err != nil {
		return []CommandError{{Path: "evidence.publish", Message: err.Error()}}
	}
	if publish == nil || publish.Status != "recorded" || strings.TrimSpace(publish.PRURL) == "" {
		issues = append(issues, CommandError{
			Path:    "evidence.publish",
			Message: "record publish evidence with a PR URL before entering required post-merge bookkeeping",
		})
	} else if strings.TrimSpace(publish.PRURL) != prURL {
		issues = append(issues, CommandError{
			Path:    "pr",
			Message: fmt.Sprintf("land PR URL %q does not match the latest publish evidence %q", prURL, publish.PRURL),
		})
	}

	ciRecord, err := evidence.LoadLatestCI(s.Workdir, planStem, runstate.CurrentRevision(state))
	if err != nil {
		return []CommandError{{Path: "evidence.ci", Message: err.Error()}}
	}
	if ciRecord == nil || (ciRecord.Status != "success" && ciRecord.Status != "not_applied") {
		issues = append(issues, CommandError{
			Path:    "evidence.ci",
			Message: "record passing or explicit not_applied CI evidence before entering required post-merge bookkeeping",
		})
	}

	syncRecord, err := evidence.LoadLatestSync(s.Workdir, planStem, runstate.CurrentRevision(state))
	if err != nil {
		return []CommandError{{Path: "evidence.sync", Message: err.Error()}}
	}
	if syncRecord == nil || (syncRecord.Status != "fresh" && syncRecord.Status != "not_applied") {
		issues = append(issues, CommandError{
			Path:    "evidence.sync",
			Message: "record fresh or explicit not_applied sync evidence before entering required post-merge bookkeeping",
		})
	}

	return issues
}
