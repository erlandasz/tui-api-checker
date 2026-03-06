package reqeditor

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
	"github.com/erlandas/postmaniux/internal/envmanager"
)

type editMode int

const (
	modeNone editMode = iota
	modeURL
	modeKVKey
	modeKVValue
	modeBody
)

type field int

const (
	fieldMethod field = iota
	fieldURL
	fieldContent
)

type kvRow struct {
	Key   string
	Value string
}

var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

type Tab int

const (
	TabHeaders Tab = iota
	TabParams
	TabBody
)

func (t Tab) String() string {
	switch t {
	case TabHeaders:
		return "Headers"
	case TabParams:
		return "Params"
	case TabBody:
		return "Body"
	}
	return ""
}

// SendRequestMsg tells the parent to execute this request.
type SendRequestMsg struct{ Request domain.Request }

type Model struct {
	request   domain.Request
	activeTab Tab
	focused   bool
	width     int
	height    int
	knownVars map[string]bool

	editMode      editMode
	activeField   field
	editBuf       string
	kvRows        []kvRow
	kvCursor      int
	bodyLines     []string
	bodyCursorRow int
	bodyCursorCol int
}

func New() Model {
	return Model{}
}

func (m *Model) SetRequest(r domain.Request) {
	m.request = r
	m.syncKVFromRequest()
	m.syncBodyFromRequest()
}
func (m *Model) SetEnvironment(envVars map[string]string) {
	m.knownVars = envmanager.KnownVars(envVars)
}
func (m *Model) SetSize(w, h int)            { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)           { m.focused = f }
func (m Model) Focused() bool                { return m.focused }
func (m Model) Editing() bool                { return m.editMode != modeNone }
func (m Model) Request() domain.Request      { return m.request }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle edit modes first
		if m.editMode != modeNone {
			return m.updateEditMode(msg)
		}

		// modeNone keybindings
		switch msg.String() {
		case "j", "down":
			if m.activeField == fieldContent && (m.activeTab == TabHeaders || m.activeTab == TabParams) {
				if m.kvCursor < len(m.kvRows)-1 {
					m.kvCursor++
				}
			} else if m.activeField < fieldContent {
				m.activeField++
			}
		case "k", "up":
			if m.activeField == fieldContent && (m.activeTab == TabHeaders || m.activeTab == TabParams) {
				if m.kvCursor > 0 {
					m.kvCursor--
				} else {
					m.activeField = fieldURL
				}
			} else if m.activeField > fieldMethod {
				m.activeField--
			}
		case "a":
			if m.activeField == fieldContent && (m.activeTab == TabHeaders || m.activeTab == TabParams) {
				m.kvRows = append(m.kvRows, kvRow{})
				m.kvCursor = len(m.kvRows) - 1
				m.editMode = modeKVKey
				m.editBuf = ""
			}
		case "d":
			if m.activeField == fieldContent && (m.activeTab == TabHeaders || m.activeTab == TabParams) {
				if len(m.kvRows) > 0 {
					m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
					if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
						m.kvCursor--
					}
					m.syncKVToRequest()
				}
			}
		case "e", "enter":
			if m.activeField == fieldContent && (m.activeTab == TabHeaders || m.activeTab == TabParams) {
				if len(m.kvRows) > 0 {
					m.editMode = modeKVKey
					m.editBuf = m.kvRows[m.kvCursor].Key
				}
			} else if m.activeField == fieldContent && m.activeTab == TabBody {
				m.editMode = modeBody
			} else if m.activeField == fieldURL {
				m.editMode = modeURL
				m.editBuf = m.request.URL
			}
		case "m":
			if m.activeField == fieldMethod {
				current := m.request.Method
				if current == "" {
					current = "GET"
				}
				next := httpMethods[0]
				for i, method := range httpMethods {
					if method == current {
						next = httpMethods[(i+1)%len(httpMethods)]
						break
					}
				}
				m.request.Method = next
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
			m.syncKVFromRequest()
			m.syncBodyFromRequest()
		case "shift+tab":
			m.activeTab = (m.activeTab + 2) % 3
			m.syncKVFromRequest()
			m.syncBodyFromRequest()
		case "ctrl+s":
			return m, func() tea.Msg {
				return SendRequestMsg{Request: m.request}
			}
		}
	}
	return m, nil
}

