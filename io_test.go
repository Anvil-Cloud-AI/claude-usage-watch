package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestFetchUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("anthropic-beta") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"five_hour":{"utilization":12,"resets_at":null}}`))
	}))
	defer srv.Close()
	ctx := context.Background()

	got, err := fetchUsage(ctx, srv.Client(), srv.URL, "good")
	if err != nil || len(got) != 1 || got[0].Utilization != 12 {
		t.Fatalf("got %+v, %v", got, err)
	}

	if _, err := fetchUsage(ctx, srv.Client(), srv.URL, "bad"); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("want ErrTokenExpired on 401, got %v", err)
	}
}

func TestFetchUsageServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	if _, err := fetchUsage(context.Background(), srv.Client(), srv.URL, "x"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.json")

	s, err := loadState(path)
	if err != nil || len(s) != 0 {
		t.Fatalf("missing file should load empty state, got %v, %v", s, err)
	}

	want := alertState{"five_hour": "2026-10-07T12:00:00Z"}
	if err := saveState(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadState(path)
	if err != nil || got["five_hour"] != want["five_hour"] {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestLabels(t *testing.T) {
	for name, want := range map[string]string{
		"five_hour": "5-hour session", "seven_day": "Weekly (all models)",
		"seven_day_opus": "Weekly (Opus)", "seven_day_sonnet": "Weekly (Sonnet)", "other": "other",
	} {
		if got := (Window{Name: name}).label(); got != want {
			t.Errorf("label(%q) = %q, want %q", name, got, want)
		}
	}
}
