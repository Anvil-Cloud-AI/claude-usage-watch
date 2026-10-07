// Command claude-usage-watch polls your Claude plan usage and shows a macOS
// notification when any limit window crosses a threshold.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultThreshold = 90.0
	defaultInterval  = 5 * time.Minute
	minInterval      = time.Minute
	requestTimeout   = 30 * time.Second
)

func main() {
	threshold := flag.Float64("threshold", defaultThreshold, "alert when a window's utilization reaches this percent")
	interval := flag.Duration("interval", defaultInterval, "how often to check (minimum 1m)")
	once := flag.Bool("once", false, "print current usage and exit")
	testNotify := flag.Bool("test-notify", false, "send a test notification and exit")
	flag.Parse()

	log.SetFlags(log.LstdFlags)

	if *threshold <= 0 || *threshold > 100 {
		log.Fatalf("-threshold must be in (0, 100], got %v", *threshold)
	}
	if *interval < minInterval {
		log.Fatalf("-interval must be at least %v", minInterval)
	}

	if *testNotify {
		if err := notify("Claude usage", "Test notification from claude-usage-watch"); err != nil {
			log.Fatal(err)
		}
		return
	}

	client := &http.Client{Timeout: requestTimeout}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *once {
		windows, err := check(ctx, client)
		if err != nil {
			log.Fatal(err)
		}
		printWindows(windows, time.Now())
		return
	}

	if err := watch(ctx, client, *threshold, *interval); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func check(ctx context.Context, client *http.Client) ([]Window, error) {
	token, err := loadToken(time.Now())
	if err != nil {
		return nil, err
	}
	return fetchUsage(ctx, client, usageURL, token)
}

func watch(ctx context.Context, client *http.Client, threshold float64, interval time.Duration) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	state, err := loadState(path)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	log.Printf("watching Claude usage every %v, alerting at %.0f%%", interval, threshold)
	expiredNotified := false
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		windows, err := check(ctx, client)
		switch {
		case errors.Is(err, ErrTokenExpired):
			log.Print(err)
			if !expiredNotified {
				expiredNotified = true
				if nerr := notify("Claude usage watch", "Token expired. Open Claude Code to refresh it."); nerr != nil {
					log.Print(nerr)
				}
			}
		case err != nil:
			log.Printf("check failed: %v", err) // transient; retry next tick
		default:
			expiredNotified = false
			state = handleWindows(windows, threshold, state, path)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func handleWindows(windows []Window, threshold float64, prev alertState, path string) alertState {
	now := time.Now()
	alerts, next := windowsToAlert(windows, threshold, prev)
	for _, w := range alerts {
		msg := fmt.Sprintf("%s at %.0f%%", w.label(), w.Utilization)
		if r := w.resetsIn(now); r != "" {
			msg += fmt.Sprintf(", resets in %s", r)
		}
		log.Print("ALERT: " + msg)
		if err := notify("Claude usage high", msg); err != nil {
			log.Print(err)
		}
	}
	if err := saveState(path, next); err != nil {
		log.Printf("save state: %v", err)
	}
	return next
}

func printWindows(windows []Window, now time.Time) {
	for _, w := range windows {
		line := fmt.Sprintf("%-22s %5.1f%%", w.label(), w.Utilization)
		if r := w.resetsIn(now); r != "" {
			line += "  (resets in " + r + ")"
		}
		fmt.Println(line)
	}
}
