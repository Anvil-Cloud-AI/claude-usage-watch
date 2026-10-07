package main

import (
	"errors"
	"testing"
	"time"
)

func TestParseUsageSkipsNullAndNonWindowFields(t *testing.T) {
	body := []byte(`{
		"five_hour": {"utilization": 42.5, "resets_at": "2026-10-07T12:00:00Z"},
		"seven_day": {"utilization": 91, "resets_at": "2026-10-10T00:00:00Z"},
		"seven_day_opus": null,
		"extra_usage": {"is_enabled": false}
	}`)

	got, err := parseUsage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 windows, got %d: %+v", len(got), got)
	}
	if got[0].Name != "five_hour" || got[0].Utilization != 42.5 {
		t.Errorf("unexpected first window: %+v", got[0])
	}
	if got[1].Name != "seven_day" || got[1].ResetsAt != "2026-10-10T00:00:00Z" {
		t.Errorf("unexpected second window: %+v", got[1])
	}
}

func TestParseUsageErrorsWhenNoWindows(t *testing.T) {
	if _, err := parseUsage([]byte(`{"error": "nope"}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := parseUsage([]byte(`not json`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestWindowsToAlertOncePerPeriod(t *testing.T) {
	high := []Window{{Name: "five_hour", Utilization: 95, ResetsAt: "A"}}

	alerts, state := windowsToAlert(high, 90, alertState{})
	if len(alerts) != 1 {
		t.Fatalf("first crossing should alert, got %d", len(alerts))
	}

	alerts, state = windowsToAlert(high, 90, state)
	if len(alerts) != 0 {
		t.Fatalf("same period should not re-alert, got %d", len(alerts))
	}

	newPeriod := []Window{{Name: "five_hour", Utilization: 92, ResetsAt: "B"}}
	alerts, _ = windowsToAlert(newPeriod, 90, state)
	if len(alerts) != 1 {
		t.Fatalf("new period should alert, got %d", len(alerts))
	}
}

func TestWindowsToAlertRearmsAfterDroppingBelow(t *testing.T) {
	prev := alertState{"five_hour": "A"}
	low := []Window{{Name: "five_hour", Utilization: 10, ResetsAt: "A"}}

	alerts, state := windowsToAlert(low, 90, prev)
	if len(alerts) != 0 || len(state) != 0 {
		t.Fatalf("below threshold: want no alerts and cleared state, got %v %v", alerts, state)
	}
	if _, ok := prev["five_hour"]; !ok {
		t.Fatal("prev state must not be mutated")
	}
}

func TestParseToken(t *testing.T) {
	now := time.UnixMilli(1_000_000)

	tok, err := parseToken([]byte(`{"claudeAiOauth":{"accessToken":"abc","expiresAt":2000000}}`), now)
	if err != nil || tok != "abc" {
		t.Fatalf("got %q, %v", tok, err)
	}

	_, err = parseToken([]byte(`{"claudeAiOauth":{"accessToken":"abc","expiresAt":500000}}`), now)
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired, got %v", err)
	}

	if _, err = parseToken([]byte(`{}`), now); err == nil {
		t.Fatal("want error for missing token")
	}
}

func TestResetsIn(t *testing.T) {
	now := time.Date(2026, 10, 7, 10, 0, 0, 0, time.UTC)
	cases := map[string]string{
		"2026-10-07T12:30:00Z": "2h 30m",
		"2026-10-09T13:00:00Z": "2d 3h",
		"2026-10-07T09:00:00Z": "now",
		"":                     "",
	}
	for in, want := range cases {
		if got := (Window{ResetsAt: in}).resetsIn(now); got != want {
			t.Errorf("resetsIn(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAppleScriptString(t *testing.T) {
	if got := appleScriptString(`say "hi" \ bye`); got != `"say \"hi\" \\ bye"` {
		t.Errorf("got %s", got)
	}
}
