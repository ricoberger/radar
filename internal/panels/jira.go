package panels

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

const jiraDefaultFlagField = "customfield_10002"

type jiraParams struct {
	jql       string
	limit     int
	flagField string
}

// WorkItem is a Jira work item row.
type WorkItem struct {
	Key         string
	Status      string
	StatusColor string
	Summary     string
}

func readJiraParams(params map[string]any) jiraParams {
	return jiraParams{
		jql:       strParam(params, "jql", ""),
		limit:     intParam(params, "limit", 50),
		flagField: strParam(params, "flagField", jiraDefaultFlagField),
	}
}

func validateJiraParams(params map[string]any, trail string) error {
	jql, ok := params["jql"].(string)
	if !ok || strings.TrimSpace(jql) == "" {
		return errf(`%s: "params.jql" is required for jira panels`, trail)
	}
	if v, ok := params["flagField"]; ok {
		if _, isStr := v.(string); !isStr {
			return errf(`%s: "params.flagField" must be a string`, trail)
		}
	}
	return nil
}

// jiraStatusColor is fzfjira's status color, keyed by the status category
// color name.
func jiraStatusColor(colorName string) string {
	switch colorName {
	case "green":
		return "green"
	case "yellow", "brown":
		return "yellow"
	case "blue-gray":
		return "blue"
	case "warm-red":
		return "red"
	default:
		return "white"
	}
}

func jiraClean(s string) string {
	if s == "" {
		return "-"
	}
	return clean(s)
}

type jiraNamed struct {
	Name string `json:"name"`
}

type jiraUser struct {
	DisplayName string `json:"displayName"`
}

type jiraIssueRef struct {
	Key    string `json:"key"`
	Fields *struct {
		Summary   string     `json:"summary"`
		Issuetype *jiraNamed `json:"issuetype"`
		Status    *jiraNamed `json:"status"`
	} `json:"fields"`
}

type jiraWorkItem struct {
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields"`
}

type jiraFields struct {
	Summary string `json:"summary"`
	Status  *struct {
		Name           string `json:"name"`
		StatusCategory *struct {
			ColorName string `json:"colorName"`
		} `json:"statusCategory"`
	} `json:"status"`
	Issuetype *jiraNamed `json:"issuetype"`
	Priority  *jiraNamed `json:"priority"`
	Assignee  *jiraUser  `json:"assignee"`
	Reporter  *jiraUser  `json:"reporter"`
	Labels    []string   `json:"labels"`
	Parent    *jiraIssueRef
	Created   string `json:"created"`
	Updated   string `json:"updated"`
	Comment   *struct {
		Total    int `json:"total"`
		Comments []struct {
			Author  *jiraUser       `json:"author"`
			Created string          `json:"created"`
			Body    json.RawMessage `json:"body"`
		} `json:"comments"`
	} `json:"comment"`
	Subtasks   []jiraIssueRef `json:"subtasks"`
	Issuelinks []struct {
		Type *struct {
			Name    string `json:"name"`
			Inward  string `json:"inward"`
			Outward string `json:"outward"`
		} `json:"type"`
		OutwardIssue *jiraIssueRef `json:"outwardIssue"`
		InwardIssue  *jiraIssueRef `json:"inwardIssue"`
	} `json:"issuelinks"`
	Description json.RawMessage `json:"description"`
}

func fetchWorkItems(p jiraParams) ([]WorkItem, error) {
	if demo.Enabled() {
		return demoJira(), nil
	}
	stdout, err := run(30*time.Second, "acli",
		"jira", "workitem", "search",
		"--jql", p.jql, "--limit", itoa(p.limit), "--json")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stdout) == "" {
		return []WorkItem{}, nil
	}
	var raw []jiraWorkItem
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return nil, err
	}
	// Server order is preserved (like fzfjira).
	items := make([]WorkItem, 0, len(raw))
	for _, item := range raw {
		var f jiraFields
		_ = json.Unmarshal(item.Fields, &f)
		status, colorName := "-", "medium-gray"
		if f.Status != nil {
			if f.Status.Name != "" {
				status = f.Status.Name
			}
			if f.Status.StatusCategory != nil && f.Status.StatusCategory.ColorName != "" {
				colorName = f.Status.StatusCategory.ColorName
			}
		}
		items = append(items, WorkItem{
			Key:         item.Key,
			Status:      status,
			StatusColor: jiraStatusColor(colorName),
			Summary:     jiraClean(f.Summary),
		})
	}
	return items, nil
}