func (m Model) updateEditMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch m.editMode {
	case modeURL:
		return m.updateURLMode(msg)
	case modeKVKey:
		return m.updateKVKeyMode(msg)
	case modeKVValue:
		return m.updateKVValueMode(msg)
	case modeBody:
		return m.updateBodyMode(msg)
	}
	return m, nil
}

func (m Model) updateURLMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.request.URL = m.editBuf
		m.editMode = modeNone
	case "esc", "escape":
		m.editMode = modeNone
	case "backspace":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
	default:
		key := msg.Key()
		if key.Text != "" {
			m.editBuf += key.Text
		}
	}
	return m, nil
}

func (m Model) updateKVKeyMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.kvRows[m.kvCursor].Key = m.editBuf
		m.editMode = modeKVValue
		m.editBuf = m.kvRows[m.kvCursor].Value
	case "esc", "escape":
		if m.kvRows[m.kvCursor].Key == "" && m.kvRows[m.kvCursor].Value == "" {
			m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
			if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
				m.kvCursor--
			}
		}
		m.editMode = modeNone
	case "backspace":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
	default:
		key := msg.Key()
		if key.Text != "" {
			m.editBuf += key.Text
		}
	}
	return m, nil
}

func (m Model) updateKVValueMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.kvRows[m.kvCursor].Value = m.editBuf
		m.syncKVToRequest()
		m.editMode = modeNone
	case "esc", "escape":
		// Sync whatever we have (key was already saved in KVKey mode)
		m.syncKVToRequest()
		m.editMode = modeNone
	case "backspace":
		if len(m.editBuf) > 0 {
			m.editBuf = m.editBuf[:len(m.editBuf)-1]
		}
	default:
		key := msg.Key()
		if key.Text != "" {
			m.editBuf += key.Text
		}
	}
	return m, nil
}

func (m Model) updateBodyMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	// Ensure bodyLines always has at least one element
	if len(m.bodyLines) == 0 {
		m.bodyLines = []string{""}
	}

	switch msg.String() {
	case "esc", "escape":
		m.syncBodyToRequest()
		m.editMode = modeNone
	case "enter":
		line := m.bodyLines[m.bodyCursorRow]
		m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol]
		rest := line[m.bodyCursorCol:]
		newLines := make([]string, 0, len(m.bodyLines)+1)
		newLines = append(newLines, m.bodyLines[:m.bodyCursorRow+1]...)
		newLines = append(newLines, rest)
		if m.bodyCursorRow+1 < len(m.bodyLines) {
			newLines = append(newLines, m.bodyLines[m.bodyCursorRow+1:]...)
		}
		m.bodyLines = newLines
		m.bodyCursorRow++
		m.bodyCursorCol = 0
	case "backspace":
		if m.bodyCursorCol > 0 {
			line := m.bodyLines[m.bodyCursorRow]
			m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol-1] + line[m.bodyCursorCol:]
			m.bodyCursorCol--
		} else if m.bodyCursorRow > 0 {
			prev := m.bodyLines[m.bodyCursorRow-1]
			m.bodyCursorCol = len(prev)
			m.bodyLines[m.bodyCursorRow-1] = prev + m.bodyLines[m.bodyCursorRow]
			m.bodyLines = append(m.bodyLines[:m.bodyCursorRow], m.bodyLines[m.bodyCursorRow+1:]...)
			m.bodyCursorRow--
		}
	case "left":
		if m.bodyCursorCol > 0 {
			m.bodyCursorCol--
		}
	case "right":
		if m.bodyCursorCol < len(m.bodyLines[m.bodyCursorRow]) {
			m.bodyCursorCol++
		}
	case "up":
		if m.bodyCursorRow > 0 {
			m.bodyCursorRow--
			if m.bodyCursorCol > len(m.bodyLines[m.bodyCursorRow]) {
				m.bodyCursorCol = len(m.bodyLines[m.bodyCursorRow])
			}
		}
	case "down":
		if m.bodyCursorRow < len(m.bodyLines)-1 {
			m.bodyCursorRow++
			if m.bodyCursorCol > len(m.bodyLines[m.bodyCursorRow]) {
				m.bodyCursorCol = len(m.bodyLines[m.bodyCursorRow])
			}
		}
	default:
		key := msg.Key()
		if key.Text != "" {
			line := m.bodyLines[m.bodyCursorRow]
			m.bodyLines[m.bodyCursorRow] = line[:m.bodyCursorCol] + key.Text + line[m.bodyCursorCol:]
			m.bodyCursorCol += len(key.Text)
		}
	}
	return m, nil
}

