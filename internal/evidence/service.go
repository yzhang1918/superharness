package evidence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/catu-ai/easyharness/internal/contracts"
	"github.com/catu-ai/easyharness/internal/inputschema"
	"github.com/catu-ai/easyharness/internal/plan"
	"github.com/catu-ai/easyharness/internal/runstate"
)

var recordIDPattern = regexp.MustCompile(`^(ci|publish|sync)-([0-9]+)\.json$`)
var saveState = runstate.SaveState

type Service struct {
	Workdir       string
	Now           func() time.Time
	AfterMutation func(Result) error
}

type Result = contracts.EvidenceSubmitResult
type Artifacts = contracts.EvidenceArtifacts
type NextAction = contracts.NextAction
type CommandError = contracts.ErrorDetail
type CIInput = contracts.EvidenceCIInput
type PublishInput = contracts.EvidencePublishInput
type SyncInput = contracts.EvidenceSyncInput
type CIRecord = contracts.EvidenceCIRecord
type PublishRecord = contracts.EvidencePublishRecord
type SyncRecord = contracts.EvidenceSyncRecord

func (s Service) Submit(kind string, inputBytes []byte) Result {
	now := s.now().Format(time.RFC3339)
	planPath, relPlanPath, planStem, state, statePath, release, result := s.loadCurrentArchivedPlan()
	if result != nil {
		result.Command = "evidence submit"
		return *result
	}
	defer release()

	kind = strings.TrimSpace(strings.ToLower(kind))
	switch kind {
	case "ci":
		var input CIInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.ci", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("ci", issues)
		}
		if issues := validateCIInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("ci", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := CIRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			Provider:   strings.TrimSpace(input.Provider),
			URL:        strings.TrimSpace(input.URL),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		originalState := cloneState(state)
		statePath, err = updateStateAfterEvidence(s.Workdir, planStem, relPlanPath, state, statePath, kind, recordID, recordPath, now, func(next *runstate.State) {
			next.LatestCI = &runstate.CIState{SnapshotID: recordID, Status: record.Status}
		})
		if err != nil {
			issues := rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
			issues = append([]CommandError{{Path: "state", Message: err.Error()}}, issues...)
			return errorResult("evidence submit", "Unable to update local harness state.", issues)
		}
		return s.finalizeMutation(successResult(planPath, statePath, kind, recordID, recordPath, "Recorded CI evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
		})
	case "publish":
		var input PublishInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.publish", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("publish", issues)
		}
		if issues := validatePublishInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("publish", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := PublishRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			PRURL:      strings.TrimSpace(input.PRURL),
			Branch:     strings.TrimSpace(input.Branch),
			Base:       strings.TrimSpace(input.Base),
			Commit:     strings.TrimSpace(input.Commit),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		originalState := cloneState(state)
		statePath, err = updateStateAfterEvidence(s.Workdir, planStem, relPlanPath, state, statePath, kind, recordID, recordPath, now, func(next *runstate.State) {
			if record.Status == "recorded" {
				next.LatestPublish = &runstate.Publish{AttemptID: recordID, PRURL: record.PRURL}
				return
			}
			next.LatestPublish = nil
		})
		if err != nil {
			issues := rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
			issues = append([]CommandError{{Path: "state", Message: err.Error()}}, issues...)
			return errorResult("evidence submit", "Unable to update local harness state.", issues)
		}
		return s.finalizeMutation(successResult(planPath, statePath, kind, recordID, recordPath, "Recorded publish evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
		})
	case "sync":
		var input SyncInput
		if issues := inputschema.DecodeAndValidate("inputs.evidence.sync", "input", inputBytes, &input); len(issues) > 0 {
			return invalidInputIssuesResult("sync", issues)
		}
		if issues := validateSyncInput(input); len(issues) > 0 {
			return invalidInputIssuesResult("sync", issues)
		}
		recordID, recordPath, err := nextRecordLocation(s.Workdir, planStem, kind)
		if err != nil {
			return errorResult("evidence submit", "Unable to determine the next evidence record ID.", []CommandError{{Path: "record_id", Message: err.Error()}})
		}
		record := SyncRecord{
			RecordID:   recordID,
			Kind:       kind,
			PlanPath:   relPlanPath,
			PlanStem:   planStem,
			Revision:   runstate.CurrentRevision(state),
			RecordedAt: now,
			Status:     strings.ToLower(strings.TrimSpace(input.Status)),
			BaseRef:    strings.TrimSpace(input.BaseRef),
			HeadRef:    strings.TrimSpace(input.HeadRef),
			Reason:     strings.TrimSpace(input.Reason),
		}
		if err := writeJSONFile(recordPath, record); err != nil {
			return errorResult("evidence submit", "Unable to persist the evidence artifact.", []CommandError{{Path: "record", Message: err.Error()}})
		}
		originalState := cloneState(state)
		statePath, err = updateStateAfterEvidence(s.Workdir, planStem, relPlanPath, state, statePath, kind, recordID, recordPath, now, func(next *runstate.State) {
			switch record.Status {
			case "fresh":
				next.Sync = &runstate.SyncState{Freshness: "fresh", Conflicts: false}
			case "stale":
				next.Sync = &runstate.SyncState{Freshness: "stale", Conflicts: false}
			case "conflicted":
				next.Sync = &runstate.SyncState{Freshness: "stale", Conflicts: true}
			default:
				next.Sync = nil
			}
		})
		if err != nil {
			issues := rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
			issues = append([]CommandError{{Path: "state", Message: err.Error()}}, issues...)
			return errorResult("evidence submit", "Unable to update local harness state.", issues)
		}
		return s.finalizeMutation(successResult(planPath, statePath, kind, recordID, recordPath, "Recorded sync evidence for the current archived candidate."), func() []CommandError {
			return rollbackEvidenceMutation(s.Workdir, planStem, originalState, statePath, recordPath)
		})
	default:
		return errorResult("evidence submit", "Evidence kind is invalid.", []CommandError{{
			Path:    "kind",
			Message: "kind must be one of: ci, publish, sync",
		}})
	}
}

