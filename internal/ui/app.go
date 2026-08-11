package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ricoberger/radar/internal/config"
)

// DashboardState pairs a dashboard config with its panel instances (aligned
// with the config.FlattenPanels order).
type DashboardState struct {
	Name   string
	Layout *config.LayoutNode
	Panels []Panel
}

// App is the root Bubble Tea model.
type App struct {
	dashboards []DashboardState
	panelsByID map[string]Panel
	active     int
	focus      map[int]int // per-dashboard focused index (1-based)
	zoomed     bool
	width      int
	height     int
}

// NewApp builds the root model.
func NewApp(dashboards []DashboardState) *App {
	byID := map[string]Panel{}
	for _, d := range dashboards {
		for _, p := range d.Panels {
			byID[p.ID()] = p
		}
	}
	return &App{
		dashboards: dashboards,
		panelsByID: byID,
		focus:      map[int]int{},
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.tick(), tea.Batch(a.fetchDue()...))
}

func (a *App) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return heartbeatMsg(t)
	})
}

func (a *App) panels() []Panel {
	return a.dashboards[a.active].Panels
}

func (a *App) focusedIndex() int {
	n := len(a.panels())
	i, ok := a.focus[a.active]
	if !ok {
		i = 1
	}
	return min(i, n)
}

func (a *App) focusedPanel() Panel {
	return a.panels()[a.focusedIndex()-1]
}

