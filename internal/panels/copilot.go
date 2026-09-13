package panels

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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

// copilotStates is the set of valid Copilot agent-task states.
var copilotStates = map[string]bool{
	"queued":           true,
	"in_progress":      true,
	"idle":             true,
	"waiting_for_user": true,
	"completed":        true,
	"failed":           true,
	"timed_out":        true,
	"cancelled":        true,
}

// defaultCopilotStates mirrors the web "Agents" view
// (github.com/copilot/agents), which lists every non-archived task
// regardless of state.
var defaultCopilotStates = []string{
	"queued", "in_progress", "idle", "waiting_for_user",
	"completed", "failed", "timed_out", "cancelled",
}

// CopilotSession is one Copilot coding-agent (cloud) task from the
// GitHub "agent tasks" API.
type CopilotSession struct {
	ID        string
	Name      string
	State     string
	UpdatedAt string
	RepoID    int64
}

// fetchCopilotSessions lists Copilot agent tasks via the GitHub API,
// excluding archived tasks (like the web UI) and keeping only the requested
// states and the newest limit entries.
func fetchCopilotSessions(limit int, states map[string]bool) ([]CopilotSession, error) {
	if demo.Enabled() {
		return demoCopilotSessions(limit, states), nil
	}
	perPage := limit
	if perPage > 100 {
		perPage = 100
	}
	if perPage < 1 {
		perPage = 1
	}
	// The gh CLI cannot filter archived tasks, so query the underlying API
	// directly. is_archived=false matches the web UI, which hides archived
	// tasks.
	path := fmt.Sprintf(
		"/agents/tasks?per_page=%d&is_archived=false&sort=updated_at&direction=desc",
		perPage,
	)
	if s := copilotStateQuery(states); s != "" {
		path += "&state=" + s
	}
	stdout, err := run(30*time.Second, "gh", "api", path)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Tasks []struct {
			ID         string  `json:"id"`
			Name       string  `json:"name"`
			State      string  `json:"state"`
			UpdatedAt  string  `json:"updated_at"`
			ArchivedAt *string `json:"archived_at"`
			Repository struct {
				ID int64 `json:"id"`
			} `json:"repository"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
		return nil, err
	}
	items := make([]CopilotSession, 0, len(resp.Tasks))
	for _, t := range resp.Tasks {
		if t.ArchivedAt != nil {
			continue
		}
		if len(states) > 0 && !states[t.State] {
			continue
		}
		name := t.Name
		if name == "" {
			name = "-"
		}
		items = append(items, CopilotSession{
			ID:        t.ID,
			Name:      name,
			State:     t.State,
			UpdatedAt: t.UpdatedAt,
			RepoID:    t.Repository.ID,
		})
	}
	return items, nil
}

// copilotStateQuery joins the requested states into the comma-separated value
// expected by the agent-tasks API "state" filter.
func copilotStateQuery(states map[string]bool) string {
	if len(states) == 0 {
		return ""
	}
	names := make([]string, 0, len(states))
	for n := range states {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// copilotStateStyle maps a task state to its icon and named color.
func copilotStateStyle(state string) (icon, color string) {
	switch state {
	case "in_progress":
		return iconCopilotInProgress, "blue"
	case "idle", "waiting_for_user":
		return iconCopilotIdle, "yellow"
	case "queued":
		return iconCopilotQueued, "gray"
	case "completed":
		return iconCopilotCompleted, "green"
	case "failed", "timed_out":
		return iconCopilotFailed, "red"
	case "cancelled":
		return iconCopilotCancelled, "gray"
	default:
		return iconCopilotUnknown, "gray"
	}
}

// openCopilotTask resolves the task's repository name and opens the task in the
// browser. The agent-tasks list API only returns the repository id, so the
// name is looked up on demand.
func openCopilotTask(repoID int64, taskID string) {
	full, err := run(30*time.Second, "gh", "api",
		fmt.Sprintf("/repositories/%d", repoID), "-q", ".full_name")
	if err != nil {
		return
	}
	full = strings.TrimSpace(full)
	if full == "" {
		return
	}
	openExternal("https://github.com/" + full + "/tasks/" + taskID)
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
	if enter && len(p.items) > 0 && !demo.Enabled() {
		it := p.items[p.list.Clamp(len(p.items))]
		return func() tea.Msg {
			openCopilotTask(it.RepoID, it.ID)
			return nil
		}
	}
	return nil
}

func (p *copilotPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.items) == 0 {
		content = line(w, dim("No agents"))
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
