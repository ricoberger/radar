// Package ui contains the root Bubble Tea model, the layout engine and the
// shared rendering building blocks (panel frame, select list, styles).
package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Mauve is the Catppuccin Macchiato Mauve accent color used for the focused
// panel, the active dashboard and the selection marker (same as the fzf
// pointer).
var Mauve = lipgloss.Color("#c6a0f6")

// Named colors matching Ink's color names (ANSI 16 palette).
var (
	Red     = lipgloss.Red
	Green   = lipgloss.Green
	Yellow  = lipgloss.Yellow
	Blue    = lipgloss.Blue
	Magenta = lipgloss.Magenta
	White   = lipgloss.White
	Gray    = lipgloss.BrightBlack
)

// NamedColor resolves the Ink color names used by the panels to a color.
func NamedColor(name string) color.Color {
	switch name {
	case "red":
		return Red
	case "green":
		return Green
	case "yellow":
		return Yellow
	case "blue":
		return Blue
	case "magenta":
		return Magenta
	case "white":
		return White
	case "gray":
		return Gray
	default:
		return White
	}
}

// Dim renders s with the terminal's faint attribute, like Ink's dimColor.
func Dim(s string) string {
	return lipgloss.NewStyle().Faint(true).Render(s)
}

// Bold renders s bold.
func Bold(s string) string {
	return lipgloss.NewStyle().Bold(true).Render(s)
}

// Colored renders s in the given named color.
func Colored(name, s string) string {
	return lipgloss.NewStyle().Foreground(NamedColor(name)).Render(s)
}
