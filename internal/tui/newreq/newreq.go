package newreq

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
)

// RequestCreatedMsg is sent when the user confirms a new request name.
type RequestCreatedMsg struct {
	Collection string
	Request    domain.Request
}

// CancelledMsg is sent when the user cancels the modal.
type CancelledMsg struct{}

// Model is a modal overlay for creating a new request.
type Model struct {
	visible    bool
	input      string
	collection string
}

// New returns a new, hidden Model.
func New() Model { return Model{} }

// Show makes the modal visible and sets the target collection.
func (m *Model) Show(collection string) {
	m.visible = true
	m.collection = collection
	m.input = ""
}

// Toggle flips the modal visibility.
func (m *Model) Toggle() { m.visible = !m.visible }

// Visible reports whether the modal is currently shown.
func (m Model) Visible() bool { return m.visible }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "escape":
			m.visible = false
			return m, func() tea.Msg { return CancelledMsg{} }
		case "enter":
			if m.input == "" {
				return m, nil
			}
			req := domain.Request{Name: m.input, Method: "GET"}
			col := m.collection
			m.visible = false
			m.input = ""
			return m, func() tea.Msg {
				return RequestCreatedMsg{Collection: col, Request: req}
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			key := msg.Key()
			if key.Text != "" {
				m.input += key.Text
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.visible {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	s := title.Render("New Request") + "\n\n"
	s += "Name: " + m.input + "\u2588\n\n"
	s += dim.Render("Enter to confirm, Esc to cancel")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
