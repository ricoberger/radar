package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var markerStyle = lipgloss.NewStyle().Foreground(Mauve)

// Marker returns the two-column selection marker prefix.
func Marker(selected bool) string {
	if selected {
		return markerStyle.Render("▌ ")
	}
	return "  "
}

// List tracks a persistent selection index (survives dashboard switches).
type List struct {
	Selected int
}

// Clamp returns the selection clamped to a list of length n.
func (l *List) Clamp(n int) int {
	i := l.Selected
	if i > n-1 {
		i = n - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

// Handle processes list navigation keys for a list of length n. It returns
// moved=true when the key changed the selection and enter=true for the
// enter key. Any other key is the caller's responsibility.
func (l *List) Handle(key string, n int) (moved, enter bool) {
	if n == 0 {
		return false, false
	}
	i := l.Clamp(n)
	switch key {
	case "j", "down":
		l.Selected = min(i+1, n-1)
		return true, false
	case "k", "up":
		l.Selected = max(i-1, 0)
		return true, false
	case "g":
		l.Selected = 0
		return true, false
	case "G":
		l.Selected = n - 1
		return true, false
	case "enter":
		l.Selected = i
		return false, true
	}
	return false, false
}

// Window computes the first visible row so the selection stays centered
// while scrolling (same math as the Ink SelectList).
func Window(length, selected, visible int) int {
	if length <= visible {
		return 0
	}
	start := selected - visible/2
	if start < 0 {
		start = 0
	}
	if start > length-visible {
		start = length - visible
	}
	return start
}

// ListView renders the visible window of rows for the given content height.
// reserve holds back lines used by the caller (e.g. a table header).
func ListView(rows []string, selected, height, reserve int) string {
	visible := max(1, height-reserve)
	start := Window(len(rows), selected, visible)
	end := min(start+visible, len(rows))
	if start >= end {
		return ""
	}
	return strings.Join(rows[start:end], "\n")
}
