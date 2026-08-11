package panels

import (
	"fmt"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/ui"
)

// Registry returns the panel type registry: default title, refresh interval,
// derived titles and params validation per panel type.
func Registry() config.Registry {
	return config.Registry{
		"apple-calendar": {
			Title:          "Calendar",
			Interval:       300,
			DeriveTitle:    appleCalendarTitle,
			ValidateParams: validateAppleCalendarParams,
		},
		"ricoberger-notes": {
			Title:          "Daily Note",
			Interval:       300,
			DeriveTitle:    ricobergerNotesTitle,
			ValidateParams: validateRicobergerNotesParams,
		},
		"apple-mail": {
			Title:          "Mail",
			Interval:       300,
			ValidateParams: validateAppleMailParams,
		},
		"alertmanager": {
			Title:          "Alerts",
			Interval:       60,
			ValidateParams: validateAlertmanagerParams,
		},
		"github-prs": {
			Title:          "Pull Requests",
			Interval:       300,
			ValidateParams: validateGithubSearchParams("github-prs"),
		},
		"github-issues": {
			Title:          "Issues",
			Interval:       300,
			ValidateParams: validateGithubSearchParams("github-issues"),
		},
		"github-notifications": {
			Title:          "GitHub Notifications",
			Interval:       300,
			ValidateParams: validateGithubNotificationsParams,
		},
		"jira": {
			Title:          "Jira",
			Interval:       300,
			ValidateParams: validateJiraParams,
		},
		"kubectl-issues": {
			Title:          "Kubernetes Issues",
			Interval:       60,
			ValidateParams: validateKubectlIssuesParams,
		},
		"httpmonitor": {
			Title:          "HTTP Monitor",
			Interval:       10,
			ValidateParams: validateHttpMonitorParams,
		},
	}
}

// New builds the panel instance for a flattened panel config.
func New(fp config.FlatPanel, editor string) (ui.Panel, error) {
	switch fp.Type {
	case "apple-calendar":
		return newAppleCalendarPanel(fp, editor), nil
	case "ricoberger-notes":
		return newRicobergerNotesPanel(fp, editor), nil
	case "apple-mail":
		return newAppleMailPanel(fp, editor), nil
	case "alertmanager":
		return newAlertmanagerPanel(fp, editor), nil
	case "github-prs":
		return newGithubSearchPanel(fp, editor, "prs", iconPR, "No pull requests"), nil
	case "github-issues":
		return newGithubSearchPanel(fp, editor, "issues", iconIssue, "No issues"), nil
	case "github-notifications":
		return newGithubNotificationsPanel(fp, editor), nil
	case "jira":
		return newJiraPanel(fp, editor), nil
	case "kubectl-issues":
		return newKubectlIssuesPanel(fp, editor), nil
	case "httpmonitor":
		return newHTTPMonitorPanel(fp, editor), nil
	}
	return nil, fmt.Errorf("unknown panel type %q", fp.Type)
}

func validateGithubNotificationsParams(params map[string]any, trail string) error {
	return validateOpenParam(params, trail)
}