func LoadLatestCI(workdir string, state *runstate.State) (*CIRecord, error) {
	if state == nil {
		return nil, nil
	}
	if state.LatestEvidence != nil && state.LatestEvidence.CI != nil {
		return loadRecord[CIRecord](workdir, state.LatestEvidence.CI.Path)
	}
	if state.LatestCI != nil {
		return &CIRecord{
			RecordID: state.LatestCI.SnapshotID,
			Kind:     "ci",
			Status:   strings.ToLower(strings.TrimSpace(state.LatestCI.Status)),
		}, nil
	}
	return nil, nil
}

func LoadLatestPublish(workdir string, state *runstate.State) (*PublishRecord, error) {
	if state == nil {
		return nil, nil
	}
	if state.LatestEvidence != nil && state.LatestEvidence.Publish != nil {
		return loadRecord[PublishRecord](workdir, state.LatestEvidence.Publish.Path)
	}
	if state.LatestPublish != nil {
		return &PublishRecord{
			RecordID: state.LatestPublish.AttemptID,
			Kind:     "publish",
			Status:   "recorded",
			PRURL:    strings.TrimSpace(state.LatestPublish.PRURL),
		}, nil
	}
	return nil, nil
}

func LoadLatestSync(workdir string, state *runstate.State) (*SyncRecord, error) {
	if state == nil {
		return nil, nil
	}
	if state.LatestEvidence != nil && state.LatestEvidence.Sync != nil {
		return loadRecord[SyncRecord](workdir, state.LatestEvidence.Sync.Path)
	}
	if state.Sync != nil {
		status := strings.ToLower(strings.TrimSpace(state.Sync.Freshness))
		if state.Sync.Conflicts {
			status = "conflicted"
		}
		return &SyncRecord{
			RecordID: "legacy-sync",
			Kind:     "sync",
			Status:   status,
		}, nil
	}
	return nil, nil
}

