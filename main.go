// Command radar is a personal TUI dashboard for macOS.
package main

import (
	_ "embed"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/panels"
	"github.com/ricoberger/radar/internal/ui"
)

//go:embed swift/apple-calendar-helper.swift
var calendarHelperSource string

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}

func main() {
	panels.CalendarHelperSource = calendarHelperSource

	args := os.Args[1:]
	configPath := ""
	demoMode := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--demo":
			demoMode = true
		case "--config":
			i++
			if i >= len(args) || args[i] == "" {
				fail("Missing value for --config.")
			}
			configPath = args[i]
		default:
			fail(fmt.Sprintf(
				"Unknown argument: %s.\nUsage: radar [--config <config.yaml>] [--demo]",
				args[i],
			))
		}
	}

	registry := panels.Registry()
	var cfg *config.Config
	if demoMode {
		demo.Enable()
		cfg = demo.Config()
	} else {
		loaded, err := config.Load(configPath, registry)
		if err != nil {
			fail(err.Error())
		}
		cfg = loaded
	}

	dashboards := make([]ui.DashboardState, len(cfg.Dashboards))
	for i, d := range cfg.Dashboards {
		flat := config.FlattenPanels(d.Layout, i, registry)
		instances := make([]ui.Panel, len(flat))
		for j, fp := range flat {
			panel, err := panels.New(fp, cfg.Editor)
			if err != nil {
				fail(err.Error())
			}
			instances[j] = panel
		}
		dashboards[i] = ui.DashboardState{
			Name:   d.Name,
			Layout: d.Layout,
			Panels: instances,
		}
	}

	app := ui.NewApp(dashboards)
	if _, err := tea.NewProgram(app).Run(); err != nil {
		fail(err.Error())
	}
}
