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
make status
make logs
make uninstall
```

## Changing the threshold

The alert threshold defaults to **90%**. It can be any value above 0 and up to 100. One threshold applies to every limit window.

**Background service:** run `make install` again with the new value. This rebuilds the binary and replaces the running agent, so nothing needs to be uninstalled first:

```sh
make install THRESHOLD=80               # alert at 80%
make install THRESHOLD=75 INTERVAL=2m   # alert at 75%, check every 2 minutes
```

Running plain `make install` resets both settings to their defaults (90% and 5m).

To see what the installed agent is using:

```sh
grep -A1 -E 'threshold|interval' ~/Library/LaunchAgents/com.anvilcloud.claude-usage-watch.plist
```

**Running it directly:** pass the flags yourself:

```sh
go run . -threshold 80 -interval 2m
```

| Flag | Make variable | Default | Notes |
|---|---|---|---|
| `-threshold` | `THRESHOLD` | `90` | Percent; must be above 0 and at most 100 |
| `-interval` | `INTERVAL` | `5m` | Go duration (`90s`, `2m`, `1h`); minimum `1m` |

Each window alerts once per reset period. If you lower the threshold partway through a period in which you were already alerted, that window won't alert again until it resets.

## Keychain access and state

The first time it runs, macOS may ask whether `claude-usage-watch` can access the Keychain item. Choose **Always Allow**. Alert state is stored in `~/Library/Caches/claude-usage-watch/state.json`.

## Notifications

Notifications are sent with `osascript`, so they appear under **Script Editor** in System Settings → Notifications. If you don't see them, make sure notifications are enabled there.

## Requirements

macOS and Go 1.22 or newer. Build with a current, patched Go release: this tool makes HTTPS requests, so it relies on the standard library's TLS fixes.

## License

[MIT](LICENSE). Free to use, modify and redistribute.

Not affiliated with or endorsed by Anthropic.
