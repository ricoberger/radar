// Package panels implements the radar panel types. Each panel owns its data
// (which doubles as the session cache), fetches it from external tools and
// renders its own rows.
package panels

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/ui"
)

// base carries the state shared by all panels: identity, fetch bookkeeping,
// frame metadata, size and list selection.
type base struct {
	id          string
	index       int
	title       string
	interval    time.Duration
	editor      string
	hasData     bool
	err         error
	inFlight    bool
	lastUpdated time.Time
	lastAttempt time.Time
	width       int
	height      int
	list        ui.List
}

func newBase(id string, index int, title string, intervalSeconds int, editor string) base {
	return base{
		id:       id,
		index:    index,
		title:    title,
		interval: time.Duration(intervalSeconds) * time.Second,
		editor:   editor,
	}
}

func (b *base) ID() string { return b.id }

func (b *base) Due(now time.Time) bool {
	return !b.inFlight && now.Sub(b.lastAttempt) >= b.interval
}

func (b *base) Activate() { b.lastAttempt = b.lastUpdated }

func (b *base) Resize(width, height int) tea.Cmd {
	b.width, b.height = width, height
	return nil
}

// beginFetch marks the panel in-flight; callers must have checked inFlight.
func (b *base) beginFetch() {
	b.inFlight = true
	b.lastAttempt = time.Now()
}

// applyMeta consumes the fetch bookkeeping of a result and reports whether
// the fetch succeeded (stale data is kept on errors).
func (b *base) applyMeta(msg ui.FetchMsg) bool {
	b.inFlight = false
	if msg.Err != nil {
		b.err = msg.Err
		return false
	}
	b.err = nil
	b.lastUpdated = time.Now()
	return true
}

// contentSize returns the size of the area inside the frame and header.
func (b *base) contentSize() (w, h int) {
	return b.width - 4, b.height - 4
}

func (b *base) frame(content string, focused bool) string {
	return ui.Frame(b.width, b.height, ui.FrameState{
		Index:       b.index,
		Title:       b.title,
		Loading:     b.inFlight,
		LastUpdated: b.lastUpdated,
		Err:         b.err,
	}, focused, content)
}

// editorCmd builds the external editor invocation for a file.
func (b *base) editorCmd(file string) *exec.Cmd {
	return execCmd(strings.Fields(b.editor), file)
}

// execCmd builds a command from pre-split fields plus a trailing argument.
func execCmd(fields []string, arg string) *exec.Cmd {
	if len(fields) == 0 {
		fields = []string{"vim"}
	}
	return exec.Command(fields[0], append(fields[1:], arg)...)
}

// run executes a command and returns stdout; on failure the error is the
// first line of stderr (or the exec error message), like the Ink app.
func run(timeout time.Duration, command string, args ...string) (string, error) {
	return runWithInput(timeout, "", nil, command, args...)
}

// runWithInput is run with stdin content and extra environment variables.
func runWithInput(timeout time.Duration, input string, env []string, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		detail := strings.SplitN(strings.TrimSpace(stderr.String()), "\n", 2)[0]
		if detail == "" {
			detail = err.Error()
		}
		return "", errors.New(detail)
	}
	return stdout.String(), nil
}

// openExternal opens a URL with the macOS open command, detached.
func openExternal(url string) {
	cmd := exec.Command("open", url)
	_ = cmd.Start()
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
}

// pbcopy puts text on the clipboard, ignoring errors.
func pbcopy(text string) {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run()
}

// Param helpers tolerating the loose YAML typing of the Ink app.

func strParam(params map[string]any, key, def string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return def
}

func intParam(params map[string]any, key string, def int) int {
	switch v := params[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return def
}

func boolParam(params map[string]any, key string, def bool) bool {
	if v, ok := params[key].(bool); ok {
		return v
	}
	return def
}

func strSliceParam(params map[string]any, key string) ([]string, bool) {
	raw, ok := params[key].([]any)
	if !ok {
		return nil, params[key] == nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// collapseSpaces collapses all whitespace runs into single spaces and trims.
func collapseSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func errf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
