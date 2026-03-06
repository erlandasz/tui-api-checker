package tree

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/erlandas/postmaniux/internal/domain"
)

// Why: each node is either a collection (folder) or a request (leaf).
// Keeping them in a flat list with depth simplifies cursor navigation.
type Node struct {
	Name       string
	IsFolder   bool
	Expanded   bool
	Depth      int
	Collection string
	Path       string // folder path, e.g. "Auth/Login". Empty for top-level collection folders.
	Request    *domain.Request
}

type Model struct {
	nodes   []Node
	cursor  int
	focused bool
	width   int
	height  int
}

// Why: messages for parent to know what was selected
type RequestSelectedMsg struct {
	Collection string
	Request    domain.Request
}

type NewRequestMsg struct {
	Collection string
}

func New(collections []domain.Collection) Model {
	var nodes []Node
	for _, col := range collections {
		nodes = append(nodes, Node{
			Name:       col.Name,
			IsFolder:   true,
			Depth:      0,
			Collection: col.Name,
		})
		for i := range col.Requests {
			nodes = append(nodes, Node{
				Name:       col.Requests[i].Name,
				IsFolder:   false,
				Depth:      1,
				Collection: col.Name,
				Request:    &col.Requests[i],
			})
		}
	}
	return Model{nodes: nodes}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) { m.focused = f }
func (m Model) Focused() bool      { return m.focused }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		visible := m.visibleNodes()
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			if m.cursor < len(visible) {
				node := visible[m.cursor]
				if node.IsFolder {
					m.toggleFolder(node.Collection, node.Path)
				} else if node.Request != nil {
					return m, func() tea.Msg {
						return RequestSelectedMsg{
							Collection: node.Collection,
							Request:    *node.Request,
						}
					}
				}
			}
		case "n":
			if m.cursor < len(visible) {
				node := visible[m.cursor]
				col := node.Collection
				return m, func() tea.Msg {
					return NewRequestMsg{Collection: col}
				}
			}
		}
	}
	return m, nil
}

func (m *Model) toggleFolder(collection, path string) {
	for i := range m.nodes {
		if m.nodes[i].IsFolder && m.nodes[i].Collection == collection && m.nodes[i].Path == path {
			m.nodes[i].Expanded = !m.nodes[i].Expanded
			break
		}
	}
}

// Why: visibleNodes filters to only show requests under expanded folders.
func (m Model) visibleNodes() []Node {
	var visible []Node
	showChildren := false
	for _, n := range m.nodes {
		if n.IsFolder {
			visible = append(visible, n)
			showChildren = n.Expanded
		} else if showChildren {
			visible = append(visible, n)
		}
	}
	return visible
}

func (m *Model) AddRequest(collection string, req domain.Request) {
	// Find insertion point: after last node of this collection
	insertIdx := -1
	folderIdx := -1
	for i, n := range m.nodes {
		if n.IsFolder && n.Collection == collection {
			folderIdx = i
			m.nodes[i].Expanded = true
			insertIdx = i + 1
		} else if folderIdx >= 0 && n.Collection == collection {
			insertIdx = i + 1
		} else if folderIdx >= 0 && n.Collection != collection {
			break
		}
	}
	if insertIdx < 0 {
		return
	}

	newNode := Node{
		Name:       req.Name,
		IsFolder:   false,
		Depth:      1,
		Collection: collection,
		Request:    &req,
	}

	// Insert node at position
	m.nodes = append(m.nodes, Node{})
	copy(m.nodes[insertIdx+1:], m.nodes[insertIdx:])
	m.nodes[insertIdx] = newNode

	// Move cursor to new node in visible list
	for i, n := range m.visibleNodes() {
		if !n.IsFolder && n.Request != nil && n.Request.Name == req.Name && n.Collection == collection {
			m.cursor = i
			break
		}
	}
}

func (m Model) View() string {
	visible := m.visibleNodes()

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s string
	s += titleStyle.Render("Collections") + "\n\n"

	for i, node := range visible {
		prefix := "  "
		if m.focused && i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}

		indent := ""
		for j := 0; j < node.Depth; j++ {
			indent += "  "
		}

		label := node.Name
		if node.IsFolder {
			arrow := ">"
			if node.Expanded {
				arrow = "v"
			}
			label = fmt.Sprintf("%s %s", arrow, node.Name)
		} else if node.Request != nil {
			method := dimStyle.Render(node.Request.Method)
			label = fmt.Sprintf("%s %s", method, node.Name)
		}

		s += prefix + indent + label + "\n"
	}

	return s
}
