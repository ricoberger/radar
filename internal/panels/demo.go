package panels

import (
	"fmt"
	"strings"
	"time"
)

// Fixture data for --demo mode, ported verbatim from the Ink app's demo.ts.

func minutesAgo(minutes int) string {
	return time.Now().Add(-time.Duration(minutes) * time.Minute).
		UTC().Format("2006-01-02T15:04:05.000Z")
}

func isoUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// demoCalendarEvents: times are relative to the given day (and to now for
// the running event) so the panel always shows a lively, current-looking
// schedule.
func demoCalendarEvents(days []time.Time) []CalendarEvent {
	at := func(day time.Time, hour, minute, duration int) (string, string) {
		start := time.Date(day.Year(), day.Month(), day.Day(),
			hour, minute, 0, 0, day.Location())
		end := start.Add(time.Duration(duration) * time.Minute)
		return isoUTC(start), isoUTC(end)
	}
	room := "Room 4.2"
	var events []CalendarEvent
	for _, day := range days {
		add := func(title, calendar string, start, end string, allDay bool, location *string) {
			events = append(events, CalendarEvent{
				Title: title, Calendar: calendar,
				Start: start, End: end, IsAllDay: allDay, Location: location,
			})
		}
		s, e := at(day, 0, 0, 24*60)
		add("On-call", "Work", s, e, true, nil)
		s, e = at(day, 9, 30, 15)
		add("Daily Standup", "Work", s, e, false, nil)
		s, e = at(day, 10, 0, 30)
		add("Incident Review INC-217", "Work", s, e, false, nil)
		add("Focus Time", "Work", minutesAgo(30), minutesAgo(-60), false, nil)
		s, e = at(day, 12, 30, 45)
		add("Lunch", "Private", s, e, false, nil)
		s, e = at(day, 14, 0, 30)
		add("1:1 with Dana", "Work", s, e, false, nil)
		s, e = at(day, 15, 0, 60)
		add("Platform Sync", "Work", s, e, false, &room)
		s, e = at(day, 16, 30, 45)
		add("Postgres 17 upgrade planning", "Work", s, e, false, nil)
		s, e = at(day, 18, 30, 90)
		add("Climbing", "Private", s, e, false, nil)
	}
	return events
}

func demoNote() string {
	note := `# {date}

## Todos

- [x] Review checkout-service deployment
- [x] Rotate the registry pull credentials
- [ ] Finish the incident report for INC-217
- [ ] Prepare the platform sync agenda
- [ ] Review DEMO-433 (rate limiting for the payments API)
- [ ] Schedule the Postgres 17 upgrade for stage-eu1

## Notes

The new ingress setup is live on **prod-eu1**. Latency dropped by ~15%,
error rate unchanged. Rollout to {bt}prod-us1{bt} is planned for tomorrow.

Standup: search-indexer image pulls are failing on prod-us1 since the
registry migration — hubot is on it, workaround is a manual pull.

## Links

- [Ingress rollout plan](https://example.com/ingress-rollout)
- [INC-217 timeline](https://example.com/inc-217)
`
	note = strings.ReplaceAll(note, "{bt}", "`")
	return strings.Replace(note, "{date}", time.Now().Format("2006-01-02"), 1)
}

func demoMail() []MailMessage {
	messages := []struct {
		sender  string
		subject string
		minutes int
		account string
	}{
		{"Grafana", "Alert: HighMemoryUsage resolved", 12, "Work"},
		{"Dana Miller", "Re: Platform sync agenda", 48, "Work"},
		{"GitHub", "acme/checkout-service: v2.4.0 released", 95, "Work"},
		{"Statuspage", "Scheduled maintenance on Saturday", 130, "Private"},
		{"Jira", "DEMO-421 was moved to In Progress", 180, "Work"},
		{"Alex Chen", "Lunch next week?", 240, "Private"},
		{"PagerDuty", "On-call handoff: you are next", 320, "Work"},
		{"Confluence", `Dana Miller edited "Ingress rollout plan"`, 410, "Work"},
		{"Newsletter", "This week in Kubernetes", 520, "Private"},
	}
	out := make([]MailMessage, len(messages))
	for i, m := range messages {
		out[i] = MailMessage{
			ID:      fmt.Sprintf("demo-%d", i),
			Sender:  m.sender,
			Subject: m.subject,
			Date:    minutesAgo(m.minutes),
			Account: m.account,
		}
	}
	return out
}

