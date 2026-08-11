package ui

import (
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Panel is a self-contained dashboard panel: it owns its data (which doubles
// as the session cache), fetches it on demand and renders itself at the size
// assigned by the layout engine.
type Panel interface {
	ID() string
	// Due reports whether a periodic fetch should be dispatched now.
	Due(now time.Time) bool
	// Activate re-anchors the fetch schedule to the last successful fetch,
	// mirroring the Ink remount behavior on dashboard switches.
	Activate()
	// Fetch marks the panel in-flight and returns a command producing a
	// FetchMsg. Returns nil while a fetch is already running.
	Fetch() tea.Cmd
	// Apply consumes a panel-scoped message (fetch result, render result)
	// and may return a follow-up command.
	Apply(msg PanelMsg) tea.Cmd
	// Resize stores the panel's frame size and may return a command (e.g.
	// re-rendering width-dependent content).
	Resize(width, height int) tea.Cmd
	// HandleKey processes a key while the panel is focused.
	HandleKey(msg tea.KeyPressMsg) tea.Cmd
	// View renders the framed panel at the last size set by Resize.
	View(focused bool) string
}

// PanelMsg is a message addressed to a single panel.
type PanelMsg interface {
	PanelID() string
}

// FetchMsg carries the result of a panel fetch.
type FetchMsg struct {
	ID   string
	Data any
	Err  error
}

func (m FetchMsg) PanelID() string { return m.ID }

// ForceRefreshMsg asks the app to fetch a panel immediately, bypassing the
// interval (r/R keys, after external actions).
type ForceRefreshMsg struct {
	ID string
}

func (m ForceRefreshMsg) PanelID() string { return m.ID }

// ExecMsg asks the app to hand the terminal to an external command (editor,
// fzfgh) and to deliver After once it exits.
type ExecMsg struct {
	Cmd   *exec.Cmd
	After tea.Msg
}

// heartbeatMsg drives the 1s tick used for age display and fetch scheduling.
type heartbeatMsg time.Time
