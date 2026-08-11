package panels

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

// CalendarHelperSource is the Swift source of the EventKit helper, injected
// by main from the embedded swift/apple-calendar-helper.swift.
var CalendarHelperSource string

// CalendarEvent is one event reported by the helper.
type CalendarEvent struct {
	Title    string  `json:"title"`
	Calendar string  `json:"calendar"`
	Start    string  `json:"start"`
	End      string  `json:"end"`
	IsAllDay bool    `json:"isAllDay"`
	Location *string `json:"location"`
}

type calendarParams struct {
	day  string // yesterday | today | tomorrow
	view string // day | week
}

var calendarDayOffsets = map[string]int{"yesterday": -1, "today": 0, "tomorrow": 1}

func readCalendarParams(params map[string]any) calendarParams {
	return calendarParams{
		day:  strParam(params, "day", "today"),
		view: strParam(params, "view", "day"),
	}
}

func validateAppleCalendarParams(params map[string]any, trail string) error {
	if v, ok := params["day"]; ok {
		if _, valid := calendarDayOffsets[fmt.Sprint(v)]; !valid {
			return errf(`%s: "day" must be one of yesterday, today, tomorrow`, trail)
		}
	}
	if v, ok := params["view"]; ok && v != "day" && v != "week" {
		return errf(`%s: "view" must be "day" or "week"`, trail)
	}
	return nil
}

func appleCalendarTitle(params map[string]any) string {
	p := readCalendarParams(params)
	if p.view == "week" {
		return "Calendar · Week"
	}
	switch p.day {
	case "yesterday":
		return "Calendar · Yesterday"
	case "tomorrow":
		return "Calendar · Tomorrow"
	}
	return "Calendar · Today"
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func addDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

func localDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// viewDays returns the days shown for the configured view, computed at call
// time so a dashboard running past midnight stays current. Week =
// Monday-Sunday containing today.
func viewDays(p calendarParams) []time.Time {
	today := startOfDay(time.Now())
	if p.view == "week" {
		monday := addDays(today, -((int(today.Weekday()) + 6) % 7))
		days := make([]time.Time, 7)
		for i := range days {
			days[i] = addDays(monday, i)
		}
		return days
	}
	return []time.Time{addDays(today, calendarDayOffsets[p.day])}
}

var (
	helperCompileMu sync.Mutex
)

// calendarHelperPath returns the compiled EventKit helper, lazily building
// it with swiftc into the user cache directory, keyed by the source hash.
func calendarHelperPath() (string, error) {
	sum := sha256.Sum256([]byte(CalendarHelperSource))
	hash := hex.EncodeToString(sum[:])[:12]
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, "radar")
	bin := filepath.Join(dir, "apple-calendar-helper-"+hash)

	helperCompileMu.Lock()
	defer helperCompileMu.Unlock()
	if _, err := os.Stat(bin); err == nil {
		return bin, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	src := bin + ".swift"
	if err := os.WriteFile(src, []byte(CalendarHelperSource), 0o644); err != nil {
		return "", err
	}
	defer os.Remove(src)
	tmp := fmt.Sprintf("%s.tmp-%d", bin, os.Getpid())
	if _, err := run(120*time.Second, "swiftc", "-O", "-o", tmp, src); err != nil {
		return "", fmt.Errorf("compiling calendar helper: %s", err)
	}
	if err := os.Rename(tmp, bin); err != nil {
		return "", err
	}
	return bin, nil
}

func fetchCalendarEvents(days []time.Time) ([]CalendarEvent, error) {
	if demo.Enabled() {
		return demoCalendarEvents(days), nil
	}
	helper, err := calendarHelperPath()
	if err != nil {
		return nil, err
	}
	start := days[0]
	end := addDays(days[len(days)-1], 1)
	stdout, err := run(30*time.Second, helper, localDate(start), localDate(end))
	if err != nil {
		return nil, err
	}
	var events []CalendarEvent
	if err := json.Unmarshal([]byte(stdout), &events); err != nil {
		return nil, err
	}
	return events, nil
}

type calendarRow struct {
	kind  string // header | spacer | event
	day   time.Time
	event CalendarEvent
}

func parseISO(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.Local()
}

func overlapsDay(event CalendarEvent, day time.Time) bool {
	dayEnd := addDays(day, 1)
	return parseISO(event.Start).Before(dayEnd) && parseISO(event.End).After(day)
}

// buildCalendarRows: multi-day events repeat under every day they cover; the
// label reflects the part of the event that falls on that day.
func buildCalendarRows(events []CalendarEvent, days []time.Time) []calendarRow {
	if len(days) == 1 {
		var rows []calendarRow
		for _, event := range events {
			if overlapsDay(event, days[0]) {
				rows = append(rows, calendarRow{kind: "event", event: event, day: days[0]})
			}
		}
		return rows
	}
	var rows []calendarRow
	for i, day := range days {
		if i > 0 {
			rows = append(rows, calendarRow{kind: "spacer"})
		}
		rows = append(rows, calendarRow{kind: "header", day: day})
		for _, event := range events {
			if overlapsDay(event, day) {
				rows = append(rows, calendarRow{kind: "event", event: event, day: day})
			}
		}
	}
	return rows
}

func formatTime(t time.Time) string {
	return t.Format("15:04")
}

func timeLabel(event CalendarEvent, day time.Time) string {
	dayEnd := addDays(day, 1)
	start := parseISO(event.Start)
	end := parseISO(event.End)
	if event.IsAllDay || (!start.After(day) && !end.Before(dayEnd)) {
		return "all-day"
	}
	startsToday := !start.Before(day)
	endsToday := !end.After(dayEnd)
	if startsToday && endsToday {
		return formatTime(start) + " – " + formatTime(end)
	}
	if startsToday {
		return formatTime(start)
	}
	return "– " + formatTime(end)
}

// eventColor returns "blue" while the event is running.
func eventColor(event CalendarEvent, day time.Time) string {
	if timeLabel(event, day) == "all-day" {
		return ""
	}
	now := time.Now()
	start := parseISO(event.Start)
	end := parseISO(event.End)
	if !start.After(now) && !end.Before(now) {
		return "blue"
	}
	return ""
}

var germanWeekdays = [...]string{
	"Sonntag", "Montag", "Dienstag", "Mittwoch",
	"Donnerstag", "Freitag", "Samstag",
}

func dayHeading(day time.Time) string {
	return fmt.Sprintf("%s %02d.%02d.",
		germanWeekdays[int(day.Weekday())], day.Day(), int(day.Month()))
}

func sameDay(a, b time.Time) bool {
	return startOfDay(a).Equal(startOfDay(b))
}

type appleCalendarPanel struct {
	base
	params calendarParams
	events []CalendarEvent
}

func newAppleCalendarPanel(fp config.FlatPanel, editor string) *appleCalendarPanel {
	return &appleCalendarPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readCalendarParams(fp.Params),
	}
}

