package curlimport

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/curlparse"
	"github.com/erlandas/postmaniux/internal/domain"
)

type screen int

const (
	screenPaste  screen = iota // paste curl command
	screenSelect               // select collection
)

// ImportedMsg is sent when the user completes the import flow.
type ImportedMsg struct {
	Collection string
	Request    domain.Request
}

// CancelledMsg is sent when the user cancels.
type CancelledMsg struct{}

type Model struct {
	visible     bool
	screen      screen
	input       string
	collections []string
	cursor      int
	parseErr    string
	parsed      domain.Request
}

func New() Model { return Model{} }

func (m *Model) Show(collections []string) {
	m.visible = true
	m.screen = screenPaste
	m.input = ""
	m.collections = collections
	m.cursor = 0
	m.parseErr = ""
	m.parsed = domain.Request{}
}

func (m Model) Visible() bool { return m.visible }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.screen {
		case screenPaste:
			return m.updatePaste(msg)
		case screenSelect:
			return m.updateSelect(msg)
		}
	}
	return m, nil
}

func (m Model) updatePaste(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.visible = false
		return m, func() tea.Msg { return CancelledMsg{} }
	case "enter":
		if m.input == "" {
			return m, nil
		}
		req, err := curlparse.Parse(m.input)
		if err != nil || req.URL == "" {
			m.parseErr = "Invalid curl command"
			return m, nil
		}
		m.parsed = req
		m.parseErr = ""
		m.screen = screenSelect
		m.cursor = 0
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
			m.parseErr = ""
		}
	default:
		if t := msg.Key().Text; t != "" {
			m.input += t
			m.parseErr = ""
		}
	}
	return m, nil
}

func (m Model) updateSelect(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape":
		m.screen = screenPaste
	case "j", "down":
		if m.cursor < len(m.collections)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.collections) == 0 {
			return m, nil
		}
		col := m.collections[m.cursor]
		req := m.parsed
		m.visible = false
		return m, func() tea.Msg {
			return ImportedMsg{Collection: col, Request: req}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	var s string

	switch m.screen {
	case screenPaste:
		s = title.Render("Import curl") + "\n\n"
		// Show truncated input to keep modal compact
		display := m.input
		if len(display) > 60 {
			display = "..." + display[len(display)-57:]
		}
		s += "curl: " + display + "\u2588\n"
		if m.parseErr != "" {
			s += errStyle.Render(m.parseErr) + "\n"
		}
		s += "\n" + dim.Render("Paste curl command, Enter to continue")

	case screenSelect:
		s = title.Render("Select collection") + "\n\n"
		// Show parsed preview
		preview := dim.Render(m.parsed.Method+" "+truncate(m.parsed.URL, 50)) + "\n\n"
		s += preview
		if len(m.collections) == 0 {
			s += errStyle.Render("No collections found")
		} else {
			for i, name := range m.collections {
				prefix := "  "
				if i == m.cursor {
					prefix = cursorStyle.Render("> ")
				}
				s += prefix + name + "\n"
			}
		}
		s += "\n" + dim.Render("Enter to confirm, Esc to go back")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(66).
		Render(s)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

