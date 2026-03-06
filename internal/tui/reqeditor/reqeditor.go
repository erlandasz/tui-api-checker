package reqeditor

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
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

func (m *Model) SetRequest(r domain.Request) { m.request = r }
func (m *Model) SetSize(w, h int)            { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)           { m.focused = f }
func (m Model) Focused() bool                { return m.focused }
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
			if m.activeField < fieldContent {
				m.activeField++
			}
		case "k", "up":
			if m.activeField > fieldMethod {
				m.activeField--
			}
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
		case "shift+tab":
			m.activeTab = (m.activeTab + 2) % 3
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
	}
	return m, nil
}

func (m Model) updateURLMode(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	return m, nil
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
	s += urlPrefix + m.request.URL + "\n\n"

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
	case TabHeaders:
		s += m.renderMap(m.request.Headers)
	case TabParams:
		s += m.renderMap(m.request.Params)
	case TabBody:
		if m.request.Body != "" {
			s += m.request.Body
		} else {
			s += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(no body)")
		}
	}

	return s
}

func (m Model) renderMap(kv map[string]string) string {
	if len(kv) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("(empty)")
	}
	var lines []string
	for k, v := range kv {
		lines = append(lines, fmt.Sprintf("  %s: %s", k, v))
	}
	return strings.Join(lines, "\n")
}