func (p *appleCalendarPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		events, err := fetchCalendarEvents(viewDays(params))
		return ui.FetchMsg{ID: id, Data: events, Err: err}
	}
}

func (p *appleCalendarPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.events = m.Data.([]CalendarEvent)
		p.hasData = true
	}
	return nil
}

func (p *appleCalendarPanel) rows() ([]calendarRow, []int) {
	rows := buildCalendarRows(p.events, viewDays(p.params))
	var eventRows []int
	for i, r := range rows {
		if r.kind == "event" {
			eventRows = append(eventRows, i)
		}
	}
	return rows, eventRows
}

func (p *appleCalendarPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, eventRows := p.rows()
	_, enter := p.list.Handle(msg.String(), len(eventRows))
	if enter {
		openExternal("ical://")
	}
	return nil
}

func (p *appleCalendarPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	rows, eventRows := p.rows()
	if p.hasData && len(rows) == 0 {
		content = line(w, dim("No events "+p.params.day))
	} else {
		selectedRow := -1
		if focused && len(eventRows) > 0 {
			selectedRow = eventRows[p.list.Clamp(len(eventRows))]
		}
		now := time.Now()
		rendered := make([]string, len(rows))
		for i, r := range rows {
			switch r.kind {
			case "spacer":
				rendered[i] = " "
			case "header":
				color := ""
				if sameDay(r.day, now) {
					color = "blue"
				}
				rendered[i] = headerLine(w, dayHeading(r.day), color)
			default:
				color := eventColor(r.event, r.day)
				rendered[i] = row(w, i == selectedRow,
					colored(color, padEnd(timeLabel(r.event, r.day), 14)+r.event.Title),
					dimColored(color, " · "+r.event.Calendar),
				)
			}
		}
		content = ui.ListView(rendered, selectedRow, h, 0)
	}
	return p.frame(content, focused)
}
