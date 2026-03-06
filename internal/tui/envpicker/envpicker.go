package envpicker

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
)

type EnvSelectedMsg struct{ Env domain.Environment }
type DismissMsg struct{}
type EnvSavedMsg struct{ Env domain.Environment }

type screen int

const (
	screenList screen = iota
	screenEdit
)

type editMode int

const (
	editNone editMode = iota
	editKey
	editValue
)

type kvRow struct {
	Key   string
	Value string
}

type Model struct {
	envs    []domain.Environment
	cursor  int
	visible bool

	screen   screen
	editIdx  int
	kvRows   []kvRow
	kvCursor int
	editMode editMode
	editBuf  string
}

func New(envs []domain.Environment) Model {
	return Model{envs: envs}
}

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
		case "j", "down":
			if m.cursor < len(m.envs)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(m.envs) {
				env := m.envs[m.cursor]
				m.visible = false
				return m, func() tea.Msg { return EnvSelectedMsg{Env: env} }
			}
		case "escape", "ctrl+e":
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
	cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	s := title.Render("Switch Environment") + "\n\n"
	for i, env := range m.envs {
		prefix := "  "
		if i == m.cursor {
			prefix = cursor.Render("> ")
		}
		s += prefix + env.Name + "\n"
	}
	s += "\nEnter to select, Esc to cancel"

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}