func demoAlerts() []Alert {
	alerts := []struct {
		name     string
		severity string
		state    string
		summary  string
		minutes  int
		exists   bool
	}{
		{"KubePodCrashLooping", "critical", "active", "Pod checkout/worker-6d9f is crash looping", 35, true},
		{"BlackboxProbeFailed", "critical", "active", "probe for https://status.acme.dev failed", 8, false},
		{"HighMemoryUsage", "error", "active", "payments-api memory above 90% for 15m", 140, false},
		{"KafkaConsumerLag", "error", "active", "checkout-events consumer lag above 100k messages", 75, false},
		{"KubeNodeNotReady", "error", "active", "node prod-us1-worker-3 is NotReady for 10m", 48, false},
		{"CertificateExpiresSoon", "warning", "active", "cert for api.acme.dev expires in 13 days", 2000, false},
		{"TargetDown", "warning", "active", "25% of search-indexer targets are down", 55, false},
		{"KubeHpaMaxedOut", "warning", "active", "HPA payments/payments-api at max replicas for 30m", 95, false},
		{"HighErrorRate", "warning", "active", "5xx rate on checkout-service above 1% for 10m", 22, false},
		{"PersistentVolumeFillingUp", "warning", "active", "PV postgres-data-2 will be full in 4 days", 430, false},
		{"KubeJobFailed", "warning", "active", "job search/reindex-29ln4 failed to complete", 190, false},
		{"PostgresReplicationLag", "info", "active", "replica lag on stage-eu1 is 42s", 260, false},
		{"CronJobSuspended", "info", "active", "cronjob checkout/cleanup has been suspended for 7d", 1600, false},
		{"DeploymentReplicasMismatch", "info", "suppressed", "search-indexer has 2/3 ready replicas", 600, false},
		{"NodeDiskPressure", "info", "suppressed", "node prod-eu1-worker-8 is under disk pressure", 780, false},
		{"WatchdogMissing", "info", "suppressed", "Watchdog alert from stage-eu1 was not received", 1100, false},
	}
	out := make([]Alert, len(alerts))
	for i, a := range alerts {
		out[i] = Alert{
			Fingerprint:       fmt.Sprintf("demo-%d", i),
			AmID:              "demo",
			Name:              a.name,
			Severity:          a.severity,
			State:             a.state,
			Summary:           a.summary,
			Alertmanager:      "prod-eu1",
			StartsAt:          minutesAgo(a.minutes),
			Markdown:          "# " + a.name + "\n\n" + a.summary + "\n",
			Actions:           map[string]string{"source": "https://example.com"},
			AnalysisAvailable: true,
			AnalysisExists:    a.exists,
			AnalysisRunning:   false,
		}
	}
	return out
}

func demoJira() []WorkItem {
	items := []struct {
		key, status, statusColor, summary string
	}{
		{"DEMO-421", "In Progress", "yellow", "Migrate checkout-service to the new ingress"},
		{"DEMO-433", "In Review", "yellow", "Add rate limiting to the payments API"},
		{"DEMO-429", "In Progress", "yellow", "Fix stale search results after reindexing"},
		{"DEMO-407", "Open", "blue", "Evaluate OpenTelemetry sampling strategies"},
		{"DEMO-402", "Open", "blue", "Add SLO dashboards for the checkout flow"},
		{"DEMO-395", "On Hold", "blue", "Upgrade PostgreSQL clusters to 17"},
		{"DEMO-390", "Done", "green", "Rotate the registry pull credentials"},
		{"DEMO-388", "Done", "green", "Document the on-call escalation policy"},
	}
	out := make([]WorkItem, len(items))
	for i, item := range items {
		out[i] = WorkItem{
			Key:         item.key,
			Status:      item.status,
			StatusColor: item.statusColor,
			Summary:     item.summary,
		}
	}
	return out
}

