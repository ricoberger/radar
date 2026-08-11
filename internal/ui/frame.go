package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// FrameState is the metadata shown in a panel frame header.
type FrameState struct {
	Index       int
	Title       string
	Loading     bool
	LastUpdated time.Time
	Err         error
}

// FormatAge renders a duration like Ink's formatAge: 42s, 5m, 3h, 2d.
func FormatAge(d time.Duration) string {
	s := int(d.Seconds())
	if s < 0 {
		s = 0
	}
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	h := m / 60
	if h < 24 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dd", h/24)
}

// Truncate cuts s to width columns, appending … when truncated (matches
// Ink's wrap="truncate" via cli-truncate).
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, "…")
}

// padTo pads or hard-clips a styled line to exactly width columns.
func padTo(s string, width int) string {
	w := ansi.StringWidth(s)
	if w > width {
		return ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", width-w)
}

// Frame renders a panel frame: rounded border, one space of horizontal
// padding, a header line ("{index} {title}" plus loading/error/age status)
// with a single-line bottom border, and the content clipped below it.
func Frame(width, height int, st FrameState, focused bool, content string) string {
	if width < 4 || height < 4 {
		return ""
	}
	innerW := width - 4 // border + paddingX on both sides
	var borderColor color.Color = Gray
	if focused {
		borderColor = Mauve
	}
	border := lipgloss.NewStyle().Foreground(borderColor)

	// Header: title left, status right (gap >= 1).
	var status []string
	if st.Loading && st.LastUpdated.IsZero() {
		status = append(status, Dim("…"))
	}
	if st.Err != nil {
		msg := strings.SplitN(strings.TrimSpace(st.Err.Error()), "\n", 2)[0]
		status = append(status, Colored("red", "! "+msg))
	} else if !st.LastUpdated.IsZero() {
		status = append(status, Dim(FormatAge(time.Since(st.LastUpdated))))
	}
	statusStr := strings.Join(status, " ")

	titleStyle := lipgloss.NewStyle().Bold(true)
	if focused {
		titleStyle = titleStyle.Foreground(Mauve)
	}
	title := titleStyle.Render(
		Truncate(fmt.Sprintf("%d %s", st.Index, st.Title), innerW),
	)
	titleW := ansi.StringWidth(title)
	statusMax := innerW - titleW - 1
	if ansi.StringWidth(statusStr) > statusMax {
		statusStr = Truncate(statusStr, statusMax)
	}
	header := title
	if gap := innerW - titleW - ansi.StringWidth(statusStr); gap > 0 && statusStr != "" {
		header += strings.Repeat(" ", gap) + statusStr
	}

	contentH := height - 4 // borders, header, header underline
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > contentH {
		contentLines = contentLines[:contentH]
	}

	pad := " "
	var b strings.Builder
	b.WriteString(border.Render("╭" + strings.Repeat("─", width-2) + "╮"))
	writeRow := func(line string) {
		b.WriteString("\n")
		b.WriteString(border.Render("│"))
		b.WriteString(pad + padTo(line, innerW) + pad)
		b.WriteString(border.Render("│"))
	}
	writeRow(header)
	writeRow(border.Render(strings.Repeat("─", innerW)))
	for i := 0; i < contentH; i++ {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		writeRow(line)
	}
	b.WriteString("\n")
	b.WriteString(border.Render("╰" + strings.Repeat("─", width-2) + "╯"))
	return b.String()
}
