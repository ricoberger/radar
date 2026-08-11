package panels

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

const (
	iconUnread = "\uea71"
	iconRead   = "\uebb5"
)

// Notification is one item of the GitHub inbox.
type Notification struct {
	ID         string
	IsUnread   bool
	Reason     string
	Title      string
	Type       string
	Repository string
	UpdatedAt  string
	URL        string
	PRState    string
	IssueState string
	IsDraft    bool
	Conclusion string
}

var notificationRepo = regexp.MustCompile(`github\.com/([^/]+/[^/]+)`)

// fetchNotifications reads the inbox through the gh-notifications helper
// (GraphQL), which provides the state fields behind fzfgh's colored type
// icons.
func fetchNotifications(limit int) ([]Notification, error) {
	if demo.Enabled() {
		return demoNotifications(), nil
	}
	stdout, err := run(30*time.Second, "gh-notifications", "list")
	if err != nil {
		return nil, err
	}
	var raw []struct {
		ID            string `json:"id"`
		IsUnread      bool   `json:"isUnread"`
		Reason        string `json:"reason"`
		Title         string `json:"title"`
		LastUpdatedAt string `json:"lastUpdatedAt"`
		URL           string `json:"url"`
		Subject       struct {
			Typename         string `json:"__typename"`
			PullRequestState string `json:"pullRequestState"`
			IssueState       string `json:"issueState"`
			IsDraft          bool   `json:"isDraft"`
			Conclusion       string `json:"conclusion"`
		} `json:"subject"`
	}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, err
	}
	if len(raw) > limit {
		raw = raw[:limit]
	}
	items := make([]Notification, 0, len(raw))
	for _, n := range raw {
		title := n.Title
		if title == "" {
			title = "-"
		}
		repo := "-"
		if m := notificationRepo.FindStringSubmatch(n.URL); m != nil {
			repo = m[1]
		}
		items = append(items, Notification{
			ID:         n.ID,
			IsUnread:   n.IsUnread,
			Reason:     strings.ReplaceAll(strings.ToLower(n.Reason), "_", " "),
			Title:      title,
			Type:       n.Subject.Typename,
			Repository: repo,
			UpdatedAt:  n.LastUpdatedAt,
			URL:        n.URL,
			PRState:    n.Subject.PullRequestState,
			IssueState: n.Subject.IssueState,
			IsDraft:    n.Subject.IsDraft,
			Conclusion: n.Subject.Conclusion,
		})
	}
	return items, nil
}

// typeIcon is fzfgh's type icon + color mapping.
func typeIcon(n Notification) (icon, color string) {
	switch n.Type {
	case "PullRequest":
		color := "gray"
		switch {
		case n.IsDraft:
			color = "gray"
		case n.PRState == "OPEN":
			color = "green"
		case n.PRState == "MERGED":
			color = "magenta"
		case n.PRState == "CLOSED":
			color = "red"
		}
		return "\uf407", color
	case "Issue":
		color := "gray"
		switch n.IssueState {
		case "OPEN":
			color = "green"
		case "CLOSED":
			color = "magenta"
		}
		return "\uf41b", color
	case "Release":
		return "\uf412", "gray"
	case "CheckSuite":
		if n.Conclusion == "SUCCESS" {
			return "\uf52e", "green"
		}
		return "\uf52e", "red"
	case "WorkflowRun":
		return "\uf52e", "red"
	case "Commit":
		return "\uf417", "gray"
	case "Gist":
		return "\uf480", "gray"
	case "Discussion", "TeamDiscussion":
		return "\uf442", "gray"
	default:
		return "\uf128", "gray"
	}
}

type githubNotificationsPanel struct {
	base
	limit int
	open  string
	items []Notification
}

func newGithubNotificationsPanel(fp config.FlatPanel, editor string) *githubNotificationsPanel {
	return &githubNotificationsPanel{
		base:  newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		limit: intParam(fp.Params, "limit", 50),
		open:  readOpenParam(fp.Params),
	}
}

func (p *githubNotificationsPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, limit := p.id, p.limit
	return func() tea.Msg {
		items, err := fetchNotifications(limit)
		return ui.FetchMsg{ID: id, Data: items, Err: err}
	}
}

func (p *githubNotificationsPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.items = m.Data.([]Notification)
		p.hasData = true
	}
	return nil
}

func (p *githubNotificationsPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, enter := p.list.Handle(msg.String(), len(p.items))
	if enter {
		return openGithubItem(p.items[p.list.Clamp(len(p.items))].URL, p.open)
	}
	return nil
}

func (p *githubNotificationsPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.items) == 0 {
		content = line(w, dimColored("green", "No notifications ✓"))
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.items))
		}
		rows := make([]string, len(p.items))
		for i, n := range p.items {
			statusIcon, statusColor := iconRead, "gray"
			if n.IsUnread {
				statusIcon, statusColor = iconUnread, "yellow"
			}
			icon, color := typeIcon(n)
			rows[i] = row(w, i == selected,
				colored(statusColor, statusIcon),
				plain(" "),
				colored(color, icon),
				plain(" ["+n.Type+"] "+n.Repository+": "+n.Title+
					" ("+n.Reason+" - "+reltime(n.UpdatedAt)+")"),
			)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