// adfToMd converts a Jira ADF document to Markdown text via md's raw mode,
// falling back to a short note when md is not installed (like fzfjira).
func adfToMd(adf json.RawMessage) string {
	out, err := runWithInput(20*time.Second, string(adf),
		[]string{"MD_ADF=true", "MD_RAW=true"}, "md")
	if err != nil {
		return "_(install md to render this content)_"
	}
	return strings.TrimRight(out, " \t\n\r")
}

func fmtdate(s string) string {
	if s == "" {
		return ""
	}
	if len(s) > 16 {
		s = s[:16]
	}
	return strings.Replace(s, "T", " ", 1)
}

func refStatus(ref *jiraIssueRef) string {
	if ref != nil && ref.Fields != nil && ref.Fields.Status != nil &&
		ref.Fields.Status.Name != "" {
		return ref.Fields.Status.Name
	}
	return "-"
}

func refSummary(ref *jiraIssueRef) string {
	if ref != nil && ref.Fields != nil {
		return jiraClean(ref.Fields.Summary)
	}
	return "-"
}

func refKey(ref *jiraIssueRef) string {
	if ref != nil && ref.Key != "" {
		return ref.Key
	}
	return "-"
}

// emitMarkdown assembles the whole work item as a single Markdown document,
// mirroring fzfjira's emit_markdown: metadata, description, comments,
// sub-tasks, links.
func emitMarkdown(wi jiraWorkItem, flagField string) string {
	var f jiraFields
	_ = json.Unmarshal(wi.Fields, &f)
	var extra map[string]any
	_ = json.Unmarshal(wi.Fields, &extra)

	var lines []string
	push := func(ls ...string) { lines = append(lines, ls...) }

	push(fmt.Sprintf("# %s  %s", wi.Key, jiraClean(f.Summary)), "")
	push(fmt.Sprintf("- **Key:** `%s`", wi.Key))
	status := "-"
	if f.Status != nil && f.Status.Name != "" {
		status = f.Status.Name
	}
	push("- **Status:** " + status)
	if f.Issuetype != nil && f.Issuetype.Name != "" {
		push("- **Type:** " + f.Issuetype.Name)
	}
	if f.Priority != nil && f.Priority.Name != "" {
		push("- **Priority:** " + f.Priority.Name)
	}
	if f.Assignee != nil && f.Assignee.DisplayName != "" {
		push("- **Assignee:** " + f.Assignee.DisplayName)
	}
	if f.Reporter != nil && f.Reporter.DisplayName != "" {
		push("- **Reporter:** " + f.Reporter.DisplayName)
	}
	if len(f.Labels) > 0 {
		labels := make([]string, len(f.Labels))
		for i, label := range f.Labels {
			labels[i] = "`" + label + "`"
		}
		push("- **Labels:** " + strings.Join(labels, " "))
	}
	if f.Parent != nil {
		kind := "Parent"
		if f.Parent.Fields != nil && f.Parent.Fields.Issuetype != nil &&
			f.Parent.Fields.Issuetype.Name == "Epic" {
			kind = "Epic"
		}
		push(fmt.Sprintf("- **%s:** `%s` · %s",
			kind, refKey(f.Parent), refSummary(f.Parent)))
	}
	if flagsRaw, ok := extra[flagField].([]any); ok && len(flagsRaw) > 0 {
		values := make([]string, 0, len(flagsRaw))
		for _, flag := range flagsRaw {
			value := ""
			if m, ok := flag.(map[string]any); ok {
				value, _ = m["value"].(string)
			}
			values = append(values, value)
		}
		push("- **Flagged:** " + strings.Join(values, ", "))
	}
	if f.Created != "" {
		push("- **Created:** " + fmtdate(f.Created))
	}
	if f.Updated != "" {
		push("- **Updated:** " + fmtdate(f.Updated))
	}
	push("", "---", "")

	if isJSONObject(f.Description) {
		push(adfToMd(f.Description))
	} else {
		push("_No description provided._")
	}
	push("")

	if f.Comment != nil && len(f.Comment.Comments) > 0 {
		comments := f.Comment.Comments
		total := f.Comment.Total
		if total == 0 {
			total = len(comments)
		}
		if len(comments) < total {
			push(fmt.Sprintf("# Comments (showing %d of %d)", len(comments), total), "")
		} else {
			push(fmt.Sprintf("# Comments (%d)", total), "")
		}
		for _, comment := range comments {
			author := "?"
			if comment.Author != nil && comment.Author.DisplayName != "" {
				author = comment.Author.DisplayName
			}
			push(fmt.Sprintf("**%s** · %s", author, fmtdate(comment.Created)), "")
			push(adfToMd(comment.Body), "")
		}
	}

	if len(f.Subtasks) > 0 {
		push(fmt.Sprintf("# Sub-tasks (%d)", len(f.Subtasks)), "")
		for i := range f.Subtasks {
			subtask := &f.Subtasks[i]
			push(fmt.Sprintf("- `%s` · %s · %s",
				refKey(subtask), refStatus(subtask), refSummary(subtask)))
		}
		push("")
	}

	var links []string
	for _, link := range f.Issuelinks {
		issue := link.OutwardIssue
		dir := ""
		if issue != nil {
			if link.Type != nil {
				dir = link.Type.Outward
				if dir == "" {
					dir = link.Type.Name
				}
			}
		} else {
			issue = link.InwardIssue
			if link.Type != nil {
				dir = link.Type.Inward
				if dir == "" {
					dir = link.Type.Name
				}
			}
		}
		if issue == nil {
			continue
		}
		if dir == "" {
			dir = "-"
		}
		links = append(links, fmt.Sprintf("- %s · `%s` · %s · %s",
			dir, refKey(issue), refStatus(issue), refSummary(issue)))
	}
	if len(links) > 0 {
		push(fmt.Sprintf("# Links (%d)", len(links)), "")
		push(links...)
		push("")
	}

	doc := strings.Join(lines, "\n")
	return strings.TrimRight(doc, "\n") + "\n"
}

