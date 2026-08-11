package panels

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

const httpMonitorDefaultTimeout = 2 * time.Second

type httpTarget struct {
	name     string
	url      string
	method   string
	body     string
	hasBody  bool
	username string
	password string
	token    string
	insecure bool
	timeout  time.Duration
}

// CheckResult holds the phase durations of one check in milliseconds; a
// phase that did not happen (TLS on plain HTTP, or anything after the point
// of failure) stays nil.
type CheckResult struct {
	Name             string
	URL              string
	Status           int
	DNSLookup        *float64
	TCPConnection    *float64
	TLSHandshake     *float64
	ServerProcessing *float64
	ContentTransfer  *float64
	Total            *float64
}

func readHTTPTargets(params map[string]any) []httpTarget {
	raw, _ := params["targets"].([]any)
	targets := make([]httpTarget, 0, len(raw))
	for _, item := range raw {
		t, _ := item.(map[string]any)
		method := strParam(t, "method", "")
		if method == "" {
			method = "GET"
		}
		body, hasBody := t["body"].(string)
		timeout := httpMonitorDefaultTimeout
		switch v := t["timeout"].(type) {
		case int:
			if v > 0 {
				timeout = time.Duration(v) * time.Second
			}
		case float64:
			if v > 0 {
				timeout = time.Duration(v * float64(time.Second))
			}
		}
		targets = append(targets, httpTarget{
			name:     fmt.Sprint(t["name"]),
			url:      fmt.Sprint(t["url"]),
			method:   strings.ToUpper(method),
			body:     body,
			hasBody:  hasBody,
			username: strParam(t, "username", ""),
			password: strParam(t, "password", ""),
			token:    strParam(t, "token", ""),
			insecure: boolParam(t, "insecure", false),
			timeout:  timeout,
		})
	}
	return targets
}

var httpURLRe = regexp.MustCompile(`^https?://`)

func validateHttpMonitorParams(params map[string]any, trail string) error {
	targets, ok := params["targets"].([]any)
	if !ok || len(targets) == 0 {
		return errf(
			`%s: "params.targets" is required for httpmonitor panels and must be a non-empty list`,
			trail,
		)
	}
	for i, target := range targets {
		at := fmt.Sprintf(`%s: "params.targets[%d]`, trail, i)
		t, isMap := target.(map[string]any)
		if !isMap {
			return errf(`%s" must be an object`, at)
		}
		name, _ := t["name"].(string)
		if strings.TrimSpace(name) == "" {
			return errf(`%s.name" is required`, at)
		}
		url, _ := t["url"].(string)
		if !httpURLRe.MatchString(url) {
			return errf(`%s.url" is required and must be a http(s) URL`, at)
		}
		for _, field := range []string{"method", "body", "username", "password", "token"} {
			if v, ok := t[field]; ok {
				if _, isStr := v.(string); !isStr {
					return errf(`%s.%s" must be a string`, at, field)
				}
			}
		}
		if v, ok := t["insecure"]; ok {
			if _, isBool := v.(bool); !isBool {
				return errf(`%s.insecure" must be a boolean`, at)
			}
		}
		if v, ok := t["timeout"]; ok {
			valid := false
			switch n := v.(type) {
			case int:
				valid = n > 0
			case float64:
				valid = n > 0
			}
			if !valid {
				return errf(`%s.timeout" must be a positive number of seconds`, at)
			}
		}
	}
	return nil
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func allCipherSuites() []uint16 {
	var ids []uint16
	for _, s := range tls.CipherSuites() {
		ids = append(ids, s.ID)
	}
	for _, s := range tls.InsecureCipherSuites() {
		ids = append(ids, s.ID)
	}
	return ids
}

// checkTarget runs one check, timing the connection phases with httptrace.
// Every check uses a fresh connection, redirects are not followed and the
// response body is consumed so the content transfer time is measured. Never
// errors: failures return status 0 and the phases reached so far.
func checkTarget(target httpTarget) CheckResult {
	result := CheckResult{Name: target.name, URL: target.url, Status: 0}
	start := time.Now()
	var dnsDone, connectDone, requestSent, responseAt time.Time

	finishTotal := func() {
		total := ms(time.Since(start))
		result.Total = &total
	}

	ctx, cancel := context.WithTimeout(context.Background(), target.timeout)
	defer cancel()

	trace := &httptrace.ClientTrace{
		DNSDone: func(httptrace.DNSDoneInfo) {
			dnsDone = time.Now()
			d := ms(dnsDone.Sub(start))
			result.DNSLookup = &d
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				return
			}
			connectDone = time.Now()
			from := start
			if !dnsDone.IsZero() {
				from = dnsDone
			}
			d := ms(connectDone.Sub(from))
			result.TCPConnection = &d
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			from := start
			if !connectDone.IsZero() {
				from = connectDone
			}
			d := ms(time.Since(from))
			result.TLSHandshake = &d
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			requestSent = time.Now()
		},
	}

	var bodyReader io.Reader
	if target.hasBody {
		bodyReader = strings.NewReader(target.body)
	}
	req, err := http.NewRequestWithContext(
		httptrace.WithClientTrace(ctx, trace),
		target.method, target.url, bodyReader,
	)
	if err != nil {
		finishTotal()
		return result
	}
	if target.username != "" && target.password != "" {
		credentials := base64.StdEncoding.EncodeToString(
			[]byte(target.username + ":" + target.password))
		req.Header.Set("Authorization", "Basic "+credentials)
	} else if target.token != "" {
		req.Header.Set("Authorization", "Bearer "+target.token)
	}

	transport := &http.Transport{
		DisableKeepAlives: true,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: target.insecure,
			// Offer every cipher suite Go implements (including legacy
			// RSA-key-exchange suites Go excludes by default) so checks
			// succeed against old servers, matching Node's OpenSSL defaults.
			CipherSuites: allCipherSuites(),
		},
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		finishTotal()
		return result
	}
	responseAt = time.Now()
	result.Status = resp.StatusCode
	if !requestSent.IsZero() {
		d := ms(responseAt.Sub(requestSent))
		result.ServerProcessing = &d
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	d := ms(time.Since(responseAt))
	result.ContentTransfer = &d
	finishTotal()
	return result
}

