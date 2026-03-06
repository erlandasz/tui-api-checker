package reqeditor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/ratatuile/internal/domain"
	"github.com/erlandas/ratatuile/internal/envmanager"
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

// CopyAsCurlMsg tells the parent to copy this request as a curl command.
type CopyAsCurlMsg struct{ Request domain.Request }

// SaveRequestMsg tells the parent to persist this request to disk.
type SaveRequestMsg struct {
	Collection string
	Request    domain.Request
}

type Model struct {
	request    domain.Request
	collection string
	activeTab  Tab
	focused    bool
	width      int
	height     int
	knownVars  map[string]bool

	editMode      editMode
	activeField   field
	editBuf       string
	editCursor    int
	acMatches     []string
	acCursor      int
	kvRows        []kvRow
	kvCursor      int
	bodyLines     []string
	bodyCursorRow int
	bodyCursorCol int
}

func New() Model {
	return Model{}
}

func (m *Model) SetRequest(collection string, r domain.Request) {
	m.collection = collection
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
				m.editCursor = 0
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
					m.editCursor = len(m.editBuf)
				}
			} else if m.activeField == fieldContent && m.activeTab == TabBody {
				m.editMode = modeBody
			} else if m.activeField == fieldURL {
				m.editMode = modeURL
				m.editBuf = m.request.URL
				m.editCursor = len(m.editBuf)
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
			col := m.collection
			req := m.request
			return m, func() tea.Msg {
				return SaveRequestMsg{Collection: col, Request: req}
			}
		case "ctrl+enter":
			return m, func() tea.Msg {
				return SendRequestMsg{Request: m.request}
			}
		case "ctrl+y":
			return m, func() tea.Msg {
				return CopyAsCurlMsg{Request: m.request}
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
	default:
		m.handleEditBufKey(msg)
	}
	return m, nil
}

func (m Model) updateKVKeyMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.kvRows[m.kvCursor].Key = m.editBuf
		m.editMode = modeKVValue
		m.editBuf = m.kvRows[m.kvCursor].Value
		m.editCursor = len(m.editBuf)
	case "esc", "escape":
		if m.kvRows[m.kvCursor].Key == "" && m.kvRows[m.kvCursor].Value == "" {
			m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
			if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
				m.kvCursor--
			}
		}
		m.editMode = modeNone
	default:
		m.handleEditBufKey(msg)
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
		m.syncKVToRequest()
		m.editMode = modeNone
	default:
		m.handleEditBufKey(msg)
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

func (m *Model) handleEditBufKey(msg tea.KeyPressMsg) {
	// Autocomplete navigation when dropdown is active
	if len(m.acMatches) > 0 {
		switch msg.String() {
		case "down":
			if m.acCursor < len(m.acMatches)-1 {
				m.acCursor++
			}
			return
		case "up":
			if m.acCursor > 0 {
				m.acCursor--
			}
			return
		case "tab":
			m.acComplete()
			return
		}
	}

	switch msg.String() {
	case "left":
		if m.editCursor > 0 {
			m.editCursor--
		}
	case "right":
		if m.editCursor < len(m.editBuf) {
			m.editCursor++
		}
	case "backspace":
		if m.editCursor > 0 {
			m.editBuf = m.editBuf[:m.editCursor-1] + m.editBuf[m.editCursor:]
			m.editCursor--
		}
	default:
		key := msg.Key()
		if key.Text != "" {
			m.editBuf = m.editBuf[:m.editCursor] + key.Text + m.editBuf[m.editCursor:]
			m.editCursor += len(key.Text)
		}
	}
	m.updateAutocomplete()
}

// updateAutocomplete checks if cursor is inside {{...}} and populates matches.
func (m *Model) updateAutocomplete() {
	m.acMatches = nil
	m.acCursor = 0

	// Find the last {{ before cursor
	before := m.editBuf[:m.editCursor]
	openIdx := strings.LastIndex(before, "{{")
	if openIdx < 0 {
		return
	}
	// Check there's no }} between {{ and cursor
	after := before[openIdx:]
	if strings.Contains(after, "}}") {
		return
	}
	// Extract partial variable name typed so far
	partial := before[openIdx+2:]

	var matches []string
	for name := range m.knownVars {
		if strings.HasPrefix(name, partial) {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	m.acMatches = matches
}

// acComplete inserts the selected autocomplete match into the edit buffer.
func (m *Model) acComplete() {
	if len(m.acMatches) == 0 {
		return
	}
	match := m.acMatches[m.acCursor]

	before := m.editBuf[:m.editCursor]
	openIdx := strings.LastIndex(before, "{{")
	if openIdx < 0 {
		return
	}
	// Replace from {{ to cursor with {{match}}
	replacement := "{{" + match + "}}"
	m.editBuf = m.editBuf[:openIdx] + replacement + m.editBuf[m.editCursor:]
	m.editCursor = openIdx + len(replacement)
	m.acMatches = nil
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
		s += "> " + m.renderEditBuf() + "\n"
		s += m.renderAutocomplete() + "\n"
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
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, m.renderEditBuf(), m.highlightVars(row.Value)))
			if ac := m.renderAutocomplete(); ac != "" {
				lines = append(lines, ac)
			}
		} else if i == m.kvCursor && m.editMode == modeKVValue {
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, row.Key, m.renderEditBuf()))
			if ac := m.renderAutocomplete(); ac != "" {
				lines = append(lines, ac)
			}
		} else {
			lines = append(lines, fmt.Sprintf("%s%s: %s", prefix, row.Key, m.highlightVars(row.Value)))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderEditBuf() string {
	c := m.editCursor
	if c > len(m.editBuf) {
		c = len(m.editBuf)
	}
	return m.editBuf[:c] + "\u2588" + m.editBuf[c:]
}

func (m Model) renderAutocomplete() string {
	if len(m.acMatches) == 0 {
		return ""
	}
	acStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("236"))
	acSelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("86"))

	var lines []string
	for i, match := range m.acMatches {
		style := acStyle
		if i == m.acCursor {
			style = acSelStyle
		}
		lines = append(lines, style.Render(" {{"+match+"}} "))
	}
	return "    " + strings.Join(lines, "\n    ") + "\n"
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
