package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const keychainService = "Claude Code-credentials"

// ErrTokenExpired means Claude Code's stored OAuth token has lapsed. We never
// refresh it ourselves: doing so would rotate the refresh token out from under
// Claude Code. Opening Claude Code refreshes it.
var ErrTokenExpired = errors.New("Claude Code OAuth token has expired; open Claude Code to refresh it")

type credentialsFile struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"` // unix millis
	} `json:"claudeAiOauth"`
}

// loadToken reads Claude Code's OAuth access token from the macOS Keychain,
// falling back to ~/.claude/.credentials.json (used on Linux).
func loadToken(now time.Time) (string, error) {
	raw, err := readKeychain()
	if err != nil {
		raw, err = readCredentialsFile()
		if err != nil {
			return "", fmt.Errorf("no Claude Code credentials found (Keychain or ~/.claude/.credentials.json): %w", err)
		}
	}
	return parseToken(raw, now)
}

func parseToken(raw []byte, now time.Time) (string, error) {
	var creds credentialsFile
	if err := json.Unmarshal(raw, &creds); err != nil {
		return "", fmt.Errorf("parse credentials: %w", err)
	}
	tok := creds.ClaudeAiOauth.AccessToken
	if tok == "" {
		return "", errors.New("credentials contain no claudeAiOauth.accessToken")
	}
	if exp := creds.ClaudeAiOauth.ExpiresAt; exp > 0 && now.After(time.UnixMilli(exp)) {
		return "", ErrTokenExpired
	}
	return tok, nil
}

func readKeychain() ([]byte, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", keychainService, "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("keychain lookup: %w", err)
	}
	return []byte(strings.TrimSpace(string(out))), nil
}

func readCredentialsFile() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(home, ".claude", ".credentials.json")) // #nosec G304 -- fixed path under $HOME
}
