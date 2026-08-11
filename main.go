// Command radar is a personal TUI dashboard for macOS.
package main

import (
	_ "embed"
	"flag"
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

	var (
		configPath string
		demoMode   bool
	)
	flag.StringVar(&configPath, "config", "",
		"Path to the YAML config file (defaults to $RADAR_CONFIG or ~/.config/radar/config.yaml).")
	flag.BoolVar(&demoMode, "demo", false,
		"Run with built-in demo data instead of loading a config file.")
	flag.Parse()

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
