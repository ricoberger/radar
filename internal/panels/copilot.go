package panels

import (
	"encoding/json"
	"os/exec"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

const (
	iconCopilotInProgress = "\uf04b"
	iconCopilotIdle       = "\uf059"
	iconCopilotQueued     = "\uf017"
	iconCopilotCompleted  = "\uf058"
	iconCopilotFailed     = "\uf057"
	iconCopilotCancelled  = "\uf05e"
	iconCopilotUnknown    = "\uf059"
)

// copilotStates is the set of valid Copilot agent-task session states.
var copilotStates = map[string]bool{
	"queued":      true,
	"in_progress": true,
	"idle":        true,
	"completed":   true,
	"failed":      true,
	"cancelled":   true,
}

// defaultCopilotStates mirrors the web "Sessions" sidebar, which hides
// terminal (cancelled) sessions.
var defaultCopilotStates = []string{"queued", "in_progress", "idle", "completed", "failed"}

// CopilotSession is one Copilot coding-agent (cloud) session from
// `gh agent-task list`.
type CopilotSession struct {
	ID        string
	Name      string
	State     string
	UpdatedAt string
}

// fetchCopilotSessions lists Copilot agent-task sessions, keeping only the
// requested states and the newest limit entries.
func fetchCopilotSessions(limit int, states map[string]bool) ([]CopilotSession, error) {
	var raw []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		State     string `json:"state"`
		UpdatedAt string `json:"updatedAt"`
	}
	if demo.Enabled() {
		return demoCopilotSessions(limit, states), nil
	}
	stdout, err := run(30*time.Second, "gh", "agent-task", "list",
		"--json", "id,name,state,updatedAt", "--limit", itoa(limit))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, err
	}
	items := make([]CopilotSession, 0, len(raw))
	for _, s := range raw {
		if !states[s.State] {
			continue
		}
		name := s.Name
		if name == "" {
			name = "-"
		}
		items = append(items, CopilotSession{
			ID:        s.ID,
			Name:      name,
			State:     s.State,
			UpdatedAt: s.UpdatedAt,
		})
	}
	return items, nil
}

// copilotStateStyle maps a session state to its icon and named color.
func copilotStateStyle(state string) (icon, color string) {
	switch state {
	case "in_progress":
		return iconCopilotInProgress, "blue"
	case "idle":
		return iconCopilotIdle, "yellow"
	case "queued":
		return iconCopilotQueued, "gray"
	case "completed":
		return iconCopilotCompleted, "green"
	case "failed":
		return iconCopilotFailed, "red"
	case "cancelled":
		return iconCopilotCancelled, "gray"
	default:
		return iconCopilotUnknown, "gray"
	}
}

// openCopilotSession opens a session in the browser via the gh CLI,
// detached and fire-and-forget.
func openCopilotSession(id string) {
	cmd := exec.Command("gh", "agent-task", "view", id, "--web") //nolint:noctx // Fire-and-forget, detached.
	_ = cmd.Start()
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}
}

type copilotPanel struct {
	base
	limit  int
	states map[string]bool
	items  []CopilotSession
}

func newCopilotPanel(fp config.FlatPanel, editor string) *copilotPanel {
	return &copilotPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		limit:  intParam(fp.Params, "limit", 50),
		states: copilotStatesParam(fp.Params),
	}
}

// copilotStatesParam reads the "states" filter, falling back to the default
// set that hides cancelled sessions.
func copilotStatesParam(params map[string]any) map[string]bool {
	names, ok := strSliceParam(params, "states")
	if !ok || len(names) == 0 {
		names = defaultCopilotStates
	}
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
}

func (p *copilotPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, limit, states := p.id, p.limit, p.states
	return func() tea.Msg {
		items, err := fetchCopilotSessions(limit, states)
		return ui.FetchMsg{ID: id, Data: items, Err: err}
	}
}

func (p *copilotPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.items = m.Data.([]CopilotSession)
		p.hasData = true
	}
	return nil
}

func (p *copilotPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, enter := p.list.Handle(msg.String(), len(p.items))
	if enter && len(p.items) > 0 {
		openCopilotSession(p.items[p.list.Clamp(len(p.items))].ID)
	}
	return nil
}

func (p *copilotPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.items) == 0 {
		content = line(w, dim("No sessions"))
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.items))
		}
		rows := make([]string, len(p.items))
		for i, s := range p.items {
			icon, color := copilotStateStyle(s.State)
			rows[i] = row(w, i == selected,
				colored(color, icon),
				plain(" "+s.Name+" ("+reltime(s.UpdatedAt)+")"),
			)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
