package tree

import (
	"fmt"
	"strings"

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
			Path:       "",
		})

		// Track which sub-folder paths we've already created
		seen := make(map[string]bool)

		for i := range col.Requests {
			req := &col.Requests[i]
			parts := strings.Split(req.Name, " / ")

			// Create intermediate folder nodes for each prefix
			for j := 0; j < len(parts)-1; j++ {
				folderPath := strings.Join(parts[:j+1], "/")
				if seen[folderPath] {
					continue
				}
				seen[folderPath] = true
				nodes = append(nodes, Node{
					Name:       parts[j],
					IsFolder:   true,
					Depth:      j + 1,
					Collection: col.Name,
					Path:       folderPath,
				})
			}

			// Add the request leaf node with just the final name segment
			nodes = append(nodes, Node{
				Name:       parts[len(parts)-1],
				IsFolder:   false,
				Depth:      len(parts),
				Collection: col.Name,
				Path:       strings.Join(parts[:len(parts)-1], "/"),
				Request:    req,
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
	hideBelow := -1 // -1 means nothing hidden
	for _, n := range m.nodes {
		// If we're hiding and this node is deeper than the collapsed folder, skip it
		if hideBelow >= 0 && n.Depth > hideBelow {
			continue
		}
		// We've reached a node at same or lesser depth — stop hiding
		hideBelow = -1

		visible = append(visible, n)

		// If this is a collapsed folder, hide everything deeper
		if n.IsFolder && !n.Expanded {
			hideBelow = n.Depth
		}
	}
	return visible
}

func (m *Model) AddRequest(collection string, req domain.Request) {
	insertIdx := -1
	for i, n := range m.nodes {
		if n.Collection == collection {
			insertIdx = i + 1
			// Expand the top-level collection folder
			if n.IsFolder && n.Path == "" {
				m.nodes[i].Expanded = true
			}
		} else if insertIdx >= 0 {
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
		Path:       "",
		Request:    &req,
	}

	m.nodes = append(m.nodes, Node{})
	copy(m.nodes[insertIdx+1:], m.nodes[insertIdx:])
	m.nodes[insertIdx] = newNode

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
