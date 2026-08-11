// Package demo implements --demo mode: a built-in config and a module-level
// flag that makes every panel fetch return fake fixture data.
package demo

import (
	"os"

	"github.com/ricoberger/radar/internal/config"
)

var enabled = false

// Enable turns on demo mode.
func Enable() { enabled = true }

// Enabled reports whether demo mode is on.
func Enabled() bool { return enabled }

// Config returns the built-in demo configuration (same layout as the
// repository's config.yaml). Keymaps stay wired and may hit real tools.
func Config() *config.Config {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	return &config.Config{
		Editor: editor,
		Dashboards: []config.Dashboard{
			{
				Name: "Demo",
				Layout: &config.LayoutNode{
					Direction: "row",
					Children: []*config.LayoutNode{
						{
							Direction: "column",
							Weight:    2,
							Children: []*config.LayoutNode{
								{
									Direction: "column",
									Children: []*config.LayoutNode{
										{
											Panel: "httpmonitor",
											Params: map[string]any{
												"targets": []any{
													map[string]any{"name": "Website", "url": "https://acme.com"},
													map[string]any{"name": "API", "url": "https://api.acme.com/health"},
												},
											},
										},
										{
											Panel:  "alertmanager",
											Weight: 2,
											Params: map[string]any{"filter": "all"},
										},
									},
								},
								{
									Direction: "column",
									Children: []*config.LayoutNode{
										{
											Panel: "kubectl-issues",
											Params: map[string]any{
												"command":  "pods",
												"contexts": []any{"prod-eu1", "stage-eu1", "dev-eu1"},
												"args":     []any{"-A"},
											},
										},
									},
								},
							},
						},
						{
							Direction: "column",
							Weight:    2,
							Children: []*config.LayoutNode{
								{
									Direction: "column",
									Children: []*config.LayoutNode{
										{
											Panel:  "jira",
											Params: map[string]any{"jql": "assignee = currentUser()"},
										},
										{Panel: "github-notifications", Weight: 2},
									},
								},
								{
									Direction: "column",
									Children: []*config.LayoutNode{
										{
											Panel:  "github-prs",
											Params: map[string]any{"query": "author:@me is:open"},
										},
										{
											Panel:  "github-issues",
											Params: map[string]any{"query": "assignee:@me is:open"},
										},
									},
								},
							},
						},
						{
							Direction: "column",
							Children: []*config.LayoutNode{
								{Panel: "apple-calendar"},
								{
									Panel:  "ricoberger-notes",
									Weight: 2,
									Params: map[string]any{"dir": "~/notes"},
								},
								{Panel: "apple-mail"},
							},
						},
					},
				},
			},
		},
	}
}