// fetchDue collects fetch commands for the active dashboard's due panels.
func (a *App) fetchDue() []tea.Cmd {
	now := time.Now()
	var cmds []tea.Cmd
	for _, p := range a.panels() {
		if p.Due(now) {
			if cmd := p.Fetch(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

// chrome returns the number of lines used by the tab bar and the footer.
func (a *App) chrome() (tabs, footer int) {
	footer = 1
	if len(a.dashboards) > 1 {
		tabs = 1
	}
	return tabs, footer
}

// applyLayout recomputes the visible panels' sizes.
func (a *App) applyLayout() []tea.Cmd {
	if a.width <= 0 || a.height <= 0 {
		return nil
	}
	tabs, footer := a.chrome()
	h := a.height - tabs - footer
	var cmds []tea.Cmd
	if a.zoomed {
		if cmd := a.focusedPanel().Resize(a.width, h); cmd != nil {
			cmds = append(cmds, cmd)
		}
		return cmds
	}
	d := a.dashboards[a.active]
	counter := 0
	walkLayout(d.Layout, a.width, h, func(w, ph int) {
		if cmd := d.Panels[counter].Resize(w, ph); cmd != nil {
			cmds = append(cmds, cmd)
		}
		counter++
	})
	return cmds
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		return a, tea.Batch(a.applyLayout()...)
	case heartbeatMsg:
		cmds := append(a.fetchDue(), a.tick())
		return a, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		return a.handleKey(msg)
	case ForceRefreshMsg:
		if p, ok := a.panelsByID[msg.ID]; ok {
			return a, p.Fetch()
		}
		return a, nil
	case ExecMsg:
		after := msg.After
		return a, tea.ExecProcess(msg.Cmd, func(error) tea.Msg {
			return after
		})
	case PanelMsg:
		if p, ok := a.panelsByID[msg.PanelID()]; ok {
			return a, p.Apply(msg)
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch s {
	case "q", "ctrl+c":
		return a, tea.Quit
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		index := int(s[0] - '0')
		if index <= len(a.panels()) {
			a.focus[a.active] = index
			return a, tea.Batch(a.applyLayout()...)
		}
		return a, nil
	case "[", "]":
		if len(a.dashboards) > 1 {
			delta := 1
			if s == "[" {
				delta = -1
			}
			n := len(a.dashboards)
			a.active = (a.active + delta + n) % n
			a.zoomed = false
			for _, p := range a.panels() {
				p.Activate()
			}
			cmds := append(a.applyLayout(), a.fetchDue()...)
			return a, tea.Batch(cmds...)
		}
		return a, nil
	case "tab", "shift+tab":
		n := len(a.panels())
		prev := a.focusedIndex()
		if s == "shift+tab" {
			a.focus[a.active] = ((prev-2+n)%n + 1)
		} else {
			a.focus[a.active] = (prev % n) + 1
		}
		return a, tea.Batch(a.applyLayout()...)
	case "z":
		a.zoomed = !a.zoomed
		return a, tea.Batch(a.applyLayout()...)
	case "r":
		return a, a.focusedPanel().Fetch()
	case "R":
		var cmds []tea.Cmd
		for _, p := range a.panels() {
			if cmd := p.Fetch(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return a, tea.Batch(cmds...)
	}
	return a, a.focusedPanel().HandleKey(msg)
}

// splitSizes distributes total cells over children by weight using
// cumulative rounding, so the sum is exactly total.
func splitSizes(children []*config.LayoutNode, total int) []int {
	var sum float64
	for _, c := range children {
		sum += c.EffectiveWeight()
	}
	sizes := make([]int, len(children))
	acc := 0.0
	prev := 0
	for i, c := range children {
		acc += c.EffectiveWeight() / sum * float64(total)
		next := int(acc + 0.5)
		sizes[i] = next - prev
		prev = next
	}
	return sizes
}

// walkLayout traverses the layout tree assigning sizes to panel leaves in
// visual order.
func walkLayout(node *config.LayoutNode, w, h int, visit func(w, h int)) {
	if node.IsPanel() {
		visit(w, h)
		return
	}
	if node.Direction == "column" {
		sizes := splitSizes(node.Children, h)
		for i, child := range node.Children {
			walkLayout(child, w, sizes[i], visit)
		}
		return
	}
	sizes := splitSizes(node.Children, w)
	for i, child := range node.Children {
		walkLayout(child, sizes[i], h, visit)
	}
}

func (a *App) renderNode(node *config.LayoutNode, w, h int, counter *int, focusedIndex int) string {
	d := a.dashboards[a.active]
	if node.IsPanel() {
		i := *counter
		*counter = i + 1
		return d.Panels[i].View(i+1 == focusedIndex)
	}
	blocks := make([]string, len(node.Children))
	if node.Direction == "column" {
		sizes := splitSizes(node.Children, h)
		for i, child := range node.Children {
			blocks[i] = a.renderNode(child, w, sizes[i], counter, focusedIndex)
		}
		return lipgloss.JoinVertical(lipgloss.Left, blocks...)
	}
	sizes := splitSizes(node.Children, w)
	for i, child := range node.Children {
		blocks[i] = a.renderNode(child, sizes[i], h, counter, focusedIndex)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, blocks...)
}

func (a *App) View() tea.View {
	if a.width <= 0 || a.height <= 0 {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	tabs, footer := a.chrome()
	layoutH := a.height - tabs - footer

	var parts []string
	if tabs > 0 {
		var names []string
		for i, d := range a.dashboards {
			if i == a.active {
				names = append(names, lipgloss.NewStyle().
					Bold(true).Foreground(Mauve).Render(d.Name))
			} else {
				names = append(names, Dim(d.Name))
			}
		}
		parts = append(parts, " "+strings.Join(names, "  "))
	}

	if a.zoomed {
		parts = append(parts, a.focusedPanel().View(true))
	} else {
		counter := 0
		parts = append(parts, a.renderNode(
			a.dashboards[a.active].Layout, a.width, layoutH,
			&counter, a.focusedIndex(),
		))
	}

	footerText := ""
	if len(a.dashboards) > 1 {
		footerText = "[/] dashboard · "
	}
	footerText += fmt.Sprintf(
		"1-%d focus · tab cycle · j/k select · enter open · z zoom · r/R refresh · q quit",
		min(len(a.panels()), 9),
	)
	parts = append(parts, " "+Dim(Truncate(footerText, a.width-2)))

	v := tea.NewView(strings.Join(parts, "\n"))
	v.AltScreen = true
	return v
}
