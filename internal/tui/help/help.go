package help

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type DismissMsg struct{}

type Model struct {
	visible bool
}

func New() Model { return Model{} }

func (m *Model) Toggle()      { m.visible = !m.visible }
func (m Model) Visible() bool { return m.visible }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "?", "escape":
			m.visible = false
			return m, func() tea.Msg { return DismissMsg{} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	key := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	s := title.Render("Keybindings") + "\n\n"
	bindings := [][2]string{
		{"Ctrl+W", "Cycle pane focus"},
		{"j/k", "Navigate up/down"},
		{"Enter", "Expand/select"},
		{"Tab", "Switch editor tab"},
		{"Ctrl+S", "Send request"},
		{"Ctrl+E", "Switch environment (e to edit vars)"},
		{"m", "Cycle HTTP method"},
		{"e / Enter", "Edit focused field"},
		{"a", "Add header/param"},
		{"d", "Delete header/param"},
		{"Esc", "Cancel / exit edit mode"},
		{"?", "Toggle this help"},
		{"n", "New request (in tree)"},
		{"q / Ctrl+C", "Quit"},
	}
	for _, b := range bindings {
		s += key.Render(b[0]) + "  " + dim.Render(b[1]) + "\n"
	}
	s += "\nPress ? or Esc to close"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
