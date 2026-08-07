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
`bin/calendar-helper`. The first run of the calendar panel triggers a macOS prompt to
grant your terminal access to your calendars.

## Keybindings

| Key               | Action                                        |
| ----------------- | --------------------------------------------- |
| `[`/`]`           | previous / next dashboard                     |
| `1`-`9`           | focus panel                                   |
| `tab`/`shift-tab` | cycle focus                                   |
| `j`/`k`           | select item / scroll                          |
| `g`/`G`           | first / last item                             |
| `enter`           | open item in native app / edit the daily note |
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
dailyNotesDir: ~/Documents/GitHub/ricoberger/notes/daily
alertmanagerUrl: http://127.0.0.1:9093
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
                - panel: calendar
                - panel: note
            - weight: 1
              panel: alerts
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
        - panel: mail
```

`dashboards` is a list of named dashboards shown as tabs at the top and switched with
`[` / `]`; the tab bar is hidden when only one dashboard is defined. Only the active
dashboard's panels are refreshed — switching back shows cached data instantly and
refetches only panels whose refresh interval has elapsed.

A layout node is either a split (`direction` + `children`) or a panel. Every node accepts
`weight` (flex ratio, default `1`); panel nodes additionally accept `title`, `interval`
(refresh in seconds) and type-specific `params`.

### Panel types

| Type                   | Params                | Description                                 |
| ---------------------- | --------------------- | ------------------------------------------- |
| `calendar`             | –                     | Today's events from all Apple calendars     |
| `note`                 | –                     | Today's daily note, scrollable              |
| `mail`                 | –                     | Unread messages across all Mail.app inboxes |
| `alerts`               | `filter` (name)       | Alerts from an Alertmanager.app filter      |
| `github-prs`           | `query`, `limit` (20) | Pull requests from a `gh search prs` query  |
| `github-notifications` | `limit` (50)          | Unread GitHub notifications                 |

Panels keep their last data when a refresh fails and show the error in the header.
