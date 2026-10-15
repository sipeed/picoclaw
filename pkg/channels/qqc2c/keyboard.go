package qqc2c

import (
	"fmt"
	"net/url"
	"strings"
)

// ---------------------------------------------------------------------------
// keyboard (message buttons)
//
//	{ "keyboard": { "content": { "rows": [ { "buttons": [ {...} ] } ] } } }
//
// Platform limits: at most 5 rows, at most 5 buttons per row.
// action.type: 0 = open URL, 1 = callback (INTERACTION_CREATE), 2 = command
// (inserts "@bot <data>" into the input box).
// ---------------------------------------------------------------------------

const (
	keyboardMaxRows        = 5
	keyboardMaxButtonsPer  = 5
)

type Keyboard struct {
	Content *KeyboardContent `json:"content,omitempty"`
	ID      string           `json:"id,omitempty"` // template id (unused)
}

type KeyboardContent struct {
	Rows []KeyboardRow `json:"rows"`
}

type KeyboardRow struct {
	Buttons []Button `json:"buttons"`
}

type Button struct {
	ID         string       `json:"id,omitempty"`
	RenderData ButtonRender `json:"render_data"`
	Action     ButtonAction `json:"action"`
}

type ButtonRender struct {
	Label        string `json:"label"`
	VisitedLabel string `json:"visited_label"`
	Style        int    `json:"style"` // 0 grey outline, 1 blue outline
}

type ButtonAction struct {
	Type           int              `json:"type"` // 0 url, 1 callback, 2 command
	Permission     ButtonPermission `json:"permission"`
	Data           string           `json:"data"`
	Reply          bool             `json:"reply,omitempty"`
	Enter          bool             `json:"enter,omitempty"`
	UnsupportTips  string           `json:"unsupport_tips,omitempty"`
}

type ButtonPermission struct {
	Type           int      `json:"type"` // 0 specific users, 1 admins, 2 everyone
	SpecifyUserIDs []string `json:"specify_user_ids,omitempty"`
}

// CallbackButton builds a button that emits an INTERACTION_CREATE with `data`.
func CallbackButton(id, label, data string) Button {
	return Button{
		ID: id,
		RenderData: ButtonRender{Label: label, VisitedLabel: label, Style: 1},
		Action: ButtonAction{
			Type:          1,
			Permission:    ButtonPermission{Type: 2},
			Data:          data,
			UnsupportTips: "请升级QQ客户端以使用按钮",
		},
	}
}

// CommandButton builds a button that inserts "@bot <cmd>" and sends it.
func CommandButton(id, label, cmd string) Button {
	return Button{
		ID: id,
		RenderData: ButtonRender{Label: label, VisitedLabel: label, Style: 1},
		Action: ButtonAction{
			Type:          2,
			Permission:    ButtonPermission{Type: 2},
			Data:          cmd,
			Enter:         true,
			UnsupportTips: "请升级QQ客户端以使用指令按钮",
		},
	}
}

// BuildKeyboard assembles rows, silently trimming to the platform limits.
func BuildKeyboard(rows [][]Button) *Keyboard {
	out := &KeyboardContent{}
	for i, row := range rows {
		if i >= keyboardMaxRows {
			break
		}
		if len(row) > keyboardMaxButtonsPer {
			row = row[:keyboardMaxButtonsPer]
		}
		if len(row) == 0 {
			continue
		}
		out.Rows = append(out.Rows, KeyboardRow{Buttons: row})
	}
	if len(out.Rows) == 0 {
		return nil
	}
	return &Keyboard{Content: out}
}

// ---------------------------------------------------------------------------
// command panel (markdown embedded tags)
//
// Only markdown messages support these tags; `text` must be url-encoded and is
// limited to 100 characters.
// ---------------------------------------------------------------------------

// CmdEnterTag renders a clickable tag that sends `text` immediately.
func CmdEnterTag(text string) string {
	return fmt.Sprintf(`<qqbot-cmd-enter text="%s" />`, url.QueryEscape(text))
}

// CmdInputTag renders a clickable tag that inserts `text` into the input box.
func CmdInputTag(text, show string) string {
	if show == "" {
		show = text
	}
	return fmt.Sprintf(`<qqbot-cmd-input text="%s" show="%s" reference="false" />`,
		url.QueryEscape(text), url.QueryEscape(show))
}

// Command describes one entry of the command panel.
type Command struct {
	Name string // shown label
	Text string // inserted/sent text (usually "/name")
	Desc string // short description
}

// RenderCommandPanel renders the built-in capabilities as a markdown panel with
// clickable command tags, so users can drive PicoClaw from the phone keyboard.
func RenderCommandPanel(title string, cmds []Command) string {
	var b strings.Builder
	if title == "" {
		title = "可用指令"
	}
	b.WriteString("**" + title + "**\n\n")
	for _, cmd := range cmds {
		label := strings.TrimSpace(cmd.Name)
		if label == "" {
			label = cmd.Text
		}
		b.WriteString("- " + CmdEnterTag(cmd.Text) + "  " + cmd.Desc + "\n")
	}
	return b.String()
}
