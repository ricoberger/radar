# radar

Personal TUI radar built with [Ink](https://github.com/vadimdemedes/ink) — everything on
your radar in one terminal: Apple Calendar events, Apple Mail unread messages, the daily
note, Alertmanager.app alerts and GitHub pull requests / notifications in configurable
dashboards.

## Requirements

- Node.js >= 20 and `swiftc` (Xcode toolchain) for the calendar helper
- `gh` (authenticated) for the GitHub panels
- [Alertmanager.app](https://github.com/ricoberger/Alertmanager) running for the alerts panel
- Mail.app running for the mail panel

## Install

```sh
npm install
npm link
mkdir -p ~/.config/radar && cp config.yaml ~/.config/radar/config.yaml
radar
```

To try it without linking, run `npm run dev` — it uses the `config.yaml` from this
repository.

The build compiles the TypeScript sources to `dist/` and the Swift EventKit helper to
`bin/apple-calendar-helper`. The first run of the apple-calendar panel triggers a macOS prompt to
grant your terminal access to your calendars.

## Keybindings

| Key               | Action                                        |
| ----------------- | --------------------------------------------- |
| `[`/`]`           | previous / next dashboard                     |
| `1`-`9`           | focus panel                                   |
| `tab`/`shift-tab` | cycle focus                                   |
| `j`/`k`           | select item / scroll                          |
| `g`/`G`           | first / last item                             |
| `enter`           | open / view item, edit the daily note         |
| `z`               | zoom focused panel                            |
| `r`/`R`           | refresh focused / all panels                  |
| `?`               | help                                          |
| `q`               | quit                                          |

## Configuration

A config file is required — without one the application exits with an error. It is read
from the first of: the path given as argument (`radar <config.yaml>`), the
`RADAR_CONFIG` environment variable, or `~/.config/radar/config.yaml`. Only
`dashboards` is required, the remaining keys have defaults:

```yaml
editor: nvim # defaults to $EDITOR

dashboards:
  - name: Main
    layout:
      direction: row
      children:
        - weight: 3
          direction: column
          children:
            - weight: 3
              direction: row
              children:
                - panel: apple-calendar
                - panel: ricoberger-notes
                  params:
                    dir: ~/Documents/GitHub/ricoberger/notes/daily
            - weight: 1
              panel: alertmanager
              interval: 30
              params:
                filter: team-core
        - weight: 1
          direction: column
          children:
            - panel: github-prs
              title: Created PRs
              params:
                query: 'author:@me is:open'
            - panel: github-prs
              title: PRs Review Requested
              params:
                query: 'review-requested:@me is:open'
            - panel: github-notifications
              weight: 2
  - name: Mail
    layout:
      children:
        - panel: apple-mail
  - name: Week
    layout:
      panel: apple-calendar
      params:
        view: week
```

`dashboards` is a list of named dashboards shown as tabs at the top and switched with
`[` / `]`; the tab bar is hidden when only one dashboard is defined. Only the active
dashboard's panels are refreshed — switching back shows cached data instantly and
refetches only panels whose refresh interval has elapsed.

A layout node is either a split (`direction` + `children`) or a panel. Every node accepts
`weight` (flex ratio, default `1`); panel nodes additionally accept `title`, `interval`
(refresh in seconds) and type-specific `params`.

### Panel types

| Type                   | Params                                    | Description                                |
| ---------------------- | ----------------------------------------- | ------------------------------------------ |
| `apple-calendar`       | `day`, `view`                             | Events from all Apple calendars            |
| `ricoberger-notes`     | `dir` (required), `day`                   | A daily note, rendered with `md`           |
| `apple-mail`           | `messages`, `limit`, `include`, `exclude` | Messages from the Mail.app inboxes         |
| `alertmanager`         | `url`, `filter` / `alertmanager`          | Alerts from Alertmanager.app               |
| `github-prs`           | `query`, `limit` (20)                     | Pull requests from a `gh search prs` query |
| `github-notifications` | `limit` (50)                              | Unread GitHub notifications                |

The apple-calendar panel shows a single day by default: `day` selects `yesterday`, `today`
(default) or `tomorrow`. With `view: week` it shows the full week (Monday–Sunday,
today highlighted, multi-day events repeated under every day they cover) and `day` is
ignored. The default title reflects the configuration (e.g. `Calendar · Tomorrow`,
`Calendar · Week`); an explicit `title` always wins. Events that are currently
running are shown in green.

The ricoberger-notes panel shows a daily note from `dir` (layout
`<dir>/YYYY/MM/YYYY-MM-DD.md`, created from `<dir>/template.md`): `day` selects
`today` (default) or `yesterday`. The note is rendered as Markdown with the
[`md`](https://github.com/ricoberger/md) binary if it is on the `PATH`
(YAML frontmatter is stripped), otherwise as plain text. Enter opens the note
in the configured editor; yesterday's note is never created retroactively.

The apple-mail panel shows the newest `limit` (default `10`) messages across the
inboxes of all accounts as `sender · subject · age · account`. `messages` selects
`unread` (default) or `all`. `include` / `exclude` are lists of account names (as
shown in Mail.app) to fetch from; if both are set they are ignored and all accounts
are fetched.

The alertmanager panel shows the alerts of one
[Alertmanager.app](https://github.com/ricoberger/Alertmanager) filter or
alertmanager: exactly one of `filter` / `alertmanager` (the name as shown in
the app) is required. `url` sets the API base URL (default
`http://127.0.0.1:9093`). Each row shows the analysis state (green = analysis
exists, yellow = running), severity, name, summary, age and alertmanager;
suppressed alerts are dimmed. Keys on the selected alert: `enter` view the
alert Markdown in the editor, `o`/`s`/`b`/`d`/`p` open the source / silence /
runbook / dashboard / panel link in the browser (ignored when the alert has no
such link), `a` open the finished analysis in the editor or start a new run,
`y` copy the source URL to the clipboard.

Panels keep their last data when a refresh fails and show the error in the header.
