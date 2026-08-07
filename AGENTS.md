# AGENTS.md

Guidance for AI agents working on this repository.

## Project

radar is a personal TUI dashboard for macOS built with
[Ink](https://github.com/vadimdemedes/ink) (React for terminals). It renders
configurable dashboards with panels for Apple Calendar, Apple Mail, a daily
note, Alertmanager alerts, GitHub pull requests / issues / notifications and
Jira work items.

- Runtime: Node.js >= 20, ESM only (`"type": "module"`), macOS only.
- Language: TypeScript with `NodeNext` module resolution — relative imports
  use the `.js` extension of the compiled output (e.g.
  `import { x } from './utils.js'`), never `.ts`. Do not "fix" this.
- No test framework. Verification is done by building and running the app in
  tmux (see below).

## Commands

| Command          | Purpose                                                  |
| ---------------- | -------------------------------------------------------- |
| `npm run build`  | Compile TypeScript to `dist/` and build the Swift helper |
| `npm run dev`    | Build, then run with the repo `config.yaml`              |
| `npm run format` | Prettier (also organizes imports)                        |

Always run `npm run build` after changing source files and fix all compiler
errors. Run `npm run format` before committing.

## Architecture

```
src/
  index.tsx     Entry point: loads config, alt-screen handling, renders <App/>
  app.tsx       Root component: dashboard tabs, global keymap, layout renderer,
                help overlay, zoom, footer
  config.ts     Config loading/validation (YAML). Path: argv[2] → $RADAR_CONFIG
                → ~/.config/radar/config.yaml. Missing config = error + exit 1
  types.ts      Config / layout / panel type definitions
  store.ts      Module-level session state: panel data cache, focus, key
                registry. Survives remounts (e.g. editor suspend/resume)
  context.ts    React context (focus, zoom, screen size)
  utils.ts      exec helpers, time formatting
  components/   PanelFrame (border + title), SelectList
  hooks/
    usePanelData.ts  Interval fetching with cache-age-aware initial delay,
                     stale-data-on-error
    usePanelKeys.ts  Key handler registration; useListNavigation adds j/k/g/G/
                     enter and an optional onKey fallback for panel-specific
                     keys
    useScreenSize.ts
  panels/       One file per panel type; each contains BOTH data fetching and
                rendering. registry.ts maps panel type name → component
swift/          apple-calendar-helper.swift (EventKit); compiled by
                scripts/build-apple-calendar-helper.sh into gitignored bin/
```

Key design decisions:

- Panels are self-contained: data fetching lives in the panel file, not in a
  separate data layer.
- Panel ids are namespaced per dashboard: `d{i}:p{n}:{type}`.
- Only the active dashboard mounts and fetches; switching back re-uses the
  cache if it is fresher than the panel interval.
- One key handler per panel id (keyRegistry). Global keys are handled in
  app.tsx first; unhandled keys go to the focused panel.
- External data comes from CLIs / local APIs: `gh` (GitHub search),
  `gh-notifications` (GitHub inbox), `acli` (Jira), AppleScript via
  `osascript` (Mail), the compiled Swift helper (Calendar), HTTP on
  `127.0.0.1:9093` (Alertmanager.app), the filesystem (daily note).

## Configuration

Users must provide a YAML config (see `config.yaml` for the reference used by
`npm run dev`, and README.md for docs). It defines a `dashboards` list, each
with a `name` and a `layout` tree of `row` / `column` / panel nodes. When
changing config semantics, update `config.ts` validation, `types.ts`,
`config.yaml` and README.md together.

## Verifying changes

Run the TUI in tmux and inspect the output:

```sh
tmux new-session -d -s radar-test -x 200 -y 50 'node dist/index.js config.yaml'
sleep 3
tmux capture-pane -t radar-test -p        # add -e to keep colors
tmux send-keys -t radar-test 'q'          # quit; also: kill-session
```

Kill leftover sessions/processes before re-testing. Panels depend on the local
machine (Mail, Calendar permissions, gh auth, Alertmanager.app); an error
message inside a panel frame is expected when a backend is unavailable — the
app itself must still render.

## Conventions

- Prettier is the source of truth for style (`.prettierrc`, 80 cols, single
  quotes, trailing commas). Import organization must stay compatible with the
  TypeScript `source.organizeImports` action: builtin/external imports first,
  blank line, then project imports; sorted within groups.
- react-jsx transform: never `import React` for JSX; use
  `import { type ReactNode } from 'react'` style type imports.
- Keep README.md in sync with keybinding, panel and config changes.
- Commits follow Conventional Commits.
- Do not commit `dist/`, `bin/` or `node_modules/` (gitignored).
