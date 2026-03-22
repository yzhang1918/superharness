package runstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CurrentPlan struct {
	PlanPath           string `json:"plan_path,omitempty"`
	LastLandedPlanPath string `json:"last_landed_plan_path,omitempty"`
	LastLandedAt       string `json:"last_landed_at,omitempty"`
}

type State struct {
	ExecutionStartedAt string       `json:"execution_started_at,omitempty"`
	CurrentNode        string       `json:"current_node,omitempty"`
	PlanPath           string       `json:"plan_path,omitempty"`
	PlanStem           string       `json:"plan_stem,omitempty"`
	Revision           int          `json:"revision,omitempty"`
	Reopen             *ReopenState `json:"reopen,omitempty"`
	ActiveReviewRound  *ReviewRound `json:"active_review_round,omitempty"`
	LatestEvidence     *EvidenceSet `json:"latest_evidence,omitempty"`
	Land               *LandState   `json:"land,omitempty"`

	// Transitional cache fields retained until status fully stops reading v0.1
	// handoff signals directly from state.json.
	LatestCI      *CIState   `json:"latest_ci,omitempty"`
	Sync          *SyncState `json:"sync,omitempty"`
	LatestPublish *Publish   `json:"latest_publish,omitempty"`
}

type ReopenState struct {
	Mode          string `json:"mode"`
	ReopenedAt    string `json:"reopened_at,omitempty"`
	BaseStepCount int    `json:"base_step_count,omitempty"`
}

type ReviewRound struct {
	RoundID    string `json:"round_id"`
	Kind       string `json:"kind"`
	Trigger    string `json:"trigger,omitempty"`
	Aggregated bool   `json:"aggregated"`
	Decision   string `json:"decision,omitempty"`
}

type EvidenceSet struct {
	CI      *EvidencePointer `json:"ci,omitempty"`
	Publish *EvidencePointer `json:"publish,omitempty"`
	Sync    *EvidencePointer `json:"sync,omitempty"`
}

type EvidencePointer struct {
	Kind       string `json:"kind"`
	RecordID   string `json:"record_id"`
	Path       string `json:"path"`
	RecordedAt string `json:"recorded_at,omitempty"`
}

type LandState struct {
	PRURL       string `json:"pr_url,omitempty"`
	Commit      string `json:"commit,omitempty"`
	LandedAt    string `json:"landed_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type CIState struct {
	SnapshotID string `json:"snapshot_id"`
	Status     string `json:"status"`
}

type SyncState struct {
	Freshness string `json:"freshness"`
	Conflicts bool   `json:"conflicts"`
}

type Publish struct {
	AttemptID string `json:"attempt_id"`
	PRURL     string `json:"pr_url"`
}

type reviewAggregate struct {
	Decision string `json:"decision"`
}

type reviewManifest struct {
	Trigger string `json:"trigger"`
	Target  string `json:"target"`
}

func LoadCurrentPlan(workdir string) (*CurrentPlan, error) {
	path := filepath.Join(workdir, ".local", "harness", "current-plan.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var current CurrentPlan
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, fmt.Errorf("parse current-plan.json: %w", err)
	}
	return &current, nil
}

func SaveCurrentPlan(workdir, planPath string) (string, error) {
	return saveCurrentPlan(workdir, CurrentPlan{PlanPath: planPath})
}

func SaveLandedPlan(workdir, planPath, landedAt string) (string, error) {
	return saveCurrentPlan(workdir, CurrentPlan{
		LastLandedPlanPath: planPath,
		LastLandedAt:       landedAt,
	})
}

func saveCurrentPlan(workdir string, current CurrentPlan) (string, error) {
	path := filepath.Join(workdir, ".local", "harness", "current-plan.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal current-plan.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func LoadState(workdir, planStem string) (*State, string, error) {
	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, path, nil
		}
		return nil, path, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, path, fmt.Errorf("parse state.json: %w", err)
	}
	return &state, path, nil
}

func SaveState(workdir, planStem string, state *State) (string, error) {
	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal state.json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func CurrentRevision(state *State) int {
	if state == nil || state.Revision <= 0 {
		return 1
	}
	return state.Revision
}

func EffectiveReviewDecision(workdir, planStem string, round *ReviewRound) (string, bool, error) {
	if round == nil {
		return "", false, nil
	}
	if decision := strings.TrimSpace(round.Decision); decision != "" {
		return decision, true, nil
	}
	if !round.Aggregated || strings.TrimSpace(round.RoundID) == "" {
		return "", false, nil
	}

	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", round.RoundID, "aggregate.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read aggregate.json for %s: %w", round.RoundID, err)
	}

	var aggregate reviewAggregate
	if err := json.Unmarshal(data, &aggregate); err != nil {
		return "", false, fmt.Errorf("parse aggregate.json for %s: %w", round.RoundID, err)
	}
	if decision := strings.TrimSpace(aggregate.Decision); decision != "" {
		return decision, true, nil
	}
	return "", false, nil
}

func EffectiveReviewTrigger(workdir, planStem string, round *ReviewRound) (string, bool, error) {
	if round == nil {
		return "", false, nil
	}
	if trigger := strings.TrimSpace(round.Trigger); trigger != "" {
		return trigger, true, nil
	}
	if strings.TrimSpace(round.RoundID) == "" {
		return "", false, nil
	}

	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", round.RoundID, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read manifest.json for %s: %w", round.RoundID, err)
	}

	var manifest reviewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", false, fmt.Errorf("parse manifest.json for %s: %w", round.RoundID, err)
	}
	if trigger := strings.TrimSpace(manifest.Trigger); trigger != "" {
		return trigger, true, nil
	}
	return "", false, nil
}

func EffectiveReviewTarget(workdir, planStem string, round *ReviewRound) (string, bool, error) {
	if round == nil || strings.TrimSpace(round.RoundID) == "" {
		return "", false, nil
	}

	path := filepath.Join(workdir, ".local", "harness", "plans", planStem, "reviews", round.RoundID, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read manifest.json for %s: %w", round.RoundID, err)
	}

	var manifest reviewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", false, fmt.Errorf("parse manifest.json for %s: %w", round.RoundID, err)
	}
	if target := strings.TrimSpace(manifest.Target); target != "" {
		return target, true, nil
	}
	return "", false, nil
}