func demoKubectlIssues() kubectlData {
	columns := []string{"CONTEXT", "NAMESPACE", "NAME", "READY", "STATUS", "RESTARTS", "AGE"}
	rows := [][]string{
		{"prod-eu1", "checkout", "worker-6d9f7b5c4-x2x9p", "0/1", "CrashLoopBackOff", "212 (2m ago)", "3d"},
		{"prod-eu1", "checkout", "worker-6d9f7b5c4-p8wn4", "0/1", "CrashLoopBackOff", "198 (4m ago)", "3d"},
		{"prod-eu1", "payments", "payments-api-79c9b6-kx2lm", "1/1", "Running", "14 (35m ago)", "12d"},
		{"prod-eu1", "payments", "payments-api-79c9b6-r7d2s", "1/1", "Running", "11 (52m ago)", "12d"},
		{"prod-eu1", "ingress", "ingress-nginx-c8b5d-t7pqm", "1/1", "Running", "3 (2h ago)", "9d"},
		{"prod-eu1", "kafka", "kafka-broker-2", "1/2", "Running", "6 (1h ago)", "45d"},
		{"prod-us1", "search", "search-indexer-1", "0/1", "ImagePullBackOff", "0", "4h"},
		{"prod-us1", "search", "search-indexer-2", "0/1", "ImagePullBackOff", "0", "4h"},
		{"prod-us1", "monitoring", "node-exporter-w4kd9", "0/1", "Error", "7 (18m ago)", "31d"},
		{"prod-us1", "monitoring", "prometheus-1", "1/2", "OOMKilled", "4 (12m ago)", "31d"},
		{"prod-us1", "checkout", "checkout-api-5f7dd9-m3znq", "1/1", "Running", "2 (3h ago)", "6d"},
		{"prod-us1", "kube-system", "coredns-787d4b-b5msj", "1/1", "Running", "1 (6h ago)", "31d"},
		{"stage-eu1", "default", "load-test-jx8j2", "0/1", "Pending", "0", "25m"},
		{"stage-eu1", "checkout", "checkout-migrate-29ln4", "0/1", "Completed", "0", "2h"},
		{"stage-eu1", "search", "reindex-29ln4-x8kfx", "0/1", "Error", "0", "3h"},
		{"stage-eu1", "postgres", "postgres-2", "1/1", "Running", "5 (30m ago)", "18d"},
		{"dev-eu1", "checkout", "worker-8b4f2c1d9-q6wtr", "0/1", "ContainerCreating", "0", "2m"},
		{"dev-eu1", "sandbox", "debug-shell-mona", "1/1", "Running", "0", "5h"},
	}
	out := make([]map[string]string, len(rows))
	for i, cells := range rows {
		row := map[string]string{}
		for j, column := range columns {
			row[column] = cells[j]
		}
		out[i] = row
	}
	return kubectlData{columns: columns, rows: out}
}

func demoGithubSearch(kind string) []SearchItem {
	type item struct {
		number  int
		title   string
		repo    string
		author  string
		state   string
		isDraft bool
		minutes int
	}
	prs := []item{
		{512, "feat(api): add idempotency keys to checkout", "acme/checkout-service", "mona", "open", false, 90},
		{508, "fix(worker): retry failed webhook deliveries", "acme/checkout-service", "mona", "open", true, 400},
		{122, "chore(deps): bump opentelemetry to 1.32", "acme/payments-api", "renovate", "open", false, 1500},
		{119, "feat(metrics): expose queue depth per shard", "acme/payments-api", "mona", "open", false, 1900},
		{87, "docs: document the reindexing runbook", "acme/search-indexer", "mona", "open", true, 2400},
		{45, "feat(ui): add dark mode toggle", "acme/statuspage", "mona", "open", false, 3100},
	}
	issues := []item{
		{307, "Search results are stale after reindexing", "acme/search-indexer", "hubot", "open", false, 200},
		{301, "Add dark mode to the status dashboard", "acme/statuspage", "mona", "open", false, 900},
		{298, "Image pulls fail on prod-us1 since registry migration", "acme/search-indexer", "octocat", "open", false, 1400},
		{291, "Webhook deliveries are not retried on 5xx", "acme/checkout-service", "hubot", "open", false, 2000},
		{284, "Flaky test: TestCheckoutTimeout", "acme/checkout-service", "octocat", "open", false, 2600},
		{278, "Expose replication lag as a metric", "acme/payments-api", "mona", "open", false, 3300},
	}
	items := issues
	urlKind := "issues"
	if kind == "prs" {
		items = prs
		urlKind = "pull"
	}
	out := make([]SearchItem, len(items))
	for i, it := range items {
		out[i] = SearchItem{
			Number:     it.number,
			Title:      it.title,
			Repository: it.repo,
			Author:     it.author,
			State:      it.state,
			IsDraft:    it.isDraft,
			CreatedAt:  minutesAgo(it.minutes),
			URL: fmt.Sprintf("https://github.com/%s/%s/%d",
				it.repo, urlKind, it.number),
		}
	}
	return out
}

