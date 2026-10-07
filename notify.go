package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// notify shows a macOS notification via AppleScript.
func notify(title, message string) error {
	script := fmt.Sprintf("display notification %s with title %s sound name \"Glass\"",
		appleScriptString(message), appleScriptString(title))
	if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// appleScriptString quotes s as an AppleScript string literal.
func appleScriptString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(s) + `"`
}
