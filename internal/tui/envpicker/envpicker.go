package envpicker

import (
	"fmt"

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

func (m *Model) enterEditScreen() {
	env := m.envs[m.cursor]
	m.editIdx = m.cursor
	m.screen = screenEdit
	m.kvRows = nil
	for k, v := range env.Variables {
		m.kvRows = append(m.kvRows, kvRow{Key: k, Value: v})
	}
	m.kvCursor = 0
	m.editMode = editNone
}

func (m *Model) buildEnvFromRows() domain.Environment {
	env := m.envs[m.editIdx]
	env.Variables = make(map[string]string)
	for _, row := range m.kvRows {
		if row.Key != "" {
			env.Variables[row.Key] = row.Value
		}
	}
	m.envs[m.editIdx] = env
	return env
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.screen == screenEdit {
			return m.updateEditScreen(msg)
		}
		return m.updateListScreen(msg)
	}
	return m, nil
}

func (m Model) updateListScreen(msg tea.KeyPressMsg) (Model, tea.Cmd) {
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
	case "e":
		if len(m.envs) > 0 {
			m.enterEditScreen()
		}
	case "escape", "ctrl+e":
		m.visible = false
		return m, func() tea.Msg { return DismissMsg{} }
	}
	return m, nil
}

func (m Model) updateEditScreen(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch m.editMode {
	case editKey:
		return m.updateEditKey(msg)
	case editValue:
		return m.updateEditValue(msg)
	default:
		return m.updateEditNone(msg)
	}
}

func (m Model) updateEditNone(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.kvCursor < len(m.kvRows)-1 {
			m.kvCursor++
		}
	case "k", "up":
		if m.kvCursor > 0 {
			m.kvCursor--
		}
	case "a":
		m.kvRows = append(m.kvRows, kvRow{})
		m.kvCursor = len(m.kvRows) - 1
		m.editMode = editKey
		m.editBuf = ""
	case "d":
		if len(m.kvRows) > 0 {
			m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
			if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
				m.kvCursor--
			}
		}
	case "e", "enter":
		if len(m.kvRows) > 0 {
			m.editMode = editKey
			m.editBuf = m.kvRows[m.kvCursor].Key
		}
	case "escape":
		env := m.buildEnvFromRows()
		m.screen = screenList
		return m, func() tea.Msg { return EnvSavedMsg{Env: env} }
	}
	return m, nil
}

func (m Model) updateEditKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.kvRows[m.kvCursor].Key = m.editBuf
		m.editMode = editValue
		m.editBuf = m.kvRows[m.kvCursor].Value
	case "escape":
		if m.kvRows[m.kvCursor].Key == "" && m.kvRows[m.kvCursor].Value == "" {
			m.kvRows = append(m.kvRows[:m.kvCursor], m.kvRows[m.kvCursor+1:]...)
			if m.kvCursor >= len(m.kvRows) && m.kvCursor > 0 {
				m.kvCursor--
			}
		}
		m.editMode = editNone
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

func (m Model) updateEditValue(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.kvRows[m.kvCursor].Value = m.editBuf
		m.editMode = editNone
	case "escape":
		m.editMode = editNone
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

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	var s string
	if m.screen == screenEdit {
		s = m.viewEditScreen()
	} else {
		s = m.viewListScreen()
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(s)
}

func (m Model) viewListScreen() string {
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
	s += "\nEnter select, e edit, Esc cancel"
	return s
}

func (m Model) viewEditScreen() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	envName := m.envs[m.editIdx].Name
	s := title.Render("Edit: "+envName) + "\n\n"

	if len(m.kvRows) == 0 {
		s += dim.Render("(empty)")
	} else {
		for i, row := range m.kvRows {
			prefix := "  "
			if i == m.kvCursor {
				prefix = cursorStyle.Render("> ")
			}
			if i == m.kvCursor && m.editMode == editKey {
				s += fmt.Sprintf("%s%s\u2588: %s", prefix, m.editBuf, row.Value)
			} else if i == m.kvCursor && m.editMode == editValue {
				s += fmt.Sprintf("%s%s: %s\u2588", prefix, row.Key, m.editBuf)
			} else {
				s += fmt.Sprintf("%s%s: %s", prefix, row.Key, row.Value)
			}
			s += "\n"
		}
	}

	s += "\na add, d delete, e edit, Esc save & back"
	return s
}
