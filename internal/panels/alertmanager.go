package panels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

const (
	alertmanagerDefaultURL = "http://127.0.0.1:9093"
	iconAnalyze            = "\uf0d0"
)

// Alert is one alert from the Alertmanager.app API.
type Alert struct {
	Fingerprint       string
	AmID              string
	Name              string
	Severity          string
	State             string
	Summary           string
	Alertmanager      string
	StartsAt          string
	Markdown          string
	Actions           map[string]string
	AnalysisAvailable bool
	AnalysisExists    bool
	AnalysisRunning   bool
}

type alertmanagerParams struct {
	url          string
	filter       string
	alertmanager string
}

func readAlertmanagerParams(params map[string]any) alertmanagerParams {
	return alertmanagerParams{
		url:          strings.TrimSuffix(strParam(params, "url", alertmanagerDefaultURL), "/"),
		filter:       strParam(params, "filter", ""),
		alertmanager: strParam(params, "alertmanager", ""),
	}
}

func validateAlertmanagerParams(params map[string]any, trail string) error {
	if v, ok := params["url"]; ok {
		if _, isStr := v.(string); !isStr {
			return errf(`%s: "params.url" must be a string`, trail)
		}
	}
	filter, _ := params["filter"].(string)
	alertmanager, _ := params["alertmanager"].(string)
	hasFilter := strings.TrimSpace(filter) != ""
	hasAlertmanager := strings.TrimSpace(alertmanager) != ""
	if hasFilter == hasAlertmanager {
		return errf(
			`%s: exactly one of "params.filter" or "params.alertmanager" is required for alertmanager panels`,
			trail,
		)
	}
	return nil
}

