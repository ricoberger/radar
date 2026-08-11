// Package config loads and validates the radar YAML configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LayoutNode is either a panel node (Panel != "") or a split node with
// Children laid out in Direction.
type LayoutNode struct {
	Panel    string         `yaml:"panel"`
	Title    string         `yaml:"title"`
	Weight   float64        `yaml:"weight"`
	Interval int            `yaml:"interval"`
	Params   map[string]any `yaml:"params"`

	Direction string        `yaml:"direction"`
	Children  []*LayoutNode `yaml:"children"`
}

// IsPanel reports whether the node is a panel leaf.
func (n *LayoutNode) IsPanel() bool { return n.Panel != "" }

// EffectiveWeight returns the node weight, defaulting to 1.
func (n *LayoutNode) EffectiveWeight() float64 {
	if n.Weight > 0 {
		return n.Weight
	}
	return 1
}

// Dashboard is a named layout tree.
type Dashboard struct {
	Name   string      `yaml:"name"`
	Layout *LayoutNode `yaml:"layout"`
}

// Config is the root configuration.
type Config struct {
	Editor     string      `yaml:"editor"`
	Dashboards []Dashboard `yaml:"dashboards"`
}

// FlatPanel is a panel leaf resolved with defaults, in visual order.
type FlatPanel struct {
	ID       string
	Type     string
	Index    int
	Title    string
	Interval int
	Params   map[string]any
}

// PanelDefaults describes a panel type: default title and refresh interval,
// an optional title derived from params and an optional params validator.
type PanelDefaults struct {
	Title          string
	Interval       int
	DeriveTitle    func(params map[string]any) string
	ValidateParams func(params map[string]any, trail string) error
}

// Registry maps panel type names to their defaults.
type Registry map[string]PanelDefaults

func (r Registry) available() string {
	names := make([]string, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	// Stable order for error messages.
	for i := range names {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return strings.Join(names, ", ")
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "radar", "config.yaml")
}

func validateLayout(node *LayoutNode, trail string, reg Registry) error {
	if node == nil {
		return fmt.Errorf(
			`%s: a layout node needs either "panel" or a non-empty "children" list`,
			trail,
		)
	}
	if node.IsPanel() {
		defaults, ok := reg[node.Panel]
		if !ok {
			return fmt.Errorf(
				"%s: unknown panel type %q (available: %s)",
				trail, node.Panel, reg.available(),
			)
		}
		if defaults.ValidateParams != nil {
			params := node.Params
			if params == nil {
				params = map[string]any{}
			}
			return defaults.ValidateParams(params, trail)
		}
		return nil
	}
	if len(node.Children) == 0 {
		return fmt.Errorf(
			`%s: a layout node needs either "panel" or a non-empty "children" list`,
			trail,
		)
	}
	if node.Direction != "" && node.Direction != "row" && node.Direction != "column" {
		return fmt.Errorf(`%s: "direction" must be "row" or "column"`, trail)
	}
	for i, child := range node.Children {
		if err := validateLayout(child, fmt.Sprintf("%s.children[%d]", trail, i), reg); err != nil {
			return err
		}
	}
	return nil
}

// Load reads the config from argPath, $RADAR_CONFIG or the default path and
// validates it against the panel registry.
func Load(argPath string, reg Registry) (*Config, error) {
	configPath := argPath
	if configPath == "" {
		configPath = os.Getenv("RADAR_CONFIG")
	}
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	// The path is user-supplied by design (--config flag / $RADAR_CONFIG).
	raw, err := os.ReadFile(configPath) //nolint:gosec
	if err != nil {
		//nolint:staticcheck // User-facing CLI message, capitalized on purpose.
		return nil, fmt.Errorf(
			"No config file found at %s.\nCreate it or pass a path: radar --config <config.yaml>",
			configPath,
		)
	}

	cfg := &Config{Editor: defaultEditor()}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		//nolint:staticcheck // User-facing CLI message, capitalized on purpose.
		return nil, fmt.Errorf("Failed to parse %s: %s", configPath, err)
	}
	if cfg.Editor == "" {
		cfg.Editor = defaultEditor()
	}
	if len(cfg.Dashboards) == 0 {
		return nil, fmt.Errorf(`%s: "dashboards" must be a non-empty list`, configPath)
	}
	for i := range cfg.Dashboards {
		d := &cfg.Dashboards[i]
		if d.Layout == nil {
			return nil, fmt.Errorf(`dashboards[%d]: missing "layout"`, i)
		}
		if err := validateLayout(d.Layout, fmt.Sprintf("dashboards[%d].layout", i), reg); err != nil {
			return nil, err
		}
		if d.Name == "" {
			d.Name = fmt.Sprintf("Dashboard %d", i+1)
		}
	}
	return cfg, nil
}

func defaultEditor() string {
	if e := os.Getenv("EDITOR"); e != "" {
		return e
	}
	return "vim"
}

// FlattenPanels lists the panel leaves of a layout in visual order and
// resolves id, title and interval.
func FlattenPanels(node *LayoutNode, dashboardIndex int, reg Registry) []FlatPanel {
	var panels []FlatPanel
	var walk func(n *LayoutNode)
	walk = func(n *LayoutNode) {
		if n.IsPanel() {
			index := len(panels) + 1
			defaults := reg[n.Panel]
			params := n.Params
			if params == nil {
				params = map[string]any{}
			}
			title := n.Title
			if title == "" && defaults.DeriveTitle != nil {
				title = defaults.DeriveTitle(params)
			}
			if title == "" {
				title = defaults.Title
			}
			interval := n.Interval
			if interval == 0 {
				interval = defaults.Interval
			}
			panels = append(panels, FlatPanel{
				ID:       fmt.Sprintf("d%d:p%d:%s", dashboardIndex, index, n.Panel),
				Type:     n.Panel,
				Index:    index,
				Title:    title,
				Interval: interval,
				Params:   params,
			})
			return
		}
		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(node)
	return panels
}

// ExpandHome expands a leading ~/ to the user's home directory.
func ExpandHome(p string) string {
	if p == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}
