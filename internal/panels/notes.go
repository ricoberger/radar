package panels

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

type notesParams struct {
	dir string
	day string // today | yesterday
}

// DailyNote is the fetched note; Content is nil when the file is missing.
type DailyNote struct {
	Path    string
	Content *string
}

// notesRenderedMsg carries the markdown-rendered note lines.
type notesRenderedMsg struct {
	ID      string
	Lines   []string
	Content string
	Width   int
}

func (m notesRenderedMsg) PanelID() string { return m.ID }

func readNotesParams(params map[string]any) notesParams {
	day := "today"
	if params["day"] == "yesterday" {
		day = "yesterday"
	}
	return notesParams{
		dir: config.ExpandHome(strParam(params, "dir", "")),
		day: day,
	}
}

func validateRicobergerNotesParams(params map[string]any, trail string) error {
	dir, ok := params["dir"].(string)
	if !ok || strings.TrimSpace(dir) == "" {
		return errf(`%s: "params.dir" is required for ricoberger-notes panels`, trail)
	}
	if v, ok := params["day"]; ok && v != "today" && v != "yesterday" {
		return errf(`%s: "params.day" must be "today" or "yesterday"`, trail)
	}
	return nil
}

func ricobergerNotesTitle(params map[string]any) string {
	if params["day"] == "yesterday" {
		return "Daily Note · Yesterday"
	}
	return "Daily Note"
}

func noteDate(day string) time.Time {
	date := time.Now()
	if day == "yesterday" {
		date = date.AddDate(0, 0, -1)
	}
	return date
}

func notePath(dir string, date time.Time) string {
	year := date.Format("2006")
	month := date.Format("01")
	return filepath.Join(dir, year, month, date.Format("2006-01-02")+".md")
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n.*?\n---\n+`)

func stripFrontmatter(content string) string {
	return frontmatterRe.ReplaceAllString(content, "")
}

func fetchDailyNote(p notesParams) (DailyNote, error) {
	path := notePath(p.dir, noteDate(p.day))
	if demo.Enabled() {
		content := demoNote()
		return DailyNote{Path: path, Content: &content}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return DailyNote{Path: path, Content: nil}, nil
	}
	content := stripFrontmatter(string(raw))
	return DailyNote{Path: path, Content: &content}, nil
}

// ensureTodayNote creates today's note from template.md (if present) and
// returns its path.
func ensureTodayNote(dir string) string {
	target := notePath(dir, time.Now())
	if _, err := os.Stat(target); err == nil {
		return target
	}
	today := strings.TrimSuffix(filepath.Base(target), ".md")
	template := "# " + today + "\n"
	if raw, err := os.ReadFile(filepath.Join(dir, "template.md")); err == nil {
		template = strings.ReplaceAll(string(raw), "yyyy-mm-dd", today)
	}
	_ = os.MkdirAll(filepath.Dir(target), 0o755)
	_ = os.WriteFile(target, []byte(template), 0o644)
	return target
}

var ansiRe = regexp.MustCompile("\x1b\\[[0-9;]*m")

func trimBlankEdges(lines []string) []string {
	isBlank := func(line string) bool {
		return strings.TrimSpace(ansiRe.ReplaceAllString(line, "")) == ""
	}
	start, end := 0, len(lines)
	for start < end && isBlank(lines[start]) {
		start++
	}
	for end > start && isBlank(lines[end-1]) {
		end--
	}
	return lines[start:end]
}

type ricobergerNotesPanel struct {
	base
	params notesParams
	note   *DailyNote
	lines  []string
	// rendered tracks which (content, width) produced lines.
	renderedContent string
	renderedWidth   int
	offset          int
}

func newRicobergerNotesPanel(fp config.FlatPanel, editor string) *ricobergerNotesPanel {
	return &ricobergerNotesPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readNotesParams(fp.Params),
	}
}

func (p *ricobergerNotesPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		note, err := fetchDailyNote(params)
		return ui.FetchMsg{ID: id, Data: note, Err: err}
	}
}

// renderCmd renders the note content through the `md` binary (glamour),
// falling back to plain text when it is not installed or fails.
func (p *ricobergerNotesPanel) renderCmd() tea.Cmd {
	if p.note == nil || p.note.Content == nil {
		return nil
	}
	w, _ := p.contentSize()
	if w <= 0 {
		return nil
	}
	if p.renderedContent == *p.note.Content && p.renderedWidth == w {
		return nil
	}
	id, content, width := p.id, *p.note.Content, w
	return func() tea.Msg {
		out, err := runWithInput(20*time.Second, content,
			[]string{"MD_WORD_WRAP=" + itoa(width)}, "md")
		var lines []string
		if err != nil {
			lines = strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		} else {
			lines = trimBlankEdges(strings.Split(out, "\n"))
		}
		return notesRenderedMsg{ID: id, Lines: lines, Content: content, Width: width}
	}
}

func (p *ricobergerNotesPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	switch m := msg.(type) {
	case ui.FetchMsg:
		if p.applyMeta(m) {
			note := m.Data.(DailyNote)
			p.note = &note
			p.hasData = true
			return p.renderCmd()
		}
	case notesRenderedMsg:
		if p.note != nil && p.note.Content != nil &&
			m.Content == *p.note.Content {
			p.lines = m.Lines
			p.renderedContent = m.Content
			p.renderedWidth = m.Width
		}
	}
	return nil
}

func (p *ricobergerNotesPanel) Resize(width, height int) tea.Cmd {
	p.width, p.height = width, height
	return p.renderCmd()
}

func (p *ricobergerNotesPanel) maxOffset() int {
	_, h := p.contentSize()
	visible := max(1, h)
	return max(0, len(p.lines)-visible)
}

func (p *ricobergerNotesPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		var target string
		if p.params.day == "today" {
			target = ensureTodayNote(p.params.dir)
		} else if p.note != nil && p.note.Content != nil {
			target = p.note.Path
		}
		if target == "" {
			return nil
		}
		cmd := p.editorCmd(target)
		id := p.id
		return func() tea.Msg {
			return ui.ExecMsg{Cmd: cmd, After: ui.ForceRefreshMsg{ID: id}}
		}
	case "j", "down":
		p.offset = min(p.offset+1, p.maxOffset())
	case "k", "up":
		p.offset = max(p.offset-1, 0)
	case "g":
		p.offset = 0
	case "G":
		p.offset = p.maxOffset()
	}
	return nil
}

func (p *ricobergerNotesPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && p.note != nil && p.note.Content == nil {
		text := "No note for yesterday"
		if p.params.day == "today" {
			text = "No note for today — press enter to create it"
		}
		content = line(w, dim(text))
	} else if len(p.lines) > 0 {
		visible := max(1, h)
		offset := min(p.offset, p.maxOffset())
		endIndex := min(offset+visible, len(p.lines))
		out := make([]string, 0, endIndex-offset)
		for _, l := range p.lines[offset:endIndex] {
			out = append(out, ui.Truncate(l, w))
		}
		content = strings.Join(out, "\n")
	}
	return p.frame(content, focused)
}