func (m *Model) syncKVFromRequest() {
	m.kvRows = nil
	var src map[string]string
	switch m.activeTab {
	case TabHeaders:
		src = m.request.Headers
	case TabParams:
		src = m.request.Params
	}
	for k, v := range src {
		m.kvRows = append(m.kvRows, kvRow{Key: k, Value: v})
	}
	m.kvCursor = 0
}

func (m *Model) syncKVToRequest() {
	kv := make(map[string]string)
	for _, row := range m.kvRows {
		if row.Key != "" {
			kv[row.Key] = row.Value
		}
	}
	switch m.activeTab {
	case TabHeaders:
		m.request.Headers = kv
	case TabParams:
		m.request.Params = kv
	}
}

func (m *Model) syncBodyFromRequest() {
	if m.request.Body == "" {
		m.bodyLines = []string{""}
	} else {
		m.bodyLines = strings.Split(m.request.Body, "\n")
	}
	m.bodyCursorRow = 0
	m.bodyCursorCol = 0
}

func (m *Model) syncBodyToRequest() {
	m.request.Body = strings.Join(m.bodyLines, "\n")
}

func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	activeTabStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true)
	inactiveTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	methodStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))

	var s string

	// Method + URL line
	method := m.request.Method
	if method == "" {
		method = "GET"
	}
	s += titleStyle.Render("Request") + "\n"

	methodPrefix := "  "
	if m.focused && m.activeField == fieldMethod {
		methodPrefix = "> "
	}
	urlPrefix := "  "
	if m.focused && m.activeField == fieldURL {
		urlPrefix = "> "
	}

	s += methodPrefix + methodStyle.Render(method) + "\n"
	if m.editMode == modeURL {
		s += "> " + m.editBuf + "\u2588\n\n"
	} else {
		s += urlPrefix + m.highlightVars(m.request.URL) + "\n\n"
	}

	// Tab bar
	tabs := []Tab{TabHeaders, TabParams, TabBody}
	var tabLine []string
	for _, t := range tabs {
		if t == m.activeTab {
			tabLine = append(tabLine, activeTabStyle.Render(t.String()))
		} else {
			tabLine = append(tabLine, inactiveTabStyle.Render(t.String()))
		}
	}
	contentPrefix := "  "
	if m.focused && m.activeField == fieldContent {
		contentPrefix = "> "
	}
	s += contentPrefix + strings.Join(tabLine, "  |  ") + "\n\n"

	// Tab content
	switch m.activeTab {
	case TabHeaders, TabParams:
		s += m.renderKVRows()
	case TabBody:
		if m.editMode == modeBody {
			s += m.renderBodyEdit()
		} else {
			if m.request.Body != "" {
				s += m.highlightVars(m.request.Body)
			} else {
				s += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(no body)")
			}
		}
	}

	return s
}

func (m Model) renderKVRows() string {
	if len(m.kvRows) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(empty)")
	}
	var lines []string
	for i, row := range m.kvRows {
		prefix := "  "
		if m.focused && m.activeField == fieldContent && i == m.kvCursor {
			prefix = "> "
		}
		if i == m.kvCursor && m.editMode == modeKVKey {
			lines = append(lines, fmt.Sprintf("%s%s\u2588: %s", prefix, m.editBuf, m.highlightVars(row.Value)))
		} else if i == m.kvCursor && m.editMode == modeKVValue {
			lines = append(lines, fmt.Sprintf("%s%s: %s\u2588", prefix, row.Key, m.editBuf))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, row.Key, m.highlightVars(row.Value)))
		}
	}
	return strings.Join(lines, "\n")
}

var varPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func (m Model) highlightVars(s string) string {
	goodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	badStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	return varPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from {{name}}
		name := match[2 : len(match)-2]
		if m.knownVars[name] {
			return goodStyle.Render(match)
		}
		return badStyle.Render(match)
	})
}

func (m Model) renderBodyEdit() string {
	bodyLines := m.bodyLines
	if len(bodyLines) == 0 {
		bodyLines = []string{""}
	}
	var lines []string
	for i, line := range bodyLines {
		if i == m.bodyCursorRow {
			col := m.bodyCursorCol
			if col > len(line) {
				col = len(line)
			}
			lines = append(lines, line[:col]+"\u2588"+line[col:])
		} else {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