func loadRecord[T any](workdir, relPath string) (*T, error) {
	if strings.TrimSpace(relPath) == "" {
		return nil, nil
	}
	path := filepath.Join(workdir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record T
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &record, nil
}

func validateCIInput(input CIInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "pending", "success", "failed":
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be pending, success, failed, or not_applied"}}
	}
}

func validatePublishInput(input PublishInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "recorded":
		if strings.TrimSpace(input.PRURL) == "" {
			return []CommandError{{Path: "input.pr_url", Message: "pr_url is required when status=recorded"}}
		}
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be recorded or not_applied"}}
	}
}

func validateSyncInput(input SyncInput) []CommandError {
	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case "fresh", "stale", "conflicted":
		return nil
	case "not_applied":
		if strings.TrimSpace(input.Reason) == "" {
			return []CommandError{{Path: "input.reason", Message: "reason is required when status=not_applied"}}
		}
		return nil
	default:
		return []CommandError{{Path: "input.status", Message: "status must be fresh, stale, conflicted, or not_applied"}}
	}
}

func nextRecordLocation(workdir, planStem, kind string) (string, string, error) {
	dir := filepath.Join(workdir, ".local", "harness", "plans", planStem, "evidence", kind)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", err
	}
	maxID := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := recordIDPattern.FindStringSubmatch(entry.Name())
		if matches == nil || matches[1] != kind {
			continue
		}
		n, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		if n > maxID {
			maxID = n
		}
	}
	recordID := fmt.Sprintf("%s-%03d", kind, maxID+1)
	return recordID, filepath.Join(dir, recordID+".json"), nil
}

func updateStateAfterEvidence(workdir, planStem, relPlanPath string, state *runstate.State, statePath, kind, recordID, recordPath, recordedAt string, mutate func(next *runstate.State)) (string, error) {
	if state == nil {
		state = &runstate.State{}
	}
	state.PlanPath = relPlanPath
	state.PlanStem = planStem
	if state.Revision <= 0 {
		state.Revision = 1
	}
	if state.LatestEvidence == nil {
		state.LatestEvidence = &runstate.EvidenceSet{}
	}
	relRecordPath, err := filepath.Rel(workdir, recordPath)
	if err != nil {
		return statePath, err
	}
	pointer := &runstate.EvidencePointer{
		Kind:       kind,
		RecordID:   recordID,
		Path:       filepath.ToSlash(relRecordPath),
		RecordedAt: recordedAt,
	}
	switch kind {
	case "ci":
		state.LatestEvidence.CI = pointer
	case "publish":
		state.LatestEvidence.Publish = pointer
	case "sync":
		state.LatestEvidence.Sync = pointer
	}
	mutate(state)
	return saveState(workdir, planStem, state)
}

func successResult(planPath, statePath, kind, recordID, recordPath, summary string) Result {
	return Result{
		OK:      true,
		Command: "evidence submit",
		Summary: summary,
		Artifacts: &Artifacts{
			PlanPath:       planPath,
			LocalStatePath: statePath,
			RecordID:       recordID,
			RecordPath:     recordPath,
			Kind:           kind,
		},
		NextAction: []NextAction{
			{Command: nil, Description: "Run harness status to refresh the archived candidate summary and next actions."},
		},
	}
}

func invalidInputResult(kind string, err error) Result {
	return errorResult("evidence submit", fmt.Sprintf("%s evidence input is invalid.", kind), []CommandError{{Path: "input", Message: err.Error()}})
}

func invalidInputIssuesResult(kind string, issues []CommandError) Result {
	return Result{
		OK:      false,
		Command: "evidence submit",
		Summary: fmt.Sprintf("%s evidence input is invalid.", kind),
		Errors:  issues,
	}
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

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0o644)
}

