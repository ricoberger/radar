# AGENTS.md

Guidance for AI agents working on this repository.

## Project

radar is a personal TUI dashboard for macOS built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) v2 and
[Lip Gloss](https://github.com/charmbracelet/lipgloss) v2. It renders
configurable dashboards with panels for Apple Calendar, Apple Mail, a daily
note, Alertmanager alerts, GitHub pull requests / issues / notifications, Jira
work items, Kubernetes issues and HTTP checks.

- Runtime: Go (see `go.mod`). Releases cover macOS, Linux and Windows; the
  apple-calendar and apple-mail panels only work on macOS.
- Note: bubbletea/lipgloss v2 use the `charm.land/...` module paths.
- No tests. Verification is done by building and running the app in tmux (see
  below).

## Commands

| Command                         | Purpose                         |
| ------------------------------- | ------------------------------- |
| `go build -o radar .`           | Build the `radar` binary        |
| `go run . --config config.yaml` | Run with the repo `config.yaml` |
| `go run . --demo`               | Run with built-in demo data     |
| `gofmt -w .`                    | Format                          |

Always run `go build ./...` after changing source files and fix all compiler
errors. Run `gofmt -w .` before committing.

## Architecture

```
main.go              Entry point: parses --config/--demo flags, loads config,
                     embeds the Swift calendar helper source (go:embed),
                     builds panel instances, runs the Bubble Tea program
internal/
  config/config.go   Config loading/validation (YAML). Path: --config →
                     $RADAR_CONFIG → ~/.config/radar/config.yaml. Missing
                     config = error + exit 1. LayoutNode tree, Registry of
                     panel defaults, FlattenPanels
  demo/demo.go       --demo mode: module-level flag and built-in demo config
  ui/
    app.go           Root model: dashboard tabs, global keymap, weighted
                     row/column layout renderer, zoom, footer, 1s heartbeat
                     tick that triggers due panel fetches; external commands
                     run via tea.ExecProcess
    panel.go         Panel interface (ID, Due, Activate, Fetch, Apply, Resize,
                     HandleKey, View) and panel-addressed messages (FetchMsg,
                     ForceRefreshMsg, ExecMsg)
    frame.go         Panel frame: rounded border, header with index/title,
                     right-aligned status (loading/error/age), underline
    list.go          List selection state, j/k/g/G/enter handling, centered
                     scroll window math
    theme.go         Colors (mauve accent, named ANSI colors), style helpers
  panels/            One file per panel type; each contains BOTH data fetching
                     and rendering. registry.go maps panel type name →
                     defaults + factory. base.go holds shared panel state
                     (fetch cadence, stale-data-on-error, exec helpers,
                     param readers); render.go the segment/row builders;
                     demo.go all demo fixtures
swift/               apple-calendar-helper.swift (EventKit); source is
                     embedded in the binary and compiled with swiftc on first
                     use into ~/Library/Caches/radar, keyed by source hash
```

Key design decisions:

- Panels are self-contained: data fetching lives in the panel file, not in a
  separate data layer.
- Panel ids are namespaced per dashboard: `d{i}:p{n}:{type}`.
- Panel instances live for the whole session; their state is the cache. Only
  the active dashboard fetches (a 1s heartbeat asks each panel whether a fetch
  is due). Switching back re-uses the data if it is fresher than the panel
  interval.
- Fetches run as tea.Cmds and return panel-addressed messages implementing
  `PanelMsg`; the app routes them to the owning panel via `Apply`.
- Global keys are handled in ui/app.go first; unhandled keys go to the focused
  panel's `HandleKey`.
- External data comes from CLIs / local APIs: `gh` (GitHub search),
  `gh-notifications` (GitHub inbox), `acli` (Jira), `kubectl issues`
  (Kubernetes), AppleScript via `osascript` (Mail), the compiled Swift helper
  (Calendar), HTTP on `127.0.0.1:9093` (Alertmanager.app), the filesystem
  (daily note).

## Configuration

Users must provide a YAML config (see `config.yaml` for the reference config,
and README.md for docs). It defines a `dashboards` list, each with
a `name` and a `layout` tree of `row` / `column` / panel nodes. When changing
config semantics, update `internal/config/config.go` validation,
`internal/panels/registry.go`, `config.yaml` and README.md together.

## Verifying changes

Run the TUI in tmux and inspect the output:

```sh
go build -o radar .
tmux new-session -d -s radar-test -x 200 -y 50 './radar --config config.yaml'
sleep 3
tmux capture-pane -t radar-test -p        # add -e to keep colors
tmux send-keys -t radar-test 'q'          # quit; also: kill-session
```

Kill leftover sessions/processes before re-testing. Panels depend on the local
machine (Mail, Calendar permissions, gh auth, Alertmanager.app); an error
message inside a panel frame is expected when a backend is unavailable — the
app itself must still render. `./radar --demo` runs every panel with built-in
fake data and no external dependencies (except `md` for the notes panel), which
is useful for screenshots and layout checks.

## Conventions

- gofmt is the source of truth for style; `golangci-lint run ./...` (see
  `.golangci.yaml`) must be clean.
- Keep README.md in sync with keybinding, panel and config changes.
- Commits follow Conventional Commits.
- Do not commit the `radar` binary (gitignored).
