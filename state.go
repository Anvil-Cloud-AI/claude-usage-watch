package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// alertState records, per window, the reset timestamp of the period we already
// alerted for. A window re-arms when its resets_at changes (a new period began)
// or its utilization drops back below the threshold.
type alertState map[string]string

// windowsToAlert returns the windows at or over threshold that have not been
// alerted for this period, plus the updated state. It does not mutate prev.
func windowsToAlert(windows []Window, threshold float64, prev alertState) ([]Window, alertState) {
	next := make(alertState, len(windows))
	var alerts []Window
	for _, w := range windows {
		if w.Utilization < threshold {
			continue
		}
		next[w.Name] = w.ResetsAt
		if seen, ok := prev[w.Name]; ok && seen == w.ResetsAt {
			continue
		}
		alerts = append(alerts, w)
	}
	return alerts, next
}

func statePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "claude-usage-watch", "state.json"), nil
}

func loadState(path string) (alertState, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return alertState{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s alertState
	if err := json.Unmarshal(b, &s); err != nil {
		return alertState{}, nil // corrupt state: start fresh rather than fail
	}
	return s, nil
}

func saveState(path string, s alertState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