func demoNotifications() []Notification {
	items := []struct {
		isUnread   bool
		reason     string
		title      string
		nType      string
		repository string
		prState    string
		issueState string
		isDraft    bool
		conclusion string
		minutes    int
	}{
		{true, "review requested", "feat(api): add idempotency keys to checkout", "PullRequest", "acme/checkout-service", "OPEN", "", false, "", 25},
		{true, "mention", "Search results are stale after reindexing", "Issue", "acme/search-indexer", "", "OPEN", false, "", 130},
		{false, "subscribed", "fix(worker): retry failed webhook deliveries", "PullRequest", "acme/checkout-service", "MERGED", "", false, "", 300},
		{false, "subscribed", "v2.4.0", "Release", "acme/checkout-service", "", "", false, "", 700},
		{true, "assign", "Image pulls fail on prod-us1 since registry migration", "Issue", "acme/search-indexer", "", "OPEN", false, "", 850},
		{false, "author", "docs: document the reindexing runbook", "PullRequest", "acme/search-indexer", "OPEN", "", true, "", 1100},
		{false, "comment", "Expose replication lag as a metric", "Issue", "acme/payments-api", "", "CLOSED", false, "", 1300},
		{false, "ci activity", "Nightly build", "CheckSuite", "acme/payments-api", "", "", false, "FAILURE", 1500},
		{false, "state change", "feat(ui): add dark mode toggle", "PullRequest", "acme/statuspage", "CLOSED", "", false, "", 1800},
	}
	out := make([]Notification, len(items))
	for i, it := range items {
		out[i] = Notification{
			ID:         fmt.Sprintf("demo-%d", i),
			IsUnread:   it.isUnread,
			Reason:     it.reason,
			Title:      it.title,
			Type:       it.nType,
			Repository: it.repository,
			UpdatedAt:  minutesAgo(it.minutes),
			URL:        "https://github.com/" + it.repository,
			PRState:    it.prState,
			IssueState: it.issueState,
			IsDraft:    it.isDraft,
			Conclusion: it.conclusion,
		}
	}
	return out
}

func demoCopilotSessions(limit int, states map[string]bool) []CopilotSession {
	items := []struct {
		state   string
		name    string
		minutes int
	}{
		{"in_progress", "Add Copilot session panel", 1},
		{"idle", "Rightsize monitoring workloads", 12},
		{"idle", "Improve Alt+G keymap functionality", 47},
		{"queued", "Migrate config loader to viper", 63},
		{"completed", "feat(api): add idempotency keys to checkout", 180},
		{"failed", "Reimplement project in Go", 320},
		{"cancelled", "Investigate waypoint metrics scraping", 900},
	}
	out := make([]CopilotSession, 0, len(items))
	for i, it := range items {
		if !states[it.state] {
			continue
		}
		out = append(out, CopilotSession{
			ID:        fmt.Sprintf("demo-%d", i),
			Name:      it.name,
			State:     it.state,
			UpdatedAt: minutesAgo(it.minutes),
		})
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func demoHttpMonitor() []CheckResult {
	fp := func(v float64) *float64 { return &v }
	sum := func(vs ...*float64) *float64 {
		total := 0.0
		for _, v := range vs {
			if v != nil {
				total += *v
			}
		}
		return &total
	}
	type target struct {
		name, url string
		status    int
		phases    []*float64 // dns, tcp, tls, server, content
	}
	targets := []target{
		{"Website", "https://acme.com", 200, []*float64{fp(8), fp(12), fp(34), fp(87), fp(14)}},
		{"API", "https://api.acme.com/health", 200, []*float64{fp(5), fp(9), fp(28), fp(42), fp(3)}},
		{"Docs", "https://docs.acme.com", 200, []*float64{fp(11), fp(14), fp(41), fp(132), fp(27)}},
		{"Auth", "https://auth.acme.com/healthz", 204, []*float64{fp(6), fp(10), fp(31), fp(55), fp(1)}},
		{"Grafana", "https://grafana.acme.com", 302, []*float64{fp(7), fp(11), fp(36), fp(61), fp(2)}},
		{"Registry", "https://registry.acme.com/v2/", 401, []*float64{fp(9), fp(13), fp(39), fp(74), fp(4)}},
		{"Staging API", "https://api.staging.acme.com/health", 503, []*float64{fp(12), fp(18), fp(47), fp(812), fp(6)}},
		{"Legacy CRM", "https://crm.acme.com", 0, []*float64{fp(15), nil, nil, nil, nil}},
	}
	out := make([]CheckResult, len(targets))
	for i, t := range targets {
		total := sum(t.phases...)
		if t.status == 0 {
			total = fp(2000)
		}
		out[i] = CheckResult{
			Name:             t.name,
			URL:              t.url,
			Status:           t.status,
			DNSLookup:        t.phases[0],
			TCPConnection:    t.phases[1],
			TLSHandshake:     t.phases[2],
			ServerProcessing: t.phases[3],
			ContentTransfer:  t.phases[4],
			Total:            total,
		}
	}
	return out
}
