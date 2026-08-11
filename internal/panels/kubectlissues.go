package panels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

type kubectlIssuesParams struct {
	command  string
	contexts []string
	args     []string
}

// kubectlData preserves the JSON key order of the first row (the plugin's
// JSON keys match the CLI table headers).
type kubectlData struct {
	columns []string
	rows    []map[string]string
}

func readKubectlIssuesParams(params map[string]any) kubectlIssuesParams {
	contexts, _ := strSliceParam(params, "contexts")
	args, _ := strSliceParam(params, "args")
	return kubectlIssuesParams{
		command:  strParam(params, "command", ""),
		contexts: contexts,
		args:     args,
	}
}

func validateKubectlStringList(params map[string]any, name, trail string) error {
	if _, ok := params[name]; !ok {
		return nil
	}
	if _, valid := strSliceParam(params, name); !valid {
		return errf(`%s: "params.%s" must be a list of strings`, trail, name)
	}
	if _, isList := params[name].([]any); !isList {
		return errf(`%s: "params.%s" must be a list of strings`, trail, name)
	}
	return nil
}

func validateKubectlIssuesParams(params map[string]any, trail string) error {
	command, ok := params["command"].(string)
	if !ok || strings.TrimSpace(command) == "" {
		return errf(`%s: "params.command" is required for kubectl-issues panels`, trail)
	}
	for _, name := range []string{"contexts", "args"} {
		if err := validateKubectlStringList(params, name, trail); err != nil {
			return err
		}
	}
	if args, ok := strSliceParam(params, "args"); ok {
		for _, arg := range args {
			if arg == "-o" || arg == "--output" {
				return errf(
					`%s: "params.args" must not contain "-o" / "--output", the panel always uses JSON`,
					trail,
				)
			}
		}
	}
	return nil
}

// parseOrderedRows decodes a JSON array of objects, returning the rows and
// the key order of the first object (Go maps do not preserve it).
func parseOrderedRows(data []byte) ([]map[string]string, []string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, nil, err
	}
	if tok == nil {
		return []map[string]string{}, nil, nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '[' {
		return nil, nil, fmt.Errorf("expected a JSON array")
	}
	var rows []map[string]string
	var columns []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, nil, err
		}
		if d, ok := tok.(json.Delim); !ok || d != '{' {
			return nil, nil, fmt.Errorf("expected a JSON object")
		}
		row := map[string]string{}
		for dec.More() {
			keyTok, err := dec.Token()
			if err != nil {
				return nil, nil, err
			}
			key := keyTok.(string)
			var value any
			if err := dec.Decode(&value); err != nil {
				return nil, nil, err
			}
			row[key] = fmt.Sprint(value)
			if len(rows) == 0 {
				columns = append(columns, key)
			}
		}
		if _, err := dec.Token(); err != nil { // closing }
			return nil, nil, err
		}
		rows = append(rows, row)
	}
	if rows == nil {
		rows = []map[string]string{}
	}
	return rows, columns, nil
}

func fetchKubectlIssues(p kubectlIssuesParams) (kubectlData, error) {
	if demo.Enabled() {
		return demoKubectlIssues(), nil
	}
	args := []string{"issues", p.command}
	for _, context := range p.contexts {
		args = append(args, "--context", context)
	}
	args = append(args, p.args...)
	args = append(args, "-o", "json")
	stdout, err := run(60*time.Second, "kubectl", args...)
	if err != nil {
		return kubectlData{}, err
	}
	if strings.TrimSpace(stdout) == "" {
		return kubectlData{rows: []map[string]string{}}, nil
	}
	rows, columns, err := parseOrderedRows([]byte(stdout))
	if err != nil {
		return kubectlData{}, err
	}
	return kubectlData{columns: columns, rows: rows}, nil
}

func kubectlColumnWidths(rows []map[string]string, names []string) []int {
	widths := make([]int, len(names))
	for i, name := range names {
		w := len([]rune(name))
		for _, row := range rows {
			if l := len([]rune(row[name])); l > w {
				w = l
			}
		}
		if i != len(names)-1 && w < 1 {
			w = 1
		}
		widths[i] = w
	}
	return widths
}

func formatTableRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if i == len(widths)-1 {
			parts[i] = cell
		} else {
			parts[i] = padEnd(cell, widths[i])
		}
	}
	return strings.Join(parts, "   ")
}

type kubectlIssuesPanel struct {
	base
	params kubectlIssuesParams
	data   kubectlData
}

func newKubectlIssuesPanel(fp config.FlatPanel, editor string) *kubectlIssuesPanel {
	return &kubectlIssuesPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readKubectlIssuesParams(fp.Params),
	}
}

func (p *kubectlIssuesPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		data, err := fetchKubectlIssues(params)
		return ui.FetchMsg{ID: id, Data: data, Err: err}
	}
}

func (p *kubectlIssuesPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.data = m.Data.(kubectlData)
		p.hasData = true
	}
	return nil
}

func (p *kubectlIssuesPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	p.list.Handle(msg.String(), len(p.data.rows))
	return nil
}

func (p *kubectlIssuesPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.data.rows) == 0 {
		content = line(w, dimColored("green", "No issues ✓"))
	} else if len(p.data.rows) > 0 {
		names := p.data.columns
		widths := kubectlColumnWidths(p.data.rows, names)
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.data.rows))
		}
		header := line(w, dim("  "+formatTableRow(names, widths)))
		rows := make([]string, len(p.data.rows))
		for i, r := range p.data.rows {
			cells := make([]string, len(names))
			for j, name := range names {
				cells[j] = r[name]
			}
			rows[i] = row(w, i == selected, plain(formatTableRow(cells, widths)))
		}
		content = header + "\n" + ui.ListView(rows, selected, h, 1)
	}
	return p.frame(content, focused)
}
