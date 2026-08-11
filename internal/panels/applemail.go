package panels

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ricoberger/radar/internal/config"
	"github.com/ricoberger/radar/internal/demo"
	"github.com/ricoberger/radar/internal/ui"
)

// MailMessage is one inbox message.
type MailMessage struct {
	Subject string `json:"subject"`
	Sender  string `json:"sender"`
	Date    string `json:"date"`
	ID      string `json:"id"`
	Account string `json:"account"`
}

type appleMailParams struct {
	Messages string   `json:"messages"`
	Limit    int      `json:"limit"`
	Include  []string `json:"include"`
	Exclude  []string `json:"exclude"`
}

func readAppleMailParams(params map[string]any) appleMailParams {
	messages := strParam(params, "messages", "unread")
	include, _ := strSliceParam(params, "include")
	exclude, _ := strSliceParam(params, "exclude")
	if include == nil {
		include = []string{}
	}
	if exclude == nil {
		exclude = []string{}
	}
	return appleMailParams{
		Messages: messages,
		Limit:    intParam(params, "limit", 10),
		Include:  include,
		Exclude:  exclude,
	}
}

func validateAppleMailParams(params map[string]any, trail string) error {
	if v, ok := params["messages"]; ok && v != "unread" && v != "all" {
		return errf(`%s: "messages" must be "unread" or "all"`, trail)
	}
	if v, ok := params["limit"]; ok {
		n, isInt := v.(int)
		if !isInt || n < 1 {
			return errf(`%s: "limit" must be a positive integer`, trail)
		}
	}
	for _, key := range []string{"include", "exclude"} {
		if _, ok := params[key]; ok {
			if _, isList := params[key].([]any); !isList {
				return errf(`%s: "%s" must be a list of account names`, trail, key)
			}
		}
	}
	return nil
}

// buildMailScript returns the JXA script reading the INBOX of every
// (matching) account with bulk property fetches - one Apple Event per
// property instead of five per message. If include and exclude are both set
// the filters are ignored and everything is fetched.
func buildMailScript(params appleMailParams) string {
	paramsJSON, _ := json.Marshal(params)
	return `
function run() {
  const params = ` + string(paramsJSON) + `;
  const Mail = Application("Mail");
  if (!Mail.running()) {
    return "NOT_RUNNING";
  }
  const useInclude = params.include.length > 0 && params.exclude.length === 0;
  const useExclude = params.exclude.length > 0 && params.include.length === 0;
  const out = [];
  for (const acct of Mail.accounts()) {
    const account = acct.name();
    if (useInclude && params.include.indexOf(account) === -1) continue;
    if (useExclude && params.exclude.indexOf(account) !== -1) continue;
    let msgs;
    try {
      msgs = acct.mailboxes.byName("INBOX").messages;
      if (params.messages === "unread") {
        msgs = msgs.whose({ readStatus: false });
      }
      const subjects = msgs.subject();
      const senders = msgs.sender();
      const dates = msgs.dateReceived();
      const ids = msgs.messageId();
      for (let i = 0; i < subjects.length; i++) {
        out.push({
          subject: subjects[i],
          sender: senders[i],
          date: dates[i],
          id: ids[i],
          account: account,
        });
      }
    } catch (error) {
      continue;
    }
  }
  out.sort((a, b) => b.date - a.date);
  return JSON.stringify(out.slice(0, params.limit));
}
`
}

func fetchMail(params appleMailParams) ([]MailMessage, error) {
	if demo.Enabled() {
		return demoMail(), nil
	}
	stdout, err := run(30*time.Second, "osascript",
		"-l", "JavaScript", "-e", buildMailScript(params))
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "NOT_RUNNING" {
		return nil, errf("Mail.app is not running")
	}
	var messages []MailMessage
	if err := json.Unmarshal([]byte(trimmed), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

func mailURL(message MailMessage) string {
	return "message://%3C" + url.QueryEscape(message.ID) + "%3E"
}

var senderNameRe = regexp.MustCompile(`^"?([^"<]+)"?\s*<`)

func senderName(sender string) string {
	if m := senderNameRe.FindStringSubmatch(sender); m != nil {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(sender)
}

type appleMailPanel struct {
	base
	params   appleMailParams
	messages []MailMessage
}

func newAppleMailPanel(fp config.FlatPanel, editor string) *appleMailPanel {
	return &appleMailPanel{
		base:   newBase(fp.ID, fp.Index, fp.Title, fp.Interval, editor),
		params: readAppleMailParams(fp.Params),
	}
}

func (p *appleMailPanel) Fetch() tea.Cmd {
	if p.inFlight {
		return nil
	}
	p.beginFetch()
	id, params := p.id, p.params
	return func() tea.Msg {
		messages, err := fetchMail(params)
		return ui.FetchMsg{ID: id, Data: messages, Err: err}
	}
}

func (p *appleMailPanel) Apply(msg ui.PanelMsg) tea.Cmd {
	if m, ok := msg.(ui.FetchMsg); ok && p.applyMeta(m) {
		p.messages = m.Data.([]MailMessage)
		p.hasData = true
	}
	return nil
}

func (p *appleMailPanel) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	_, enter := p.list.Handle(msg.String(), len(p.messages))
	if enter {
		openExternal(mailURL(p.messages[p.list.Clamp(len(p.messages))]))
	}
	return nil
}

func (p *appleMailPanel) View(focused bool) string {
	w, h := p.contentSize()
	content := ""
	if p.hasData && len(p.messages) == 0 {
		if p.params.Messages == "unread" {
			content = line(w, dimColored("green", "Inbox zero ✓"))
		} else {
			content = line(w, dim("No messages"))
		}
	} else {
		selected := -1
		if focused {
			selected = p.list.Clamp(len(p.messages))
		}
		rows := make([]string, len(p.messages))
		for i, message := range p.messages {
			age := "?"
			if t, err := time.Parse(time.RFC3339, message.Date); err == nil {
				age = ui.FormatAge(time.Since(t))
			}
			rows[i] = row(w, i == selected,
				colored("yellow", senderName(message.Sender)),
				dim(" · "),
				plain(message.Subject),
				dim(" · "+age+" · "+message.Account),
			)
		}
		content = ui.ListView(rows, selected, h, 0)
	}
	return p.frame(content, focused)
}