func getJSON(url string, timeout time.Duration, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// alertsPath resolves the configured filter / alertmanager name to its id
// and returns the path serving its alerts.
func alertsPath(p alertmanagerParams) (string, error) {
	kind, name := "alertmanagers", p.alertmanager
	if p.filter != "" {
		kind, name = "filters", p.filter
	}
	var sources []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := getJSON(p.url+"/api/"+kind, 5*time.Second, &sources); err != nil {
		return "", err
	}
	for _, s := range sources {
		if s.Name == name {
			return "/api/" + kind + "/" + s.ID + "/alerts", nil
		}
	}
	names := make([]string, 0, len(sources))
	for _, s := range sources {
		names = append(names, s.Name)
	}
	label := "alertmanager"
	if kind == "filters" {
		label = "filter"
	}
	return "", fmt.Errorf("%s %q not found (available: %s)",
		label, name, strings.Join(names, ", "))
}

func fetchAlerts(p alertmanagerParams) ([]Alert, error) {
	if demo.Enabled() {
		return demoAlerts(), nil
	}
	path, err := alertsPath(p)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		Fingerprint  string `json:"fingerprint"`
		AlertName    string `json:"alertName"`
		Severity     string `json:"severity"`
		State        string `json:"state"`
		Summary      string `json:"summary"`
		Markdown     string `json:"markdown"`
		Alertmanager struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"alertmanager"`
		Actions  map[string]*string `json:"actions"`
		Analysis struct {
			Available bool `json:"available"`
			Exists    bool `json:"exists"`
			Running   bool `json:"running"`
		} `json:"analysis"`
		Raw struct {
			StartsAt string `json:"startsAt"`
		} `json:"raw"`
	}
	if err := getJSON(p.url+path, 5*time.Second, &raw); err != nil {
		return nil, err
	}
	// Server order is preserved (like fzfalertmanager).
	alerts := make([]Alert, 0, len(raw))
	for _, a := range raw {
		severity := a.Severity
		if severity == "" {
			severity = "none"
		}
		state := a.State
		if state == "" {
			state = "active"
		}
		actions := map[string]string{}
		for k, v := range a.Actions {
			if v != nil && *v != "" {
				actions[k] = *v
			}
		}
		summary := collapseSpaces(a.Summary)
		alerts = append(alerts, Alert{
			Fingerprint:       a.Fingerprint,
			AmID:              a.Alertmanager.ID,
			Name:              a.AlertName,
			Severity:          severity,
			State:             state,
			Summary:           summary,
			Alertmanager:      a.Alertmanager.Name,
			StartsAt:          a.Raw.StartsAt,
			Markdown:          a.Markdown,
			Actions:           actions,
			AnalysisAvailable: a.Analysis.Available,
			AnalysisExists:    a.Analysis.Exists,
			AnalysisRunning:   a.Analysis.Running,
		})
	}
	return alerts, nil
}

func severityColor(severity string) string {
	switch severity {
	case "critical":
		return "magenta"
	case "error":
		return "red"
	case "warning":
		return "yellow"
	case "info":
		return "blue"
	default:
		return "gray"
	}
}

func analysisColor(alert Alert) string {
	if alert.AnalysisExists {
		return "green"
	}
	if alert.AnalysisRunning {
		return "yellow"
	}
	return "gray"
}

type alertmanagerPanel struct {
	base
	params alertmanagerParams
	alerts []Alert
}

func newAlertmanagerPanel(fp config.FlatPanel, editor string) *alertmanagerPanel {
	return &alertmanagerPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readAlertmanagerParams(fp.Params),
	}
}

func (p *alertmanagerPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		alerts, err := fetchAlerts(params)
		return ui.FetchMsg{ID: id, Data: alerts, Err: err}
	}
}

func (p *alertmanagerPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.alerts = m.Data.([]Alert)
		p.hasData = true
	}
	return nil
}

// viewMarkdown writes the alert markdown to a temp file and opens it in the
// editor, mirroring fzfalertmanager's view action.
func (p *alertmanagerPanel) viewMarkdown(alert Alert) tea.Cmd {
	file := filepath.Join(os.TempDir(), "radar-alert-"+alert.Fingerprint+".md")
	if err := os.WriteFile(file, []byte(alert.Markdown), 0o600); err != nil {
		return nil
	}
	cmd := p.editorCmd(file)
	return func() tea.Msg { return ui.ExecMsg{Cmd: cmd} }
}

// analyze opens the finished analysis in the editor, or starts a run when
// none exists yet; a running analysis is left alone (icon already shows it).
func (p *alertmanagerPanel) analyze(alert Alert) tea.Cmd {
	base := fmt.Sprintf("%s/api/alertmanagers/%s/alerts/%s",
		p.params.url, alert.AmID, alert.Fingerprint)
	if alert.AnalysisExists {
		editor := p.editor
		return func() tea.Msg {
			var result struct {
				FilePath string `json:"filePath"`
			}
			if err := getJSON(base+"/analysis", 5*time.Second, &result); err != nil {
				return nil
			}
			if result.FilePath == "" {
				return nil
			}
			if _, err := os.Stat(result.FilePath); err != nil {
				return nil
			}
			fields := strings.Fields(editor)
			cmd := execCmd(fields, result.FilePath)
			return ui.ExecMsg{Cmd: cmd}
		}
	}
	if alert.AnalysisRunning || !alert.AnalysisAvailable {
		return nil
	}
	id := p.id
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/analyze", nil)
		if err == nil {
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
		return ui.ForceRefreshMsg{ID: id}
	}
}

func (p *alertmanagerPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	moved, enter := p.list.Handle(key, len(p.alerts))
	if moved {
		return nil
	}
	if len(p.alerts) == 0 {
		return nil
	}
	alert := p.alerts[p.list.Clamp(len(p.alerts))]
	if enter {
		return p.viewMarkdown(alert)
	}
	switch key {
	case "o", "s", "b", "d", "p":
		links := map[string]string{
			"o": "source", "s": "silence", "b": "runbook",
			"d": "dashboard", "p": "panel",
		}
		if url := alert.Actions[links[key]]; url != "" {
			openExternal(url)
		}
	case "a":
		return p.analyze(alert)
	case "y":
		text := alert.Actions["source"]
		if text == "" {
			text = alert.Fingerprint
		}
		pbcopy(text)
	}
	return nil
}

func (p *alertmanagerPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.alerts) == 0 {
		content = line(w, dimColored("green", "No alerts ✓"))
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.alerts))
		}
		rows := make([]string, len(p.alerts))
		for i, alert := range p.alerts {
			age := "?"
			if alert.StartsAt != "" {
				if t, err := time.Parse(time.RFC3339, alert.StartsAt); err == nil {
					age = ui.FormatAge(time.Since(t))
				}
			}
			segs := []seg{
				colored(analysisColor(alert), iconAnalyze+" "),
				colored(severityColor(alert.Severity), padEnd(alert.Severity, 9)),
				plain(" " + alert.Name),
			}
			if alert.Summary != "" {
				segs = append(segs, dim(" "+alert.Summary))
			}
			segs = append(segs, dim("  "+age+"  "+alert.Alertmanager))
			rows[i] = rowWith(w, i == selected, alert.State == "suppressed", segs...)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