func (s Service) loadCurrentArchivedPlan() (string, string, string, *runstate.State, string, func(), *Result) {
	release := func() {}
	planPath, err := plan.DetectCurrentPath(s.Workdir)
	if err != nil {
		return "", "", "", nil, "", release, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	planStem := strings.TrimSuffix(filepath.Base(planPath), filepath.Ext(planPath))
	release, err = runstate.AcquireStateMutationLock(s.Workdir, planStem)
	if err != nil {
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Another local state mutation is already in progress.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}

	planPath, err = plan.DetectCurrentPathLocked(s.Workdir, planStem)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to determine the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	doc, err := plan.LoadFile(planPath)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read the current plan.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath, err := filepath.Rel(s.Workdir, planPath)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to relativize the current plan path.",
			Errors:  []CommandError{{Path: "plan", Message: err.Error()}},
		}
	}
	relPlanPath = filepath.ToSlash(relPlanPath)
	state, statePath, err := runstate.LoadState(s.Workdir, planStem)
	if err != nil {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Unable to read local harness state.",
			Errors:  []CommandError{{Path: "state", Message: err.Error()}},
		}
	}
	if doc.DerivedPlanStatus() != "archived" || doc.DerivedLifecycle(state) != "awaiting_merge_approval" {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Evidence commands require the current archived candidate.",
			Errors: []CommandError{{
				Path:    "plan.lifecycle",
				Message: fmt.Sprintf("current plan is status=%q lifecycle=%q", doc.DerivedPlanStatus(), doc.DerivedLifecycle(state)),
			}},
		}
	}
	if state != nil && (strings.TrimSpace(state.CurrentNode) == "land" ||
		(state.Land != nil &&
			strings.TrimSpace(state.Land.LandedAt) != "" &&
			strings.TrimSpace(state.Land.CompletedAt) == "")) {
		release()
		return "", "", "", nil, "", func() {}, &Result{
			OK:      false,
			Summary: "Evidence commands are not allowed after merge confirmation enters land cleanup.",
			Errors: []CommandError{{
				Path:    "state.current_node",
				Message: "current archived candidate is already in land cleanup",
			}},
		}
	}
	return planPath, relPlanPath, planStem, state, statePath, release, nil
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
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
	if state.LatestEvidence != nil {
		evidenceSet := *state.LatestEvidence
		cloned.LatestEvidence = &evidenceSet
		if state.LatestEvidence.CI != nil {
			ciPtr := *state.LatestEvidence.CI
			cloned.LatestEvidence.CI = &ciPtr
		}
		if state.LatestEvidence.Publish != nil {
			publishPtr := *state.LatestEvidence.Publish
			cloned.LatestEvidence.Publish = &publishPtr
		}
		if state.LatestEvidence.Sync != nil {
			syncPtr := *state.LatestEvidence.Sync
			cloned.LatestEvidence.Sync = &syncPtr
		}
	}
	if state.Land != nil {
		land := *state.Land
		cloned.Land = &land
	}
	if state.LatestCI != nil {
		ci := *state.LatestCI
		cloned.LatestCI = &ci
	}
	if state.Sync != nil {
		sync := *state.Sync
		cloned.Sync = &sync
	}
	if state.LatestPublish != nil {
		publish := *state.LatestPublish
		cloned.LatestPublish = &publish
	}
	return &cloned
}

func rollbackEvidenceMutation(workdir, planStem string, originalState *runstate.State, statePath, recordPath string) []CommandError {
	issues := make([]CommandError, 0, 2)
	if originalState != nil {
		if _, err := runstate.SaveState(workdir, planStem, originalState); err != nil {
			issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback local state: %v", err)})
		}
	} else if statePath != "" {
		if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
			issues = append(issues, CommandError{Path: "state", Message: fmt.Sprintf("rollback local state: %v", err)})
		}
	}
	if err := os.Remove(recordPath); err != nil && !os.IsNotExist(err) {
		issues = append(issues, CommandError{Path: "record", Message: fmt.Sprintf("rollback evidence artifact: %v", err)})
	}
	return issues
}
