package panels

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ricoberger/radar/internal/ui"
)

// seg is a styled fragment of a list row.
type seg struct {
	text  string
	color string // named color, "" for the default foreground
	dim   bool
}

func plain(text string) seg         { return seg{text: text} }
func colored(c, text string) seg    { return seg{text: text, color: c} }
func dim(text string) seg           { return seg{text: text, dim: true} }
func dimColored(c, text string) seg { return seg{text: text, color: c, dim: true} }

// row renders a selectable list row: marker, styled segments (all bold when
// selected, like Ink's nested <Text bold>), truncated to width.
func row(width int, selected bool, segs ...seg) string {
	return rowWith(width, selected, false, segs...)
}

// rowWith is row with an extra dimAll flag applied to the whole row (used
// for suppressed alerts).
func rowWith(width int, selected, dimAll bool, segs ...seg) string {
	var b strings.Builder
	marker := "  "
	if selected {
		style := lipgloss.NewStyle().Foreground(ui.Mauve).Bold(true)
		if dimAll {
			style = style.Faint(true)
		}
		marker = style.Render("▌ ")
	}
	b.WriteString(marker)
	for _, s := range segs {
		style := lipgloss.NewStyle()
		if s.color != "" {
			style = style.Foreground(ui.NamedColor(s.color))
		}
		if s.dim || dimAll {
			style = style.Faint(true)
		}
		if selected {
			style = style.Bold(true)
		}
		b.WriteString(style.Render(s.text))
	}
	return ui.Truncate(b.String(), width)
}

// line renders a non-selectable row (headers etc.), truncated to width.
func line(width int, segs ...seg) string {
	var b strings.Builder
	for _, s := range segs {
		style := lipgloss.NewStyle()
		if s.color != "" {
			style = style.Foreground(ui.NamedColor(s.color))
		}
		if s.dim {
			style = style.Faint(true)
		}
		b.WriteString(style.Render(s.text))
	}
	return ui.Truncate(b.String(), width)
}

// padEnd pads s with spaces to at least n characters (TS String.padEnd).
func padEnd(s string, n int) string {
	if r := []rune(s); len(r) < n {
		return s + strings.Repeat(" ", n-len(r))
	}
	return s
}

// headerLine renders a bold (optionally colored) non-selectable header row.
func headerLine(width int, text, color string) string {
	style := lipgloss.NewStyle().Bold(true)
	if color != "" {
		style = style.Foreground(ui.NamedColor(color))
	}
	return ui.Truncate(style.Render(text), width)
}

// pad pads or slices s to exactly n characters (Ink app's pad()).
func pad(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s + strings.Repeat(" ", n-len(r))
}

// clean replaces tabs and newlines with spaces.
func clean(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}
