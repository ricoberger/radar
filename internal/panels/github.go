package panels

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

func itoa(n int) string { return strconv.Itoa(n) }

// Shared helpers for the github-prs, github-issues and github-notifications
// panels, mirroring fzfgh's list rendering and view action.

func readOpenParam(params map[string]any) string {
	if params["open"] == "fzfgh" {
		return "fzfgh"
	}
	return "browser"
}

func validateOpenParam(params map[string]any, trail string) error {
	if v, ok := params["open"]; ok && v != "browser" && v != "fzfgh" {
		return errf(`%s: "params.open" must be "browser" or "fzfgh"`, trail)
	}
	return nil
}

var githubItemURL = regexp.MustCompile(`github\.com/[^/]+/[^/]+/(pull|issues)/\d+`)

// openGithubItem opens a GitHub URL either in the browser or inside fzfgh.
// fzfgh's view command only supports pull request and issue URLs; anything
// else falls back to the browser.
func openGithubItem(url, mode string) tea.Cmd {
	if mode == "fzfgh" && githubItemURL.MatchString(url) {
		return func() tea.Msg {
			return ui.ExecMsg{Cmd: exec.Command("fzfgh", "view", url)}
		}
	}
	openExternal(url)
	return nil
}

// reltime is fzfgh's relative time: "just now", "5m ago", ..., "2w ago",
// then "Mar 05".
func reltime(iso string) string {
	date, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return ""
	}
	seconds := int(time.Since(date).Seconds())
	switch {
	case seconds < 60:
		return "just now"
	case seconds < 3600:
		return itoa(seconds/60) + "m ago"
	case seconds < 86400:
		return itoa(seconds/3600) + "h ago"
	case seconds < 604800:
		return itoa(seconds/86400) + "d ago"
	case seconds < 2592000:
		return itoa(seconds/604800) + "w ago"
	}
	return date.UTC().Format("Jan 02")
}

// SearchItem is a pull request or issue from gh search.
type SearchItem struct {
	Number     int
	Title      string
	Repository string
	Author     string
	State      string
	IsDraft    bool
	CreatedAt  string
	URL        string
}

func searchGithub(kind, query string, limit int) ([]SearchItem, error) {
	if demo.Enabled() {
		return demoGithubSearch(kind), nil
	}
	fields := []string{
		"number", "title", "repository", "author", "state", "createdAt", "url",
	}
	if kind == "prs" {
		fields = append(fields, "isDraft")
	}
	args := []string{
		"search", kind, "--json", strings.Join(fields, ","),
		"--limit", itoa(limit),
	}
	args = append(args, strings.Fields(query)...)
	stdout, err := run(30*time.Second, "gh", args...)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Number     int    `json:"number"`
		Title      string `json:"title"`
		Repository struct {
			NameWithOwner string `json:"nameWithOwner"`
		} `json:"repository"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
		State     string `json:"state"`
		IsDraft   bool   `json:"isDraft"`
		CreatedAt string `json:"createdAt"`
		URL       string `json:"url"`
	}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0, len(raw))
	for _, item := range raw {
		author := item.Author.Login
		if author == "" {
			author = "-"
		}
		items = append(items, SearchItem{
			Number:     item.Number,
			Title:      item.Title,
			Repository: item.Repository.NameWithOwner,
			Author:     author,
			State:      strings.ToLower(item.State),
			IsDraft:    item.IsDraft,
			CreatedAt:  item.CreatedAt,
			URL:        item.URL,
		})
	}
	return items, nil
}

// searchItemColor is fzfgh's icon color: draft = gray, open = green,
// closed = mauve.
func searchItemColor(item SearchItem) string {
	if item.IsDraft {
		return "gray"
	}
	if item.State == "closed" {
		return "magenta"
	}
	return "green"
}

func validateGithubSearchParams(panelType string) func(map[string]any, string) error {
	return func(params map[string]any, trail string) error {
		query, ok := params["query"].(string)
		if !ok || strings.TrimSpace(query) == "" {
			return errf(`%s: "params.query" is required for %s panels`, trail, panelType)
		}
		return validateOpenParam(params, trail)
	}
}
