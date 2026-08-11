package panels

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/ui"
)

const (
	iconPR    = "\uf407"
	iconIssue = "\uf41b"
)

// githubSearchPanel implements both the github-prs and github-issues panels.
type githubSearchPanel struct {
	base
	kind  string // "prs" or "issues"
	icon  string
	empty string
	query string
	limit int
	open  string
	items []SearchItem
}

func newGithubSearchPanel(fp config.FlatPanel, editor, kind, icon, empty string) *githubSearchPanel {
	return &githubSearchPanel{
		base:  newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		kind:  kind,
		icon:  icon,
		empty: empty,
		query: strParam(fp.Params, "query", ""),
		limit: intParam(fp.Params, "limit", 20),
		open:  readOpenParam(fp.Params),
	}
}

func (p *githubSearchPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, kind, query, limit := p.id, p.kind, p.query, p.limit
	return func() tea.Msg {
		items, err := searchGithub(kind, query, limit)
		return ui.FetchMsg{ID: id, Data: items, Err: err}
	}
}

func (p *githubSearchPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.items = m.Data.([]SearchItem)
		p.hasData = true
	}
	return nil
}

func (p *githubSearchPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, enter := p.list.Handle(msg.String(), len(p.items))
	if enter {
		return openGithubItem(p.items[p.list.Clamp(len(p.items))].URL, p.open)
	}
	return nil
}

func (p *githubSearchPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.items) == 0 {
		content = line(w, dim(p.empty))
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.items))
		}
		rows := make([]string, len(p.items))
		for i, item := range p.items {
			rows[i] = row(w, i == selected,
				colored(searchItemColor(item), p.icon),
				plain(fmt.Sprintf(" [#%d] %s: %s  (%s · %s)",
					item.Number, item.Repository, item.Title,
					item.Author, reltime(item.CreatedAt))),
			)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