func isJSONObject(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return strings.HasPrefix(trimmed, "{")
}

type jiraPanel struct {
	base
	params jiraParams
	items  []WorkItem
}

func newJiraPanel(fp config.FlatPanel, editor string) *jiraPanel {
	return &jiraPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readJiraParams(fp.Params),
	}
}

func (p *jiraPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		items, err := fetchWorkItems(params)
		return ui.FetchMsg{ID: id, Data: items, Err: err}
	}
}

func (p *jiraPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.items = m.Data.([]WorkItem)
		p.hasData = true
	}
	return nil
}

// viewWorkItem fetches the full work item, writes it as a Markdown document
// to a temp file and opens it in the editor, mirroring fzfjira's view action.
func (p *jiraPanel) viewWorkItem(key string) tea.Cmd {
	flagField, editor := p.params.flagField, p.editor
	return func() tea.Msg {
		stdout, err := run(30*time.Second, "acli",
			"jira", "workitem", "view", key, "--fields", "*all", "--json")
		if err != nil {
			return nil
		}
		var wi jiraWorkItem
		if err := json.Unmarshal([]byte(stdout), &wi); err != nil {
			return nil
		}
		doc := emitMarkdown(wi, flagField)
		file := filepath.Join(os.TempDir(), "radar-jira-"+key+".md")
		if err := os.WriteFile(file, []byte(doc), 0o644); err != nil {
			return nil
		}
		return ui.ExecMsg{Cmd: execCmd(strings.Fields(editor), file)}
	}
}

func (p *jiraPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	moved, enter := p.list.Handle(key, len(p.items))
	if moved || len(p.items) == 0 {
		return nil
	}
	item := p.items[p.list.Clamp(len(p.items))]
	if enter {
		return p.viewWorkItem(item.Key)
	}
	switch key {
	case "o":
		go func() {
			_, _ = run(20*time.Second, "acli", "jira", "workitem", "view", item.Key, "--web")
		}()
	case "y":
		pbcopy(item.Key)
	}
	return nil
}

func (p *jiraPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.items) == 0 {
		content = line(w, dim("No work items"))
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.items))
		}
		rows := make([]string, len(p.items))
		for i, item := range p.items {
			rows[i] = row(w, i == selected,
				plain(pad(item.Key, 12)+" "),
				colored(item.StatusColor, pad(item.Status, 14)),
				plain(" "+item.Summary),
			)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
