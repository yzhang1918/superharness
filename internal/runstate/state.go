package runstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type CurrentPlan struct {
	PlanPath string `json:"plan_path"`
}

type State struct {
	PlanPath          string       `json:"plan_path,omitempty"`
	PlanStem          string       `json:"plan_stem,omitempty"`
	ActiveReviewRound *ReviewRound `json:"active_review_round,omitempty"`
	LatestCI          *CIState     `json:"latest_ci,omitempty"`
	Sync              *SyncState   `json:"sync,omitempty"`
	LatestPublish     *Publish     `json:"latest_publish,omitempty"`
}

type ReviewRound struct {
	RoundID    string `json:"round_id"`
	Kind       string `json:"kind"`
	Aggregated bool   `json:"aggregated"`
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
