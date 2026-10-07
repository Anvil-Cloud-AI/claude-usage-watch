# claude-usage-watch

Polls your Claude plan usage (the numbers behind Claude Code's `/usage` and claude.ai's usage page) and shows a macOS notification when any limit window, such as the 5-hour session or the weekly limit, crosses a threshold (default 90%). Each window alerts once per reset period.

## How it works

- Reads the OAuth token Claude Code already stores in the macOS Keychain (`Claude Code-credentials`). If that entry is missing, it falls back to `~/.claude/.credentials.json`.
- Calls `GET https://api.anthropic.com/api/oauth/usage`. **This endpoint is undocumented** and may change or break without notice.
- The program never refreshes the token itself. That would rotate the refresh token and could sign Claude Code out. If the token expires, you get one notification asking you to open Claude Code, which refreshes it.

## Usage

```sh
go run . -once          # print current usage and exit
go run . -test-notify   # check that notifications show up
make test

make install            # build to ~/.local/bin and start a launchd agent at login
make install THRESHOLD=85 INTERVAL=10m
make status
make logs
make uninstall
```

The first time it runs, macOS may ask whether `claude-usage-watch` can access the Keychain item. Choose **Always Allow**. Alert state is stored in `~/Library/Caches/claude-usage-watch/state.json`.

## Notifications

Notifications are sent with `osascript`, so they appear under **Script Editor** in System Settings → Notifications. If you don't see them, make sure notifications are enabled there.

## Requirements

macOS and Go 1.22 or newer. Build with a current, patched Go release: this tool makes HTTPS requests, so it relies on the standard library's TLS fixes.

## License

[MIT](LICENSE). Free to use, modify and redistribute.

Not affiliated with or endorsed by Anthropic.