func fetchChecks(targets []httpTarget) []CheckResult {
	if demo.Enabled() {
		return demoHttpMonitor()
	}
	results := make([]CheckResult, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Add(1)
		go func(i int, target httpTarget) {
			defer wg.Done()
			results[i] = checkTarget(target)
		}(i, target)
	}
	wg.Wait()
	return results
}

func httpStatusColor(status int) string {
	if status == 0 || status >= 500 {
		return "red"
	}
	if status >= 400 {
		return "yellow"
	}
	return "green"
}

func formatDuration(msValue *float64) string {
	if msValue == nil {
		return "-"
	}
	v := *msValue
	if v < 1000 {
		return fmt.Sprintf("%dms", int(v+0.5))
	}
	return fmt.Sprintf("%.2fs", v/1000)
}

var httpColumns = []struct {
	header string
	cell   func(CheckResult) string
}{
	{"Name", func(r CheckResult) string { return r.Name }},
	{"Status", func(r CheckResult) string { return itoa(r.Status) }},
	{"Total", func(r CheckResult) string { return formatDuration(r.Total) }},
	{"DNS Lookup", func(r CheckResult) string { return formatDuration(r.DNSLookup) }},
	{"TCP Connection", func(r CheckResult) string { return formatDuration(r.TCPConnection) }},
	{"TLS Handshake", func(r CheckResult) string { return formatDuration(r.TLSHandshake) }},
	{"Server Processing", func(r CheckResult) string { return formatDuration(r.ServerProcessing) }},
	{"Content Transfer", func(r CheckResult) string { return formatDuration(r.ContentTransfer) }},
}

type httpMonitorPanel struct {
	base
	targets []httpTarget
	results []CheckResult
}

func newHTTPMonitorPanel(fp config.FlatPanel, editor string) *httpMonitorPanel {
	return &httpMonitorPanel{
		base:    newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		targets: readHTTPTargets(fp.Params),
	}
}

func (p *httpMonitorPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, targets := p.id, p.targets
	return func() tea.Msg {
		return ui.FetchMsg{ID: id, Data: fetchChecks(targets)}
	}
}

func (p *httpMonitorPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.results = m.Data.([]CheckResult)
		p.hasData = true
	}
	return nil
}

func (p *httpMonitorPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, enter := p.list.Handle(msg.String(), len(p.results))
	if enter {
		openExternal(p.results[p.list.Clamp(len(p.results))].URL)
	}
	return nil
}

func (p *httpMonitorPanel) View(focused bool) string {
	w, h := p.contentSize()
	cells := make([][]string, len(p.results))
	for i, result := range p.results {
		cells[i] = make([]string, len(httpColumns))
		for j, column := range httpColumns {
			cells[i][j] = column.cell(result)
		}
	}
	widths := make([]int, len(httpColumns))
	for j, column := range httpColumns {
		widths[j] = len([]rune(column.header))
		for i := range cells {
			if l := len([]rune(cells[i][j])); l > widths[j] {
				widths[j] = l
			}
		}
	}
	headers := make([]string, len(httpColumns))
	for j, column := range httpColumns {
		headers[j] = column.header
	}
	header := line(w, dim("  "+formatTableRow(headers, widths)))

	selected := -1
	if focused {
		selected = p.list.Clamp(len(p.results))
	}
	rows := make([]string, len(p.results))
	for i, result := range p.results {
		rest := make([]string, 0, len(httpColumns)-2)
		for j := 2; j < len(httpColumns); j++ {
			if j == len(httpColumns)-1 {
				rest = append(rest, cells[i][j])
			} else {
				rest = append(rest, padEnd(cells[i][j], widths[j]))
			}
		}
		rows[i] = row(w, i == selected,
			plain(padEnd(cells[i][0], widths[0])+"   "),
			colored(httpStatusColor(result.Status), padEnd(cells[i][1], widths[1])),
			plain("   "+strings.Join(rest, "   ")),
		)
	}
	content := header
	if len(rows) > 0 {
		content += "\n" + ui.ListView(rows, selected, h, 1)
	}
	return p.frame(content, focused)
}
