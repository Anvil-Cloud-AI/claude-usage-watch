package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// usageURL is the undocumented endpoint Claude Code's /usage command reads.
// It may change without notice.
const usageURL = "https://api.anthropic.com/api/oauth/usage"

const maxBodyBytes = 1 << 20

// Window is one rate-limit window, e.g. "five_hour" or "seven_day".
type Window struct {
	Name        string
	Utilization float64 // percent, 0-100
	ResetsAt    string  // RFC 3339 as returned by the API; empty if unknown
}

type rawWindow struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    *string  `json:"resets_at"`
}

func fetchUsage(ctx context.Context, client *http.Client, url, token string) ([]Window, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "claude-usage-watch")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request usage: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read usage response: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrTokenExpired
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usage endpoint returned HTTP %d", resp.StatusCode)
	}
	return parseUsage(body)
}

// parseUsage accepts any top-level object whose value has a numeric
// "utilization" field, so new windows the API adds are picked up automatically.
// Null windows (e.g. no Opus-specific limit on this plan) are skipped.
func parseUsage(body []byte) ([]Window, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(body, &top); err != nil {
		return nil, fmt.Errorf("parse usage response: %w", err)
	}

	windows := make([]Window, 0, len(top))
	for name, raw := range top {
		var rw rawWindow
		if json.Unmarshal(raw, &rw) != nil || rw.Utilization == nil {
			continue
		}
		w := Window{Name: name, Utilization: *rw.Utilization}
		if rw.ResetsAt != nil {
			w.ResetsAt = *rw.ResetsAt
		}
		windows = append(windows, w)
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("usage response contained no windows: %s", truncate(body, 200))
	}
	sort.Slice(windows, func(i, j int) bool { return windows[i].Name < windows[j].Name })
	return windows, nil
}

// label turns "seven_day_opus" into a human-readable name.
func (w Window) label() string {
	switch w.Name {
	case "five_hour":
		return "5-hour session"
	case "seven_day":
		return "Weekly (all models)"
	case "seven_day_opus":
		return "Weekly (Opus)"
	case "seven_day_sonnet":
		return "Weekly (Sonnet)"
	default:
		return w.Name
	}
}

// resetsIn formats the time until reset, or "" if unknown.
func (w Window) resetsIn(now time.Time) string {
	t, err := time.Parse(time.RFC3339, w.ResetsAt)
	if err != nil {
		return ""
	}
	d := t.Sub(now).Round(time.Minute)
	if d <= 0 {
		return "now"
	}
	h, m := int(d.Hours()), int(d.Minutes())%60
	if h >= 24 {
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
